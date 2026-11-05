package memory

import "time"

type noteRecord struct {
	id        string
	title     string
	body      string
	archived  bool
	createdAt time.Time
	editedAt  time.Time
	version   int
}
