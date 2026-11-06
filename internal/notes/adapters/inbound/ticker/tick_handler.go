// Package ticker checks for due reminders on every clock tick.
package ticker

import (
	"context"
	"log/slog"
	"runtime/debug"
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

// checkReminders recovers a panic, because no middleware runs here, and a panic
// would otherwise end the process and every note it holds in memory.
func (handler TickHandler) checkReminders(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			handler.logger.ErrorContext(ctx, logCheckPanicked, logKeyPanic, recovered, logKeyStack, string(debug.Stack()))
		}
	}()
	triggered, err := handler.noteUseCases.TriggerDueReminders(ctx)
	// A check cut short by shutdown isn't a fault.
	if err != nil && ctx.Err() == nil {
		handler.logger.ErrorContext(ctx, logCheckFailed, logKeyError, err)
	}
	if triggered > 0 {
		handler.logger.InfoContext(ctx, logRemindersTriggered, logKeyCount, triggered)
	}
}
