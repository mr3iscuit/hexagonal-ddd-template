package httpapi

import (
	"context"
	"crypto/rand"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// assignRequestID keeps a caller's safe X-Request-Id and generates one
// otherwise, because the value reaches the logs. Chi's own middleware adds the
// hostname and keeps any value a caller sends.
func assignRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-Id")
		if !isSafeRequestID(requestID) {
			requestID = rand.Text()
		}
		responseWriter.Header().Set("X-Request-Id", requestID)
		ctx := context.WithValue(request.Context(), middleware.RequestIDKey, requestID)
		next.ServeHTTP(responseWriter, request.WithContext(ctx))
	})
}

func isSafeRequestID(requestID string) bool {
	if requestID == "" || len(requestID) > maximumRequestIDLength {
		return false
	}
	for _, character := range requestID {
		isLetter := (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')
		isDigit := character >= '0' && character <= '9'
		if !isLetter && !isDigit && character != '-' && character != '_' && character != '.' {
			return false
		}
	}
	return true
}

func requestIDFrom(ctx context.Context) string {
	return middleware.GetReqID(ctx)
}
