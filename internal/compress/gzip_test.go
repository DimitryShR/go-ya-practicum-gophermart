package compress

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriterAndReader(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := NewWriter(recorder)

	if _, err := writer.Write([]byte("hello gzip")); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	writer.Flush()
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("unexpected Content-Encoding: %q", recorder.Header().Get("Content-Encoding"))
	}

	reader, err := NewReader(io.NopCloser(bytes.NewReader(recorder.Body.Bytes())))
	if err != nil {
		t.Fatalf("NewReader() returned error: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() returned error: %v", err)
	}
	if string(body) != "hello gzip" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestWriterHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := NewWriter(recorder)
	writer.WriteHeader(http.StatusCreated)
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("unexpected Content-Encoding: %q", recorder.Header().Get("Content-Encoding"))
	}
}
