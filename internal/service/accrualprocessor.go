package service

import (
	"context"
	"errors"
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
	pauseUntil   time.Time
}

// Создает обработчик фонового обновления статусов заказов
func NewAccrualProcessor(repo AccrualRepository, client AccrualClient, pollInterval time.Duration) *AccrualProcessor {
	return &AccrualProcessor{
		repo:         repo,
		client:       client,
		pollInterval: pollInterval,
		batchSize:    defaultBatchSize,
	}
}

// Запускает бесконечный цикл опроса accrual-сервиса
func (p *AccrualProcessor) Run(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		p.processBatch(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Обработка заказов пакетом для синхронизации с accrual сервисом
func (p *AccrualProcessor) processBatch(ctx context.Context) {
	if !p.pauseUntil.IsZero() && time.Now().Before(p.pauseUntil) {
		return
	}

	orders, err := p.repo.ListOrdersForProcessing(ctx, p.batchSize)
	if err != nil {
		logger.Log.Error("cannot load orders for accrual sync", "error", err)
		return
	}

	// Хранит значение задержки максимальное для всего пакета заказов
	var RetryAfter time.Duration
	for _, order := range orders {
		if err := p.processOrder(ctx, order.Number); err != nil {
			// Обрабатываем ошибку rateLimitError и обновляем значения RetryAfter
			var rateLimitError accrual.RateLimitError
			if errors.As(err, &rateLimitError) {
				if rateLimitError.RetryAfter > RetryAfter {
					RetryAfter = rateLimitError.RetryAfter
				}
				logger.Log.Warn("accrual service rate limited requests", "order", order.Number, "retry_after", rateLimitError.RetryAfter.String())
				continue
			}
			logger.Log.Warn("cannot sync order with accrual service", "order", order.Number, "error", err)
		}
	}

	if RetryAfter > 0 {
		p.pauseUntil = time.Now().Add(RetryAfter)
	}
}

// Обработка заказа для синхронизации с accrual сервисом
func (p *AccrualProcessor) processOrder(ctx context.Context, orderNumber string) error {
	var accrualOrder models.AccrualOrder
	err := withRetry(ctx, func() error {
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

	status := accrualOrder.Status.ToOrderStatus()
	return p.repo.ApplyAccrualResult(ctx, orderNumber, status, accrualOrder.Accrual)
}

// Возвращает true, если ошибка допускает повторную попытку запроса.
func IsTemporaryError(err error) bool {
	var temporaryError accrual.TemporaryError
	return errors.As(err, &temporaryError)
}
