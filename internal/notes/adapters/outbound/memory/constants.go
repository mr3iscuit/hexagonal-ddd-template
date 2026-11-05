package memory

// maximumOutboxRecords caps the outbox, because nothing reads it yet and it
// would otherwise grow until the process dies.
const maximumOutboxRecords = 10000

// maximumNotes caps the notes the adapter holds, because the API has no auth,
// and a loop of creates would otherwise grow the map until the process dies.
const maximumNotes = 10000

const (
	damagedIDFormat    = "stored note has a damaged ID: %v"
	damagedTitleFormat = "stored note has a damaged title: %v"
	damagedBodyFormat  = "stored note has a damaged body: %v"
	damagedNoteFormat  = "stored note is damaged: %v"
)
