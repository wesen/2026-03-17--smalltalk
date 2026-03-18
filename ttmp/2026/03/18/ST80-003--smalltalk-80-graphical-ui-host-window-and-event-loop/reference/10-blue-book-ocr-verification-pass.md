---
Title: Blue Book OCR Verification Pass
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
    - Path: data/method.oops
      Note: Image-side selector inventory used to confirm UI/timer/display methods exist
    - Path: pkg/interpreter/interpreter.go
      Note: Primitive dispatch table and field-index constants audited against the OCR pack
    - Path: pkg/interpreter/interpreter_test.go
      Note: Focused interpreter regression set used during the OCR verification pass
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/04-primitive-audit.csv
      Note: OCR-extracted primitive table used as the main audit source
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/05-display-and-bitblt-audit.md
      Note: OCR-extracted display and BitBlt field-order reference used to verify constants
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-18T12:39:27.598281067-04:00
WhatFor: ""
WhenToUse: ""
---


# Blue Book OCR Verification Pass

## Goal

Capture a post-OCR verification pass that checks the intern's Blue Book extraction against the current VM/UI implementation, records what matches, and isolates the remaining discrepancies that still need engineering work.

## Context

The OCR pack under `reference/ocr-bluebook/` is now good enough to use as an audit source instead of just a transcription experiment. This pass compares the extracted class-layout and primitive-reference material against the Go VM, the live image method inventory, and a focused set of regression tests.

This pass intentionally uses:
- the Blue Book OCR pack in this ticket
- the current repository source
- the image-side selector inventory in `data/method.oops`

This pass intentionally does not use any external Smalltalk-80 implementation as a reference.

## Quick Reference

### Inputs used

- `reference/ocr-bluebook/02-class-layouts.csv`
- `reference/ocr-bluebook/04-primitive-audit.csv`
- `reference/ocr-bluebook/05-display-and-bitblt-audit.md`
- `reference/ocr-bluebook/07-open-questions.md`
- `pkg/interpreter/interpreter.go`
- `pkg/interpreter/interpreter_test.go`
- `pkg/ui/ui.go`
- `pkg/ui/ui_test.go`
- `cmd/st80-ui/main.go`
- `data/method.oops`

### Verified layout/index mappings

The OCR-derived layout tables match the current interpreter constants for:

- `Point`
  - `PointXIndex = 0`
  - `PointYIndex = 1`
- `Rectangle`
  - `RectangleOriginIndex = 0`
  - `RectangleCornerIndex = 1`
- `Form`
  - `FormBitsIndex = 0`
  - `FormWidthIndex = 1`
  - `FormHeightIndex = 2`
  - `FormOffsetIndex = 3`
- `BitBlt`
  - `BitBltDestFormIndex = 0`
  - `BitBltSourceFormIndex = 1`
  - `BitBltHalftoneFormIndex = 2`
  - `BitBltCombinationRuleIndex = 3`
  - `BitBltDestXIndex = 4`
  - `BitBltDestYIndex = 5`
  - `BitBltWidthIndex = 6`
  - `BitBltHeightIndex = 7`
  - `BitBltSourceXIndex = 8`
  - `BitBltSourceYIndex = 9`
  - `BitBltClipXIndex = 10`
  - `BitBltClipYIndex = 11`
  - `BitBltClipWidthIndex = 12`
  - `BitBltClipHeightIndex = 13`

### Verified I/O/UI primitive coverage

Audited OCR primitives and current status:

| Primitive | Selector | Status in VM |
|-----------|----------|--------------|
| 90 | `primMousePt` | implemented |
| 91 | `primCursorLocPut:` | implemented |
| 92 | `cursorLink:` | implemented |
| 93 | `primInputSemaphore:` | implemented |
| 94 | `primSampleInterval:` | implemented |
| 95 | `primInputWord` | implemented |
| 96 | `copyBits` | implemented |
| 97 | `snapshotPrimitive` | missing from `dispatchInputOutputPrimitives` |
| 98 | `secondClockInto:` | implemented |
| 99 | `millisecondClockInto:` | implemented |
| 100 | `signal:atMilliseconds:` | implemented |
| 101 | `beCursor` | implemented |
| 102 | `beDisplay` | implemented |

### Image-side selector presence

The image-side compiled methods for the currently relevant UI/timer selectors are present in `data/method.oops`, including:

- `<InputState>primCursorLocPut:`
- `<InputState>primMousePt`
- `<InputState>primSampleInterval:`
- `<InputState>primInputWord`
- `<InputState>primInputSemaphore:`
- `<Time class>secondClockInto:`
- `<Time class>millisecondClockInto:`
- `<ProcessorScheduler>signal:atMilliseconds:`
- `<DisplayScreen>beDisplay`
- `<Cursor>beCursor`

### Validation commands and current results

Commands run during this verification pass:

```bash
go test ./pkg/ui ./cmd/st80-ui
go test ./pkg/interpreter -run 'TestTrace2SendSelectorsMatch|TestTrace3DisplayStartupSendSelectorsMatch|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles|TestPrimitive(MousePointReturnsConfiguredPoint|CursorLocPutUpdatesCursorAndReturnsReceiver|CursorLocPutUpdatesMouseWhenLinked|InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord|SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal)'
rg -n 'primCursorLocPut|primMousePt|primSampleInterval|primInputWord|primInputSemaphore|secondClockInto|millisecondClockInto|signal:atMilliseconds:|beDisplay|beCursor' data/method.oops
rg -n 'case 9[0-9]|case 100|case 101|case 102|snapshotPrimitive|dispatchInputOutputPrimitives' pkg/interpreter/interpreter.go
```

Observed results:

- `go test ./pkg/ui ./cmd/st80-ui` passed.
- The focused interpreter regression set passed.
- The OCR layout tables matched the current interpreter constant definitions.
- Primitive `97` is the concrete audited gap in the current I/O dispatch table.

### Verification findings

1. The intern's OCR pack is good enough to drive real VM audits.
2. The currently implemented class/field ordering for the UI/display path matches the Blue Book extraction.
3. The active UI/timer primitive surface matches the OCR audit except for primitive `97`.
4. `go test ./pkg/...` is not currently a practical blanket verifier because `pkg/interpreter` contains long-running diagnostic tests that can keep the suite busy for minutes; targeted verification is currently more trustworthy and more repeatable.

## Usage Examples

Use this note when doing any follow-up VM audit:

1. Start with the OCR tables for field order and primitive expectations.
2. Confirm the relevant constants or primitive cases in `pkg/interpreter/interpreter.go`.
3. Confirm the selector actually exists in `data/method.oops`.
4. Run the smallest focused test subset that exercises the audited behavior.
5. If a discrepancy remains, record whether it is:
   - a missing primitive
   - a field-order mismatch
   - an argument-decoding mismatch
   - a validation-gap problem in the test suite

Use this note specifically for:

- checking future OCR-driven audits against live code
- reviewing whether UI/timer/input primitives still match the Blue Book extraction
- reviewing why primitive `97` remains open
- reviewing why package-wide verification should not yet rely on `go test ./pkg/...` alone

## Related

- `reference/03-bluebook-ocr-extraction-instructions-for-intern.md`
- `reference/04-bitblt-field-order-bug-writeup.md`
- `reference/05-bitblt-copyloop-row-advance-bug-writeup.md`
- `reference/09-offscreen-input-exercise-note.md`
