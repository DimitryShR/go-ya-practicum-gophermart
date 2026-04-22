package models

// Определяет статус заказа во внешней системе начислений
type AccrualStatus string

const (
	// Заказ зарегистрирован, но еще не рассчитан
	AccrualStatusRegistered AccrualStatus = "REGISTERED"
	// Расчет начисления еще идет
	AccrualStatusProcessing AccrualStatus = "PROCESSING"
	// В начислении отказано
	AccrualStatusInvalid AccrualStatus = "INVALID"
	// Начисление успешно рассчитано
	AccrualStatusProcessed AccrualStatus = "PROCESSED"
)

// Описывает ответ внешней системы начислений
type AccrualOrder struct {
	Order   string        `json:"order"`
	Status  AccrualStatus `json:"status"`
	Accrual *float64      `json:"accrual,omitempty"`
}

// Преобразует статус внешней системы в статус внутреннего заказа
func (s AccrualStatus) ToOrderStatus() OrderStatus {
	switch s {
	case AccrualStatusProcessed:
		return OrderStatusProcessed
	case AccrualStatusInvalid:
		return OrderStatusInvalid
	case AccrualStatusRegistered, AccrualStatusProcessing:
		return OrderStatusProcessing
	default:
		return OrderStatusNew
	}
}
