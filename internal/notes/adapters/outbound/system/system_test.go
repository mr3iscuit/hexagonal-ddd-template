package system_test

import (
	"testing"

	"github.com/mr3iscuit/hexagonal-ddd-template/internal/notes/adapters/outbound/system"
)

func TestTheClockKeepsItsMonotonicReading(t *testing.T) {
	if now := (system.SystemClock{}).Now(); now.Round(0).Equal(now) && now.String() == now.Round(0).String() {
		t.Errorf("%v has no monotonic reading", now)
	}
}

func TestGeneratedNoteIDsAreUnique(t *testing.T) {
	first, err := system.RandomIDGenerator{}.NextNoteID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := system.RandomIDGenerator{}.NextNoteID()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first.IsZero() {
		t.Errorf("first = %v, second = %v", first, second)
	}
}
