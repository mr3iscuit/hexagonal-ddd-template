package application

// ListNotesQuery selects either the active notes or the archived ones.
type ListNotesQuery struct {
	Archived bool
}
