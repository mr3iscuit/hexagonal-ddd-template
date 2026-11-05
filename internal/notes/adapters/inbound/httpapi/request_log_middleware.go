package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// logRequests writes one line per request, so a successful request can be
// traced too. It defers the line, because an aborted response ends in a panic.
func logRequests(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			wrapped := middleware.NewWrapResponseWriter(responseWriter, request.ProtoMajor)
			defer func() {
				logger.InfoContext(request.Context(), logRequestHandled,
					logKeyRequestID, requestIDFrom(request.Context()), logKeyMethod, request.Method, logKeyPath, request.URL.Path, logKeyStatus, wrapped.Status())
			}()
			next.ServeHTTP(wrapped, request)
		})
	}
}
