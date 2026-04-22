package middleware

import (
	"errors"
	"net/http"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/auth"
)

// Проверяет JWT-токен и добавляет пользователя в контекст запроса
func RequireAuth(manager *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := manager.AuthenticateRequest(r)
			if err != nil {
				if errors.Is(err, auth.ErrTokenNotFound) || errors.Is(err, auth.ErrTokenInvalid) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
		})
	}
}
