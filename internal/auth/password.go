package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

const (
	defaultIterations = 120000
	defaultSaltSize   = 16
	defaultKeyLength  = 32
)

// Реализует безопасное хранение паролей
type PasswordManager struct {
	iterations int
	saltSize   int
	keyLength  int
}

// Создает менеджер паролей со значениями по умолчанию
func NewPasswordManager() *PasswordManager {
	return &PasswordManager{
		iterations: defaultIterations,
		saltSize:   defaultSaltSize,
		keyLength:  defaultKeyLength,
	}
}

// Преобразует пароль в строку для хранения
func (m *PasswordManager) Hash(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", ErrEmpryPassword
	}

	salt := make([]byte, m.saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := derivePBKDF2SHA256([]byte(password), salt, m.iterations, m.keyLength)
	return fmt.Sprintf(
		"pbkdf2$sha256$%d$%s$%s",
		m.iterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Проверяет соответствие хеша от пароля ранее сохраненному хэшу
func (m *PasswordManager) Compare(encodedHash, password string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 5 || parts[0] != "pbkdf2" || parts[1] != "sha256" {
		return false, ErrInvalidPasswordHash
	}

	iterations, err := strconv.Atoi(parts[2])
	if err != nil || iterations <= 0 {
		return false, ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, ErrInvalidPasswordHash
	}

	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidPasswordHash
	}

	actualKey := derivePBKDF2SHA256([]byte(password), salt, iterations, len(expectedKey))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

// Вычисляет хеш PBKDF2SHA256 от пароля
func derivePBKDF2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	hashLength := sha256.Size
	blocks := (keyLength + hashLength - 1) / hashLength
	derived := make([]byte, 0, blocks*hashLength)

	for block := 1; block <= blocks; block++ {
		u := hmacSHA256(password, appendBlockIndex(salt, block))
		t := make([]byte, len(u))
		copy(t, u)

		for iteration := 1; iteration < iterations; iteration++ {
			u = hmacSHA256(password, u)
			for idx := range t {
				t[idx] ^= u[idx]
			}
		}

		derived = append(derived, t...)
	}

	return derived[:keyLength]
}

// Вычисляет HMAC-SHA256 от переданных данных
func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// Добавляет к соли порядковый номер блока
func appendBlockIndex(salt []byte, block int) []byte {
	payload := make([]byte, len(salt)+4)
	copy(payload, salt)
	binary.BigEndian.PutUint32(payload[len(salt):], uint32(block))
	return payload
}
