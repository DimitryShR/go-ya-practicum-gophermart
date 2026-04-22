package service

import (
	"context"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
)

// Описывает операции хранилища для работы с пользователем
type UserRepository interface {
	// Cохраняет нового пользователя
	CreateUser(ctx context.Context, login, passwordHash string) (models.User, error)
	// Возвращает пользователя по логину
	UserByLogin(ctx context.Context, login string) (models.User, error)
}

// Описывает операции хранилища для заказов, баланса и списаний
type LoyaltyRepository interface {
	// Регистрирует новый заказ пользователя
	CreateOrder(ctx context.Context, UserID int64, orderNumber string) (models.Order, error)
	// Возвращает заказы пользователя от новых к старым
	ListOrdersByUser(ctx context.Context, UserID int64) ([]models.Order, error)
	// Возвращает текущий и уже списанный баланс пользователя
	GetBalance(ctx context.Context, UserID int64) (models.Balance, error)
	// Атомарно списывает бонусы и сохраняет факт списания
	CreateWithdrawal(ctx context.Context, UserID int64, orderNumber string, sum float64) error
	// Возвращает историю списаний пользователя
	ListWithdrawalsByUser(ctx context.Context, UserID int64) ([]models.Withdrawal, error)
}

// Описывает операции хранилища для фоновой обработки начислений
type AccrualRepository interface {
	// Возвращает заказы, которые нужно синхронизировать с accrual-сервисом
	ListOrdersForProcessing(ctx context.Context, limit int) ([]models.Order, error)
	// Обновляет статус заказа и, при необходимости, баланс пользователя
	ApplyAccrualResult(ctx context.Context, orderNumber string, status models.OrderStatus, accrual *float64) error
}

// Описывает контракт внешнего клиента начислений
type AccrualClient interface {
	// Получает состояние заказа во внешней системе начислений
	GetOrderAccrual(ctx context.Context, orderNumber string) (models.AccrualOrder, error)
}
