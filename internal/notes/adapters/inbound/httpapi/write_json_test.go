package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingResponseWriter struct{ http.ResponseWriter }

func (failingResponseWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

func logsOfAFailedWrite(ctx context.Context) string {
	logs := &strings.Builder{}
	logger := slog.New(slog.NewTextHandler(logs, nil))
	handler := assignRequestID(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		_ = writeJSON(failingResponseWriter{responseWriter}, request, logger, "application/json", http.StatusOK, NoteListResponseDTO{})
	}))
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/notes", nil)
	request.Header.Set("X-Request-Id", "trace-123")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	return logs.String()
}

func TestAFailedWriteIsLoggedWithItsRequestID(t *testing.T) {
	logs := logsOfAFailedWrite(context.Background())
	for _, expected := range []string{"level=ERROR", "requestId=trace-123", "broken pipe"} {
		if !strings.Contains(logs, expected) {
			t.Errorf("%q is missing from the logs: %s", expected, logs)
		}
	}
}

func TestAWriteThatFailsBecauseTheClientLeftIsNotAServerFault(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	logs := logsOfAFailedWrite(cancelled)
	for _, expected := range []string{"level=INFO", "requestId=trace-123"} {
		if !strings.Contains(logs, expected) {
			t.Errorf("%q is missing from the logs: %s", expected, logs)
		}
	}
}
