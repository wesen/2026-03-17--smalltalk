---
Title: Headless copyBits Quiescent Loop Writeup
Ticket: ST80-002
Status: active
Topics:
    - vm
    - smalltalk
    - sdl
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go
      Note: Temporary headless `primitiveCopyBits` success path used to move the image out of the notifier/debugger chain and into a stable scheduler loop (commit c1384ff)
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go
      Note: Long-run state probe showing the image in a priority-1 ProcessorScheduler loop rather than a notifier/debugger path
ExternalSources: []
Summary: Intern-facing writeup of the tactical headless `copyBits` implementation that removed the immediate `primitiveFailed` notifier path and produced a stable quiescent scheduler loop, but is not yet a real BitBlt/display implementation.
LastUpdated: 2026-03-18T13:05:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Headless copyBits Quiescent Loop Writeup

## Goal

Explain why the temporary `primitiveCopyBits` change in commit `c1384ff` is useful, what evidence it unlocked, and why it must not be confused with a finished BitBlt implementation.

## Context

After fixing the long-run notifier-path primitives (`perform:`, `beCursor`, `cursorLink:`, `asOop`), the two-million-cycle state probe still ended inside:
- `BitBlt>>copyBitsAgain`
- `BitBlt>>copyBits`
- `Object>>primitiveFailed`
- `NotifierView ...`

That was the next real frontier. The image was no longer failing on generic control/input/runtime gaps. It was failing in the display pipeline itself.

At that point there were two choices:

1. implement a real BitBlt engine immediately
2. first test whether the image can otherwise reach a quiescent loop if `copyBits` simply succeeds in headless mode

I chose the second option first because it is a fast way to separate:
- "is the image still blocked on notifier/debugger churn?"

from:
- "is a fully correct graphical BitBlt engine required before the rest of the VM can settle?"

## The Tactical Change

`primitiveCopyBits` now:
- pops the BitBlt receiver
- pushes it back
- reports primitive success

It does **not**:
- copy pixels
- update any display bitmap
- perform clipping
- apply combination rules
- handle halftone forms

So this is not a real BitBlt implementation. It is a tactical headless success path.

## Result

The result was significant:

- `go test ./...` remains green
- `go test ./pkg/interpreter -run TestLogStateAtTwoMillionCycles -v` no longer reports a notifier/debugger sender chain
- `go run ./cmd/st80 data/VirtualImage 5000000` runs cleanly

More importantly, the long-run state changed qualitatively.

Instead of living in a notifier/debugger chain, the VM now sits in a small repeating loop:

```text
[cycle 500000]  ctx=0x6664 ip=12 sp=5 bc=153 method=0x666E rcvr=0x6626
[cycle 1000000] ctx=0x6664 ip=11 sp=6 bc=113 method=0x666E rcvr=0x6626
[cycle 1500000] ctx=0x6664 ip=10 sp=5 bc=163 method=0x666E rcvr=0x6626
...
```

The associated long-run state probe shows:
- receiver class `0x6626` = `ProcessorScheduler`
- active process priority `1`
- active process list `0x6670`
- a one-frame sender chain with `sender = nil`

That is strong evidence that the image has reached a low-priority scheduler/quiescent loop rather than an active error-reporting path.

## Why This Matters

This tactical primitive answered an important question:

- yes, removing the immediate `copyBits` primitive failure is enough to let the image settle into a stable long-run loop

That is valuable because it means:
- the interpreter core is now much closer to quiescent correctness
- the remaining display work can be treated as real output/rendering work, not as a blocker that still prevents the image from settling at all

## What This Does Not Mean

It does **not** mean BitBlt is done.

The current implementation is insufficient for:
- real rendering
- visual correctness
- any future graphical UI ticket that expects the image to draw correctly

Before the interpreter/runtime can be called complete in a graphical sense, the temporary `copyBits` stub must be replaced with a real implementation or an equivalently faithful bridge to host rendering.

## Minimal Review Checklist

If an intern reviews this step later, they should verify:

1. `primitiveCopyBits` is explicitly documented as tactical/headless.
2. The post-fix long-run state is a scheduler loop, not a notifier/debugger chain.
3. No one mistakes this change for a final graphics solution.
4. The next tasks explicitly call for replacing this with real BitBlt/display semantics before a UI ticket proceeds.

## Validation Commands

```bash
go test ./...
go test ./pkg/interpreter -run TestLogStateAtTwoMillionCycles -v
go run ./cmd/st80 data/VirtualImage 5000000
```

Expected post-fix behavior:
- no notifier/debugger chain rooted in `BitBlt>>copyBits`
- stable repeating low-priority scheduler loop

## Related

- See `reference/01-diary.md` Step 9 for the chronological implementation details.
- See `reference/05-long-run-notifier-path-primitive-gap-writeup.md` for the earlier primitive-gap fixes that made the `copyBits` frontier visible.
