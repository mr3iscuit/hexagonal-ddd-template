package memory

import "time"

type outboxRecord struct {
	eventName  string
	noteID     string
	occurredAt time.Time
}
