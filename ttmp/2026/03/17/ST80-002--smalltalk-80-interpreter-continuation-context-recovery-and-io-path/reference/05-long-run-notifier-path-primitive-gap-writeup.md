---
Title: Long-Run Notifier Path Primitive Gap Writeup
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
      Note: Implements primitivePerform, primitivePerformWithArgs, primitiveCursorLink, primitiveBeCursor, and correct asOop/asObject semantics (commit d0346da)
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go
      Note: Adds a hard trace2 regression and a 2,000,000-cycle state probe that exposed and then moved the notifier/debugger frontier (commit d0346da)
ExternalSources: []
Summary: Intern-facing writeup of the post-allocator runtime frontier where the VM was stable for millions of cycles but still fell into the Smalltalk notifier/debugger path because several control/input/system primitives were stubbed or incorrectly implemented.
LastUpdated: 2026-03-18T12:35:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Long-Run Notifier Path Primitive Gap Writeup

## Goal

Explain the next runtime defect cluster after the context-body allocator fix: the interpreter was no longer crashing early or corrupting contexts late, but it still reached the Smalltalk notifier/debugger path during long runs because several primitives were either missing or wrong.

## Context

Once the allocator fix landed, the VM could again run cleanly through:
- `800000` cycles
- `2000000` cycles

That looked good at first, but the important question was not just "does it panic?" It was:

- "What is the image doing at two million cycles?"

The first long-run state probe showed the VM was not idling. Instead, the sender chain led into `NotifierView class>>openDebugger:contents:label:displayAt:` after `Object>>primitiveFailed`.

That means runtime stability had improved, but semantic completeness had not. The image was still tripping over unsupported primitives and opening its own debugger UI in response.

## Quick Reference

### Symptom

At two million cycles, the active sender chain repeatedly ended inside:
- `NotifierView class>>openDebugger:contents:label:displayAt:`
- `NotifierView class>>openContext:label:contents:`
- `Object>>error:`
- `Object>>primitiveFailed`

This was not random corruption. It was the image's normal failure-handling path for an unimplemented or misimplemented primitive.

### First concrete blocker

The first chain identified after the allocator fix bottomed out at:

- `Object>>perform:`
- `Object>>perform:withArguments:`

Those primitives were still explicit stubs in the Go interpreter.

### Later blockers after fixing `perform:`

Once `perform:` and `perform:withArguments:` were implemented, the notifier path moved deeper and exposed:

1. `Cursor>>beCursor` (`primitive #101`) was missing.
2. `Cursor class>>cursorLink:` (`primitive #92`) was missing.
3. `Object>>asOop` (`primitive #75`) was not missing, but wrong.

After fixing those, the notifier/debugger path moved deeper again and now bottoms out in:

- `BitBlt>>copyBits`
- `BitBlt>>copyBitsAgain`

That is the current display-path frontier after commit `d0346da`.

## Root Causes

### 1. `primitivePerform` and `primitivePerformWithArgs` were still stubs

The Blue Book treats `perform:` as a specialized send:
- the first argument becomes the real selector
- the remaining arguments become the real message arguments
- the selector argument must be removed from the stack before activation
- the resulting method's argument count must match the transformed send

The interpreter still had:

```go
interp.primitiveFail() // Complex — defer
```

for both primitive indices `83` and `84`.

That was enough to make the image open the debugger later, even though the VM itself stayed mechanically stable.

### 2. `primitiveBeCursor` and `primitiveCursorLink` were unimplemented

Once the image got farther into its UI/update path, cursor configuration primitives became necessary.

The minimal functional behavior needed here was not full host integration. The image mainly needed the VM to:
- accept a designated cursor form
- accept whether cursor and mouse position should be linked

Without that, the image still raised `primitiveFailed`.

### 3. `primitiveAsOop` was semantically wrong

This was subtler than a stub.

The Blue Book specifies:
- `asOop` works only for non-SmallInteger receivers
- it returns the receiver OOP with bit 0 set (`receiver bitOr: 1`)

The existing implementation instead did:

```go
interp.pushInteger(int(rcvr >> 1))
```

That is not equivalent.

It is wrong for two reasons:
- it accepts SmallInteger receivers when it should fail
- it routes the result through `pushInteger`, which can fail when the derived value does not fit the SmallInteger range

So `asOop` was not just incomplete. It was actively producing the wrong failure behavior.

## Final Fix

Commit `d0346da` landed the following repairs.

### `primitivePerform`

Implemented according to the Blue Book:
- save the original perform selector
- replace `messageSelector` with the selector argument from the stack
- look up the real method on the real receiver
- verify that the looked-up method takes one fewer argument than the original perform send
- shift arguments down over the selector slot
- reduce `argumentCount`
- execute the looked-up method directly

### `primitivePerformWithArgs`

Implemented the array-driven perform path:
- pop the array argument
- verify it is an `Array`
- verify the active context has room for its elements
- pop the selector argument
- push the array elements as ordinary arguments
- look up the real method
- verify argument count
- execute the looked-up method or restore the original stack shape on failure

### `primitiveBeCursor`

Added minimal VM-side cursor designation:
- remember the designated cursor form in an interpreter register
- return the receiver

### `primitiveCursorLink`

Added minimal VM-side cursor-link bookkeeping:
- accept only `true` or `false`
- update an interpreter boolean register
- return the receiver

### `primitiveAsOop`

Corrected to the Blue Book semantics:
- fail for SmallInteger receivers
- push `receiver | 1` directly

### `primitiveAsObject`

Added the inverse operation:
- require a SmallInteger receiver
- derive `newOop := receiver & 0xFFFE`
- succeed only when that OOP names a valid object

## Result

After these fixes:
- `go test ./...` passes
- `trace2` is now a hard regression check in the test suite
- the two-million-cycle state probe remains green
- the notifier/debugger path no longer bottoms out at `perform:`, `beCursor`, `cursorLink:`, or `asOop`

The current frontier moved deeper into the real display subsystem:
- `BitBlt>>copyBits`

That is progress. It means the earlier notifier path was real, reproducible, and successfully removed rather than merely hidden.

## Why This Matters

This cluster is a good example of a misleading "stable but not correct" state.

The VM was already capable of:
- long runs
- valid contexts
- correct block creation

but the image still could not proceed normally because a few higher-level primitives were missing or wrong. If we had only watched for panics, we could have mistaken this for success.

The long-run sender-chain probe was what exposed the difference.

## Minimal Review Checklist

If an intern reviews this later, they should verify:

1. `perform:` really rewrites the stack to a normal send shape before `executeNewMethod`.
2. `perform:withArguments:` restores the original stack shape if the transformed send fails validation.
3. `asOop` fails for SmallInteger receivers and pushes `receiver | 1` directly for non-SmallInteger receivers.
4. `asObject` only succeeds for valid object pointers.
5. The long-run state moves forward rather than reopening the same notifier path.

## Validation Commands

```bash
go test ./...
go test ./pkg/interpreter -run TestLogStateAtTwoMillionCycles -v
```

Expected post-fix behavior:
- no notifier/debugger chain rooted in `perform:`
- no notifier/debugger chain rooted in `beCursor`
- no notifier/debugger chain rooted in `cursorLink:`
- no notifier/debugger chain rooted in `asOop`

## Related

- See `reference/01-diary.md` Step 8 for the chronological debugging narrative.
- See `reference/04-context-body-reuse-and-segment-wrap-writeup.md` for the earlier allocator fix that made this long-run primitive frontier visible.
