package service

import (
	"context"
	"errors"
	"strings"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/repository"
)

// Описывает результат попытка зарегистрировать номер заказа
type UploadOrderResult int

const (
	// Заказ принят в обработку
	UploadOrderCreated UploadOrderResult = iota
	// Пользователь уже загружал этот заказ
	UploadOrderAlreadyExists
)

// Инкапсулирует работу с заказами, балансом и списаниями
type LoyaltyService struct {
	repo LoyaltyRepository
}

// Создает новый сервис лояльности
func NewLoyaltyService(repo LoyaltyRepository) *LoyaltyService {
	return &LoyaltyService{repo: repo}
}

// Валидирует и регистрирует номер заказа
func (s *LoyaltyService) UploadOrder(ctx context.Context, userID int64, rawOrderNumber string) (UploadOrderResult, error) {
	orderNumber, err := normalizeOrderNumber(rawOrderNumber)
	if err != nil {
		return 0, err
	}

	if _, err := s.repo.CreateOrder(ctx, userID, orderNumber); err != nil {
		switch {
		case errors.Is(err, repository.ErrOrderAlreadyUploadedByUser):
			return UploadOrderAlreadyExists, nil
		case errors.Is(err, repository.ErrOrderUploadedByAnotherUser):
			return 0, ErrOrderUploadedByAnotherUser
		default:
			return 0, err
		}
	}

	return UploadOrderCreated, err
}

// Возваращает все заказы пользователя
func (s *LoyaltyService) ListOrders(ctx context.Context, userID int64) ([]models.Order, error) {
	return s.repo.ListOrdersByUser(ctx, userID)
}

// Возвращает бонусный баланс
func (s *LoyaltyService) GetBalance(ctx context.Context, userID int64) (models.Balance, error) {
	return s.repo.GetBalance(ctx, userID)
}

// Списывает бонусы со счета пользователя
func (s *LoyaltyService) Withdraw(ctx context.Context, userID int64, request models.WithdrawalRequest) error {
	orderNumber, err := normalizeOrderNumber(request.Order)
	if err != nil {
		return err
	}
	if request.Sum <= 0 {
		return ErrInvalidWithdrawal
	}

	if err := s.repo.CreateWithdrawal(ctx, userID, orderNumber, request.Sum); err != nil {
		switch {
		case errors.Is(err, repository.ErrInsufficientBalance):
			return ErrInsufficientBalance
		case errors.Is(err, repository.ErrWithdrawalOrderAlreadyExists):
			return ErrWithdrawalOrderAlreadyExists
		default:
			return err
		}
	}

	return nil
}

// Возвращает историю списаний пользователя
func (s *LoyaltyService) ListWithdrawals(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
	return s.repo.ListWithdrawalsByUser(ctx, userID)
}

// Нормализует и валидирует номер заказа по алгоритму Луна
func normalizeOrderNumber(rawOrderNumber string) (string, error) {
	orderNumber := strings.TrimSpace(rawOrderNumber)
	if !ValidLuhnNumber(orderNumber) {
		return "", ErrInvalidOrderNumber
	}
	return orderNumber, nil
}
