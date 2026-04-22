package repository

import (
	"context"
	"errors"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	retryAttempts = 4
	retryDelay    = 100 * time.Millisecond
)

// Выполняет операцию с экспоненциальным backoff
func (s *Store) withRetry(ctx context.Context, operation func() error) error {
	return retry.Do(
		operation,
		retry.Context(ctx),
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.RetryIf(isRetryablePostgresError),
	)
}

func isRetryablePostgresError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	switch pgErr.Code {

	case
		// Только указанные ошибки, т.к. ошибки соединение (такие как 08*) pgx ретраит самостоятельно
		"40001", // сериализация не удалась
		"40P01", // дедлок
		"53300": // слишком много соединений
		return true
	default:
		return false
	}
}
