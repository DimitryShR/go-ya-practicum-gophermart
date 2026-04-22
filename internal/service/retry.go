package service

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
)

const (
	retryAttempts = 4
	retryDelay    = 100 * time.Millisecond
)

// Выполняет операцию с экспоненциальным backoff
func withRetry(ctx context.Context, operation func() error, retryCondition func(error) bool) error {
	return retry.Do(
		operation,
		retry.Context(ctx),
		retry.Attempts(retryAttempts),
		retry.Delay(retryDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.RetryIf(retryCondition),
	)
}
