package domain_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/domain"
)

var remindAt = changedAt.Add(time.Hour)

func withReminder(t *testing.T) domain.Note {
	t.Helper()
	note, err := newNote(t).ScheduleReminder(remindAt, changedAt)
	if err != nil {
		t.Fatal(err)
	}
	return note
}

func TestSchedulingAReminderKeepsItsTime(t *testing.T) {
	if note := withReminder(t); !note.RemindAt().Equal(remindAt) {
		t.Errorf("remindAt = %v, want %v", note.RemindAt(), remindAt)
	}
}

func TestSchedulingAReminderRecordsItsEvent(t *testing.T) {
	note := withReminder(t)
	if !slices.Equal(eventNames(note), []string{"note.created", "reminder.scheduled"}) {
		t.Fatalf("events = %v", eventNames(note))
	}
	if event := note.PendingEvents()[1]; event.NoteID() != note.ID() || !event.OccurredAt().Equal(changedAt) {
		t.Errorf("event = %+v", event)
	}
}

func TestAReminderMustBeInTheFuture(t *testing.T) {
	testCases := []struct {
		name     string
		remindAt time.Time
		want     error
	}{
		{"one nanosecond ago", changedAt.Add(-time.Nanosecond), domain.ErrReminderNotInFuture},
		{"exactly now", changedAt, domain.ErrReminderNotInFuture},
		{"one nanosecond from now", changedAt.Add(time.Nanosecond), nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := newNote(t).ScheduleReminder(testCase.remindAt, changedAt); !errors.Is(err, testCase.want) {
				t.Errorf("err = %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestRejectsSchedulingAReminderOnAnArchivedNote(t *testing.T) {
	if _, err := archived(t).ScheduleReminder(remindAt, changedAt); !errors.Is(err, domain.ErrNoteArchived) {
		t.Errorf("err = %v", err)
	}
}

func TestCancellingAReminderClearsIt(t *testing.T) {
	scheduled := withReminder(t)
	note, err := scheduled.CancelReminder(changedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !note.RemindAt().IsZero() {
		t.Errorf("remindAt = %v", note.RemindAt())
	}
	if !slices.Equal(eventNames(note), []string{"note.created", "reminder.scheduled", "reminder.cancelled"}) {
		t.Errorf("events = %v", eventNames(note))
	}
	if !scheduled.RemindAt().Equal(remindAt) {
		t.Error("the original note changed")
	}
}

func TestRejectsCancellingWithoutAReminder(t *testing.T) {
	if _, err := newNote(t).CancelReminder(changedAt); !errors.Is(err, domain.ErrNoReminderScheduled) {
		t.Errorf("err = %v", err)
	}
}

func TestSchedulingAgainReplacesTheReminder(t *testing.T) {
	later := remindAt.Add(time.Hour)
	note, err := withReminder(t).ScheduleReminder(later, changedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !note.RemindAt().Equal(later) {
		t.Errorf("remindAt = %v, want %v", note.RemindAt(), later)
	}
}
