package logger

import (
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Константы на случай записи логов в файл
const (
	MaxSize    = 20 // мегабайт
	MaxBackups = 3
	MaxAge     = 30 // дней
	Compress   = true
)

// Глобальный логер
var Log = slog.New(slog.DiscardHandler)

func Initialize(level string, filename string) error {
	// Парсим уровень логирования
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		return err
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	// Консольный хендлер
	consoleHandler := slog.NewJSONHandler(os.Stderr, opts)

	// Если filename отсутствует, то только консольный вывод
	if filename == "" {
		Log = slog.New(consoleHandler)
		return nil
	}

	// // Файл лога без ротации
	// logFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	// if err != nil {
	// 	Log = slog.New(consoleHandler)
	// 	return nil
	// }

	// Файл лога с ротацией
	logFile := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    MaxSize, // мегабайт
		MaxBackups: MaxBackups,
		MaxAge:     MaxAge, // дней
		Compress:   Compress,
	}

	// Файловый хендлер
	fileHandler := slog.NewJSONHandler(logFile, opts)

	// Логгер с мультихендлером (консольный и файловый)
	Log = slog.New(slog.NewMultiHandler(consoleHandler, fileHandler))
	return nil
}
