package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

// businessErrors gives each core error its HTTP form. A new core error needs a
// row here, and a test fails if one is missing.
var businessErrors = []struct {
	target error
	status int
	code   string
}{
	{application.ErrNoteNotFound, http.StatusNotFound, codeNoteNotFound},
	{application.ErrNoteModifiedConcurrently, http.StatusConflict, codeNoteModifiedConcurrently},
	{application.ErrNoteLimitReached, http.StatusConflict, codeNoteLimitReached},
	{domain.ErrNoteIDRequired, http.StatusUnprocessableEntity, codeNoteIDRequired},
	{domain.ErrTitleRequired, http.StatusUnprocessableEntity, codeNoteTitleRequired},
	{domain.ErrTitleTooLong, http.StatusUnprocessableEntity, codeNoteTitleTooLong},
	{domain.ErrBodyTooLong, http.StatusUnprocessableEntity, codeNoteBodyTooLong},
	{domain.ErrTitleHasControlCharacter, http.StatusUnprocessableEntity, codeNoteTitleHasControlCharacter},
	{domain.ErrBodyHasControlCharacter, http.StatusUnprocessableEntity, codeNoteBodyHasControlCharacter},
	{domain.ErrNoteAlreadyArchived, http.StatusConflict, codeNoteAlreadyArchived},
	{domain.ErrNoteNotArchived, http.StatusConflict, codeNoteNotArchived},
	{domain.ErrNoteArchived, http.StatusConflict, codeNoteArchived},
	{domain.ErrReminderNotInFuture, http.StatusUnprocessableEntity, codeNoteReminderNotInFuture},
}

// translate turns any error into an APIError. Unknown errors are logged and
// hidden, because their text can contain internal details.
func translate(ctx context.Context, logger *slog.Logger, err error) APIError {
	var apiError APIError
	if errors.As(err, &apiError) {
		return apiError
	}
	for _, business := range businessErrors {
		if errors.Is(err, business.target) {
			// A wrapped error can quote client input, so only the core error's
			// fixed message reaches the client.
			return NewBusinessError(business.status, business.code, business.target.Error())
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		logger.InfoContext(ctx, logRequestCancelled, logKeyRequestID, requestIDFrom(ctx), logKeyError, err)
		return NewAPIError(http.StatusRequestTimeout, codeRequestCancelled, requestCancelledMessage)
	}
	logger.ErrorContext(ctx, logRequestFailed, logKeyRequestID, requestIDFrom(ctx), logKeyError, err)
	return NewAPIError(http.StatusInternalServerError, codeInternalError, internalErrorMessage)
}
