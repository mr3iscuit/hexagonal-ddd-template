package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
)

var routedMethods = []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions}

// NewRouter builds the HTTP API. Failures that never reach a handler are
// handled here, so they return the same APIError body.
func NewRouter(noteUseCases application.NoteUseCasesPort, logger *slog.Logger) chi.Router {
	router := chi.NewRouter()
	router.Use(assignRequestID, logRequests(logger), recoverPanics(logger))
	router.NotFound(func(responseWriter http.ResponseWriter, request *http.Request) {
		writeAPIError(responseWriter, request, logger, NewAPIError(http.StatusNotFound, codeRouteNotFound, routeNotFoundMessage))
	})
	router.MethodNotAllowed(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Allow", allowedMethods(router, routedPath(request)))
		writeAPIError(responseWriter, request, logger, NewAPIError(http.StatusMethodNotAllowed, codeMethodNotAllowed, methodNotAllowedMessage))
	})
	noteHandler := NewNoteHandler(noteUseCases, logger)
	router.Get("/healthz", noteHandler.handle(withoutBody(checkHealth)))
	noteHandler.Register(router)
	return router
}

func checkHealth(responseWriter http.ResponseWriter, _ *http.Request) error {
	responseWriter.WriteHeader(http.StatusNoContent)
	return nil
}

// routedPath is the path Chi matched, which keeps its percent-encoding when the
// decoded path would mean something else.
func routedPath(request *http.Request) string {
	if request.URL.RawPath != "" {
		return request.URL.RawPath
	}
	return request.URL.Path
}

// allowedMethods rebuilds the list, because Chi's custom handler drops it.
func allowedMethods(router chi.Router, path string) string {
	var methods []string
	for _, method := range routedMethods {
		if router.Match(chi.NewRouteContext(), method, path) {
			methods = append(methods, method)
		}
	}
	return strings.Join(methods, allowedMethodSeparator)
}
