package ticker_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/inbound/ticker"
	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/application"
)

const patience = time.Second

type stubUseCases struct {
	application.NoteUseCasesPort
	trigger func() (int, error)
}

func (stub stubUseCases) TriggerDueReminders(context.Context) (int, error) {
	return stub.trigger()
}

type runningHandler struct {
	ticks    chan time.Time
	cancel   context.CancelFunc
	finished chan struct{}
}

func run(useCases stubUseCases, logs io.Writer) runningHandler {
	ctx, cancel := context.WithCancel(context.Background())
	running := runningHandler{ticks: make(chan time.Time), cancel: cancel, finished: make(chan struct{})}
	handler := ticker.NewTickHandler(useCases, slog.New(slog.NewTextHandler(logs, nil)))
	go func() {
		defer close(running.finished)
		// A panic that escapes must fail a test, not end the test binary.
		defer func() { _ = recover() }()
		handler.Run(ctx, running.ticks)
	}()
	return running
}

func (running runningHandler) tick(t *testing.T) {
	t.Helper()
	select {
	case running.ticks <- time.Time{}:
	case <-time.After(patience):
		t.Fatal("nothing read the tick")
	}
}

func (running runningHandler) stop(t *testing.T) {
	t.Helper()
	running.cancel()
	select {
	case <-running.finished:
	case <-time.After(patience):
		t.Fatal("the handler did not stop")
	}
}

func TestEveryTickTriggersDueReminders(t *testing.T) {
	checks := 0
	running := run(stubUseCases{trigger: func() (int, error) {
		checks++
		return 0, nil
	}}, io.Discard)
	running.tick(t)
	running.tick(t)
	running.stop(t)
	if checks != 2 {
		t.Errorf("checks = %d", checks)
	}
}
