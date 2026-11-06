# hexagonal-ddd-template

A Go microservice for taking notes, built with Domain-Driven Design and hexagonal architecture. Chi handles routing, and it stays inside the inbound adapter.

A note has a title and a body. It can be archived out of the way, restored and deleted. It can also have one reminder, which records an event when it comes due. The [glossary](docs/domain/notes-glossary.md) defines every term and rule, and lists the assumptions that still need confirming.

## Run

```bash
make run
```

The server listens on `:8080`. Set `ADDR` to change it.

```bash
curl -i localhost:8080/notes -H 'Content-Type: application/json' -d '{"title":"Groceries","body":"Milk, eggs"}'
```

Both fields are required, and the body may be an empty string.

| Task | Command |
| --- | --- |
| Run every test with the race detector | `make test` |
| Break each business rule and check that a test fails | `make mutations` |
| Build the binary | `make build` |
| Run `go vet` | `make lint` |

## API

| Method | Path | Purpose | Success |
| --- | --- | --- | --- |
| `POST` | `/notes` | Create a note | `201` |
| `GET` | `/notes` | List active notes in reading order | `200` |
| `GET` | `/notes?archived=true` | List archived notes in reading order | `200` |
| `GET` | `/notes/{id}` | Read a note | `200` |
| `PUT` | `/notes/{id}` | Replace a note's title and body | `200` |
| `DELETE` | `/notes/{id}` | Delete a note permanently | `204` |
| `POST` | `/notes/{id}/archive` | Archive a note | `200` |
| `POST` | `/notes/{id}/restore` | Restore an archived note | `200` |
| `PUT` | `/notes/{id}/reminder` | Schedule a reminder, replacing any earlier one | `200` |
| `DELETE` | `/notes/{id}/reminder` | Cancel the reminder | `200` |
| `GET` | `/healthz` | Health check | `204` |

`POST /notes` returns the new note's path in the `Location` header. `GET /notes` returns `{"notes":[...]}`, with the most recently edited note first. Each note looks like this:

```json
{
  "id": "JUFS25PSQHN5UF6FXJAHMUIU2M",
  "title": "Groceries",
  "body": "Milk, eggs",
  "archived": false,
  "createdAt": "2026-01-02T03:04:05Z",
  "editedAt": "2026-01-02T03:04:05Z"
}
```

A note with a reminder also has `remindAt`. `PUT /notes/{id}/reminder` takes `{"remindAt":"2026-11-07T09:00:00Z"}`, which must be an RFC 3339 time in the future.

Once a second, the service looks for reminders that have come due. It removes each one, and stores a `reminder.triggered` event in the outbox along with the note. Scheduling stores `reminder.scheduled`, and cancelling stores `reminder.cancelled`. Archiving a note cancels its reminder, and restoring the note doesn't bring it back.

## Errors

Every failure returns the same body, with the content type `application/problem+json`:

```json
{
  "title": "Conflict",
  "status": 409,
  "category": "business",
  "code": "NOTE_ARCHIVED",
  "detail": "an archived note cannot be edited or given a reminder",
  "requestId": "abc-000001"
}
```

`category` is `business` or `technical`. Check `code` in client code, as `detail` can change. Request validation failures add an `errors` list.

| Status | Code | When |
| --- | --- | --- |
| 404 | `NOTE_NOT_FOUND` | The note doesn't exist |
| 409 | `NOTE_ALREADY_ARCHIVED`, `NOTE_NOT_ARCHIVED` | The note is already in that archive state |
| 409 | `NOTE_ARCHIVED` | An archived note can't be edited or given a reminder |
| 409 | `NOTE_REMINDER_NOT_SCHEDULED` | The note has no reminder to cancel |
| 409 | `NOTE_MODIFIED_CONCURRENTLY` | Another request changed the note first, so retry |
| 409 | `NOTE_LIMIT_REACHED` | The server already holds 10,000 notes, so delete one first |
| 422 | `NOTE_ID_REQUIRED`, `NOTE_TITLE_REQUIRED` | A required value is blank, or a title has only invisible characters |
| 422 | `NOTE_TITLE_TOO_LONG`, `NOTE_BODY_TOO_LONG` | The title is over 200 characters, or the body is over 10,000 |
| 422 | `NOTE_REMINDER_NOT_IN_FUTURE` | A reminder's time isn't in the future |
| 422 | `NOTE_TITLE_HAS_CONTROL_CHARACTER` | The title isn't a single line, or it contains a control character |
| 422 | `NOTE_BODY_HAS_CONTROL_CHARACTER` | The body contains a control character other than a line break or tab |
| 400 | `MALFORMED_REQUEST` | The body can't be parsed, or it isn't valid UTF-8 |
| 404 | `ROUTE_NOT_FOUND` | No such route |
| 405 | `METHOD_NOT_ALLOWED` | Wrong method, and the `Allow` header lists the right ones |
| 408 | `REQUEST_CANCELLED` | The client left, or the deadline passed |
| 413 | `REQUEST_TOO_LARGE` | The body is over 128 KiB |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | The content type isn't JSON |
| 422 | `VALIDATION_FAILED` | The request or its query string has the wrong shape, such as a repeated property |
| 500 | `INTERNAL_ERROR` | Unexpected failure, which is logged with the request ID |

Send an `X-Request-Id` header to trace a request across services. It can have up to 128 letters, digits, `-`, `_` or `.`. Any other value is replaced with a random ID, and every response returns the ID in the same header.

## Layout

```text
cmd/api/                       composition root
docs/domain/                   glossary
docs/tdd/                      evidence for features built test-first
internal/notes/                bounded context
  domain/                      Note, value objects, events, rules
  application/                 use cases, ports, commands, queries, DTOs
  adapters/
    inbound/httpapi/           HTTP to use cases
    inbound/ticker/            clock ticks to the reminder check
    outbound/memory/           in-memory repository and outbox
    outbound/system/           clock and ID generator
scripts/                       mutation check
test/architecture/             the architecture rules, enforced as tests
test/e2e/                      the real binary over real HTTP
```

Dependencies point inward: adapters to application to domain. `test/architecture` fails if any rule is broken, and each rule is proved against a planted violation in `testdata`.

## Adding a Feature

Work from the inside out, because each layer depends only on the one inside it. Business rules go in the domain, never in a handler or a use case. A use case only runs the steps in order, as `CreateNote` does.

| Step | Where | What to add |
| --- | --- | --- |
| 1. Language | `docs/domain/notes-glossary.md` | Each new term, rule and event. A rule nobody has confirmed goes under Assumptions To Confirm |
| 2. Domain | `internal/notes/domain/` | A value object, or a behaviour on `Note` that returns a new `Note` and calls `requireChange`. A past-tense event, with its name in `constants.go`. Errors in `errors.go`. New state also goes through `ReconstituteNote` |
| 3. Storage | `internal/notes/adapters/outbound/memory/` | New state in `noteRecord`, `toRecord` and `toDomain`. A new kind of read or write also goes on `NoteRepositoryPort` |
| 4. Application | `internal/notes/application/` | A command or query in its own file, a method on `NoteUseCasesPort`, the use case on `NoteApplicationService`, and new output in `NoteViewDTO` and `mapper.go` |
| 5. HTTP | `internal/notes/adapters/inbound/httpapi/` | A route in `Register`, wrapped in `withoutBody` if it takes no body. Request and response DTOs, mappers in `mapper.go`, and for each new core error, a code in `constants.go` and a row in `businessErrors` |
| 6. Documents | `README.md` | The API and error tables, and any gap you chose to leave |

Each step is test-first, and each gets its own small commits:

1. Write one failing test, and watch it fail. Add the name first without the rule, so that it fails on an assertion rather than a compile error. Commit it as `test(<feature>): <behaviour>`, and quote the failure in the message.
2. Write the smallest code that passes, and commit it as `feat(<feature>): <behaviour>`.
3. If that left something worth tidying, commit it as `refactor(<feature>): <what>` while the tests stay green.

Name tests in domain language, such as `TestRejectsEditingAnArchivedNote`. Some changes need a particular test:

| Change | Test |
| --- | --- |
| A new rule | Both sides of any limit, and an entry in `MUTATIONS` in `scripts/verify_mutations.py` for each way to break it |
| A new behaviour on `Note` | An entry in `TestANoteThatWasNeverCreatedRefusesEveryBehaviour` and `TestEveryBehaviourRefusesAChangeWithNoTime` |
| A new mapper or field | A test that checks every field, because a forgotten field compiles and just comes through empty |
| A new failure | A row in `TestEveryFailureReturnsTheStandardErrorResponse`. A new exported error also fails `TestEveryCoreErrorHasARowInBusinessErrors` until `businessErrors` has its row |
| A new field in a note response | `TestANoteHasExactlyItsDocumentedFields` |
| A new field in a request | Its list beside the DTO, which `TestRequiredFieldListsMatchTheJSONTags` compares with the JSON tags |
| A new main flow | A step in `test/e2e/server_test.go`, which runs the real binary |

`test/architecture` checks the other rules for you, including names, one type per file, constants, pointers and imports. A failure there names the file and the rule.

Three kinds of feature work differently:

- **A new technology:** write an adapter that implements the existing port, named after its technology, such as `PostgresNoteRepository` in `adapters/outbound/postgres`. It must keep the port's promises, including the version check and storing events with the note. Only `cmd/api/main.go` changes, because it's the only place that picks adapters. A technology other than Postgres also needs adding to `outboundName` in `test/architecture/rules_test.go`.
- **A new way in:** write an inbound adapter that calls the same port as HTTP, such as `adapters/inbound/ticker`, and name it after what it receives from, ending in `Handler`. Background work needs its own panic recovery, because the HTTP middleware doesn't cover it.
- **A new business area:** give it its own bounded context in `internal/<context>/`, with the same folders and a glossary in `docs/domain/`, and wire it in `cmd/api/main.go`. Contexts never import each other.

### Worked Example

Reminders were built exactly this way, so their commits show every step above on a real feature:

```bash
git log --oneline --reverse --grep='(reminder)'
```

| Steps | Layer | What to look for |
| --- | --- | --- |
| 1 to 10 | Domain | One rule at a time, each with its glossary line. `refactor(reminder): share clearing a reminder` removes duplication only once it exists |
| 11 and 12 | Storage | A new record field, and a new method on `NoteRepositoryPort` |
| 13 to 15 | Application | Use cases that only run steps in order |
| 16 to 18 | HTTP | Routes, DTOs and rows in the failure table |
| 19 to 22 | Inbound adapter | A second way into the core, driven by ticks on a channel so that tests never wait |
| 23 | Composition root | The real binary firing a real reminder on the real clock |

Use `git show <commit>` to read a step, because each message quotes what the test did. `docs/tdd/note-reminders.tdd.md` lists every step with the failure it showed, and what each test guarantees.

## Maintaining the Repo

Nothing runs the checks automatically yet, so run `make test`, `make mutations` and `make lint` before every commit except a `test` commit, and start the service with `make run` if the change reaches the API. `make lint` only runs `go vet`, so run `staticcheck ./...` too if you have it installed. A `test` commit fails on purpose, so before one, check that only the new test fails, and that it fails on an assertion.

| Failure | Meaning | What to do |
| --- | --- | --- |
| A test in `test/architecture` | The code breaks a design rule | Fix the code. Change a rule only when the design itself changes |
| `SURVIVED` from `make mutations` | A rule can break without a test noticing | Add the missing test |
| `INVALID` from `make mutations` | The mutation doesn't compile | Fix its entry in `MUTATIONS` |
| `the original text is missing` from `make mutations` | The code a mutation targets has changed | Update its entry rather than deleting it |

Keep to these habits:

- **Whole changes:** a `test` commit holds the failing test, its glossary line and any name-only scaffolding, and the `feat` commit after it holds the code and its mutations. Never end a branch on a failing commit.
- **History:** don't squash these commits, because they're the evidence. `test` commits fail on purpose, so use `git bisect skip` on them.
- **Published names:** error codes, event names, routes and JSON fields are contracts, so never rename one. Add the new name, and remove the old one only when nothing uses it. `detail` text can change, because clients must not depend on it.
- **Documents:** the glossary must match the code, so if they disagree, fix one of them. Move an assumption into the rules once it's confirmed, and remove a known gap once it's closed.
- **Removing a feature:** remove it from every layer, along with its tests, mutations and documents. If a test only used it as an example of a change, move that test to another behaviour rather than losing it. Then search the repo for its name.
- **New architecture rules:** add each one to `rules()` with a planted violation in `test/architecture/testdata/violations`, because a rule that can't fail proves nothing.
- **Dependencies:** `domain` and `application` use only the standard library, and a framework stays in the inbound adapter. Run `go mod tidy` after adding or removing one.
- **Reliable tests:** a test must fail every time its rule breaks, so test time rules with a fake clock and give fakes a fixed order. Only call `t.Fatal` on the test goroutine.

## Known Gaps

- **Storage:** notes are kept in memory, so they are lost on restart, and the server holds at most 10,000.
- **Outbox:** events are stored with each change, but nothing publishes them. The outbox is a ring that keeps the latest 10,000 and drops the oldest.
- **Reminders:** nothing delivers one yet, because `reminder.triggered` only reaches the outbox. A reminder fires within about a second of its time, using the server's clock, and it's lost on restart along with its note.
- **Stale edits:** the version check only rejects requests that overlap on the server. A client that reads a note and sends `PUT` much later still overwrites any edit made in between, because the API doesn't expose the version.
- **Retries:** `POST /notes` isn't idempotent, so retrying after a lost response creates a second note.
- **Auth:** any caller can read, change or delete any note.
- **Lists:** `GET /notes` returns every note, with no pagination or search.
- **Tightened rules:** stored notes are rebuilt through the value objects, so lowering a length limit or forbidding more characters would make older saved notes look damaged. Migrate them first.
- **`HEAD`:** `GET` routes don't answer `HEAD`, so `HEAD /healthz` returns `405`.
