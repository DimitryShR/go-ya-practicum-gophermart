package service

import "errors"

var (
	// Означает, что логин или пароль не прошли проверку
	ErrInvalidCredentials = errors.New("invalid credentials")
	// Означает, что логин уже зарегистрирован
	ErrLoginAlreadyExists = errors.New("login already exists")
	// Означает, что номер заказа не прошел валидацию
	ErrInvalidOrderNumber = errors.New("invalid order number")
	// Означает, что запрос на списание заполнен неверно
	ErrInvalidWithdrawal = errors.New("invalid withdrawal payload")
	// Означает, что пользователь уже отправлял этот заказ
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by current user")
	// Означает, что заказ уже закреплен за другим пользователем
	ErrOrderUploadedByAnotherUser = errors.New("order already uploaded by another user")
	// Означает, что на бонусном счете недостаточно средств
	ErrInsufficientBalance = errors.New("insufficient balance")
	// Означает, что номер заказа уже использовался в списании
	ErrWithdrawalOrderAlreadyExists = errors.New("withdrawal order already exists")
)
