package auth

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	defaultIssuer     = "gophermart"
	defaultCookieName = "gophermart_token"
)

type claims struct {
	UserID int64  `json:"uid"`
	Login  string `json:"login"`
	jwt.RegisteredClaims
}

// Отвечает за выпуск и проверку JWT токена
type Manager struct {
	secret     []byte
	ttl        time.Duration
	issuer     string
	cookieName string
}

// Cоздает новый менеджер токенов
func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{
		secret:     []byte(secret),
		ttl:        ttl,
		issuer:     defaultIssuer,
		cookieName: defaultCookieName,
	}
}

// Возвращает имя cookie с токеном
func (m *Manager) CookieName() string {
	return m.cookieName
}

// Выпускает подписанный JWT для пользователя
func (m *Manager) IssueToken(userID int64, login string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	})

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

// Валидирует токен и извлекает из него пользователя
func (m *Manager) ParseToken(rawToken string) (Principal, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Principal{}, ErrTokenNotFound
	}

	parsed, err := jwt.ParseWithClaims(rawToken, &claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrTokenInvalid
		}
		return m.secret, nil
	})
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	tokenClaims, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return Principal{}, ErrTokenInvalid
	}

	return Principal{
		UserID: tokenClaims.UserID,
		Login:  tokenClaims.Login,
	}, nil

}

// Извлекает токен из cookie или заголовка Authorization
//
// Приоритет отдается cookie, как это требуется в проекте
func (m *Manager) TokenFromRequest(r *http.Request) (string, error) {
	if cookie, err := r.Cookie(m.cookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value, nil
	}

	rawHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if rawHeader == "" {
		return "", ErrTokenNotFound
	}

	if strings.HasPrefix(strings.ToLower(rawHeader), "bearer") {
		return strings.TrimSpace(rawHeader[7:]), nil
	}

	return rawHeader, nil
}

// Извлекает и валидирует токен из HTTP-запроса
func (m *Manager) AuthenticateRequest(r *http.Request) (Principal, error) {
	token, err := m.TokenFromRequest(r)
	if err != nil {
		return Principal{}, err
	}
	return m.ParseToken(token)
}

// Cоздает cookie с токеном для клиентского агента
func (m *Manager) BuildCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     m.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(m.ttl.Seconds()),
		Expires:  time.Now().Add(m.ttl),
	}
}

// Cоздает cookie, которая удаляет токен на клиенте
func (m *Manager) ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
}
