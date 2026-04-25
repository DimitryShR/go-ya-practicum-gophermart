package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigParseFlagsAndEnvPrecedence(t *testing.T) {
	cfg := defaultConfig()
	err := cfg.parseFlags([]string{
		"-a", ":9000",
		"-d", "postgres://flag-db",
		"-r", "flag-accrual:8080",
		"-log-level", "debug",
		"-log-file", "test_gophermart.log",
		"-j", "flag-secret",
		"-jwt-ttl", "2h",
		"-poll-interval", "7s",
		"-shutdown-timeout", "15s",
		"-acc-concurrency", "5",
		"-local",
	})
	if err != nil {
		t.Fatalf("parseFlags() returned error: %v", err)
	}

	t.Setenv("RUN_ADDRESS", ":9100")
	t.Setenv("DATABASE_URI", "postgres://user:pass@env-db:5432/db")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "env-accrual:8081")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_FILENAME", "env-test_gophermart.log")
	t.Setenv("JWT_SECRET", "env-secret")
	t.Setenv("JWT_TTL", "3h")
	t.Setenv("ACCRUAL_POLL_INTERVAL", "9s")
	t.Setenv("SHUTDOWN_TIMEOUT", "25s")
	t.Setenv("ACCRUAL_CONCURRENCY", "7")
	t.Setenv("LOCAL", "true")

	if err := cfg.applyEnv(); err != nil {
		t.Fatalf("applyEnv() returned error: %v", err)
	}
	cfg.normalize()

	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() returned error: %v", err)
	}

	if cfg.RunAddress != ":9100" {
		t.Fatalf("unexpected RunAddress: %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://user:pass@env-db:5432/db" {
		t.Fatalf("unexpected DatabaseURI: %q", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "http://env-accrual:8081" {
		t.Fatalf("unexpected AccrualSystemAddress: %q", cfg.AccrualSystemAddress)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("unexpected LogLevel: %s", cfg.LogLevel)
	}
	if cfg.LogFilename != "env-test_gophermart.log" {
		t.Fatalf("unexpected LogFilename: %q", cfg.LogFilename)
	}
	if cfg.JWTSecret != "env-secret" {
		t.Fatalf("unexpected JWTSecret: %q", cfg.JWTSecret)
	}
	if cfg.TokenTTL != 3*time.Hour {
		t.Fatalf("unexpected TokenTTL: %s", cfg.TokenTTL)
	}
	if cfg.PollInterval != 9*time.Second {
		t.Fatalf("unexpected PollInterval: %s", cfg.PollInterval)
	}
	if cfg.ShutdownTimeout != 25*time.Second {
		t.Fatalf("unexpected ShutdownTimeout: %s", cfg.ShutdownTimeout)
	}
	if cfg.AccrualConcurrency != 7 {
		t.Fatalf("unexpected AccrualConcurrency: %d", cfg.AccrualConcurrency)
	}
	if cfg.Local != true {
		t.Fatalf("unexpected Local value: %t", cfg.Local)
	}
}

func TestConfigValidateRejectsInvalidValues(t *testing.T) {
	cfg := &config{
		RunAddress:           "",
		DatabaseURI:          "",
		AccrualSystemAddress: "",
		LogFilename:          "",
		JWTSecret:            "",
		TokenTTL:             0,
		PollInterval:         -time.Second,
		ShutdownTimeout:      0,
	}

	if err := cfg.validate(); err == nil {
		t.Fatal("validate() expected error, got nil")
	}
}

func TestDatabaseURIValidateRejectInvalidValue(t *testing.T) {
	tests := []struct {
		name          string
		databaseURI   string
		wantErr       bool
		expectedError string
	}{
		{
			name:        "valid",
			databaseURI: "postgres://user:pass@env-db:5432/db",
			wantErr:     false,
		},
		{
			name:          "no db",
			databaseURI:   "postgres://user:pass@env-db:5432",
			wantErr:       true,
			expectedError: "database URI: db name is not specified",
		},
		{
			name:          "no password",
			databaseURI:   "postgres://user@env-db:5432/db",
			wantErr:       true,
			expectedError: "database URI: password is not specified",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDatabaseURI(tt.databaseURI)
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
				if tt.expectedError != "" {
					assert.ErrorContains(t, err, tt.expectedError)
				}
				return
			}
			require.NoError(t, err, "Initialize should not return error")
		})

	}
}

func TestNormalizeHTTPAddress(t *testing.T) {
	if got := normalizeHTTPAddress("localhost:8080"); got != "http://localhost:8080" {
		t.Fatalf("unexpected normalized address: %q", got)
	}
	if got := normalizeHTTPAddress("https://localhost:8080"); got != "https://localhost:8080" {
		t.Fatalf("unexpected preserved address: %q", got)
	}
}
