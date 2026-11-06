package domain

// ReminderScheduled records that a reminder was scheduled for a note.
type ReminderScheduled struct{ noteEvent }

func (ReminderScheduled) EventName() string { return reminderScheduledEventName }
