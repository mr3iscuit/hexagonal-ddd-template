package application

import "time"

type ScheduleReminderCommand struct {
	NoteID   string
	RemindAt time.Time
}
