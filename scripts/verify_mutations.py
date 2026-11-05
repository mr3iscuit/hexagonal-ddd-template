#!/usr/bin/env python3
"""Breaks each business rule on a copy of the project and checks a test fails."""
import pathlib
import shutil
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
DOMAIN = "internal/notes/domain/"
APPLICATION = "internal/notes/application/"
MEMORY = "internal/notes/adapters/outbound/memory/"
SYSTEM = "internal/notes/adapters/outbound/system/"
HTTP = "internal/notes/adapters/inbound/httpapi/"

# Each mutation is (name, file, original text, broken text).
MUTATIONS = [
    ("title limit moves up", DOMAIN + "constants.go", "maximumTitleLength = 200", "maximumTitleLength = 201"),
    ("title limit moves down", DOMAIN + "constants.go", "maximumTitleLength = 200", "maximumTitleLength = 199"),
    ("body limit moves up", DOMAIN + "constants.go", "maximumBodyLength  = 10000", "maximumBodyLength  = 10001"),
    ("body limit moves down", DOMAIN + "constants.go", "maximumBodyLength  = 10000", "maximumBodyLength  = 9999"),
    ("title limit counts bytes", DOMAIN + "title.go", "utf8.RuneCountInString(trimmed) > maximumTitleLength", "len(trimmed)+0*utf8.RuneLen('a') > maximumTitleLength"),
    ("body limit counts bytes", DOMAIN + "body.go", "utf8.RuneCountInString(value) > maximumBodyLength", "len(value)+0*utf8.RuneLen('a') > maximumBodyLength"),
    ("title limit is off by one", DOMAIN + "title.go", "> maximumTitleLength", ">= maximumTitleLength"),
    ("body limit is off by one", DOMAIN + "body.go", "> maximumBodyLength", ">= maximumBodyLength"),
    ("a blank title is allowed", DOMAIN + "title.go", "if !strings.ContainsFunc(trimmed, isVisible) {", "if false {"),
    ("a title needs no visible character", DOMAIN + "title.go", "if !strings.ContainsFunc(trimmed, isVisible) {", 'if trimmed == "" {'),
    ("spaces count as visible", DOMAIN + "title.go", "unicode.In(character, unicode.L, unicode.N, unicode.P, unicode.S)", "unicode.IsGraphic(character)"),
    ("marks count as visible", DOMAIN + "title.go", "unicode.In(character, unicode.L, unicode.N, unicode.P, unicode.S)", "unicode.In(character, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S)"),
    ("symbols don't count as visible", DOMAIN + "title.go", "unicode.In(character, unicode.L, unicode.N, unicode.P, unicode.S)", "unicode.In(character, unicode.L, unicode.N, unicode.P)"),
    ("a blank ID is allowed", DOMAIN + "note_id.go", 'if strings.TrimSpace(value) == "" {', 'if strings.TrimSpace(value) == "never" {'),
    ("an ID is trimmed", DOMAIN + "note_id.go", "return NoteID{value: value}, nil", "return NoteID{value: strings.TrimSpace(value)}, nil"),
    ("archived note can be edited", DOMAIN + "note.go", "if note.archived {\n\t\treturn Note{}, ErrNoteArchived", "if false {\n\t\treturn Note{}, ErrNoteArchived"),
    ("an archived note can be archived", DOMAIN + "note.go", "if note.archived {\n\t\treturn Note{}, ErrNoteAlreadyArchived", "if false {\n\t\treturn Note{}, ErrNoteAlreadyArchived"),
    ("an active note can be restored", DOMAIN + "note.go", "if !note.archived {", "if false {"),
    ("archiving counts as an edit", DOMAIN + "note.go", "note.archived = true\n", "note.archived = true\n\tnote.editedAt = archivedAt\n"),
    ("restoring counts as an edit", DOMAIN + "note.go", "note.archived = false\n", "note.archived = false\n\tnote.editedAt = restoredAt\n"),
    ("editing keeps the old edit time", DOMAIN + "note.go", "note.editedAt = editedAt\n", ""),
    ("a change can have no time", DOMAIN + "note.go", "if changedAt.IsZero() {", "if false {"),
    ("a title can hold control characters", DOMAIN + "title.go", "strings.ContainsFunc(trimmed, isForbiddenInTitle)", "false && strings.ContainsFunc(trimmed, isForbiddenInTitle)"),
    ("a title can hold line separators", DOMAIN + "title.go", "unicode.In(character, unicode.Cc, unicode.Zl, unicode.Zp)", "unicode.In(character, unicode.Cc)"),
    ("a body can hold control characters", DOMAIN + "body.go", "strings.ContainsFunc(value, isForbiddenInBody)", "false && strings.ContainsFunc(value, isForbiddenInBody)"),
    ("a body can't hold line breaks", DOMAIN + "constants.go", '"\\t\\n\\r"', '"\\t"'),
    ("a body can hold NUL", DOMAIN + "constants.go", '"\\t\\n\\r"', '"\\t\\n\\r\\x00"'),
    ("a zero note can change", DOMAIN + "note.go", "if note.id.IsZero() {\n\t\treturn errNoteNotCreated", "if false {\n\t\treturn errNoteNotCreated"),
    ("reconstitution applies a business rule", DOMAIN + "note.go", "if version < 1 {", "if version < 1 || editedAt.Before(createdAt) {"),
    ("events are shared between copies", DOMAIN + "note.go", "note.pendingEvents = append(slices.Clone(note.pendingEvents), event)\n\treturn note", "note.pendingEvents = append(note.pendingEvents[:len(note.pendingEvents):len(note.pendingEvents)+0*len(slices.Clone(note.pendingEvents))], event)\n\tnote.pendingEvents[0] = event\n\treturn note"),
    ("oldest notes come first", DOMAIN + "reading_order.go", "second.editedAt.Compare(first.editedAt)", "first.editedAt.Compare(second.editedAt)"),
    ("ties have no order", DOMAIN + "reading_order.go", "return strings.Compare(first.id.value, second.id.value)", "return 0 * strings.Compare(first.id.value, second.id.value)"),
    ("use case skips saving an edit", APPLICATION + "note_application_service.go", "if err := service.noteRepository.Save(ctx, edited); err != nil {", "if err := error(nil); err != nil {"),
    ("edit loads before validating", APPLICATION + "note_application_service.go", "func (service NoteApplicationService) EditNote(ctx context.Context, command EditNoteCommand) (NoteViewDTO, error) {\n", "func (service NoteApplicationService) EditNote(ctx context.Context, command EditNoteCommand) (NoteViewDTO, error) {\n\t_, _ = service.load(ctx, command.NoteID)\n"),
    ("the archived filter is ignored", APPLICATION + "note_application_service.go", "List(ctx, query.Archived)", "List(ctx, false)"),
    ("listing skips the reading order", APPLICATION + "note_application_service.go", "toNoteViewDTOs(domain.OrderForReading(notes))", "toNoteViewDTOs(notes)"),
    ("version check is gone", MEMORY + "in_memory_note_repository.go", "if repository.records[note.ID().String()].version != note.Version() {", "if false {"),
    ("saving keeps the version", MEMORY + "in_memory_note_repository.go", "record.version = note.Version() + 1", "record.version = max(note.Version(), 1)"),
    ("an unsaved note can be removed", MEMORY + "in_memory_note_repository.go", "if note.Version() == 0 {", "if false {"),
    ("events are lost on save", MEMORY + "in_memory_note_repository.go", "repository.records[record.id] = record\n\trepository.storeEvents(note)", "repository.records[record.id] = record"),
    ("events are lost on remove", MEMORY + "in_memory_note_repository.go", "delete(repository.records, note.ID().String())\n\trepository.storeEvents(note)", "delete(repository.records, note.ID().String())"),
    ("the outbox has no cap", MEMORY + "in_memory_note_repository.go", "if len(repository.outbox) < maximumOutboxRecords {", "if true {"),
    ("the ring overwrites one place", MEMORY + "in_memory_note_repository.go", "(repository.oldestOutboxRecord + 1) % maximumOutboxRecords", "(repository.oldestOutboxRecord + 0) % maximumOutboxRecords"),
    ("the ring runs off its end", MEMORY + "in_memory_note_repository.go", "(repository.oldestOutboxRecord + 1) % maximumOutboxRecords", "repository.oldestOutboxRecord + 1"),
    ("a full outbox is copied", MEMORY + "in_memory_note_repository.go", "repository.outbox[repository.oldestOutboxRecord] = record\n", "repository.outbox = append([]outboxRecord{}, repository.outbox...)\n\t\trepository.outbox[repository.oldestOutboxRecord] = record\n"),
    ("the list ignores archived notes", MEMORY + "in_memory_note_repository.go", "if record.archived == archived {", "if !record.archived {"),
    ("damaged data is wrapped", MEMORY + "constants.go", '"stored note has a damaged title: %v"', '"stored note has a damaged title: %w"'),
    ("the note limit is gone", MEMORY + "in_memory_note_repository.go", "if note.Version() == 0 && len(repository.records) >= maximumNotes {", "if false && len(repository.records) >= maximumNotes {"),
    ("edits are refused at the note limit", MEMORY + "in_memory_note_repository.go", "if note.Version() == 0 && len(repository.records) >= maximumNotes {", "if len(repository.records) >= maximumNotes {"),
    ("one note too many fits", MEMORY + "in_memory_note_repository.go", "len(repository.records) >= maximumNotes {", "len(repository.records) > maximumNotes {"),
    ("one note too few fits", MEMORY + "in_memory_note_repository.go", "len(repository.records) >= maximumNotes {", "len(repository.records) >= maximumNotes-1 {"),
    ("a cancelled request reaches storage", MEMORY + "in_memory_note_repository.go", "func (repository *InMemoryNoteRepository) Save(ctx context.Context, note domain.Note) error {\n\tif err := ctx.Err(); err != nil {", "func (repository *InMemoryNoteRepository) Save(ctx context.Context, note domain.Note) error {\n\tif err := error(nil); err != nil {"),
    ("a refused archive is saved", APPLICATION + "note_application_service.go", "archived, err := note.Archive(service.clock.Now())\n\tif err != nil {", "archived, err := note.Archive(service.clock.Now())\n\tif false {"),
    ("a failed archive looks saved", APPLICATION + "note_application_service.go", "if err := service.noteRepository.Save(ctx, archived); err != nil {", "if err := service.noteRepository.Save(ctx, archived); false {"),
    ("a failed list looks empty", APPLICATION + "note_application_service.go", "notes, err := service.noteRepository.List(ctx, query.Archived)\n\tif err != nil {", "notes, err := service.noteRepository.List(ctx, query.Archived)\n\tif false {"),
    ("the clock strips monotonic time", SYSTEM + "system_clock.go", "return time.Now()", "return time.Now().UTC()"),
    ("responses are not in UTC", HTTP + "mapper.go", "CreatedAt: view.CreatedAt.UTC()", "CreatedAt: view.CreatedAt"),
    ("a null field counts as present", HTTP + "request_decoder.go", ' || string(value) == jsonNull', ' || (value == nil && string(value) == jsonNull)'),
    ("a repeated property is accepted", HTTP + "request_decoder.go", "if hasRepeatedProperty(content) {", "if false && hasRepeatedProperty(content) {"),
    ("a repeated property goes unseen", HTTP + "request_decoder.go", "seen[name] = true\n", "seen[name] = false\n"),
    ("an encoded ID stays encoded", HTTP + "note_handler.go", 'if request.URL.RawPath == "" {\n\t\treturn noteID', 'if true {\n\t\treturn noteID'),
    ("an ID is decoded twice", HTTP + "note_handler.go", 'if request.URL.RawPath == "" {\n\t\treturn noteID', 'if false {\n\t\treturn noteID'),
    ("a broken escape loses the ID", HTTP + "note_handler.go", "return noteID\n\t}\n\treturn decoded", 'return ""\n\t}\n\treturn decoded'),
    ("the status is sent before encoding", HTTP + "note_handler.go", "content, err := json.Marshal(body)\n", "responseWriter.WriteHeader(status)\n\tcontent, err := json.Marshal(body)\n"),
    ("an encoding failure is ignored", HTTP + "note_handler.go", "content, err := json.Marshal(body)\n\tif err != nil {", "content, err := json.Marshal(body)\n\tif err != nil && false {"),
    ("a created note has no Location", HTTP + "note_handler.go", 'responseWriter.Header().Set("Location", "/notes/"+url.PathEscape(view.ID))\n', ""),
    ("a failed response keeps its Location", HTTP + "note_handler.go", 'responseWriter.Header().Del("Location")\n', ""),
    ("the health check ignores a body", HTTP + "router.go", "noteHandler.handle(withoutBody(checkHealth))", "noteHandler.handle(checkHealth)"),
    ("the request log is dropped", HTTP + "router.go", "router.Use(assignRequestID, logRequests(logger), recoverPanics(logger))", "_ = logRequests\n\trouter.Use(assignRequestID, recoverPanics(logger))"),
    ("a wrapped error's text is sent", HTTP + "translate.go", "business.target.Error())", "err.Error())"),
    ("invalid UTF-8 is accepted", HTTP + "request_decoder.go", "if !utf8.Valid(content) {", "if !utf8.Valid(content) && false {"),
    ("lone surrogates are accepted", HTTP + "request_decoder.go", "err != nil || hasLoneSurrogate(content)", "err != nil || (false && hasLoneSurrogate(content))"),
    ("an escaped backslash reads as an escape", HTTP + "request_decoder.go", "\t\t\tposition++\n\t\t\tcontinue\n", "\t\t\tcontinue\n"),
    ("a surrogate pair reads as lone", HTTP + "request_decoder.go", "position += unicodeEscapeLength - 1", "position -= 1"),
    ("a broken query is accepted", HTTP + "request_decoder.go", "values, err := url.ParseQuery(request.URL.RawQuery)\n\tif err != nil {", "values, err := url.ParseQuery(request.URL.RawQuery)\n\tif err != nil && false {"),
    ("a client leaving reads as a broken body", HTTP + "request_decoder.go", "if cancelled := request.Context().Err(); cancelled != nil {", "if cancelled := request.Context().Err(); false {"),
    ("a failed list reaches the client as a success", HTTP + "note_handler.go", "views, err := handler.noteUseCases.ListNotes(request.Context(), toListNotesQuery(archived))\n\tif err != nil {", "views, err := handler.noteUseCases.ListNotes(request.Context(), toListNotesQuery(archived))\n\tif false {"),
    ("Allow uses the decoded path", HTTP + "router.go", 'if request.URL.RawPath != "" {', 'if false {'),
    ("an aborted request goes unlogged", HTTP + "request_log_middleware.go", "defer func() {\n\t\t\t\tlogger.InfoContext(request.Context(), logRequestHandled,\n\t\t\t\t\tlogKeyRequestID, requestIDFrom(request.Context()), logKeyMethod, request.Method, logKeyPath, request.URL.Path, logKeyStatus, wrapped.Status())\n\t\t\t}()\n\t\t\tnext.ServeHTTP(wrapped, request)\n", "next.ServeHTTP(wrapped, request)\n\t\t\tlogger.InfoContext(request.Context(), logRequestHandled,\n\t\t\t\tlogKeyRequestID, requestIDFrom(request.Context()), logKeyMethod, request.Method, logKeyPath, request.URL.Path, logKeyStatus, wrapped.Status())\n"),
    ("a failed write loses its request ID", HTTP + "note_handler.go", 'logger.Log(ctx, level, logWriteFailed, logKeyRequestID, requestIDFrom(ctx), logKeyError, err)', 'logger.Log(ctx, level, logWriteFailed, logKeyError, err)'),
    ("a client leaving is logged as a fault", HTTP + "note_handler.go", "if ctx.Err() != nil {", "if false {"),
    ("the Allow header is dropped", HTTP + "router.go", 'responseWriter.Header().Set("Allow", allowedMethods(router, routedPath(request)))', "_ = allowedMethods"),
    ("a reminder can be for right now", DOMAIN + "note.go", "if !remindAt.After(scheduledAt) {", "if remindAt.Before(scheduledAt) {"),
    ("a reminder can be in the past", DOMAIN + "note.go", "if !remindAt.After(scheduledAt) {", "if false {"),
    ("a reminder can be set on an archived note", DOMAIN + "note.go", "if note.archived {\n\t\treturn Note{}, ErrNoteArchived\n\t}\n\tif !remindAt.After(scheduledAt) {", "if false {\n\t\treturn Note{}, ErrNoteArchived\n\t}\n\tif !remindAt.After(scheduledAt) {"),
    ("a missing reminder can be cancelled", DOMAIN + "note.go", "if note.remindAt.IsZero() {\n\t\treturn Note{}, ErrNoReminderScheduled", "if false {\n\t\treturn Note{}, ErrNoReminderScheduled"),
    ("cancelling leaves the reminder", DOMAIN + "note.go", "note.remindAt = time.Time{}\n\treturn note.recording(ReminderCancelled", "return note.recording(ReminderCancelled"),
    ("archiving leaves the reminder", DOMAIN + "note.go", "if !note.remindAt.IsZero() {\n\t\tnote = note.withoutReminder(archivedAt)", "if false {\n\t\tnote = note.withoutReminder(archivedAt)"),
    ("archiving always records a cancellation", DOMAIN + "note.go", "if !note.remindAt.IsZero() {\n\t\tnote = note.withoutReminder(archivedAt)", "if true {\n\t\tnote = note.withoutReminder(archivedAt)"),
    ("triggering leaves the reminder", DOMAIN + "note.go", "note.remindAt = time.Time{}\n\treturn note.recording(ReminderTriggered", "return note.recording(ReminderTriggered"),
    ("scheduling counts as an edit", DOMAIN + "note.go", "note.remindAt = remindAt\n", "note.remindAt = remindAt\n\tnote.editedAt = scheduledAt\n"),
    ("cancelling counts as an edit", DOMAIN + "note.go", "note.remindAt = time.Time{}\n\treturn note.recording(ReminderCancelled", "note.remindAt = time.Time{}\n\tnote.editedAt = cancelledAt\n\treturn note.recording(ReminderCancelled"),
    ("triggering counts as an edit", DOMAIN + "note.go", "note.remindAt = time.Time{}\n\treturn note.recording(ReminderTriggered", "note.remindAt = time.Time{}\n\tnote.editedAt = triggeredAt\n\treturn note.recording(ReminderTriggered"),
    ("a reminder can trigger early", DOMAIN + "note.go", "if triggeredAt.Before(note.remindAt) {", "if false {"),
    ("exactly on time isn't due", DOMAIN + "note.go", "if triggeredAt.Before(note.remindAt) {", "if !triggeredAt.After(note.remindAt) {"),
    ("a missing reminder can trigger", DOMAIN + "note.go", "if note.remindAt.IsZero() {\n\t\treturn Note{}, ErrNoReminderScheduled\n\t}\n\tif triggeredAt", "if false {\n\t\treturn Note{}, ErrNoReminderScheduled\n\t}\n\tif triggeredAt"),
]


def run(directory):
    return subprocess.run(["go", "test", "./internal/..."], cwd=directory, capture_output=True, text=True)


def main():
    survivors = []
    with tempfile.TemporaryDirectory() as workspace:
        copy = pathlib.Path(workspace) / "project"
        shutil.copytree(ROOT, copy, ignore=shutil.ignore_patterns("bin", ".git"))
        if run(copy).returncode != 0:
            sys.exit("the tests fail before any mutation")
        for name, relative_path, original, broken in MUTATIONS:
            path = copy / relative_path
            source = path.read_text()
            if source.count(original) < 1:
                sys.exit(f"{name}: the original text is missing from {relative_path}")
            path.write_text(source.replace(original, broken, 1))
            built = subprocess.run(["go", "build", "./internal/..."], cwd=copy, capture_output=True, text=True)
            if built.returncode != 0:
                print(f"INVALID   {name}: {built.stderr.strip()}")
                survivors.append(name)
            elif run(copy).returncode == 0:
                print(f"SURVIVED  {name}")
                survivors.append(name)
            else:
                print(f"caught    {name}")
            path.write_text(source)
    print(f"\n{len(MUTATIONS) - len(survivors)} of {len(MUTATIONS)} mutations caught")
    sys.exit(1 if survivors else 0)


main()
