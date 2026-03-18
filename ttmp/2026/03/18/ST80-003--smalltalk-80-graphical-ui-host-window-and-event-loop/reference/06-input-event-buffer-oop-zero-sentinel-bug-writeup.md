---
Title: Input Event Buffer OOP Zero Sentinel Bug Writeup
Ticket: ST80-003
Status: active
Topics:
    - input
    - bug
    - vm
    - ui
    - intern-review
DocType: reference
Intent: implementation
Owners: []
RelatedFiles:
    - Path: pkg/interpreter/interpreter.go
      Note: Buffered input primitives and deferred semaphore signaling path
    - Path: pkg/interpreter/interpreter_test.go
      Note: Focused regression that exposed the OOP-0 semaphore bug
    - Path: pkg/ui/ui.go
      Note: Host event loop that now feeds the active input buffer
Summary: Detailed writeup of the first-pass active-input bug where treating OOP 0 as unset suppressed semaphore signaling for queued input words.
LastUpdated: 2026-03-18T17:45:00-04:00
---

# Bug summary

While implementing the Blue Book active input-event buffer, I registered a freshly allocated `Semaphore` with `primitiveInputSemaphore:` and then queued a mouse-movement event. The input words were enqueued correctly, but the deferred semaphore signals were missing.

The cause was a bad sentinel assumption in my own Go code:

```go
if interp.inputSemaphore != 0 && interp.inputSemaphore != om.NilPointer {
    interp.asynchronousSignal(interp.inputSemaphore)
}
```

I treated OOP `0` as “unset”. In this VM, that is not safe. A newly allocated object can legitimately live at OOP `0`.

# Symptom

The first focused regression failed like this:

```text
--- FAIL: TestRecordMouseMotionQueuesTimedCoordinatesAndSignalsSemaphore (0.00s)
    interpreter_test.go:441: expected 3 deferred semaphore signals, got 0
```

At the same time:

- `inputWordCount == 3`
- the queued words were correct
- only the deferred `asynchronousSignal:` side effect was missing

That combination was the clue: the bug was not in event encoding or buffer insertion. It was in the guard around signaling the registered semaphore.

# Why this happened

The interpreter already uses `0` as a convenient internal “unset” value in several places. That made it tempting to reuse `0` as a generic “no object” sentinel.

But Smalltalk OOPs are not Go pointers. In this image/object memory:

- `nil` is OOP `2`
- OOP `0` is not reserved as “no object”
- an allocation can legitimately return OOP `0`

So this guard:

```go
interp.inputSemaphore != 0
```

was not checking “did the user register a semaphore?” It was accidentally checking “did the user register a semaphore whose OOP is not zero?”

# Exact fix

The fix was to stop treating OOP `0` as a globally invalid object in this path.

The corrected guard is:

```go
if interp.inputSemaphore != om.NilPointer {
    interp.asynchronousSignal(interp.inputSemaphore)
}
```

That matches the intended semantics much better:

- `nil` means “no semaphore registered”
- any real object OOP, including `0`, is eligible to be signaled

# Why this bug matters beyond this one primitive

This is the kind of bug that can recur anywhere the VM mixes:

- Go-native zero values
- VM object references
- hand-rolled “unset” state

The general rule is:

- use `nil` (`2`) when the Smalltalk-level sentinel is “no object”
- use separate booleans when the state is “initialized vs not initialized”
- do not assume OOP `0` is invalid unless the specific subsystem guarantees that

This same pattern is especially risky in:

- cached object references
- deferred signal/timer slots
- optional designated objects like display/cursor handles
- host integration state that defaults to zero in Go structs

# Validation

The regression that failed before the fix now passes:

- the queued input words are still present
- `semaphoreIndex` increments once per queued word
- the queue contents remain unchanged

The focused validation for this slice was:

```bash
go test ./pkg/interpreter -run 'TestPrimitive(InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord|MousePointReturnsConfiguredPoint|CursorLocPutUpdatesCursorAndReturnsReceiver|CursorLocPutUpdatesMouseWhenLinked)|TestRecord(MouseMotionQueuesTimedCoordinatesAndSignalsSemaphore|MouseMotionRespectsSampleInterval|DecodedKeyQueuesOnAndOffWords)|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles'
SDL_VIDEODRIVER=dummy go test ./pkg/ui ./cmd/st80-ui
```

# Review guidance for an intern

When reviewing this bug, do not focus only on the new input buffer. The more important lesson is about VM object references versus host-language defaults.

Check these three things together:

1. what value represents Smalltalk `nil`
2. whether OOP `0` can be a real allocated object
3. whether a Go-side zero value is being reused as VM-level meaning

If a guard mixes those up, the code can look completely reasonable and still be wrong.

# Follow-up

The event-buffer path is now working at the primitive level, but the broader cleanup item remains:

- audit other interpreter fields that still use raw zero as an implicit “unset” object reference

This is not necessarily a bug everywhere today, but it is the same failure pattern and worth keeping in mind during further UI/timer work.
