package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// Означает, что логин уже занят
	ErrUserAlreadyExists = errors.New("user already exists")
	// Означает, что пользователь не найден
	ErrUserNotFound = errors.New("user not found")
	// Означает, что этот пользователь уже загружал заказ
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by current user")
	// Означает, что заказ принадлежит другому пользователю
	ErrOrderUploadedByAnotherUser = errors.New("order already uploaded by another user")
	// Означает, что у пользователя недостаточно бонусов для списания
	ErrInsufficientBalance = errors.New("insufficient balance")
	// Означает, что номер заказа уже использовался при списании
	ErrWithdrawalOrderAlreadyExists = errors.New("withdrawal order already exists")
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
