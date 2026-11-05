# hexagonal-ddd-template

A Go microservice for taking notes, built with Domain-Driven Design and hexagonal architecture. Chi handles routing, and it stays inside the inbound adapter.

A note has a title and a body. It can be archived out of the way, restored and deleted. The [glossary](docs/domain/notes-glossary.md) defines every term and rule, and lists the assumptions that still need confirming.

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

## Errors

Every failure returns the same body, with the content type `application/problem+json`:

```json
{
  "title": "Conflict",
  "status": 409,
  "category": "business",
  "code": "NOTE_ARCHIVED",
  "detail": "an archived note cannot be edited",
  "requestId": "abc-000001"
}
```

`category` is `business` or `technical`. Check `code` in client code, as `detail` can change. Request validation failures add an `errors` list.

| Status | Code | When |
| --- | --- | --- |
| 404 | `NOTE_NOT_FOUND` | The note doesn't exist |
| 409 | `NOTE_ALREADY_ARCHIVED`, `NOTE_NOT_ARCHIVED` | The note is already in that archive state |
| 409 | `NOTE_ARCHIVED` | An archived note can't be edited |
| 409 | `NOTE_MODIFIED_CONCURRENTLY` | Another request changed the note first, so retry |
| 409 | `NOTE_LIMIT_REACHED` | The server already holds 10,000 notes, so delete one first |
| 422 | `NOTE_ID_REQUIRED`, `NOTE_TITLE_REQUIRED` | A required value is blank, or a title has only invisible characters |
| 422 | `NOTE_TITLE_TOO_LONG`, `NOTE_BODY_TOO_LONG` | The title is over 200 characters, or the body is over 10,000 |
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
internal/notes/                bounded context
  domain/                      Note, value objects, events, rules
  application/                 use cases, ports, commands, queries, DTOs
  adapters/
    inbound/httpapi/           HTTP to use cases
    outbound/memory/           in-memory repository and outbox
    outbound/system/           clock and ID generator
scripts/                       mutation check
test/architecture/             the architecture rules, enforced as tests
test/e2e/                      the real binary over real HTTP
```

Dependencies point inward: adapters to application to domain. `test/architecture` fails if any rule is broken, and each rule is proved against a planted violation in `testdata`.

## Known Gaps

- **Storage:** notes are kept in memory, so they are lost on restart, and the server holds at most 10,000.
- **Outbox:** events are stored with each change, but nothing publishes them. The outbox is a ring that keeps the latest 10,000 and drops the oldest.
- **Stale edits:** the version check only rejects requests that overlap on the server. A client that reads a note and sends `PUT` much later still overwrites any edit made in between, because the API doesn't expose the version.
- **Retries:** `POST /notes` isn't idempotent, so retrying after a lost response creates a second note.
- **Auth:** any caller can read, change or delete any note.
- **Lists:** `GET /notes` returns every note, with no pagination or search.
- **Tightened rules:** stored notes are rebuilt through the value objects, so lowering a length limit or forbidding more characters would make older saved notes look damaged. Migrate them first.
- **`HEAD`:** `GET` routes don't answer `HEAD`, so `HEAD /healthz` returns `405`.
