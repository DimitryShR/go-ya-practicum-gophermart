package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	defaultRunAddress         = ":8080"
	defaultLogLevel           = "info"
	defaultJWTSecret          = "gophermart-dev-secret"
	defaultTokenTTL           = 72 * time.Hour
	defaultPollInterval       = 2 * time.Second
	defaultShutdownTimeout    = 10 * time.Second
	defaultAccrualConcurrency = 5
)

// Хранит конфигурацию сервиса
type config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	LogLevel             string
	LogFilename          string
	JWTSecret            string
	TokenTTL             time.Duration
	PollInterval         time.Duration
	ShutdownTimeout      time.Duration
	Local                bool
	AccrualConcurrency   int
}

// Создает конфигурацию приложения
//
// Приоритет источников: переменные окружения -> флаги -> значения по умолчанию
func New() (*config, error) {
	cfg := defaultConfig()
	if err := cfg.parseFlags(os.Args[1:]); err != nil {
		return nil, err
	}
	if err := cfg.applyEnv(); err != nil {
		return nil, err
	}
	cfg.normalize()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultConfig() *config {
	return &config{
		RunAddress:         defaultRunAddress,
		LogLevel:           defaultLogLevel,
		JWTSecret:          defaultJWTSecret,
		TokenTTL:           defaultTokenTTL,
		PollInterval:       defaultPollInterval,
		ShutdownTimeout:    defaultShutdownTimeout,
		AccrualConcurrency: defaultAccrualConcurrency,
	}
}

// Парсит флаги и устанавливает их значения в конфиг приложения
func (c *config) parseFlags(args []string) error {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&c.RunAddress, "a", c.RunAddress, "HTTP server address")
	fs.StringVar(&c.DatabaseURI, "d", c.DatabaseURI, "PostgreSQL connection URI")
	fs.StringVar(&c.AccrualSystemAddress, "r", c.AccrualSystemAddress, "Accrual service address")
	fs.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Log Level")
	fs.StringVar(&c.LogFilename, "log-file", c.LogFilename, "Log Filename")
	fs.StringVar(&c.JWTSecret, "j", c.JWTSecret, "JWT signing secret")
	fs.DurationVar(&c.TokenTTL, "jwt-ttl", c.TokenTTL, "JWT token lifetime")
	fs.DurationVar(&c.PollInterval, "poll-interval", c.PollInterval, "Accrual polling interval")
	fs.DurationVar(&c.ShutdownTimeout, "shutdown-timeout", c.ShutdownTimeout, "Graceful shutdown timeout")
	fs.BoolVar(&c.Local, "local", c.Local, "Run in local mode")
	fs.IntVar(&c.AccrualConcurrency, "acc-concurrency", c.AccrualConcurrency, "Accrual worker pool concurrency")

	if err := fs.Parse(args); err != nil {
		// Если запросили справку --help
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(os.Stdout, "Сервис системы лояльности Gophermart\n\n")
			fs.SetOutput(os.Stdout)
			fs.PrintDefaults()
			os.Exit(0)
		}
		return fmt.Errorf("parse flags: %w", err)
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	return nil
}

// Читает переменные окружения и устанавливает их значения в конфиг приложения
func (c *config) applyEnv() error {
	var errs []error
	applyEnvString("RUN_ADDRESS", &c.RunAddress)
	applyEnvString("DATABASE_URI", &c.DatabaseURI)
	applyEnvString("ACCRUAL_SYSTEM_ADDRESS", &c.AccrualSystemAddress)
	applyEnvString("LOG_LEVEL", &c.LogLevel)
	applyEnvString("LOG_FILENAME", &c.LogFilename)
	applyEnvString("JWT_SECRET", &c.JWTSecret)
	if err := applyEnvDuration("JWT_TTL", &c.TokenTTL); err != nil {
		errs = append(errs, fmt.Errorf("JWT_TTL: %w", err))
	}
	if err := applyEnvDuration("ACCRUAL_POLL_INTERVAL", &c.PollInterval); err != nil {
		errs = append(errs, fmt.Errorf("ACCRUAL_POLL_INTERVAL: %w", err))
	}
	if err := applyEnvDuration("SHUTDOWN_TIMEOUT", &c.ShutdownTimeout); err != nil {
		errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err))
	}
	if err := applyEnvBool("LOCAL", &c.Local); err != nil {
		errs = append(errs, fmt.Errorf("LOCAL: %w", err))
	}
	if err := applyEnvInt("ACCRUAL_CONCURRENCY", &c.AccrualConcurrency); err != nil {
		errs = append(errs, fmt.Errorf("ACCRUAL_CONCURRENCY: %w", err))
	}
	return errors.Join(errs...)
}

// Ищет переменную окружения типа string по имени и устанавливает ее значение в dst
func applyEnvString(name string, dst *string) {
	if value, ok := os.LookupEnv(name); ok {
		*dst = strings.TrimSpace(value)
	}
}

// Ищет переменную окружения типа time.Duration по имени и устанавливает ее значение в dst
func applyEnvDuration(name string, dst *time.Duration) error {
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return err
	}

	*dst = parsed
	return nil
}

// Ищет переменную окружения типа bool по имени и устанавливает ее значение в dst
func applyEnvBool(name string, dst *bool) error {
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return err
	}

	*dst = parsed
	return nil
}

// Ищем переменную окружения типа int по имени и устанавливаем ее значение в dst
func applyEnvInt(name string, dst *int) error {
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return err
	}

	*dst = parsed
	return nil
}

// Приводит параметры к нормализованному виду
func (c *config) normalize() {
	c.RunAddress = strings.TrimSpace(c.RunAddress)
	c.DatabaseURI = strings.TrimSpace(c.DatabaseURI)
	c.AccrualSystemAddress = normalizeHTTPAddress(strings.TrimSpace(c.AccrualSystemAddress))
	c.LogLevel = strings.TrimSpace(c.LogLevel)
	c.LogFilename = strings.TrimSpace(c.LogFilename)
	c.JWTSecret = strings.TrimSpace(c.JWTSecret)
}

// Добавляет схему к адресу, если она отсутствует
func normalizeHTTPAddress(address string) string {
	if address == "" {
		return ""
	}
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return address
	}
	return "http://" + address
}

func (c *config) validate() error {
	var errs []error

	if c.RunAddress == "" {
		errs = append(errs, errors.New("run address must not be empty"))
	}
	if c.DatabaseURI == "" {
		errs = append(errs, errors.New("database URI must not be empty"))
	} else {
		if err := validateDatabaseURI(c.DatabaseURI); err != nil {
			errs = append(errs, err)
		}
	}
	if c.AccrualSystemAddress == "" {
		errs = append(errs, errors.New("accrual system address must not be empty"))
	}
	if c.LogLevel == "" {
		errs = append(errs, errors.New("log level must not be empty"))
	}
	if c.JWTSecret == "" {
		errs = append(errs, errors.New("JWT secret must not be empty"))
	}
	if c.TokenTTL <= 0 {
		errs = append(errs, errors.New("JWT token TTL must be positive"))
	}
	if c.PollInterval <= 0 {
		errs = append(errs, errors.New("poll interval must be positive"))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("shutdown timeout must be positive"))
	}
	if c.AccrualConcurrency <= 0 {
		errs = append(errs, errors.New("accrual concurrency must be positive"))
	}

	return errors.Join(errs...)
}

// Валидирует DatabaseURI
func validateDatabaseURI(DatabaseURI string) error {
	cfg, err := pgx.ParseConfig(DatabaseURI)
	if err != nil {
		return err
	}

	var errs []error

	if cfg.Host == "" {
		errs = append(errs, errors.New("database URI: host is not specified"))
	}
	if cfg.Database == "" {
		errs = append(errs, errors.New("database URI: db name is not specified"))
	}
	if cfg.User == "" {
		errs = append(errs, errors.New("database URI: user is not specified"))
	}
	if cfg.Password == "" {
		errs = append(errs, errors.New("database URI: password is not specified"))
	}
	return errors.Join(errs...)
}
