package domain_test

import (
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
