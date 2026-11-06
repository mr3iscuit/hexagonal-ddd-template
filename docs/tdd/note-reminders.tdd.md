# Note Reminders: TDD Evidence

This report indexes the commits that built reminders, and what they prove. It follows the `tdd-workflow` evidence report from [affaan-m/ECC](https://github.com/affaan-m/ECC), alongside the test-driven rules in the `hexagonal-ddd` skill.

## Source

An approved plan, with two defaults that were never contradicted: the event fires when a reminder comes due, and a note has one reminder, held on the `Note` aggregate.

## User Journeys

- As a note taker, I want to set a reminder on a note, so that I'm prompted about it at a time I choose.
- As a note taker, I want to change or cancel a reminder, so that I'm not prompted about something that no longer matters.
- As a service that delivers notifications, I want an event when a reminder comes due, so that I can tell the reader.

## Task Report

Each row is one `test` commit, which was run and failed as shown, followed by one `feat` commit. Each commit message quotes its own command and result.

| # | Layer | Behaviour | RED | Failure seen | GREEN |
| --- | --- | --- | --- | --- | --- |
| 1 | Domain | scheduling keeps the reminder's time | `e4f18a3` | `remindAt = 0001-01-01 00:00:00 +0000 UTC, want 2026-01-02 05:04:05 +0000 UTC` | `a257abd` |
| 2 | Domain | scheduling records ReminderScheduled | `73a99b8` | `events = [note.created]` | `47c694d` |
| 3 | Domain | a reminder must be in the future | `e72f864` | `err = <nil>, want a reminder must be in the future` | `3dc4fbf` |
| 4 | Domain | an archived note can't get a reminder | `b409eca` | `err = <nil>` | `7b97c72` |
| 5 | Domain | cancelling clears it and records ReminderCancelled | `1b7c657` | `remindAt = 2026-01-02 05:04:05 +0000 UTC` | `114d0ef` |
| 6 | Domain | cancelling with no reminder is refused | `35eab34` | `err = <nil>` | `06df2a0` |
| 7 | Domain | archiving cancels the reminder | `ccaea7c` | `remindAt = 2026-01-02 05:04:05 +0000 UTC` | `09e6a21` |
| 8 | Domain | a due reminder triggers and records ReminderTriggered | `9ba0423` | `remindAt = 2026-01-02 05:04:05 +0000 UTC` | `64281a1` |
| 9 | Domain | a reminder can't trigger early | `564ebe9` | `the reminder triggered early` | `76dc01e` |
| 10 | Domain | a zero note or a missing time is refused | `dc555b3` | `err = the note has no reminder must not look like a client's mistake` | `9ac9bb3` |
| 11 | Storage | a reminder survives a save and a load | `5a679aa` | `got {id:soon title:Soon body: archived:false remindAt:{wall:0 ext:0 loc:<nil>} creat...` | `ec78ee7` |
| 12 | Storage | storage finds the reminders that are due | `bb049ed` | `ids = [], want [exactly-due overdue]` | `f78105e` |
| 13 | Application | the schedule use case stores the event | `1f976a1` | `got {ID:groceries Title:Groceries Body:Milk Archived:true RemindAt:0001-01-01 00:00:...` | `06978cb` |
| 14 | Application | the cancel use case stores the event | `c3ef31e` | `view = {ID: Title: Body: Archived:false RemindAt:0001-01-01 00:00:00 +0000 UTC` | `e84fb66` |
| 15 | Application | the trigger use case fires due reminders | `7458f0d` | `triggered = 0` | `50f6b4b` |
| 16 | HTTP | responses show the reminder only when set | `80a93d8` | `got {ID:id Title:a Body:b Archived:true RemindAt:0001-01-01 00:00:00 +0000 UTC Creat...` | `0593239` |
| 17 | HTTP | PUT schedules a reminder | `6e8b16c` | `got {NoteID: RemindAt:0001-01-01 00:00:00 +0000 UTC}` | `bd954da` |
| 18 | HTTP | DELETE cancels a reminder | `19ed8ff` | `a mapper lost the note ID` | `081e4bb` |
| 19 | Adapter | every tick triggers due reminders | `2c114ad` | `nothing read the tick` | `6d4add2` |
| 20 | Adapter | triggered reminders are logged | `5c2302c` | `logs = ""` | `6cd7bfc` |
| 21 | Adapter | a failed check is logged and the next tick still runs | `9f973b6` | `"level=ERROR" is missing from the logs: ""` | `f3c7870` |
| 22 | Adapter | a panic doesn't stop the checks | `2fdc747` | `nothing read the tick` | `575cb3d` |
| 23 | Root | the running service fires a real reminder | `190a9a2` | `the reminder never fired: {"time":"2026-11-06T04:07:14.452386389+04:00","level":"INF...` | `e5c570c` |

## Refactors

| Commit | Change |
| --- | --- |
| `6c637fc` | share clearing a reminder |
| `f0ea6a1` | share record filtering and rebuilding |
| `c9f6c37` | share starting the real binary |

## Tests That Passed On Arrival

A test that passes first proves nothing until it's seen to fail, so each of these was proved another way:

| Test | How it was proved |
| --- | --- |
| `TestRemindersDoNotCountAsEdits` | Three mutations make each behaviour move the edit time |
| `TestACheckThatFindsNothingStaysQuiet` | The mutation `an empty check is logged` |
| `TestAShutdownDuringACheckIsNotAFailure` | The mutation `a shutdown is logged as a failure` |
| `TestADueReminderReachesTheOutbox` | Seen failing with `storeEvents` removed from `Save`, as its commit records |

## What The Checks Caught

- The first attempt at the storage step used a parameter named `at`. The architecture test refused it as not a full word, so that step was redone with `now`, and no commit recorded the mistake.
- The mutation `a conflict stops the reminder check` didn't compile at first, because it left `errors` unused. `make mutations` reported it as `INVALID`, and its entry was fixed before the `feat` commit.
- The plan's step 19 turned out to do four separate things, so it became steps 19 to 22.

## Guarantees

| # | What is guaranteed | Test | Type | Result |
| --- | --- | --- | --- | --- |
| 1 | A reminder keeps its time, and scheduling again replaces it | `TestSchedulingAgainReplacesTheReminder` | unit | PASS |
| 2 | Scheduling, cancelling and triggering each record their event | `TestADueReminderTriggers` | unit | PASS |
| 3 | A reminder at now or 1ns ago is refused, and 1ns ahead is accepted | `TestAReminderMustBeInTheFuture` | unit | PASS |
| 4 | An archived note can't get a reminder | `TestRejectsSchedulingAReminderOnAnArchivedNote` | unit | PASS |
| 5 | Archiving cancels a reminder, and restoring doesn't bring it back | `TestRestoringDoesNotBringAReminderBack` | unit | PASS |
| 6 | A reminder can't fire 1ns early, and exactly on time counts | `TestAReminderCannotTriggerEarly` | unit | PASS |
| 7 | Reminder behaviour never moves the edit time | `TestRemindersDoNotCountAsEdits` | unit | PASS |
| 8 | A reminder survives a save and a load | `TestRecordRoundTripKeepsEveryField` | integration | PASS |
| 9 | Storage finds due reminders at the exact boundary | `TestListingDueRemindersFindsOnlyDueNotes` | integration | PASS |
| 10 | A triggered event reaches the outbox through the real use case and storage | `TestADueReminderReachesTheOutbox` | integration | PASS |
| 11 | An overlapping change skips only that note | `TestAConflictSkipsOnlyThatReminder` | unit | PASS |
| 12 | A storage failure stops the check | `TestAStorageFailureStopsTheReminderCheck` | unit | PASS |
| 13 | `PUT` schedules a reminder, and shows it in UTC | `TestSchedulingAReminderOverHTTP` | integration | PASS |
| 14 | `DELETE` cancels a reminder | `TestCancellingAReminderOverHTTP` | integration | PASS |
| 15 | Every reminder failure returns the standard error response | `TestEveryFailureReturnsTheStandardErrorResponse` | integration | PASS |
| 16 | A bad time is reported at `body.remindAt` without echoing it | `TestValidationErrorsNameEachProblemWithoutEchoingInput` | integration | PASS |
| 17 | Every tick checks once, and shutdown stops the checks | `TestEveryTickTriggersDueReminders` | unit | PASS |
| 18 | A failed check is logged, and the next tick still runs | `TestAFailedCheckIsLoggedAndTheNextTickStillRuns` | unit | PASS |
| 19 | A panic doesn't stop the checks or end the process | `TestAPanicDoesNotStopTheChecks` | unit | PASS |
| 20 | The real binary fires a real reminder on the real clock | `TestTheRealBinaryTriggersADueReminder` | e2e | PASS |

## Coverage And Verification

These results come from the last code commit, before this report was added:

| Check | Result |
| --- | --- |
| `go test -race -count=3 ./...` | All 8 packages pass |
| `go test -count=2 -shuffle=on ./...` | All 8 packages pass |
| `go vet ./...` and `staticcheck ./...` | Clean |
| `make mutations` | 125 of 125 caught, up from 82 |
| Every commit tested from a clean export | 51 of 51 as expected: 23 `test` commits failed, and the other 28 passed |
| Real binary | A reminder fired on the real clock and logged once, each refusal returned its code, and the server exited cleanly with no error logs |

| Package | Coverage |
| --- | --- |
| `domain` | 100% |
| `application` | 100% |
| `adapters/inbound/httpapi` | 99.6% |
| `adapters/inbound/ticker` | 100% |
| `adapters/outbound/memory` | 100% |
| `adapters/outbound/system` | 100% |

The one uncovered statement is a fallback in `request_decoder.go` that can't be reached. It existed before this feature.

**Known gaps:** nothing delivers `reminder.triggered` yet, because the outbox has no reader. Reminders are lost on restart along with their notes. The README's Known Gaps section lists these.

## Merging

Merge with `git merge --ff-only`, and don't squash, because these commits are the worked example the README points at. The `test` commits fail on purpose, so `git bisect` should skip them.
