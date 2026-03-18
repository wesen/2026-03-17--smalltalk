---
Title: Direct Input Exercise Note
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
    - Path: cmd/st80-exercise-snapshot/main.go
      Note: CLI entrypoint for the direct input exercise harness (commit 89c742b)
    - Path: pkg/ui/exercise.go
      Note: Direct interpreter-side input exercise path used to bypass SDL/X11 (commit 89c742b)
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-direct-input-snapshot.sh
      Note: Ticket-local wrapper used to reproduce the direct-input before/after result (commit 89c742b)
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-18T13:16:38.235137644-04:00
WhatFor: ""
WhenToUse: ""
---


# Direct Input Exercise Note

## Goal

Record the result of bypassing SDL/X11 entirely and injecting the same mouse/key sequence directly into the interpreter, so we can tell whether the remaining UI/input problem is inside the image/VM or in host-side event delivery.

## Context

The earlier off-screen `Xvfb` exercise was inconclusive in the worst way:

- before/after images showed no visible change
- the UI run log showed no input-debug lines

That left two very different possibilities open:

1. the image/VM input path was still wrong
2. the host never delivered events into SDL under the current `Xvfb` setup

The direct-input harness closes that ambiguity by injecting the same style of input directly into the interpreter and capturing before/after snapshots without any SDL or X server in the middle.

## Quick Reference

### Tooling added

- `pkg/ui/exercise.go`
- `cmd/st80-exercise-snapshot/main.go`
- `scripts/exercise-direct-input-snapshot.sh`

### Direct exercise command

```bash
BEFORE_CYCLES=50000 AFTER_CYCLES=500000 \
ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-direct-input-snapshot.sh
```

### Current result

Observed output:

```text
beforeCycles=50000 afterCycles=500000 changedPixels=10319 beforeHash=0162f0db51d0f337b9c13722d5b2dc815344aa4f9b2b9b4f79507aeb1e63586b afterHash=b680477060f52bcc2a95142d83f17e0f6405822fa4537df0b3dfb2a97b2ff13c beforeBlack=112228 afterBlack=111072
```

Interpretation:

- the raw display-word hash changed
- the framebuffer changed by `10319` pixels
- the black-pixel count changed

So the image does respond to the directly injected input sequence once delivery is guaranteed and enough cycles are allowed afterward.

### Important contrast with the shorter run

With only `AFTER_CYCLES=50000`, the result was:

- `changedPixels=28`
- identical raw display hashes

That is consistent with only a cursor-overlay delta. The longer run is the important one because it shows a real framebuffer change, not just a cursor-position artifact.

### Conclusion

This narrows the remaining problem substantially:

- the image-side input path is not dead
- the current blocker is host-side SDL/X event delivery under the off-screen `Xvfb` setup

## Usage Examples

Use the direct harness when:

- checking whether a candidate input-mapping change affects the image at all
- comparing image behavior without SDL/X11 in the loop
- proving that a no-change `Xvfb` run is a host-delivery issue rather than an image-consumption issue

Suggested workflow:

1. run the direct harness with a known input sequence
2. confirm whether the framebuffer hash changes
3. only then return to SDL/X delivery debugging if direct input works

This is now the faster iteration path for input experiments.

## Related

- `reference/09-offscreen-input-exercise-note.md`
- `reference/01-diary.md`
