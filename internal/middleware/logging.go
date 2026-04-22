package middleware

import (
	"net/http"
	"time"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/logger"
)

type responseData struct {
	statusCode int
	size       int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	data *responseData
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.data.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *loggingResponseWriter) Write(body []byte) (int, error) {
	size, err := w.ResponseWriter.Write(body)
	w.data.size += size
	return size, err
}

// Пишет в лог информацию о каждом завершенном HTTP-запросе
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		data := &responseData{}
		lw := &loggingResponseWriter{
			ResponseWriter: w,
			data:           data,
		}

		next.ServeHTTP(lw, r)

		if data.statusCode == 0 {
			data.statusCode = http.StatusOK
		}

		logger.Log.Info(
			"http request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", data.statusCode,
			"size", data.size,
			"duration", time.Since(startedAt).String(),
		)
	})
}
