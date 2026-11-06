// Package httpapi exposes the note use cases over HTTP/JSON.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
)

// NoteHandler is the inbound adapter. It depends only on the inbound port.
type NoteHandler struct {
	noteUseCases application.NoteUseCasesPort
	logger       *slog.Logger
}

func NewNoteHandler(noteUseCases application.NoteUseCasesPort, logger *slog.Logger) NoteHandler {
	return NoteHandler{noteUseCases: noteUseCases, logger: logger}
}

// Register attaches every note route to the router.
func (handler NoteHandler) Register(router chi.Router) {
	router.Post("/notes", handler.handle(handler.createNote))
	router.Get("/notes", handler.handle(withoutBody(handler.listNotes)))
	router.Get("/notes/{id}", handler.handle(withoutBody(handler.getNote)))
	router.Put("/notes/{id}", handler.handle(handler.editNote))
	router.Delete("/notes/{id}", handler.handle(withoutBody(handler.deleteNote)))
	router.Post("/notes/{id}/archive", handler.handle(withoutBody(handler.archiveNote)))
	router.Post("/notes/{id}/restore", handler.handle(withoutBody(handler.restoreNote)))
	router.Put("/notes/{id}/reminder", handler.handle(handler.scheduleReminder))
	router.Delete("/notes/{id}/reminder", handler.handle(withoutBody(handler.cancelReminder)))
}

// handle lets an endpoint stop by returning an error, so every failure goes
// through translate and returns the same body.
func (handler NoteHandler) handle(endpoint func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		if err := endpoint(responseWriter, request); err != nil {
			writeAPIError(responseWriter, request, handler.logger, translate(request.Context(), handler.logger, err))
		}
	}
}

func (handler NoteHandler) createNote(responseWriter http.ResponseWriter, request *http.Request) error {
	requestDTO, err := decodeRequest[CreateNoteRequestDTO](responseWriter, request, createNoteRequestFields)
	if err != nil {
		return err
	}
	view, err := handler.noteUseCases.CreateNote(request.Context(), toCreateNoteCommand(requestDTO))
	if err != nil {
		return err
	}
	responseWriter.Header().Set("Location", "/notes/"+url.PathEscape(view.ID))
	return handler.respond(responseWriter, request, http.StatusCreated, toNoteResponseDTO(view))
}

func (handler NoteHandler) listNotes(responseWriter http.ResponseWriter, request *http.Request) error {
	archived, err := archivedFilter(request)
	if err != nil {
		return err
	}
	views, err := handler.noteUseCases.ListNotes(request.Context(), toListNotesQuery(archived))
	if err != nil {
		return err
	}
	return handler.respond(responseWriter, request, http.StatusOK, toNoteListResponseDTO(views))
}

func (handler NoteHandler) getNote(responseWriter http.ResponseWriter, request *http.Request) error {
	view, err := handler.noteUseCases.GetNote(request.Context(), toGetNoteQuery(noteIDFrom(request)))
	if err != nil {
		return err
	}
	return handler.respond(responseWriter, request, http.StatusOK, toNoteResponseDTO(view))
}

func (handler NoteHandler) editNote(responseWriter http.ResponseWriter, request *http.Request) error {
	requestDTO, err := decodeRequest[EditNoteRequestDTO](responseWriter, request, editNoteRequestFields)
	if err != nil {
		return err
	}
	view, err := handler.noteUseCases.EditNote(request.Context(), toEditNoteCommand(noteIDFrom(request), requestDTO))
	if err != nil {
		return err
	}
	return handler.respond(responseWriter, request, http.StatusOK, toNoteResponseDTO(view))
}

func (handler NoteHandler) deleteNote(responseWriter http.ResponseWriter, request *http.Request) error {
	if err := handler.noteUseCases.DeleteNote(request.Context(), toDeleteNoteCommand(noteIDFrom(request))); err != nil {
		return err
	}
	responseWriter.WriteHeader(http.StatusNoContent)
	return nil
}

func (handler NoteHandler) archiveNote(responseWriter http.ResponseWriter, request *http.Request) error {
	view, err := handler.noteUseCases.ArchiveNote(request.Context(), toArchiveNoteCommand(noteIDFrom(request)))
	if err != nil {
		return err
	}
	return handler.respond(responseWriter, request, http.StatusOK, toNoteResponseDTO(view))
}

func (handler NoteHandler) restoreNote(responseWriter http.ResponseWriter, request *http.Request) error {
	view, err := handler.noteUseCases.RestoreNote(request.Context(), toRestoreNoteCommand(noteIDFrom(request)))
	if err != nil {
		return err
	}
	return handler.respond(responseWriter, request, http.StatusOK, toNoteResponseDTO(view))
}

func (handler NoteHandler) scheduleReminder(responseWriter http.ResponseWriter, request *http.Request) error {
	requestDTO, err := decodeRequest[ScheduleReminderRequestDTO](responseWriter, request, scheduleReminderRequestFields)
	if err != nil {
		return err
	}
	remindAt, err := time.Parse(time.RFC3339, requestDTO.RemindAt)
	if err != nil {
		return invalidRequest(FieldError{Message: invalidRemindAtMessage, Location: bodyLocation + locationSeparator + remindAtField})
	}
	view, err := handler.noteUseCases.ScheduleReminder(request.Context(), toScheduleReminderCommand(noteIDFrom(request), remindAt))
	if err != nil {
		return err
	}
	return handler.respond(responseWriter, request, http.StatusOK, toNoteResponseDTO(view))
}

func (handler NoteHandler) cancelReminder(responseWriter http.ResponseWriter, request *http.Request) error {
	view, err := handler.noteUseCases.CancelReminder(request.Context(), toCancelReminderCommand(noteIDFrom(request)))
	if err != nil {
		return err
	}
	return handler.respond(responseWriter, request, http.StatusOK, toNoteResponseDTO(view))
}

// noteIDFrom reads the ID from the path. A parameter keeps its percent-encoding
// only when Chi routes on the raw path, so decoding it any other time would
// decode it twice. A broken escape stays as it is, and matches no note.
func noteIDFrom(request *http.Request) string {
	noteID := chi.URLParam(request, noteIDParameter)
	if request.URL.RawPath == "" {
		return noteID
	}
	decoded, err := url.PathUnescape(noteID)
	if err != nil {
		return noteID
	}
	return decoded
}

func (handler NoteHandler) respond(responseWriter http.ResponseWriter, request *http.Request, status int, body any) error {
	return writeJSON(responseWriter, request, handler.logger, "application/json", status, body)
}

// writeJSON encodes before it sends the status, so a body that can't be encoded
// can still become the standard error response.
func writeJSON(responseWriter http.ResponseWriter, request *http.Request, logger *slog.Logger, contentType string, status int, body any) error {
	content, err := json.Marshal(body)
	if err != nil {
		return err
	}
	responseWriter.Header().Set("Content-Type", contentType)
	responseWriter.WriteHeader(status)
	if _, err := responseWriter.Write(append(content, '\n')); err != nil {
		ctx := request.Context()
		// A client leaving isn't a server fault.
		level := slog.LevelError
		if ctx.Err() != nil {
			level = slog.LevelInfo
		}
		logger.Log(ctx, level, logWriteFailed, logKeyRequestID, requestIDFrom(ctx), logKeyError, err)
	}
	return nil
}

func writeAPIError(responseWriter http.ResponseWriter, request *http.Request, logger *slog.Logger, apiError APIError) {
	apiError.RequestID = requestIDFrom(request.Context())
	// A response that failed must not point at a note.
	responseWriter.Header().Del("Location")
	// An APIError holds only strings and numbers, so it always encodes.
	_ = writeJSON(responseWriter, request, logger, "application/problem+json", apiError.Status, apiError)
}
