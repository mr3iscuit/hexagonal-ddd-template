package memory

import "time"

type noteRecord struct {
	id        string
	title     string
	body      string
	archived  bool
	remindAt  time.Time
	createdAt time.Time
	editedAt  time.Time
	version   int
}
