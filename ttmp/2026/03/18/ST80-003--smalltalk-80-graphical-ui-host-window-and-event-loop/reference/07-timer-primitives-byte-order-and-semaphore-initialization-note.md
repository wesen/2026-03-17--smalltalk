---
Title: Timer Primitives Byte Order and Semaphore Initialization Note
Ticket: ST80-003
Status: active
Topics:
    - timer
    - bug
    - vm
    - intern-review
DocType: reference
Intent: implementation
Owners: []
RelatedFiles:
    - Path: pkg/interpreter/interpreter.go
      Note: Host clock/timer primitives and deferred scheduler wakeup path
    - Path: pkg/interpreter/interpreter_test.go
      Note: Direct timer primitive tests and the corrected Semaphore fixture helper
Summary: Notes for intern review on the host timer primitive implementation, the chosen 32-bit byte order, and the fresh-Semaphore test initialization pitfall.
LastUpdated: 2026-03-18T18:10:00-04:00
---

# What was implemented

This slice implemented the host clock/timer primitives:

- primitive `98`
- primitive `99`
- primitive `100`

In the current image, those correspond to the time/millisecond/timer paths used by `Time class` and `ProcessorScheduler`.

The implementation now does three things:

1. writes a 32-bit second clock into a byte-indexable object
2. writes a 32-bit millisecond clock into a byte-indexable object
3. arms or immediately signals a timer semaphore based on a millisecond deadline

# Byte order decision

The Blue Book says these primitives store unsigned 32-bit integers into the first four bytes of the argument object. It does not spell out the host endianness in the same concrete way it spells out the 16-bit positive-integer boxing helper.

The implementation in this repo stores those four bytes in little-endian order:

- byte 0 = low 8 bits
- byte 1 = next 8 bits
- byte 2 = next 8 bits
- byte 3 = high 8 bits

Why this was chosen:

- it matches the existing 16-bit positive integer helper already used elsewhere in the VM
- it keeps the read/write helper pair internally consistent
- the timer primitive (`100`) reads back the same representation that the clock primitive (`99`) writes

This is a reasonable and self-consistent choice, but it is still worth keeping under review as more live image paths exercise these primitives.

# Timer scheduling path

The timer wakeup is not delivered directly inside the primitive on every interpreter step. Instead:

- primitive `100` stores:
  - the waiting semaphore
  - the deadline tick value
  - whether a timer is active
- `checkProcessSwitch` checks whether the deadline has passed
- if it has, the interpreter queues an asynchronous semaphore signal
- the existing scheduler path then converts that into the normal synchronous signal behavior

This matters because it keeps timer wakeups aligned with the rest of the VM’s asynchronous-signal model.

# The first failing test and what it really meant

The first future-timer test failed with a surprising double-signal count:

```text
expected scheduled timer to signal semaphore once, got excessSignals=2
```

At first glance, that looked like the timer implementation might be signaling twice.

That was not the real problem.

The test was constructing a fresh `Semaphore` with:

```go
interp.instantiateClassWithPointers(om.ClassSemaphorePointer, 3)
```

For pointer objects, the allocator initializes fields to `nil`.

That means the new Semaphore started as:

- `firstLink = nil`
- `lastLink = nil`
- `excessSignals = nil`

But `synchronousSignal` expects `excessSignals` to be a SmallInteger count, not `nil`.

So the failure was a fixture-construction bug. The correct test helper explicitly initializes:

- `firstLink = nil`
- `lastLink = nil`
- `excessSignals = 0`

# Why an intern should care

There are two separate lessons here.

First:

- when the image and host exchange raw multi-byte values, document and test byte order explicitly

Second:

- when using allocator-created objects as test fixtures, do not assume the default field values are semantically valid for that class

Those are easy mistakes to make in VM work because both failures can look like “the runtime logic is wrong” when the actual issue is representation or fixture state.

# Review checklist

When reviewing this slice, check:

1. that `storeUint32LE` and `fetchUint32LE` are inverses
2. that primitives `98` and `99` return the filled target object
3. that primitive `100`:
   - signals immediately when the deadline is already past
   - arms a future deadline otherwise
   - clears the timer if the first argument is not a valid `Semaphore`
4. that the timer fires through `checkProcessSwitch`, not through an ad hoc side path
5. that tests use a properly initialized `Semaphore`

# Follow-up

This note does not claim the full image-level timer behavior is finished forever. The next thing to verify is the live Smalltalk behavior:

- do `Delay` and scheduler wakeups behave correctly in the running UI?

That requires a real runtime/session check, not just direct unit coverage.
