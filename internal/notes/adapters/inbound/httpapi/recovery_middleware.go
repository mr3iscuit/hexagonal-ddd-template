package httpapi

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
)

// recoverPanics turns a panic into the standard error response. Chi's own
// Recoverer answers with an empty body.
func recoverPanics(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			wrapped := middleware.NewWrapResponseWriter(responseWriter, request.ProtoMajor)
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				// net/http uses this panic to abort a response on purpose.
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}
				logger.ErrorContext(request.Context(), logPanicRecovered,
					logKeyRequestID, requestIDFrom(request.Context()), logKeyPanic, recovered, logKeyStack, string(debug.Stack()))
				if wrapped.Status() == 0 {
					writeAPIError(wrapped, request, logger, NewAPIError(http.StatusInternalServerError, codeInternalError, internalErrorMessage))
				}
			}()
			next.ServeHTTP(wrapped, request)
		})
	}
}
