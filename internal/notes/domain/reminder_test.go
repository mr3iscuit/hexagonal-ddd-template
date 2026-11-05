package domain_test

import (
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
