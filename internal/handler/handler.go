package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/auth"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/service"
)

// Объединяет HTTP-обработчики API сервиса лояльности
type Handler struct {
	users   *service.UserService
	loyalty *service.LoyaltyService
	tokens  *auth.Manager
}

// Создает новый экземпляр HTTP-обработчиков
func New(users *service.UserService, loyalty *service.LoyaltyService, tokens *auth.Manager) *Handler {
	return &Handler{
		users:   users,
		loyalty: loyalty,
		tokens:  tokens,
	}
}

// Обрабатывает регистрацию пользователя
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	credentials, ok := decodeCredentials(w, r)
	if !ok {
		return
	}

	user, err := h.users.Register(r.Context(), credentials)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			http.Error(w, "invalid credentials", http.StatusBadRequest)
		case errors.Is(err, service.ErrLoginAlreadyExists):
			http.Error(w, "login already exists", http.StatusConflict)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	h.writeAuthResponse(w, user)
}

// Обрабатывает аутентификацию пользователя
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	credentials, ok := decodeCredentials(w, r)
	if !ok {
		return
	}

	user, err := h.users.Login(r.Context(), credentials)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	h.writeAuthResponse(w, user)
}

// Принимает номер заказа от аутентифицированного пользователя
func (h *Handler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(string(body)) == "" {
		http.Error(w, "empty order number", http.StatusBadRequest)
		return
	}

	result, err := h.loyalty.UploadOrder(r.Context(), userID, string(body))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOrderNumber):
			http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		case errors.Is(err, service.ErrOrderUploadedByAnotherUser):
			http.Error(w, "order already uploaded by another user", http.StatusConflict)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	if result == service.UploadOrderAlreadyExists {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// Возвращает историю заказов текущего пользователя
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.loyalty.ListOrders(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

// Возвращает бонусный баланс пользователя
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	balance, err := h.loyalty.GetBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, balance)
}

// Списывает бонусы со счета пользователя
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var request models.WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.loyalty.Withdraw(r.Context(), userID, request); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOrderNumber):
			http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		case errors.Is(err, service.ErrInvalidWithdrawal):
			http.Error(w, "invalid withdrawal payload", http.StatusBadRequest)
		case errors.Is(err, service.ErrInsufficientBalance):
			http.Error(w, "insufficient balance", http.StatusPaymentRequired)
		case errors.Is(err, service.ErrWithdrawalOrderAlreadyExists):
			http.Error(w, "withdrawal order already exists", http.StatusConflict)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Возвращает историю списаний текущего пользователя
func (h *Handler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.loyalty.ListWithdrawals(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusOK, withdrawals)
}
