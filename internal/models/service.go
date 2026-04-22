package models

import "time"

// Описывает логин/пароль для регистрации и входа
type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Описывает пользователя системы лояльности
type User struct {
	ID             int64
	Login          string
	PasswordHash   string
	CurrentBalance float64
	Withdrawn      float64
	CreatedAt      time.Time
}

// Определяет состояние обработки заказа внутри системы лояльности
type OrderStatus string

const (
	// Заказ принят, но еще не обработан внешней системой
	OrderStatusNew OrderStatus = "NEW"
	// Начисление по заказу вычисляется
	OrderStatusProcessing OrderStatus = "PROCESSING"
	// Заказ отклонен системой начислений
	OrderStatusInvalid OrderStatus = "INVALID"
	// Начисление по заказу успешно рассчитано
	OrderStatusProcessed OrderStatus = "PROCESSED"
)

// Возвращает true для конечных статусов обработки заказа.
func (s OrderStatus) IsFinal() bool {
	return s == OrderStatusInvalid || s == OrderStatusProcessed
}

// Описывает заказ пользователя
type Order struct {
	Number     string      `json:"number"`
	UserID     int64       `json:"-"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual,omitempty"`
	UploadedAt string      `json:"uploaded_at"`
	UpdatedAt  string      `json:"-"`
}

// Описывает состояние бонусного счета пользователя
type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// Описывает запрос на списание бонусов
type WithdrawalRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// Withdrawal описывает успешное списание бонусов
type Withdrawal struct {
	Order       string    `json:"order"`
	UserID      int64     `json:"-"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
