package auth

import "errors"

// Ошибки для хеша
var (
	// Строка хэша повреждена или имеет неожиданный формат
	ErrInvalidPasswordHash = errors.New("invalid password hash")
	// Пустая строка вместо пароля
	ErrEmpryPassword = errors.New("password must not be empty")
)

// Ошибки для токена
var (
	// ErrTokenNotFound возвращается, когда токен отсутствует в cookie и в заголовке.
	ErrTokenNotFound = errors.New("token not found")
	// ErrTokenInvalid возвращается, когда токен не прошел проверку подписи или структуры.
	ErrTokenInvalid = errors.New("token is invalid")
)
