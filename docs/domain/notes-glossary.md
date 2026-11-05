# Notes Glossary

The language of the `notes` bounded context. Names in the code must match this page.

## Terms

| Term | Code | Meaning |
| --- | --- | --- |
| Note | `Note` | The aggregate root: a piece of text a reader wants to keep |
| Note ID | `NoteID` | The identity of a note, which is never blank |
| Title | `Title` | The name of a note: required, trimmed, a single line with no control characters, at most 200 characters, and with at least one letter, number, punctuation mark or symbol |
| Body | `Body` | The text of a note: may be empty, keeps its whitespace, has no control characters apart from line breaks and tabs, and has at most 10,000 characters |
| Archived | `Note.IsArchived` | The note is put away: it's listed separately and can't be edited or given a reminder |
| Reminder | `Note.RemindAt` | A time when the reader wants to be prompted about a note. A note has at most one |
| Reading order | `OrderForReading` | The most recently edited notes first, then by ID so the order is stable |
| Event | `Event` | A fact that happened to a note |

## Behaviour

| Behaviour | Rule | Rejected with |
| --- | --- | --- |
| `Edit` | Replaces the title and body, and moves the edit time forward | `ErrNoteArchived`, `ErrTitleRequired` |
| `ScheduleReminder` | Sets the reminder, replacing any earlier one. Its time must be strictly after now, and the note must be active | `ErrReminderNotInFuture`, `ErrNoteArchived` |
| `CancelReminder` | Removes the reminder, so one must be scheduled | `ErrNoReminderScheduled` |
| `Archive` | Only an active note, and it cancels any reminder | `ErrNoteAlreadyArchived` |
| `Restore` | Only an archived note, and it doesn't bring a reminder back | `ErrNoteNotArchived` |
| `Delete` | Any note, whether active or archived | |

## Events

| Event | Published name | Happens when |
| --- | --- | --- |
| `NoteCreated` | `note.created` | A note was created |
| `NoteEdited` | `note.edited` | A note's title and body were replaced |
| `NoteArchived` | `note.archived` | A note was archived |
| `NoteRestored` | `note.restored` | A note was restored |
| `NoteDeleted` | `note.deleted` | A note was deleted |
| `ReminderScheduled` | `reminder.scheduled` | A reminder was set on a note |
| `ReminderCancelled` | `reminder.cancelled` | A note's reminder was removed |

## Assumptions To Confirm

No business rules were supplied for this service, so each of these is a choice that a domain expert should confirm.

- The limits of 200 and 10,000 characters are placeholders. Characters are Unicode code points, not bytes.
- Invisible characters don't count as a title: format characters, combining marks and spaces show nothing by themselves. Characters that look blank but count as letters or symbols, such as a Hangul filler, aren't detected.
- A title can't contain a tab, a line break, a Unicode line or paragraph separator, or any other control character. A body can contain tabs, line feeds, carriage returns and Unicode line separators, but no other control character.
- Archiving and restoring aren't edits, so they leave the edit time alone.
- An archived note is read-only, apart from restoring or deleting it.
- Editing a note to the same content still counts as an edit.
- Deleting is permanent, and any note can be deleted without archiving it first.
- A note has no owner, so every caller shares the same notes.
- The in-memory adapter holds at most 10,000 notes. This protects the process rather than being a business rule, so another adapter can set its own limit.
