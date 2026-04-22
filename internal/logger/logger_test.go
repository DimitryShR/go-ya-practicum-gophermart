package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name     string
		loglevel string
		filename string
		wantFile bool
		wantErr  bool
	}{
		{
			name:     "without file",
			loglevel: "debug",
			filename: "",
			wantFile: false,
			wantErr:  false,
		},
		{
			name:     "with file",
			loglevel: "debug",
			filename: "test_gophermart.log",
			wantFile: true,
			wantErr:  false,
		},
		{
			name:     "incorrect loglevel name",
			loglevel: "bug",
			filename: "",
			wantFile: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantFile {
				// Удаляем тестовый файл с логами после теста
				t.Cleanup(func() {
					_ = os.Remove(tt.filename)
				})
				// Удаляем файл с логами перед тестом
				_ = os.Remove(tt.filename)
			}

			err := Initialize(tt.loglevel, tt.filename)
			// Если ожидаем ошибку, проверяем, что она есть
			if tt.wantErr {
				require.Error(t, err, "Expected error but got none")
				return
			}

			require.NoError(t, err, "Initialize should not return error")

			// Записываем лог, чтобы гарантированно создался файл
			Log.Info("test message", "case", tt.name)

			// Проверяем, что файл создался
			if tt.wantFile {
				_, err := os.Stat(tt.filename)
				assert.NoError(t, err, "Log file should be created")
			}

		})
	}
}
