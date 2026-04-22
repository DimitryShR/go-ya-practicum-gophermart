package accrual

import (
	"errors"
	"fmt"
	"time"
)

var (
	// Означает, что заказ еще не известен внешней системе начислений
	ErrOrderNotRegistered = errors.New("order is not registered in accrual system")
)

// Описывает ситуацию, когда внешний сервис временно ограничил частоту запросов
type RateLimitError struct {
	RetryAfter time.Duration
}

// Возвращает текст ошибки ограничения частоты запросов
func (e RateLimitError) Error() string {
	return fmt.Sprintf("accrual rate limit exceeded, retry after %s", e.RetryAfter)
}

// Описывает временную ошибку взаимодействия с accrual-сервисом
type TemporaryError struct {
	Err error
}

// Возвращает текст ошибки
func (e TemporaryError) Error() string {
	return e.Err.Error()
}

// Извлекает вложенную ошибку
func (e TemporaryError) Unwrap() error {
	return e.Err
}
