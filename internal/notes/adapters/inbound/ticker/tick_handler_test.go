package ticker_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
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

func TestTriggeredRemindersAreLogged(t *testing.T) {
	logs := &strings.Builder{}
	running := run(stubUseCases{trigger: func() (int, error) { return 2, nil }}, logs)
	running.tick(t)
	running.stop(t)
	if !strings.Contains(logs.String(), `msg="reminders triggered" count=2`) {
		t.Errorf("logs = %q", logs)
	}
}

func TestACheckThatFindsNothingStaysQuiet(t *testing.T) {
	logs := &strings.Builder{}
	running := run(stubUseCases{trigger: func() (int, error) { return 0, nil }}, logs)
	running.tick(t)
	running.stop(t)
	if logs.Len() != 0 {
		t.Errorf("logs = %q", logs)
	}
}

func TestAFailedCheckIsLoggedAndTheNextTickStillRuns(t *testing.T) {
	logs := &strings.Builder{}
	checks := 0
	running := run(stubUseCases{trigger: func() (int, error) {
		checks++
		return 0, errors.New("storage is down")
	}}, logs)
	running.tick(t)
	running.tick(t)
	running.stop(t)
	if checks != 2 {
		t.Errorf("checks = %d", checks)
	}
	for _, expected := range []string{"level=ERROR", "storage is down"} {
		if !strings.Contains(logs.String(), expected) {
			t.Errorf("%q is missing from the logs: %q", expected, logs)
		}
	}
}

func TestAShutdownDuringACheckIsNotAFailure(t *testing.T) {
	logs := &strings.Builder{}
	var cancel context.CancelFunc
	running := run(stubUseCases{trigger: func() (int, error) {
		cancel()
		return 0, context.Canceled
	}}, logs)
	cancel = running.cancel
	running.tick(t)
	running.stop(t)
	if logs.Len() != 0 {
		t.Errorf("logs = %q", logs)
	}
}

func TestAPanicDoesNotStopTheChecks(t *testing.T) {
	logs := &strings.Builder{}
	checks := 0
	running := run(stubUseCases{trigger: func() (int, error) {
		checks++
		if checks == 1 {
			panic("secret panic value")
		}
		return 0, nil
	}}, logs)
	running.tick(t)
	running.tick(t)
	running.stop(t)
	if checks != 2 {
		t.Errorf("checks = %d", checks)
	}
	for _, expected := range []string{"level=ERROR", "secret panic value"} {
		if !strings.Contains(logs.String(), expected) {
			t.Errorf("%q is missing from the logs: %q", expected, logs)
		}
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
