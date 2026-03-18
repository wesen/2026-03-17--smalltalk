---
Title: Diary
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
    - Path: cmd/st80-ui/main.go
      Note: |-
        Windowed UI entrypoint and runtime flags (commit 8e85254)
        Windowed UI entrypoint for the new host loop (commit 8e85254)
    - Path: pkg/interpreter/interpreter.go
      Note: |-
        Stepped execution API plus display snapshot export for the SDL host loop (commit 8e85254)
        Stepped interpreter API and display snapshot export (commit 8e85254)
    - Path: pkg/ui/ui.go
      Note: |-
        SDL host window, render loop, and bitmap conversion (commit 8e85254)
        SDL host window and display bitmap presentation loop (commit 8e85254)
ExternalSources: []
Summary: Diary for the graphical host-window ticket that begins after the interpreter reaches a stable idle loop with a real BitBlt path.
LastUpdated: 2026-03-18T09:33:46.413235314-04:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the new UI-focused continuation after the interpreter ticket reached stable long-run execution with a real BitBlt implementation, and record the exact code and validation needed to expose the Smalltalk display bitmap in a host SDL window.

## Step 1: SDL Host Window, Stepped Interpreter API, and Dummy-Driver End-to-End Validation

The interpreter ticket reached the right boundary for UI work: the old headless `copyBits` stub was gone, the real in-memory BitBlt path was active, and the image again settled into a stable scheduler loop through five million cycles. That meant the next honest step was not another speculative interpreter fix. It was to expose the designated display form to a host window and prove that the VM could now drive a real rendering loop.

I kept this first UI slice intentionally narrow. The goal was not full input integration or cursor semantics yet. The goal was an end-to-end path that boots the image, advances the interpreter in bounded chunks, snapshots the Smalltalk display bitmap, expands the 1-bit words into pixels, and presents those pixels through SDL. That creates a real UI command while leaving mouse, keyboard, timer, and cursor follow-up work explicit instead of hidden.

### Prompt Context

**User prompt (verbatim):** "alright, do it, no questions asked, go as far as you can until we have a working UI"

**Assistant interpretation:** Move immediately from the stabilized interpreter into host-side UI work, create the separate UI ticket, and implement the largest reliable working slice without pausing for planning questions.

**Inferred user intent:** Turn the now-stable interpreter into something visibly usable, while preserving the same detailed, reviewable diary/commit discipline used for the VM debugging work.

**Commit (code):** 8e85254 — "Add SDL host window for display bitmap"

### What I did
- Created ticket `ST80-003` for the graphical host-window work.
- Pulled in SDL bindings via:

```bash
go get github.com/veandco/go-sdl2/sdl@master
```

- Confirmed local SDL availability with:

```bash
pkg-config --modversion sdl2
```

which reported:

```text
2.30.0
```

- Extended [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) with UI-facing execution hooks:
  - `initializeActiveContext`
  - `stepCycle`
  - `RunSteps`
  - `DisplaySnapshot`
  - `CycleCount`
- Added exported `DisplaySnapshot` data describing:
  - current display form OOP
  - width
  - height
  - raster size
  - copied display words
- Kept the existing headless `Run` path intact while refactoring it to reuse the same initialization/stepping helpers.
- Added new package [pkg/ui/ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) that:
  - loads the image
  - creates an interpreter
  - runs interpreter cycles in chunks
  - creates an SDL window/renderer/texture once the image designates a display form
  - converts 1-bit Smalltalk bitmap words to ARGB pixels
  - presents the current display every loop
  - polls SDL quit events
- Added new command [cmd/st80-ui/main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go) with flags:
  - `-image`
  - `-cycles-per-frame`
  - `-max-cycles`
  - `-scale`
  - `-title`
- Validated the full repository build:

```bash
go test ./...
```

- Validated the UI path without relying on an interactive desktop session by using SDL’s dummy backend:

```bash
SDL_VIDEODRIVER=dummy go run ./cmd/st80-ui -max-cycles 500000 -cycles-per-frame 50000 -scale 1
```

### Why
- The interpreter had reached the correct boundary for UI work. Leaving the display bitmap trapped inside the VM would have slowed the project more than it helped.
- A stepped interpreter API is the simplest host integration boundary:
  - run some cycles
  - snapshot display
  - render
  - poll host events
- Using SDL’s dummy driver was the safest way to validate the new code path in this environment because it exercises the SDL initialization/window/texture/render loop without requiring a manually observed desktop session in the middle of implementation.

### What worked
- The interpreter can now be advanced incrementally by the host without using the verbose headless CLI loop.
- The display form can now be copied out of the VM in a host-friendly shape.
- The new `st80-ui` command builds cleanly.
- `go test ./...` remains green after the UI additions.
- The end-to-end SDL path also runs cleanly under the dummy video driver:

```bash
SDL_VIDEODRIVER=dummy go run ./cmd/st80-ui -max-cycles 500000 -cycles-per-frame 50000 -scale 1
```

- The only implementation mistake in this slice was a small SDL API mismatch: I initially treated `Window.SetTitle` as returning an error. The build failure exposed that immediately, and the fix was trivial.

### What didn't work
- The first `go test ./...` after adding SDL failed because of the mistaken `SetTitle` assumption:

```text
pkg/ui/ui.go:126:16: window.SetTitle(opts.WindowTitle) (no value) used as value
```

- This slice does not yet handle:
  - mouse input
  - keyboard input
  - timer primitives
  - cursor location/link semantics on the host side
  - verified interactive behavior on a real desktop session

### What I learned
- The correct seam between the VM and the host UI is not “let SDL read object memory directly.” It is “let the interpreter expose a stable snapshot API.”
- The Smalltalk display bitmap is straightforward to render once the VM exposes width, height, raster, and word contents.
- The UI path is already useful even without input integration because it proves the runtime can drive a real host window rather than only passing long-run scheduler probes.
- SDL’s dummy driver is valuable for repository-local validation because it exercises the full host loop without turning testing into a manual process.

### What was tricky to build
- The tricky part was preserving the interpreter’s existing headless/debugging loop while also making it usable from a host render loop. The solution was to refactor the boot/step logic into shared helpers instead of bolting UI behavior into the existing CLI entrypoint.
- The other subtle piece was display conversion. The VM stores 1-bit pixels packed into 16-bit words, so the UI layer has to unpack them consistently. The renderer uses a simple rule: each word is read from most significant bit to least significant bit across the scanline, mapping set bits to black and cleared bits to white.

### What warrants a second pair of eyes
- Review the bit-unpacking direction in [pkg/ui/ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) against real rendered output on a desktop session. The code is internally consistent with the current BitBlt path, but visual confirmation is still important.
- Review whether `CyclesPerFrame=50000` is the right default once interactive input/timers are added.
- Review whether the UI command should eventually subsume the headless command or remain a dedicated separate entrypoint.

### What should be done in the future
- Feed SDL keyboard and mouse events into the corresponding Smalltalk input primitives.
- Implement host-side time support for the clock/timer primitives still marked as failing in the interpreter.
- Decide how to display or synthesize the Smalltalk cursor once cursor location primitives exist.
- Run `st80-ui` on a real desktop session and capture visual confirmation screenshots / notes in this ticket.

### Code review instructions
- Start in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `DisplaySnapshot`
  - `RunSteps`
  - `CycleCount`
- Then review [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go):
  - `Run`
  - `runLoop`
  - `processEventsAndPresent`
  - `copyDisplayBits`
- Finally review the CLI entrypoint in [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go).
- Validate with:
  - `go test ./...`
  - `SDL_VIDEODRIVER=dummy go run ./cmd/st80-ui -max-cycles 500000 -cycles-per-frame 50000 -scale 1`

### Technical details
- SDL binding added:
  - `github.com/veandco/go-sdl2/sdl v0.5.0-alpha.7.0.20250220045537-7f43f67a3a12`
- Host rendering path:
  - create window/renderer
  - create ARGB8888 streaming texture
  - copy Smalltalk display words into `[]uint32`
  - `texture.UpdateRGBA`
  - `renderer.Copy`
  - `renderer.Present`
- Commands used in this step:

```bash
docmgr ticket create-ticket --ticket ST80-003 --title 'Smalltalk-80 graphical UI host window and event loop' --topics vm,smalltalk,sdl,go
pkg-config --modversion sdl2
go get github.com/veandco/go-sdl2/sdl@master
go test ./...
SDL_VIDEODRIVER=dummy go run ./cmd/st80-ui -max-cycles 500000 -cycles-per-frame 50000 -scale 1
git commit -m "Add SDL host window for display bitmap"
```
