---
Title: Snapshot Primitive 97 Support Writeup
Ticket: ST80-003
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
    - Path: pkg/image/loader.go
      Note: Image serializer added for primitive 97 snapshot support (commit fec10f5)
    - Path: pkg/image/loader_test.go
      Note: Round-trip regression for image serialization (commit fec10f5)
    - Path: pkg/interpreter/interpreter.go
      Note: Primitive 97 dispatch and snapshot-path integration (commit fec10f5)
    - Path: pkg/interpreter/interpreter_test.go
      Note: Direct primitive 97 regression coverage (commit fec10f5)
    - Path: pkg/objectmemory/objectmemory.go
      Note: Raw word export helpers used by the serializer (commit fec10f5)
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-18T12:48:41.834188684-04:00
WhatFor: ""
WhenToUse: ""
---


# Snapshot Primitive 97 Support Writeup

## Goal

Explain the primitive-97 audit gap found during the OCR verification pass, record the exact implementation added to close it, and capture the image-format details that matter for future snapshot/debugging work.

## Context

The OCR-backed Blue Book verification pass found one concrete mismatch between the extracted I/O primitive table and the current VM: primitive `97` (`snapshotPrimitive`) was present in the audit but absent from `dispatchInputOutputPrimitives`.

This mattered because the discrepancy was no longer speculative. The OCR pack and the image-side audit had already established that:

- primitive `97` belongs in the I/O primitive family
- it is a `SystemDictionary` receiver primitive with no arguments
- its purpose is to save a snapshot of object memory

The VM already knew how to load image files, but it had no inverse path for writing the current object memory back out.

## Quick Reference

### Bug

Before this fix:

- `pkg/interpreter/interpreter.go` dispatched primitives `90..96` and `98..102`
- primitive `97` was missing entirely
- any image send that expected `snapshotPrimitive` would primitive-fail and fall back into Smalltalk

### Fix

The fix has three parts:

1. Added raw-word export helpers to `ObjectMemory`
   - `ObjectTableWords()`
   - `ObjectSpaceWords()`

2. Added `image.WriteImage(path, memory)` as the inverse of `image.LoadImage(path)`
   - writes big-endian header values
   - writes big-endian object-space words
   - aligns the object-table start to the same 512-byte boundary used by the current image format
   - writes big-endian object-table words

3. Wired primitive `97` into the interpreter
   - added `SetSnapshotPath(path string)` on `Interpreter`
   - added `case 97` in `dispatchInputOutputPrimitives`
   - implemented `primitiveSnapshot()`
   - configured the snapshot path in:
     - `pkg/ui/ui.go`
     - `pkg/ui/snapshot.go`
     - interpreter test setup

### Snapshot path semantics

Primitive `97` now writes to the interpreter's configured snapshot path. If no snapshot path has been configured, the primitive fails normally.

That is deliberate. It avoids silently writing to an invented fallback path and keeps the primitive tied to the image path the current process actually loaded.

### Stack semantics

The implemented behavior is:

- pop receiver
- write snapshot
- push receiver back on success
- restore stack and fail on error

This matches the general receiver-preserving style used by the other no-argument I/O primitives in this VM.

### File-format detail that mattered

The current `data/VirtualImage` is not laid out as:

```text
header + objectSpace + objectTable
```

There is a padding gap between object space and object table. For the checked-in image:

- header: `512` bytes
- object space: `517760` bytes
- object table: `77472` bytes
- gap before object table: `384` bytes

That gap exists because the object table starts on a `512`-byte boundary. `WriteImage` now preserves that alignment rule.

### Verification commands

```bash
gofmt -w pkg/objectmemory/objectmemory.go pkg/image/loader.go pkg/image/loader_test.go pkg/interpreter/interpreter.go pkg/interpreter/interpreter_test.go pkg/ui/ui.go pkg/ui/snapshot.go
go test ./pkg/image ./pkg/interpreter ./pkg/ui ./cmd/st80-ui -run 'TestWriteImageRoundTripsObjectMemory|TestPrimitiveSnapshotWritesImageAndReturnsReceiver|TestTrace2SendSelectorsMatch|TestTrace3DisplayStartupSendSelectorsMatch|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles|TestPrimitive(MousePointReturnsConfiguredPoint|CursorLocPutUpdatesCursorAndReturnsReceiver|CursorLocPutUpdatesMouseWhenLinked|InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord|SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal)'
```

Observed result:

```text
ok  	github.com/wesen/st80/pkg/image	0.007s
ok  	github.com/wesen/st80/pkg/interpreter	0.066s
ok  	github.com/wesen/st80/pkg/ui	0.011s [no tests to run]
?   	github.com/wesen/st80/cmd/st80-ui	[no test files]
```

## Usage Examples

Use this writeup when:

- reviewing why primitive `97` is now considered implemented
- checking how the VM decides where to write snapshots
- reviewing the on-disk image layout for future memory/debugging work
- deciding whether later work should add a separate “Save As” path instead of reusing the loaded image path

Typical review order:

1. `pkg/image/loader.go`
2. `pkg/objectmemory/objectmemory.go`
3. `pkg/interpreter/interpreter.go`
4. `pkg/interpreter/interpreter_test.go`

## Related

- `reference/10-blue-book-ocr-verification-pass.md`
- `reference/01-diary.md`
