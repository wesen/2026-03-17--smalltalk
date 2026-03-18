---
Title: 'DisplayBitmap new: LargePositiveInteger Size Bug Writeup'
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
    - Path: data/trace3
      Note: Reference startup trace used to identify the primitive-71 divergence
    - Path: pkg/interpreter/interpreter.go
      Note: Primitive 71 LargePositiveInteger size fix and positive integer decoder (commit acaa659)
    - Path: pkg/interpreter/interpreter_test.go
      Note: Trace3 startup regression and targeted startup diagnostics for the display allocation bug (commit acaa659)
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-1000.png
      Note: Post-fix direct framebuffer snapshot showing the corrected 640x480 surface
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-2000000.png
      Note: Long-run direct framebuffer snapshot after the primitive-71 fix
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-18T10:27:20.190673224-04:00
WhatFor: ""
WhenToUse: ""
---


# DisplayBitmap new: LargePositiveInteger Size Bug Writeup

## Goal

Capture the startup display-allocation bug that kept the UI locked to a `640x16` white framebuffer, explain the exact interpreter defect, and give future reviewers a fast way to validate the fix.

## Context

The UI ticket originally looked like a renderer problem because the SDL window was blank white. The direct framebuffer snapshot tool proved otherwise: the interpreter-designated display surface itself was `640x16`, all white, and unchanged across long runs.

Further startup tracing showed that `DisplayScreen class>>displayExtent:` intentionally constructs two display screens:

1. a temporary `width x 16` display
2. the real `width x fullHeight` display

The temporary screen is designated first. The real screen is supposed to replace it later via a second `currentDisplay:` send.

The reference startup trace in [trace3](/home/manuel/code/wesen/2026-03-17--smalltalk/data/trace3) shows:

- cycle `137`: first `currentDisplay:`
- cycle `721`: second `extent:offset:bits:`
- cycle `733`: second `currentDisplay:`
- cycle `757`: `restore`

In our buggy runtime, the trace diverged at cycle `721`: instead of already creating the second full-size `DisplayScreen`, execution had fallen back into bytecodes inside `Behavior>>new:`.

## Quick Reference

### Bug summary

- **Symptom:** the designated display remained `640x16`, all white.
- **Root cause:** primitive `71` (`basicNew:` / `new:`) only accepted SmallInteger sizes.
- **Trigger:** `DisplayBitmap new: 19200` passes a `LargePositiveInteger`, because `19200` is larger than the SmallInteger range used by this image.
- **Effect:** the primitive failed, Smalltalk fell back into `Behavior>>new:`, and the startup display rebuild no longer matched the trace.
- **Fix:** decode non-negative `LargePositiveInteger` size arguments in the primitive and allocate normally.

### Evidence chain

1. Direct snapshots before the fix:

```text
cycles=1000000 width=640 height=16 raster=40 blackPixels=0 whitePixels=10240
cycles=2000000 width=640 height=16 raster=40 blackPixels=0 whitePixels=10240
```

2. `currentDisplay:` bytecode/literal decode:
   - global `Display` association value is placeholder OOP `0x0340`
   - `currentDisplay:` sends `Display become: aDisplayScreen`
   - then sends `Display beDisplay`
   - `become:` itself was not the bug

3. `displayExtent:` bytecode decode:
   - first builds `width x 16`
   - later builds `width x fullHeight`
   - later sends `currentDisplay:` again

4. Trace mismatch before the fix:

```text
trace3 cycle 721 expected send "[cycle=721]  aDisplayScreen extent:offset:bits: aPoint aPoint aDisplayBitmap"
but bytecode=112 method=0x4514(<Behavior>new:) was not a send
```

5. Primitive defect:
   - old code in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) used `popInteger()` in primitive `71`
   - `popInteger()` rejects any non-SmallInteger
   - `DisplayBitmap new: 19200` therefore failed even though the size was valid

6. Post-fix trace and snapshot:

```text
cycle=733 method=0x0362(<DisplayScreen class>displayExtent:) selector=currentDisplay:
cycle=757 method=0x0362(<DisplayScreen class>displayExtent:) selector=restore
cycles=1000 width=640 height=480 raster=40 blackPixels=0 whitePixels=307200
cycles=2000000 width=640 height=480 raster=40 blackPixels=0 whitePixels=307200
```

### Code delta

- Added `positiveIntegerValueOf` in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) to decode:
  - non-negative SmallIntegers
  - `LargePositiveInteger` byte objects
- Switched primitive `71` to use that decoder instead of `popInteger()`
- Added a normal regression test in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestTrace3DisplayStartupSendSelectorsMatch`
  - validates `trace3` selector flow through cycle `757`

### Why this fix is correct

- The reference trace explicitly expects the second full-screen display allocation to succeed during startup.
- The only concrete divergence was the fallback into `Behavior>>new:` at the exact point where `DisplayBitmap new: 19200` should have been primitive-handled.
- After teaching primitive `71` to decode `LargePositiveInteger` sizes, the trace lines realigned at the expected cycles without changing the `currentDisplay:` or `become:` semantics.

### Remaining issue after the fix

- The designated display is now correctly `640x480`.
- The UI and direct snapshots still show that the full-size framebuffer remains all white.
- So this fix resolves the structural display-allocation bug, but not yet the later “why nothing draws into the corrected display surface” issue.

## Usage Examples

### Validate the regression test

```bash
go test ./pkg/interpreter
go test ./...
```

### Reproduce the original divergence signal

```bash
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpTrace3FirstMismatchUpTo750 -v
```

Before the fix, this failed at cycle `721`. After the fix, it passes.

### Check the startup selector path

```bash
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpDisplayStartupSendCycles -v
```

Expected important lines after the fix:

```text
cycle=137 ... selector=currentDisplay:
cycle=733 ... selector=currentDisplay:
cycle=757 ... selector=restore
```

### Check the designated framebuffer directly

```bash
CYCLES=1000 bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh
CYCLES=2000000 bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh
```

Expected dimensions after the fix:

```text
width=640 height=480
```

## Related

- Diary: [01-diary.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/01-diary.md)
- Startup trace: [trace3](/home/manuel/code/wesen/2026-03-17--smalltalk/data/trace3)
- Snapshot command: [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-snapshot/main.go)
- UI host loop: [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go)
- Fixed interpreter logic: [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go)
