package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const schemaName = "gophermart"

// Gрименяет SQL-миграции проекта к базе данных
//
// Для миграций открывается отдельное подключение
func RunMigrations(databaseURI string) error {
	migrationSourceURL, err := buildMigrationSourceURL()
	if err != nil {
		return err
	}

	db, err := openMigrationDB(databaseURI)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{
		SchemaName: schemaName,
	})
	if err != nil {
		return fmt.Errorf("create pgx migration driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(migrationSourceURL, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	defer migrator.Close()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// Возвращает наименование схемы
func SchemaName() string {
	return schemaName
}

func buildMigrationSourceURL() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	return "file://" + filepath.Join(workingDirectory, "migrations"), nil
}

func openMigrationDB(databaseURI string) (*sql.DB, error) {
	postgresConfig, err := pgx.ParseConfig(databaseURI)
	if err != nil {
		return nil, fmt.Errorf("parse migration database config: %w", err)
	}
	postgresConfig.RuntimeParams["search_path"] = schemaName

	return stdlib.OpenDB(*postgresConfig), nil
}
