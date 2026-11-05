package domain

const (
	maximumTitleLength = 200
	maximumBodyLength  = 10000

	// A body may keep these control characters, because they lay out its text.
	allowedBodyControlCharacters = "\t\n\r"

	noteCreatedEventName  = "note.created"
	noteEditedEventName   = "note.edited"
	noteArchivedEventName = "note.archived"
	noteRestoredEventName = "note.restored"
	noteDeletedEventName  = "note.deleted"

	reminderScheduledEventName = "reminder.scheduled"
	reminderCancelledEventName = "reminder.cancelled"
)
