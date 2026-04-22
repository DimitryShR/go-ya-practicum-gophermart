package compress

import (
	"compress/gzip"
	"io"
	"net/http"
)

// Cжимает HTTP-ответ и проксирует вызовы в исходный ResponseWriter
type writer struct {
	http.ResponseWriter
	zw           *gzip.Writer
	headerWriten bool
}

// Создает новый gzip ResponseWriter
func NewWriter(w http.ResponseWriter) *writer {
	return &writer{
		ResponseWriter: w,
		zw:             gzip.NewWriter(w),
	}
}

// Возвращает исходные HTTP-заголовки ответа
func (w *writer) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Записывает HTTP-статус и при необходимости включает gzip
func (w *writer) WriteHeader(statusCode int) {
	if w.headerWriten {
		return
	}
	w.headerWriten = true
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(statusCode)
}

// Записывает gzip-сжатое тело ответа
func (w *writer) Write(p []byte) (int, error) {
	if !w.headerWriten {
		w.WriteHeader(http.StatusOK)
	}
	return w.zw.Write(p)
}

// Принудительно сбрасывает буфер gzip-писателя
func (w *writer) Flush() {
	_ = w.zw.Flush()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Завершает запись gzip-потока
func (w *writer) Close() error {
	return w.zw.Close()
}

// Распаковывает gzip-тело HTTP-запроса
type compressReader struct {
	io.ReadCloser
	zr *gzip.Reader
}

// Создает декомпрессор для тела HTTP-запроса
func NewReader(readCloser io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(readCloser)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		ReadCloser: readCloser,
		zr:         zr,
	}, nil
}

// Читает данные из распакованного тела запроса
func (r *compressReader) Read(p []byte) (int, error) {
	return r.zr.Read(p)
}

// Освобождает ресурсы декомпрессора
func (r *compressReader) Close() error {
	if err := r.zr.Close(); err != nil {
		_ = r.ReadCloser.Close()
		return err
	}
	return r.ReadCloser.Close()
}
