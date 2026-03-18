---
Title: Real BitBlt WordArray Source-Form Bug Writeup
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
    - Path: pkg/interpreter/interpreter.go
      Note: Bug and fix for WordArray-backed BitBlt source/halftone forms (commit ea9ea41)
    - Path: pkg/interpreter/interpreter_test.go
      Note: Diagnostic reproduction of the first failing copyBits source-form shape (commit ea9ea41)
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-18T09:21:56.295109153-04:00
WhatFor: ""
WhenToUse: ""
---


# Real BitBlt WordArray Source-Form Bug Writeup

## Goal

Record the exact bug that appeared when the temporary headless `copyBits` stub was replaced by a real BitBlt implementation, explain why the first implementation still failed, and leave an intern-reviewable explanation of the final fix.

## Context

The previous ticket step proved that the interpreter could reach a stable low-priority scheduler loop if `primitiveCopyBits` simply reported success in headless mode. That was useful, but it was only a tactical checkpoint. The next real step was to implement the Blue Book BitBlt simulation in memory so the runtime could continue with actual display semantics instead of a fake primitive.

The first implementation of the copy loop was broad enough to compile and run, but by roughly cycle `1,972,594` the image re-entered `NotifierView` through `BitBlt>>copyBitsAgain` and `Object>>primitiveFailed`. That meant the issue was somewhere inside the primitive path itself, not in a higher-level sender/lookup bug.

## Quick Reference

### Symptom

At two million cycles, the image was no longer in the quiescent scheduler loop. Instead, the sender chain bottomed out in:

```text
Object>>primitiveFailed
BitBlt>>copyBitsAgain
BitBlt>>copyBits
Form>>copyBits:from:at:clippingBox:rule:mask:
```

Primitive-local diagnostics recorded the first hard failure as:

```text
lastCopyBitsFailure cycle=1972594 bitBlt=0xF0C4 detail=invalid source form oop=0x0F88
```

### Root Cause

The first real implementation of `formWordsOf` accepted only one kind of form backing store:

- valid object
- width and height stored as SmallIntegers
- `bits` object class exactly `DisplayBitmap`

That assumption was too narrow for the live image.

The retained diagnostic test showed the failing source form was:

```text
sourceForm=0x0F88 class=0x02E8(Cursor) wordLen=4
sourceForm bits=0x0F8A class=0x0A72(WordArray) wordLen=16
halftoneForm=0x100C class=0x0C52(Form) wordLen=4
halftoneForm bits=0x100E class=0x0A72(WordArray) wordLen=16
```

So the bug was not “BitBlt cannot handle Cursor/Form objects.” The bug was:

- the interpreter assumed all legal bit storage must be `DisplayBitmap`
- the image legally used `WordArray` for cursor and halftone backing storage

### Fix

`formWordsOf` now accepts any valid non-pointer word object for `bits`, not just `DisplayBitmap`.

Before:

- reject unless `fetchClassOf(bits) == 0x001E`

After:

- reject only if `bits` is invalid, not a word object, or has pointer fields

This matches the real requirement of the primitive more closely:

- BitBlt needs word-addressable backing storage
- it does not inherently require a single concrete storage class

### Why the Fix Is Correct

- The source and halftone forms that failed were structurally valid word-backed forms.
- After widening acceptance to non-pointer word objects:
  - the earlier `copyBits` primitive failure disappeared
  - `go test ./...` stayed green
  - `go run ./cmd/st80 data/VirtualImage 5000000` returned to the stable scheduler loop
- That means the new acceptance rule matches the actual image usage instead of overfitting to one concrete form class.

### Final Validation

```bash
go test ./pkg/interpreter -run TestTrace2SendSelectorsMatch -v
go test ./pkg/interpreter -run TestLogStateAtTwoMillionCycles -v
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpFirstCopyBitsFailureState -v
go test ./...
go run ./cmd/st80 data/VirtualImage 5000000
```

Expected long-run signal after the fix:

```text
[cycle 500000]  ctx=0x6664 ip=12 sp=5 bc=153 method=0x666E rcvr=0x6626
[cycle 1000000] ctx=0x6664 ip=11 sp=6 bc=113 method=0x666E rcvr=0x6626
[cycle 1500000] ctx=0x6664 ip=10 sp=5 bc=163 method=0x666E rcvr=0x6626
```

## Usage Examples

### Example 1: Review the Bug Quickly

Start in:

- [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go)

Look at:

- `formWordsOf`
- `copyBitsFailure`
- `doPrimitiveCopyBits`

The key review question is:

- does the primitive need a specific storage class, or only valid non-pointer word storage?

### Example 2: Reproduce the Original Discovery Path

Run:

```bash
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpGraphicsClassLayouts -v
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpGraphicsMethodHeaders -v
go test ./pkg/interpreter -run TestLogStateAtTwoMillionCycles -v
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpFirstCopyBitsFailureState -v
```

Use the output to verify:

- form/BitBlt field layouts
- first failing copyBits receiver state
- actual source/halftone backing-store classes

## Related

- [01-diary.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/reference/01-diary.md)
- [06-headless-copybits-quiescent-loop-writeup.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/reference/06-headless-copybits-quiescent-loop-writeup.md)
- [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go)
- [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go)
