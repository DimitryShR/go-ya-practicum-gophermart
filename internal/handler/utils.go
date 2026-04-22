package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/auth"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
)

func decodeCredentials(w http.ResponseWriter, r *http.Request) (models.Credentials, bool) {
	var credentials models.Credentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return models.Credentials{}, false
	}

	return credentials, true
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return 0, false
	}
	return principal.UserID, true
}
