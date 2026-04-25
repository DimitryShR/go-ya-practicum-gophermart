package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/accrual"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/logger"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
)

const defaultBatchSize = 20

// Синхронизирует заказы с внешней системой начислений
type AccrualProcessor struct {
	repo         AccrualRepository
	client       AccrualClient
	pollInterval time.Duration
	batchSize    int
	concurrency  int
	jobs         chan string
	mu           sync.Mutex
	wg           sync.WaitGroup
	pauseUntil   time.Time
	pauseCond    *sync.Cond
	pauseTimer   *time.Timer
}

// Создает обработчик фонового обновления статусов заказов
func NewAccrualProcessor(repo AccrualRepository, client AccrualClient,
	pollInterval time.Duration, concurrency int) *AccrualProcessor {
	p := &AccrualProcessor{
		repo:         repo,
		client:       client,
		pollInterval: pollInterval,
		batchSize:    defaultBatchSize,
		concurrency:  concurrency,
	}
	p.pauseCond = sync.NewCond(&p.mu)
	return p
}

// Запускает бесконечный цикл опроса accrual-сервиса
func (p *AccrualProcessor) Run(ctx context.Context) {
	p.jobs = make(chan string, p.batchSize)

	// Запускаем воркеры
	p.wg.Add(p.concurrency)
	for i := 0; i < p.concurrency; i++ {
		go p.worker(ctx)
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		p.processBatch(ctx)

		select {
		// graceful shutdown
		case <-ctx.Done():
			p.mu.Lock()
			p.pauseUntil = time.Time{}
			// Останавливаем таймер паузы при выходе
			if p.pauseTimer != nil {
				p.pauseTimer.Stop()
				p.pauseTimer = nil
			}
			p.pauseCond.Broadcast() // Будим всех воркеров

			p.mu.Unlock()
			close(p.jobs)
			p.wg.Wait()
			return
		case <-ticker.C:
		}
	}
}

// Берет задачу из chan jobs и обрабатывает
//
// Если в процессе сталкивается с rateLimitError, то устанавливаем глобальную пазу
func (p *AccrualProcessor) worker(ctx context.Context) {
	defer p.wg.Done()

	for number := range p.jobs {
		// Проверяем на паузу до выполнения запроса
		p.mu.Lock()
		for p.isPausedLocked() {
			// Если пауза, ждем пробуждения
			p.pauseCond.Wait()
		}
		p.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := p.processOrder(ctx, number); err != nil {
			var rateLimitError accrual.RateLimitError
			if errors.As(err, &rateLimitError) {
				// Если получаем RateLimitError, то устанавливаем паузу
				p.setPauseUntil(time.Now().Add(rateLimitError.RetryAfter))
				logger.Log.Warn("accrual service rate limited requests",
					"order", number, "retry_after", rateLimitError.RetryAfter.String())
				// Откладываем обработку на следующий запрос
				continue
			}
			logger.Log.Warn("cannot sync order with accrual service", "order", number, "error", err)
		}
	}
}

// Обработка заказов пакетом для синхронизации с accrual сервисом
func (p *AccrualProcessor) processBatch(ctx context.Context) {
	if p.isPaused() {
		return
	}

	freeSlots := cap(p.jobs) - len(p.jobs)
	if freeSlots <= 0 {
		return
	}

	orders, err := p.repo.ListOrdersForProcessing(ctx, freeSlots)
	if err != nil {
		logger.Log.Error("cannot load orders for accrual sync", "error", err)
		return
	}

	for _, order := range orders {
		select {
		case p.jobs <- order.Number:
		case <-ctx.Done():
			return
		default:
			// Канал полон - ожидаем следующего тика
			return
		}
	}
}

// Обработка заказа для синхронизации с accrual сервисом
func (p *AccrualProcessor) processOrder(ctx context.Context, orderNumber string) error {
	var accrualOrder models.AccrualOrder
	err := withRetry(ctx, func() error {
		// Получаем состояние заказа в Accrual системе
		order, err := p.client.GetOrderAccrual(ctx, orderNumber)
		if err != nil {
			return err
		}

		accrualOrder = order
		return nil
	}, IsTemporaryError)

	if err != nil {
		if errors.Is(err, accrual.ErrOrderNotRegistered) {
			return nil
		}
		return err
	}

	// Конвертируем статус и делаем обновление в системе лояльности
	status := accrualOrder.Status.ToOrderStatus()
	return p.repo.ApplyAccrualResult(ctx, orderNumber, status, accrualOrder.Accrual)
}

// Проверяет, установлена ли пауза
//
// Не включает установку блокировки
func (p *AccrualProcessor) isPausedLocked() bool {
	return !p.pauseUntil.IsZero() && time.Now().Before(p.pauseUntil)
}

// Проверяет, установлена ли пауза
//
// Включает установку блокировки
func (p *AccrualProcessor) isPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isPausedLocked()
}

// Устанавливает паузу
func (p *AccrualProcessor) setPauseUntil(t time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Округляем, чтобы установить один раз
	// А не обновлять множество раз из-за пары миллисекунд
	rounded := t.Truncate(time.Second).Add(time.Second)

	if !p.pauseUntil.IsZero() && rounded.Before(p.pauseUntil) {
		return
	}

	// Отменяем предыдущий таймер, если он запущен
	if p.pauseTimer != nil {
		p.pauseTimer.Stop()
	}

	// Сохраняем новую отметку времени
	p.pauseUntil = rounded
	deadline := rounded

	// Запускаем новый таймер
	p.pauseTimer = time.AfterFunc(time.Until(deadline), func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		// Если до завершения таймера уже были созданы новые, то новые не сбрасываем
		if !p.pauseUntil.Equal(deadline) {
			return
		}

		p.pauseUntil = time.Time{}
		p.pauseTimer = nil
		p.pauseCond.Broadcast()
	})
}

// Возвращает true, если ошибка допускает повторную попытку запроса
func IsTemporaryError(err error) bool {
	var temporaryError accrual.TemporaryError
	return errors.As(err, &temporaryError)
}
