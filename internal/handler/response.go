package handler

import (
	"encoding/json"
	"net/http"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/logger"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
)

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if payload == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Log.Error("cannot encode json response", "error", err)
	}
}

func (h *Handler) writeAuthResponse(w http.ResponseWriter, user models.User) {
	token, err := h.tokens.IssueToken(user.ID, user.Login)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, h.tokens.BuildCookie(token))
	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}
