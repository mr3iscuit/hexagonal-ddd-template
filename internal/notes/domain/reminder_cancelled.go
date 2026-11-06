package domain

// ReminderCancelled records that a note's reminder was cancelled.
type ReminderCancelled struct{ noteEvent }

func (ReminderCancelled) EventName() string { return reminderCancelledEventName }
