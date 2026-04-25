package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/accrual"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/auth"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/config"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/handler"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/logger"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/middleware"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/repository"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.New()
	if err != nil {
		return err
	}

	// Инициализируем логгер
	if err := logger.Initialize(cfg.LogLevel, cfg.LogFilename); err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}

	// Инициализируем корневой контекст с обработкой сигналов
	rootCtx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Применяем миграции
	if err := repository.RunMigrations(cfg.DatabaseURI, cfg.Local); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Создаем подключение к БД
	db, err := openDatabase(cfg.DatabaseURI, cfg.Local)
	if err != nil {
		return err
	}
	defer db.Close()

	// Контекст с таймаутом для пинга БД
	startupCtx, cancel := context.WithTimeout(rootCtx, 5*time.Second)
	defer cancel()

	// Создаем PostgreSQL репозиторий
	store := repository.NewStore(db)
	if err := store.Ping(startupCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	// Менеджеры аутентикации
	passwords := auth.NewPasswordManager()
	tokens := auth.NewManager(cfg.JWTSecret, cfg.TokenTTL)
	// Сервисы
	users := service.NewUserService(store, passwords)
	loyalty := service.NewLoyaltyService(store)

	// Создаем клиент и сервис для обращения к внешнему accrual сервису
	accrualClient := accrual.NewClient(cfg.AccrualSystemAddress, &http.Client{Timeout: 5 * time.Second})
	processor := service.NewAccrualProcessor(store, accrualClient, cfg.PollInterval, cfg.AccrualConcurrency)
	// Запускаем обработчик
	go processor.Run(rootCtx)

	// Конфигурируем хендлеры и сервер
	apiHandler := handler.New(users, loyalty, tokens)
	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: newRouter(apiHandler, tokens),
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Log.Info("starting gophermart server", "address", cfg.RunAddress)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-rootCtx.Done():
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}

// Создает подключение к БД
func openDatabase(databaseURI string, local bool) (*sql.DB, error) {
	postgresConfig, err := pgx.ParseConfig(databaseURI)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	if local {
		postgresConfig.RuntimeParams["search_path"] = repository.SchemaName()
	}
	return stdlib.OpenDB(*postgresConfig), nil
}

func newRouter(apiHandler *handler.Handler, tokens *auth.Manager) http.Handler {
	router := chi.NewRouter()
	router.Use(chimw.StripSlashes)
	router.Use(middleware.Logging)
	router.Use(middleware.Gzip)

	router.Post("/api/user/register", apiHandler.Register)
	router.Post("/api/user/login", apiHandler.Login)

	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(tokens))
		r.Post("/api/user/orders", apiHandler.UploadOrder)
		r.Get("/api/user/orders", apiHandler.ListOrders)
		r.Get("/api/user/balance", apiHandler.GetBalance)
		r.Post("/api/user/balance/withdraw", apiHandler.Withdraw)
		r.Get("/api/user/withdrawals", apiHandler.ListWithdrawals)
	})

	return router
}
