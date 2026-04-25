package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
)

// Реализует обращение к внешнему accrual сервису начислений
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Создает клиент для обращения к внешнему accrual сервису
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 5 * time.Second,
		}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

// Получает состояние заказа во внешней системе начислений
func (c *Client) GetOrderAccrual(ctx context.Context, orderNumber string) (models.AccrualOrder, error) {
	url, err := url.JoinPath(c.baseURL, "/api/orders", orderNumber)
	if err != nil {
		return models.AccrualOrder{}, fmt.Errorf("create url path for accrual request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return models.AccrualOrder{}, fmt.Errorf("create accrual request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return models.AccrualOrder{}, TemporaryError{Err: fmt.Errorf("perform accrual request: %w", err)}
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		var order models.AccrualOrder
		if err := json.NewDecoder(response.Body).Decode(&order); err != nil {
			return models.AccrualOrder{}, TemporaryError{Err: fmt.Errorf("decode accrual response: %w", err)}
		}
		return order, nil
	case http.StatusNoContent:
		return models.AccrualOrder{}, ErrOrderNotRegistered
	case http.StatusTooManyRequests:
		return models.AccrualOrder{}, RateLimitError{RetryAfter: parseRetryAfter(response.Header.Get("Retry-After"))}
	case http.StatusInternalServerError, http.StatusBadRequest, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return models.AccrualOrder{}, TemporaryError{
			Err: fmt.Errorf("temporary accrual status: %d", response.StatusCode),
		}
	default:
		return models.AccrualOrder{}, fmt.Errorf("unexpected accrual status: %d", response.StatusCode)
	}
}

// Парсит значение RetryAfter, в случае ошибки возвращает RetryAfter равный 1 минуте
func parseRetryAfter(rawValue string) time.Duration {
	value := strings.TrimSpace(rawValue)
	if value == "" {
		return time.Minute
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Minute
	}

	return time.Duration(seconds) * time.Second
}
