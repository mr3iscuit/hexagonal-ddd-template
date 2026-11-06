package domain

// ReminderTriggered records that a note's reminder came due.
type ReminderTriggered struct{ noteEvent }

func (ReminderTriggered) EventName() string { return reminderTriggeredEventName }
