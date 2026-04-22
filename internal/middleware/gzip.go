package middleware

import (
	"net/http"
	"strings"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/compress"
)

// Обрабатывает gzip-сжатие входящих и исходящих HTTP-сообщений
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := compress.NewWriter(w)
			defer cw.Close()
			w = cw
		}

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := compress.NewReader(r.Body)
			if err != nil {
				http.Error(w, "invalid gzip body", http.StatusBadRequest)
				return
			}
			defer cr.Close()
			r.Body = cr
		}

		next.ServeHTTP(w, r)
	})
}
