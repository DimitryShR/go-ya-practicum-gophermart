package auth

import "context"

type principalKey struct{}

// Содержит данные аутентифицированного пользователя
type Principal struct {
	UserID int64
	Login  string
}

// Сохраняет аутентифицированного пользователя в контекст
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// Извлекает аутентифицированного пользователя из контекста
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok
}
