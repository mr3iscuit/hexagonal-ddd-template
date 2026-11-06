// Package ticker checks for due reminders on every clock tick.
package ticker

import (
	"context"
	"log/slog"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
)

// TickHandler is the inbound adapter for clock ticks. Ticks arrive on a channel,
// so tests can send them without waiting for real time to pass.
type TickHandler struct {
	noteUseCases application.NoteUseCasesPort
	logger       *slog.Logger
}

func NewTickHandler(noteUseCases application.NoteUseCasesPort, logger *slog.Logger) TickHandler {
	return TickHandler{noteUseCases: noteUseCases, logger: logger}
}

// Run checks for due reminders on every tick, until ctx is done.
func (handler TickHandler) Run(ctx context.Context, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			handler.checkReminders(ctx)
		}
	}
}

func (handler TickHandler) checkReminders(ctx context.Context) {
	_, _ = handler.noteUseCases.TriggerDueReminders(ctx)
}
