package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
)

// Реализует работу с PostgreSQL
type Store struct {
	db *sql.DB
}

// Создает новый экземпляр PostgreSQL репозитория
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Проверяет доступность PostgreSQL
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Cохраняет нового пользователя в базе данных
func (s *Store) CreateUser(ctx context.Context, login, passwordHash string) (models.User, error) {
	var user models.User

	if err := s.withRetry(ctx, func() error {
		if err := s.db.QueryRowContext(ctx, createUserQuery, login, passwordHash).Scan(
			&user.ID,
			&user.Login,
			&user.PasswordHash,
			&user.CurrentBalance,
			&user.Withdrawn,
			&user.CreatedAt,
		); err != nil {
			if isUniqueViolation(err) {
				return ErrUserAlreadyExists
			}
			return fmt.Errorf("create user: %w", err)
		}

		return nil
	}); err != nil {
		return models.User{}, err
	}

	return user, nil
}

// Возвращает пользователя по логину
func (s *Store) UserByLogin(ctx context.Context, login string) (models.User, error) {
	var user models.User
	err := s.withRetry(ctx, func() error {
		if err := s.db.QueryRowContext(ctx, getUserByLoginQuery, login).Scan(
			&user.ID,
			&user.Login,
			&user.PasswordHash,
			&user.CurrentBalance,
			&user.Withdrawn,
			&user.CreatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFound
			}
			return fmt.Errorf("get user by login: %w", err)
		}
		return nil
	})
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

// Регистрирует новый заказ пользователя
func (s *Store) CreateOrder(ctx context.Context, userID int64, orderNumber string) (models.Order, error) {
	var (
		order    models.Order
		inserted bool
		accrual  sql.NullFloat64
	)

	err := s.withRetry(ctx, func() error {
		if err := s.db.QueryRowContext(ctx, createOrderQuery, orderNumber, userID, models.OrderStatusNew).Scan(
			&order.Number,
			&order.UserID,
			&order.Status,
			&accrual,
			&order.UploadedAt,
			&order.UpdatedAt,
			&inserted,
		); err != nil {
			return fmt.Errorf("scan order with inserted flag: %w", err)
		}

		if accrual.Valid {
			order.Accrual = &accrual.Float64
		}
		return nil
	})
	if err != nil {
		return models.Order{}, err
	}

	if inserted {
		return order, nil
	}
	if order.UserID == userID {
		return models.Order{}, ErrOrderAlreadyUploadedByUser
	}
	return models.Order{}, ErrOrderUploadedByAnotherUser
}

// Возвращает заказ по его номеру
func (s *Store) OrderByNumber(ctx context.Context, orderNumber string) (models.Order, error) {
	var order models.Order
	err := s.withRetry(ctx, func() error {
		scannedOrder, err := scanOrder(s.db.QueryRowContext(ctx, getOrderByNumberQuery, orderNumber))
		if err != nil {
			return err
		}
		order = scannedOrder
		return nil
	})
	if err != nil {
		return models.Order{}, err
	}

	return order, nil
}

// Возвращает заказы пользователя от новых к старым
func (s *Store) ListOrdersByUser(ctx context.Context, userID int64) ([]models.Order, error) {
	var orders []models.Order
	err := s.withRetry(ctx, func() error {
		rows, err := s.db.QueryContext(ctx, listOrdersByUserQuery, userID)
		if err != nil {
			return fmt.Errorf("list user orders: %w", err)
		}
		defer rows.Close()

		items := make([]models.Order, 0)
		for rows.Next() {
			order, err := scanOrder(rows)
			if err != nil {
				return err
			}
			items = append(items, order)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate user orders: %w", err)
		}

		orders = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// Возвращает заказы, которые нужно синхронизировать с accrual-сервисом
func (s *Store) ListOrdersForProcessing(ctx context.Context, limit int) ([]models.Order, error) {
	var orders []models.Order
	err := s.withRetry(ctx, func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin select processing orders transaction: %w", err)
		}
		defer tx.Rollback()

		rows, err := tx.QueryContext(ctx, listOrdersForProcessingQuery, models.OrderStatusNew, models.OrderStatusProcessing, limit)
		if err != nil {
			return fmt.Errorf("list orders for processing: %w", err)
		}
		defer rows.Close()

		items := make([]models.Order, 0, limit)
		for rows.Next() {
			order, err := scanOrder(rows)
			if err != nil {
				return err
			}
			items = append(items, order)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate processing orders: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit select processing orders transaction: %w", err)
		}

		orders = items
		return nil
	})
	if err != nil {
		return nil, err
	}

	return orders, nil
}

// Обновляет статус заказа и, при необходимости, баланс пользователя
func (s *Store) ApplyAccrualResult(ctx context.Context, number string, status models.OrderStatus, accrual *float64) error {
	return s.withRetry(ctx, func() error {
		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		if err != nil {
			return fmt.Errorf("begin apply accrual transaction: %w", err)
		}
		defer tx.Rollback()

		const selectQuery = `
			SELECT user_id, status
			FROM orders
			WHERE number = $1
			FOR UPDATE
		`

		var (
			userID        int64
			currentStatus models.OrderStatus
		)

		if err := tx.QueryRowContext(ctx, selectQuery, number).Scan(
			&userID,
			&currentStatus,
		); err != nil {
			return fmt.Errorf("lock order for accrual update: %w", err)
		}

		if currentStatus.IsFinal() {
			return nil
		}

		const updateQuery = `
			UPDATE orders
			SET status = $2, accrual = $3, updated_at = NOW()
			WHERE number = $1
		`

		var (
			accrualArg   any
			accrualValue float64
		)
		if accrual != nil {
			accrualArg = *accrual
			accrualValue = *accrual
		}

		if _, err := tx.ExecContext(ctx, updateQuery, number, status, accrualArg); err != nil {
			return fmt.Errorf("update order after accrual response: %w", err)
		}

		if status == models.OrderStatusProcessed {
			const creditQuery = `
				UPDATE users
				SET current_balance = current_balance + $2
				WHERE id = $1
			`

			if _, err := tx.ExecContext(ctx, creditQuery, userID, accrualValue); err != nil {
				return fmt.Errorf("credit user balance: %w", err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit apply accrual transaction: %w", err)
		}

		return nil
	})
}

// Возвращает текущий и уже списанный баланс пользователя
func (s *Store) GetBalance(ctx context.Context, userID int64) (models.Balance, error) {
	var balance models.Balance
	err := s.withRetry(ctx, func() error {
		err := s.db.QueryRowContext(ctx, getBalanceByUserQuery, userID).Scan(
			&balance.Current,
			&balance.Withdrawn,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFound
			}
			return fmt.Errorf("get balance: %w", err)
		}

		return nil
	})
	if err != nil {
		return models.Balance{}, err
	}

	return balance, nil
}

// Атомарно списывает бонусы и сохраняет факт списания
func (s *Store) CreateWithdrawal(ctx context.Context, userID int64, orderNumber string, sum float64) error {
	return s.withRetry(ctx, func() error {
		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		if err != nil {
			return fmt.Errorf("begin withdrawal transaction: %w", err)
		}
		defer tx.Rollback()

		const lockUserQuery = `
			SELECT current_balance
			FROM users
			WHERE id = $1
			FOR UPDATE
		`

		var currentBalance float64
		if err := tx.QueryRowContext(ctx, lockUserQuery, userID).Scan(
			&currentBalance,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFound
			}
			return fmt.Errorf("lock user for withdrawal: %w", err)
		}

		if currentBalance < sum {
			return ErrInsufficientBalance
		}

		const insertQuery = `
			INSERT INTO withdrawals (user_id, order_number, sum)
			VALUES ($1, $2, $3)
		`

		if _, err := tx.ExecContext(ctx, insertQuery, userID, orderNumber, sum); err != nil {
			if isUniqueViolation(err) {
				return ErrWithdrawalOrderAlreadyExists
			}
			return fmt.Errorf("insert withdrawal: %w", err)
		}

		const updateBalanceQuery = `
			UPDATE users
			SET current_balance = current_balance - $2,
			    withdrawn = withdrawn + $2
			WHERE id = $1
		`

		if _, err := tx.ExecContext(ctx, updateBalanceQuery, userID, sum); err != nil {
			return fmt.Errorf("debit user balance: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit withdrawal transaction: %w", err)
		}

		return nil
	})
}

// Возвращает историю списаний пользователя
func (s *Store) ListWithdrawalsByUser(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	err := s.withRetry(ctx, func() error {
		rows, err := s.db.QueryContext(ctx, listWithdrawalsByUserQuery, userID)
		if err != nil {
			return fmt.Errorf("list withdrawals: %w", err)
		}
		defer rows.Close()

		items := make([]models.Withdrawal, 0)
		for rows.Next() {
			var withdrawal models.Withdrawal
			if err := rows.Scan(
				&withdrawal.Order,
				&withdrawal.UserID,
				&withdrawal.Sum,
				&withdrawal.ProcessedAt,
			); err != nil {
				return fmt.Errorf("scan withdrawal: %w", err)
			}
			items = append(items, withdrawal)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate withdrawals: %w", err)
		}

		withdrawals = items
		return nil
	})
	if err != nil {
		return nil, err
	}

	return withdrawals, nil
}

type orderScanner interface {
	Scan(dest ...any) error
}

func scanOrder(scanner orderScanner) (models.Order, error) {
	var (
		order   models.Order
		accrual sql.NullFloat64
	)

	if err := scanner.Scan(
		&order.Number,
		&order.UserID,
		&order.Status,
		&accrual,
		&order.UploadedAt,
		&order.UpdatedAt,
	); err != nil {
		return models.Order{}, fmt.Errorf("scan order: %w", err)
	}

	if accrual.Valid {
		order.Accrual = &accrual.Float64
	}

	return order, nil
}
