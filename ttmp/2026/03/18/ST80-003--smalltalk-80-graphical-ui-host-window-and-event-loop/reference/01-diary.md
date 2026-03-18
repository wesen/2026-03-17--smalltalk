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
    - Path: cmd/st80-snapshot/main.go
      Note: Command-line entrypoint for quick framebuffer diagnostics (commit ee69a09)
    - Path: cmd/st80-ui/main.go
      Note: |-
        Windowed UI entrypoint and runtime flags (commit 8e85254)
        Windowed UI entrypoint for the new host loop (commit 8e85254)
    - Path: pkg/interpreter/interpreter.go
      Note: |-
        Stepped execution API plus display snapshot export for the SDL host loop (commit 8e85254)
        Stepped interpreter API and display snapshot export (commit 8e85254)
        Primitive 71 LargePositiveInteger size fix and positive integer decoder (commit acaa659)
        Broader positive size/index primitive decoding pass (commit d2d22d8)
    - Path: pkg/interpreter/interpreter_test.go
      Note: |-
        Trace3 startup regression and startup diagnostics for the display allocation bug (commit acaa659)
        Direct LargePositiveInteger decoder tests and broader bug-pattern coverage (commit d2d22d8)
    - Path: pkg/ui/snapshot.go
      Note: Direct framebuffer snapshot capture and PNG output (commit ee69a09)
    - Path: pkg/ui/ui.go
      Note: |-
        SDL host window, render loop, and bitmap conversion (commit 8e85254)
        SDL host window and display bitmap presentation loop (commit 8e85254)
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/10-blue-book-ocr-verification-pass.md
      Note: New OCR-backed verification checkpoint recorded in Step 15
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/11-snapshot-primitive-97-support-writeup.md
      Note: Primitive 97 checkpoint documented in Step 16
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/12-direct-input-exercise-note.md
      Note: Direct input checkpoint documented in Step 17
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/capture-ui-screenshot.sh
      Note: Reusable off-screen screenshot helper for the SDL UI
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh
      Note: Ticket-local wrapper for direct framebuffer snapshot iteration
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-1000.png
      Note: Post-fix snapshot showing the corrected 640x480 display surface
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-1000000.png
      Note: Direct framebuffer PNG at one million cycles showing the current white 640x16 surface
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-2000000.png
      Note: Long-run snapshot showing the corrected 640x480 display surface remains all white
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png
      Note: |-
        First captured UI image showing the current visible state
        Updated off-screen UI capture after the display allocation fix
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

## Step 2: Ticket-Local Screenshot Script and First Real UI Capture

After wiring the SDL host window, the next practical question was not architectural. It was simply: what does the current UI actually show? Since the user explicitly asked for the script to live in the ticket workspace, I moved the capture flow out of `/tmp` and into the ST80-003 `scripts/` directory, then used `Xvfb` plus `xwd`/ImageMagick to grab an actual off-screen window image.

That capture answered the current-state question cleanly. The UI window exists and renders, but the visible content at this point is a blank white framebuffer rather than a recognizable Smalltalk desktop. So the project has crossed from “no UI” to “host window showing the display surface,” but it has not yet reached “fully drawn interactive Smalltalk environment.”

### Prompt Context

**User prompt (verbatim):** "store all scripts in the scripts/ ticket. then run it"

**Assistant interpretation:** Move the ad hoc UI screenshot helper into the active UI ticket workspace, execute it from there, and report the actual visible result.

**Inferred user intent:** Keep auxiliary tooling tracked with the ticket instead of scattering one-off shell scripts in `/tmp`, and verify the current UI visually rather than talking about it abstractly.

### What I did
- Added ticket-local script [capture-ui-screenshot.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/capture-ui-screenshot.sh).
- Made the script write all outputs under:
  - [various/ui-capture](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture)
- The script now:
  - starts `Xvfb`
  - runs `st80-ui`
  - waits for a `Smalltalk-80` window
  - captures it with `xwd`
  - converts it to PNG
- The first run failed because the UI exited before the screenshot was taken when `-max-cycles` elapsed too early.
- Updated the script so `MAX_CYCLES` defaults to `0` for capture runs and so it explicitly detects if the UI process dies before the window is found.
- Re-ran successfully with:

```bash
bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/capture-ui-screenshot.sh
```

## Step 3: Direct Framebuffer Snapshot Diagnostics and Proof That the Display Is White Upstream of SDL

The screenshot script showed a white window, but that still left one important ambiguity: was SDL showing a white window because the host renderer was wrong, or because the VM’s own designated display form was already white? To remove that ambiguity, I added a non-SDL snapshot path that runs the interpreter headlessly, captures the display form directly, writes it to PNG, and reports framebuffer metrics.

That diagnostic immediately paid off. The current display surface is not merely blank; it is also suspiciously small. At both one million and two million cycles, the designated display snapshot is `640x16`, all white, with a stable hash. So the current bug is not “SDL only renders white.” The current bug is that the VM-visible display form being presented to SDL is itself `640x16` and contains no black pixels.

### Prompt Context

**User prompt (verbatim):** "how would you check that the input program doesn't justrender white? But yet, you should add a way to quickly snapshot diagnostic screenshots so that you can self iterate without necessarily using xfvb all the time."

**Assistant interpretation:** Add a fast framebuffer-dump path that bypasses SDL/Xvfb so the visible display contents can be diagnosed directly, and use it immediately to determine whether the display buffer itself is white.

**Inferred user intent:** Shorten the debug loop by replacing window-capture gymnastics with a direct snapshot tool, and use that tool to distinguish renderer bugs from VM/display-state bugs.

**Commit (code):** ee69a09 — "Add framebuffer snapshot diagnostics"

### What I did
- Added [snapshot.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/snapshot.go) with:
  - `SnapshotOptions`
  - `SnapshotDiagnostic`
  - `CaptureSnapshot`
- Added [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-snapshot/main.go) so snapshots can be taken directly from the command line with no SDL dependency.
- Reused the existing display unpacking path in [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) so the diagnostic PNG and SDL renderer are driven by the same bitmap interpretation.
- Added ticket-local wrapper script [dump-display-snapshot.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh).
- Ran:

```bash
bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh
CYCLES=2000000 bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh
```

- Captured diagnostic results:

```text
cycles=1000000 width=640 height=16 raster=40 blackPixels=0 whitePixels=10240 wordHash=bfe492baf731a0dbf6e1e050f5bc3fe8c1b049383194dcdf82f023bfa409f462
cycles=2000000 width=640 height=16 raster=40 blackPixels=0 whitePixels=10240 wordHash=bfe492baf731a0dbf6e1e050f5bc3fe8c1b049383194dcdf82f023bfa409f462
```

- Saved PNG snapshots at:
  - [display-1000000.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-1000000.png)
  - [display-2000000.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-2000000.png)

### Why
- The screenshot script answered “what does the window look like?” but not “where does the whiteness originate?”
- A direct framebuffer snapshot is the shortest path to the root distinction:
  - if the raw display snapshot contains content, the SDL host renderer is wrong
  - if the raw display snapshot is white, the bug is upstream in the VM/display state

### What worked
- The direct snapshot command builds and runs cleanly with `go test ./...` still green.
- The snapshot path removes any dependence on `Xvfb` for framebuffer inspection.
- The result is decisive:
  - the SDL renderer is not inventing the white window
  - the underlying designated display form is already white
  - its shape is currently `640x16`, which is itself suspicious for a supposed full display surface

### What didn't work
- The current visible-state problem did not disappear by bypassing SDL. The raw framebuffer is still white.
- That means the next bug is not in the host presentation shell.

### What I learned
- The new highest-value clue is not “all white”; it is “all white and only 16 rows tall.”
- The hash staying identical between one million and two million cycles means the designated display form is not being meaningfully updated during the quiescent period.
- The next investigation should target display/form initialization or the exact object registered by `beDisplay`, not SDL texture upload or Xvfb timing.

### What was tricky to build
- The main trick was keeping the diagnostic path honest by reusing the same bitmap unpacking logic as the SDL renderer. If the snapshot command had used a separate conversion routine, it would have been much less trustworthy as a renderer-vs-VM discriminator.

### What warrants a second pair of eyes
- Review whether `DisplaySnapshot` is looking at the correct object designated by `beDisplay`.
- Review whether the `Form` field interpretation for `DisplayScreen` is fully correct in this image, especially given the observed `640x16` surface.

### What should be done in the future
- Investigate why the designated display form is `640x16` rather than an expected full screen extent.
- Trace display initialization again with the new snapshot tool in hand.
- Keep the direct snapshot command as the primary display diagnostic while the UI remains visually blank.

### Code review instructions
- Start in [snapshot.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/snapshot.go).
- Then review [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-snapshot/main.go).
- Then inspect the wrapper script [dump-display-snapshot.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh).
- Validate with:
  - `go test ./...`
  - `bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh`
  - `CYCLES=2000000 bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh`

### Technical details
- The snapshot command reports:
  - cycle count
  - width
  - height
  - raster
  - black pixel count
  - white pixel count
  - SHA-256 hash of the raw display words
- Commands used:

```bash
go test ./...
bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh
CYCLES=2000000 bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh
git commit -m "Add framebuffer snapshot diagnostics"
```

- Saved the resulting screenshot at:
  - [st80-ui.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png)

### Why
- The user asked specifically that scripts live in the ticket workspace.
- An off-screen screenshot is the fastest way to answer “what does the UI currently show?” without depending on a live desktop session during the turn.
- Keeping the capture path in the ticket makes future UI regressions easier to reproduce and compare.

### What worked
- `Xvfb` capture works for the new SDL UI command.
- The script successfully produces a PNG artifact inside the ticket.
- The captured UI is currently a blank white window, which is a real and useful milestone: the host window exists and is rendering the display surface, but the content is not yet visually rich.

### What didn't work
- The first capture attempt failed with:

```text
X Error of failed request:  BadWindow (invalid Window parameter)
```

- That happened because the UI process had already exited before `xwd` captured the window.
- Directly executing the script path also hit a shell-level permission/noexec issue in this environment, so invoking it via `bash .../capture-ui-screenshot.sh` is the reliable path.

### What I learned
- The current visible UI state is simpler than the runtime milestone might suggest: it is a real window, but it is visually blank white at the captured point.
- A tracked screenshot script is worth keeping because it turns “what does it show?” into a repeatable artifact rather than a memory question.

### What was tricky to build
- The only tricky part was timing. Screenshot capture needs the UI window to still exist when `xwd` runs, which means a finite `-max-cycles` default is actually wrong for a reusable capture helper. Letting the script keep the UI alive until after capture is more reliable.

### What warrants a second pair of eyes
- Review whether the all-white image is expected from the current display state or whether a missing host/input/time primitive is still preventing visible drawing.
- Review whether the capture script should also store the UI run log and window tree as first-class ticket artifacts for future debugging.

### What should be done in the future
- Run the same script again after input/timer work to compare visual progress.
- Add a second script or mode that captures multiple frames over time if the UI starts animating or repainting dynamically.
- Investigate why the current rendered surface is visually blank.

### Code review instructions
- Start with [capture-ui-screenshot.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/capture-ui-screenshot.sh).
- Then inspect the captured artifact [st80-ui.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png).
- Validate with:
  - `bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/capture-ui-screenshot.sh`

### Technical details
- Output files produced by the script:
  - `st80-ui.png`
  - `st80-ui.xwd`
  - `xwininfo-tree.txt`
  - `st80-ui-run.log`
  - `xvfb.log`
- Commands used:

```bash
bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/capture-ui-screenshot.sh
```

## Step 4: Fixed `DisplayBitmap new:` for LargePositiveInteger Sizes and Restored the Full Display Surface

The blank UI investigation stopped being a renderer problem once the snapshot tool proved that the designated display surface itself was only `640x16`. The next useful step was not more SDL work. It was to explain why the startup image never replaced that temporary `640x16` display with the real full-height one that `DisplayScreen class>>displayExtent:` is supposed to build later in the boot sequence.

I used the trace, method-bytecode dumps, and targeted startup diagnostics to narrow that down. `currentDisplay:` and `become:` were behaving correctly. The real defect was earlier: primitive `71` (`basicNew:` / `new:`) only accepted SmallInteger sizes, so `DisplayBitmap new: 19200` failed during the second full-screen allocation path. That pushed the image into `Behavior>>new:` bytecodes, broke the trace at cycle `721`, and left the temporary `640x16` display designated indefinitely.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Continue the investigation aggressively from the blank-UI/display frontier without pausing for more planning, and keep the result heavily documented.

**Inferred user intent:** Push the UI ticket forward by turning the latest diagnostics into a real VM fix, and store the root-cause analysis in a form that can be reviewed later by someone else.

**Commit (code):** acaa659 — "Fix new: for LargePositiveInteger sizes"

### What I did
- Kept and expanded the startup diagnostics in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go) so I could answer three questions with traceable evidence:
  - what `DisplayScreen class>>currentDisplay:` actually sends
  - what `DisplayScreen class>>displayExtent:` actually builds
  - where our runtime first diverges from [trace3](/home/manuel/code/wesen/2026-03-17--smalltalk/data/trace3)
- Decoded `currentDisplay:` and confirmed it does:
  - class check on the passed screen
  - bitmap-class check on `aDisplayScreen bits`
  - `Display become: aDisplayScreen`
  - `Display beDisplay`
- Decoded `displayExtent:` and confirmed it intentionally builds two displays:
  - temporary `width x 16`
  - real `width x fullHeight`
- Added a targeted trace mismatch diagnostic and ran:

```bash
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpTrace3FirstMismatchUpTo750 -v
```

- That showed the first real divergence before the fix:

```text
trace3 cycle 721 expected send "[cycle=721]  aDisplayScreen extent:offset:bits: aPoint aPoint aDisplayBitmap" but bytecode=112 method=0x4514(<Behavior>new:) was not a send
```

- Read primitive `71` in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) and confirmed it used `popInteger()`, which rejects any non-SmallInteger.
- Implemented `positiveIntegerValueOf` so primitive `71` can accept:
  - non-negative SmallIntegers
  - `LargePositiveInteger` byte objects
- Rewired primitive `71` to decode its size argument with `positiveIntegerValueOf`.
- Added a real regression test:
  - `TestTrace3DisplayStartupSendSelectorsMatch`
  - validates the startup selector flow through cycle `757`
- Re-ran validation:

```bash
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpDisplayStartupSendCycles -v
go test ./pkg/interpreter
go test ./...
go run ./cmd/st80-snapshot -cycles 1000 -output /tmp/st80-1000.png
go run ./cmd/st80-snapshot -cycles 2000000 -output /tmp/st80-2000000.png
bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/capture-ui-screenshot.sh
```

### Why
- The white `640x16` framebuffer was already proven to originate inside the VM, so SDL work would have been a distraction.
- The first startup trace mismatch is the highest-value bug locator available here because it turns “UI blank” into a precise control-flow failure.
- Primitive `71` sits exactly on the path for `DisplayBitmap new: 19200`, so once the trace showed a fallback into `Behavior>>new:`, it became the correct place to look.

### What worked
- The trace mismatch disappeared after the primitive fix:

```bash
RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpTrace3FirstMismatchUpTo750 -v
```

now passes.

- The startup selector probe now shows the expected second full-screen handoff:

```text
cycle=137 ... selector=currentDisplay:
cycle=733 ... selector=currentDisplay:
cycle=757 ... selector=restore
```

- The designated display surface is now structurally correct:

```text
cycles=1000 width=640 height=480 raster=40 blackPixels=0 whitePixels=307200
cycles=2000000 width=640 height=480 raster=40 blackPixels=0 whitePixels=307200
```

- The repository test suite remains green with:

```bash
go test ./...
```

### What didn't work
- The first compile after implementing `positiveIntegerValueOf` failed because I forgot to import `strconv`:

```text
pkg/interpreter/interpreter.go:709:17: undefined: strconv
```

- Fixing the display-allocation bug did not make the UI visually non-white. The designated framebuffer is now `640x480`, but it is still all white at both `1000` and `2000000` cycles.

### What I learned
- The `640x16` surface was not a generic `become:` or `beDisplay` bug.
- `displayExtent:` always intended to replace the temporary display later; our VM simply never executed that path correctly because `new:` rejected the large size.
- A startup trace mismatch around a single send was enough to isolate a primitive bug that directly affected the UI.
- The UI frontier has moved forward: the structural display surface bug is fixed, so the next problem is actual drawing into a now-correct framebuffer.

### What was tricky to build
- The tricky part was resisting the obvious but wrong target. The early evidence made `currentDisplay:` and `become:` look suspicious because the first designated display really was `640x16`. But method decoding showed that this temporary display is intentional. The real problem was that the later full-screen replacement never completed.
- The second subtlety was integer representation. `19200` looks like an ordinary size from the host side, but in this image it is a `LargePositiveInteger`. Any primitive that handles object sizes by only accepting SmallIntegers will silently fail on legitimate startup allocations.

### What warrants a second pair of eyes
- Review `positiveIntegerValueOf` in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) for integer-overflow handling and for whether other storage-management primitives should share the same decoder later.
- Review the new regression in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go) to make sure `trace3` up to cycle `757` is the right stability boundary for this startup behavior.
- Review whether any other primitives besides `71` are still incorrectly limited to SmallInteger arguments in code paths that the image expects to feed with `LargePositiveInteger`s.

### What should be done in the future
- Investigate why the corrected `640x480` designated display still remains all white.
- Compare later drawing-related sends and framebuffer hashes against the current UI capture path.
- Feed input/time primitives once the draw path is understood, rather than assuming interactivity alone will populate the screen.

### Code review instructions
- Start in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `positiveIntegerValueOf`
  - primitive `71` in `dispatchStorageManagementPrimitives`
- Then review [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestTrace3DisplayStartupSendSelectorsMatch`
  - `TestDumpTrace3FirstMismatchUpTo750`
  - `TestDumpDisplayStartupSendCycles`
- Then confirm the new direct-snapshot state with:
  - [display-1000.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-1000.png)
  - [display-2000000.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-2000000.png)
- Validate with:
  - `go test ./pkg/interpreter`
  - `go test ./...`
  - `RUN_ST80_DIAGNOSTIC=1 go test ./pkg/interpreter -run TestDumpDisplayStartupSendCycles -v`
  - `CYCLES=1000 bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh`
  - `CYCLES=2000000 bash ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh`

### Technical details
- Primitive `71` bug condition:
  - old code path used `popInteger()`
  - `popInteger()` accepts only SmallIntegers
  - `DisplayBitmap new: 19200` therefore failed
- Trace landmarks after the fix:
  - cycle `137` first `currentDisplay:`
  - cycle `733` second `currentDisplay:`
  - cycle `757` `restore`
- Snapshot metrics after the fix:
  - `display-1000.png`: `640x480`, all white
  - `display-2000000.png`: `640x480`, all white, same hash
- Updated UI capture artifact:
  - [st80-ui.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png)

## Step 5: Broader Pass on the Same Positive-Integer Decoding Bug Pattern

Once primitive `71` was fixed, the next obvious question was whether the same mistake existed elsewhere in the interpreter. The answer was yes: a cluster of primitives were still using `popInteger()` even though their arguments are semantically positive sizes or 1-based indices, not specifically SmallIntegers. That is the same conceptual bug, just without the same startup visibility.

I treated this as a pattern pass rather than another speculative rewrite. The guiding rule is straightforward: arithmetic primitives stay SmallInteger-only unless explicitly extended, but size/index primitives should accept any non-negative integer representation the image can legally produce. I encoded that distinction by adding a shared `popPositiveInteger` helper on top of `positiveIntegerValueOf`, then moved the clear positive-index sites over to it.

### Prompt Context

**User prompt (verbatim):** "can you do a pass on that bug pattern?"

**Assistant interpretation:** Audit the interpreter for other places where the same LargePositiveInteger-vs-SmallInteger decoding mistake could appear, fix the clear cases, and validate that the broader change is safe.

**Inferred user intent:** Prevent this from becoming a recurring class of VM bugs instead of only fixing the one display allocation that first exposed it.

**Commit (code):** d2d22d8 — "Broaden positive integer primitive decoding"

### What I did
- Audited all `popInteger()` call sites in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go).
- Split the sites into two buckets:
  - primitives that really are SmallInteger arithmetic/comparison/bit operations
  - primitives whose arguments are positive sizes or 1-based indices
- Added `popPositiveInteger` in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) so those second-bucket primitives can share the same non-negative integer decoding.
- Switched these primitives to `popPositiveInteger`:
  - `primitiveAt`
  - `primitiveAtPut`
  - `primitiveStringAt`
  - `primitiveStringAtPut`
  - storage primitives `68`, `69`, `71`, `73`, and `74`
- Added direct decoder tests in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestPositiveIntegerValueOfLargePositiveInteger`
  - `TestPositiveIntegerValueOfRejectsNegativeSmallInteger`
- Re-ran:

```bash
go test ./pkg/interpreter
go test ./...
```

### Why
- The display-allocation bug came from reusing a SmallInteger-only helper in a place whose semantics were broader.
- The same misuse on index-taking primitives would be a latent correctness bug for large collections, large strings, large bitmaps, or reflective access with large integer indices.
- Converting the clear positive-index sites now is cheaper than rediscovering them later one bug at a time.

### What worked
- The broader pass stayed small and explicit.
- Both new decoder tests pass.
- The existing startup trace regression and the full suite still pass:

```bash
go test ./pkg/interpreter
go test ./...
```

- The result is a cleaner semantic split in the interpreter:
  - `popInteger` remains for genuine SmallInteger primitives
  - `popPositiveInteger` is used where the image is allowed to supply a `LargePositiveInteger`

### What didn't work
- No runtime failure surfaced during this pass. It was preventive work based on the now-proven pattern rather than a second independently observed bug.

### What I learned
- The important distinction is not “integer vs non-integer.” It is “SmallInteger-only primitive contract” vs “general non-negative integer contract.”
- Once that distinction is made explicit in helper naming, the code is much easier to audit.
- The original UI bug was a high-signal example of a broader interpreter hygiene issue.

### What was tricky to build
- The tricky part was not overgeneralizing. Some `popInteger()` sites should remain SmallInteger-only, especially the arithmetic and bit-operation primitives. Widening those silently would blur interpreter semantics and make later LargeInteger work harder to reason about.
- The right pass was therefore selective: broaden only the sites that clearly consume sizes or 1-based indices.

### What warrants a second pair of eyes
- Review whether any remaining `popInteger()` sites outside arithmetic/bit operations are still semantically too narrow.
- Review whether `primitiveSize` should eventually grow a LargePositiveInteger return path for collections larger than SmallInteger range, rather than failing back to Smalltalk.

### What should be done in the future
- Keep using the “primitive contract” distinction during future audits:
  - SmallInteger arithmetic stays narrow
  - size/index consumers use positive-integer decoding
- Revisit other result-producing primitives later if the image starts needing LargePositiveInteger results rather than only LargePositiveInteger inputs.

### Code review instructions
- Start in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `popPositiveInteger`
  - all call sites converted from `popInteger`
- Then review [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestPositiveIntegerValueOfLargePositiveInteger`
  - `TestPositiveIntegerValueOfRejectsNegativeSmallInteger`
  - `TestTrace3DisplayStartupSendSelectorsMatch`
- Validate with:
  - `go test ./pkg/interpreter`
  - `go test ./...`

### Technical details
- Converted positive-index / positive-size sites:
  - subscript primitives `60`, `61`, `63`, `64`
  - storage primitives `68`, `69`, `71`, `73`, `74`
- Left SmallInteger-only sites unchanged:
  - arithmetic primitives `1` through `18` where currently implemented
- This change is preventive hardening built directly from the primitive-71 bug pattern.

## Step 6: BitBlt Field Order Bug Fix

The next major UI/debugging breakthrough came from stopping the generic "blank white UI" investigation and reducing it to one concrete `copyBits` send. By this point, the display-allocation bug was already fixed: the designated display form was the correct `640x480` size. The remaining question was why the display still stayed effectively white.

The crucial evidence came from a very narrow diagnostic around the first `Form>>fill:rule:mask:` path. That probe showed:

- the image really does send `copyBits`
- the receiver at that send site is a real `BitBlt`
- method lookup resolves to `BitBlt>>copyBits`
- primitive decoding resolves to primitive `96`

So the bug was no longer "the image never draws" or "the primitive is not wired." The first real hint was that `copyBits` still recorded zero successful copy operations and there was no primitive failure detail either. That combination only made sense if `primitiveCopyBits` was returning early without error.

Dumping the live `BitBlt` receiver fields at that first send exposed the mismatch. The object had `640` and `480` in the slots the Go constants were interpreting as `sourceX` and `sourceY`, while the slots the Go constants were interpreting as `clipWidth` and `clipHeight` were `0` and `0`. In other words, my `BitBlt` tail-slot mapping was wrong.

The correct tail ordering is:

- `sourceX`
- `sourceY`
- `clipX`
- `clipY`
- `clipWidth`
- `clipHeight`

The Go code had assumed:

- `clipX`
- `clipY`
- `clipWidth`
- `clipHeight`
- `sourceX`
- `sourceY`

That mistake meant `primitiveCopyBits` immediately clipped every operation down to zero width and zero height, then returned success without touching the framebuffer.

### What I did
- Corrected the `BitBlt` slot constants in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `BitBltSourceXIndex = 8`
  - `BitBltSourceYIndex = 9`
  - `BitBltClipXIndex = 10`
  - `BitBltClipYIndex = 11`
  - `BitBltClipWidthIndex = 12`
  - `BitBltClipHeightIndex = 13`
- Added a normal regression test in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles`
- Captured a direct post-fix framebuffer artifact at [display-5000.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-5000.png)
- Refreshed the off-screen SDL window capture at [st80-ui.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png)
- Wrote a detailed intern-facing explanation in [04-bitblt-field-order-bug-writeup.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/04-bitblt-field-order-bug-writeup.md)

### Why
- The bug was not in SDL.
- The bug was not in display allocation anymore.
- The bug was not in primitive-number dispatch.
- The bug was a specification-alignment error: a wrong field-order assumption for a layout-sensitive primitive object.

### What worked
- The framebuffer immediately stopped being all white.
- A direct snapshot at 5000 cycles reported:

```text
cycles=5000 width=640 height=480 blackPixels=12817 whitePixels=294383
```

- The SDL capture also stopped being completely blank.
- The new regression test makes the old symptom hard to reintroduce silently.

### What didn't work
- The rendered output is still not visually correct enough to call the UI finished. The fix restored drawing, but the image currently looks structurally wrong rather than blank. That means the next frontier is likely in BitBlt/display semantics rather than allocation or slot-order plumbing.

### What I learned
- A primitive can be "wired correctly" and still be functionally dead if the receiver layout assumptions are off.
- The right debugging pattern for this class of bug is:
  - prove the send happens
  - prove the receiver class
  - dump the live object fields
  - compare that concrete object against the assumed constant ordering
- This is exactly the kind of mismatch that justifies a more systematic Blue Book audit.

## Step 7: Intern OCR Handoff For A Systematic Blue Book Audit

After the `BitBlt` field-order bug, it became clear that ad hoc debugging is not enough for the display/rendering classes. The bug itself was simple once exposed: the first live `BitBlt` object showed `640x480` in the slots I had interpreted as `sourceX/sourceY`, while the slots I had interpreted as `clipWidth/clipHeight` were `0,0`. That meant the Go VM constants were not matching the object layout implied by the image and the book. The right response is not just "fix this one constant set." The right response is to build a repeatable audit path from the Blue Book into structured reference data.

I did not want to hand an intern a vague instruction like "OCR the book and summarize it." That would likely produce a lot of text and not much implementation value. What we actually need is a machine-usable and reviewer-usable extraction: class layouts, ordered method signatures, primitive-number mappings, field-access notes, and page references that can be compared directly against [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go).

### Prompt Context

**User prompt (verbatim):** "can you make a set of instructions for an intern to do the OCR and extract all then ecessary info from the book? they're really good at OCR"

**Assistant interpretation:** Create an intern-facing reference document that explains how to OCR the Blue Book and, more importantly, how to extract the resulting text into structured implementation facts that will help the VM audit.

**Inferred user intent:** Turn OCR into a concrete audit pipeline rather than a generic documentation exercise, with emphasis on field ordering, argument ordering, primitive semantics, and exact page citation.

### What I did
- Added [03-bluebook-ocr-extraction-instructions-for-intern.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/03-bluebook-ocr-extraction-instructions-for-intern.md).
- Made the document explicit about:
  - using the Blue Book as the primary source
  - not using existing implementations as reference
  - preserving ambiguity instead of guessing
  - attaching page references to every extracted fact
- Specified the exact outputs we want:
  - OCR notes
  - page/topic index
  - class layout table
  - method signature table
  - primitive audit table
  - graphics/BitBlt narrative audit
  - object-memory narrative audit
  - open-questions list
- Prioritized the first extraction targets:
  - `BitBlt`
  - `Form`
  - `DisplayScreen`
  - bitmap/display storage classes
  - `Point`
  - `Rectangle`
  - primitive/header/object-memory topics
- Included concrete CSV schemas and examples so the intern knows what "good" output looks like.
- Updated [tasks.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/tasks.md) to add the systematic Blue Book OCR audit as a tracked follow-up.

### Why
- The `BitBlt` bug showed that a single wrong slot-order assumption can make the UI look completely wrong while the interpreter still appears superficially healthy.
- A reviewer needs to be able to answer "what exact order does the book require?" quickly and with page citations.
- Raw OCR text is not enough. The useful artifact is a structured extraction that maps directly onto VM constants and primitive implementations.

### What worked
- The handoff is concrete enough that an intern can execute it without guessing the desired output format.
- The deliverables are directly tied to the kinds of bugs we have already seen:
  - class layout mismatches
  - method argument order mismatches
  - primitive number / stack-contract mismatches
- The document is reusable for future audits outside graphics, especially object memory and primitive dispatch.

### What didn't work
- I did not yet produce the extracted tables themselves. This step is the instruction set for that work, not the completed audit.

### What I learned
- The right unit of extraction is not "chapter summary." It is "implementation fact with citation."
- The most valuable audit documents are tables, not prose, as long as they preserve order and page references.
- The classes most likely to hide layout bugs are the ones that bridge image-level objects and interpreter-level constants.

### What should be done in the future
- Have the intern produce the OCR pack under this ticket.
- Use the resulting tables to do a line-by-line audit of:
  - field-index constants
  - primitive dispatch tables
  - constructor/setter argument ordering
  - object-memory layout assumptions
- Turn any confirmed mismatch into:
  - a code fix
  - a regression test
  - a short writeup in this ticket

## Step 8: BitBlt CopyLoop Row-Advance Fix

The next graphics bug was a different class of mistake from the `BitBlt` field-order bug. After the field-order fix, the UI was no longer blank, but it was still obviously wrong: the framebuffer showed a distorted horizontal-band image, and all non-white pixels were confined to the top 256 rows. That exact `255/256` boundary was suspicious enough that I stopped looking at screenshots as "visual weirdness" and turned it into a direct invariant: are we actually writing display words beyond row 255?

The answer was no. Instrumentation showed:

```text
displayWordWrites=49612 changed=37345 writeIndexRange=0..10241 changedIndexRange=0..10227
```

Given a raster of 40 words per row, index `10240` is exactly row `256`. That meant the copy loop itself, not the snapshot/UI code, was effectively stalling in the top half of the display. The key clue was that this happened even though the very first display `BitBlt` had geometry that looked perfectly sane:

- full-screen fill at `640x480`
- later centered rectangles like `502x304`
- text blits landing inside a plausible clip rectangle

If the geometry was sane but writes never escaped row 255, the translation of the Blue Book copy loop had to be wrong.

That turned out to be exactly the issue. In the Blue Book simulation, `sourceIndex` and `destIndex` are the running indices advanced inside the inner horizontal loop, and `sourceDelta` / `destDelta` are added to those already-advanced running values after the row finishes. My Go version introduced `lineSourceIndex` and `lineDestIndex` temporaries inside each row, but then I updated the base indices like this:

```go
sourceIndex += sourceDelta
destIndex += destDelta
```

That is only correct if the base indices themselves were advanced inside the row. They were not. The real running position after a row lived in `lineSourceIndex` / `lineDestIndex`. So the code was starting each new row from the wrong place.

The fix was:

```go
sourceIndex = lineSourceIndex + sourceDelta
destIndex = lineDestIndex + destDelta
```

That re-aligned the Go implementation with the actual state transitions of the Blue Book `copyLoop`.

### What I did
- Corrected row-to-row index progression in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) so the next-row starting positions are based on the post-row running indices.
- Strengthened [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles` now also requires visible content below row 255
- Added focused diagnostics to confirm:
  - the first display `BitBlt` rectangles were plausible
  - display word writes had been artificially capped at row 255 before the fix
- Refreshed ticket artifacts:
  - [display-5000.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-5000.png)
  - [display-50000.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-50000.png)
  - [st80-ui.png](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png)
- Wrote a detailed writeup in [05-bitblt-copyloop-row-advance-bug-writeup.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/05-bitblt-copyloop-row-advance-bug-writeup.md)

### Why
- The field-order fix proved the VM could draw at all.
- The row-255 cutoff proved the remaining bug was in the copy-loop implementation, not SDL, not snapshotting, and not generic display designation.
- The Blue Book simulation was detailed enough to compare state transitions directly, not just high-level intent.

### What worked
- The post-fix `5000`-cycle framebuffer now shows a recognizable windowed scene instead of a corrupted band.
- The post-fix `50000`-cycle snapshot shows a visible `System Browser`.
- The off-screen SDL capture now also shows that recognizable UI state.
- Black-pixel counts increased sharply:

```text
cycles=50000 blackPixels=112228 whitePixels=194972
```

### What didn't work
- The UI is still not interactive. Rendering is now much more believable, but keyboard, mouse, timer, and cursor host integration remain open.
- I have not yet done a fresh long-run semantic audit of all remaining BitBlt edge cases; this fix addresses the row progression bug specifically.

### What I learned
- A literal-looking translation of a simulation can still be wrong if the meaning of the mutated variables changes.
- The best clue was not visual style. It was the exact row boundary where rendering stopped.
- Once the bug was phrased as "why do display writes stop at row 255?" the diagnosis got much faster.

## Step 9: Passive Mouse Point And Cursor Location Support

Once the UI was visually recognizable, the next gap was obvious: the host window still had no path for feeding live pointer position into the image. I did not want to jump immediately into the full asynchronous input-event buffer described in the Blue Book because that includes semaphores, event words, sample intervals, keyboard/button encoding, and the `primInputWord` machinery. That is the right destination, but not the right next slice.

The smallest useful next step was passive mouse support:

- primitive `90` (`primitiveMousePoint`)
- primitive `91` (`primitiveCursorLocPut`)

The Blue Book description for `primitiveMousePoint` is straightforward: allocate a new `Point` and store the current pointing-device location in its x and y fields. The description for `primitiveCursorLocPut` is also narrow: update the cursor location from the argument `Point`, and if the cursor is linked to the pointing device, update the pointing-device location as well.

That meant I could add useful host/input plumbing without yet implementing the buffered event stream (`93` through `95`) or the timer primitives.

### What I did
- Extended [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) with minimal input state:
  - `mouseX`
  - `mouseY`
  - `cursorX`
  - `cursorY`
- Added `SetMousePoint(x, y)` so the host loop can update the current pointer location.
- Implemented primitive `90`:
  - pops the receiver
  - allocates a fresh `Point`
  - stores the current `mouseX` / `mouseY`
  - pushes the new `Point`
- Implemented primitive `91`:
  - pops the argument `Point`
  - pops the receiver
  - updates `cursorX` / `cursorY`
  - if `cursorLinked` is true, also updates `mouseX` / `mouseY`
  - pushes the receiver back
- Updated [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) so SDL mouse motion and button events map window coordinates into the interpreter's logical display coordinates and feed them through `SetMousePoint`.
- Added direct primitive tests in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestPrimitiveMousePointReturnsConfiguredPoint`
  - `TestPrimitiveCursorLocPutUpdatesCursorAndReturnsReceiver`
  - `TestPrimitiveCursorLocPutUpdatesMouseWhenLinked`
- Validated the new slice with:

```bash
go test ./pkg/interpreter -run 'TestPrimitiveMousePointReturnsConfiguredPoint|TestPrimitiveCursorLocPutUpdatesCursorAndReturnsReceiver|TestPrimitiveCursorLocPutUpdatesMouseWhenLinked|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles'
SDL_VIDEODRIVER=dummy go run ./cmd/st80-ui -max-cycles 50000 -cycles-per-frame 500
```

### Why
- Passive pointer location is the cheapest real input bridge.
- It is directly specified in the Blue Book.
- It gives the image a host-backed pointer location without dragging in the entire event-buffer design all at once.

### What worked
- The primitive tests pass.
- The dummy SDL UI path still runs cleanly after the host event-loop changes.
- The new code stays small and isolated:
  - interpreter bookkeeping
  - primitive `90`
  - primitive `91`
  - mouse coordinate mapping in the SDL loop

### What didn't work
- This does not yet implement:
  - keyboard input
  - button state words
  - input semaphores
  - sample interval handling
  - buffered event words via primitive `95`
- The Smalltalk cursor is still not visibly overlaid by the host UI. The VM now has more accurate cursor/pointer state, but the host loop does not yet synthesize the cursor image.

### What I learned
- There is a clean incremental path for host input work: passive pointer state first, then buffered events.
- The right separation is:
  - passive polling primitives (`90`, `91`)
  - event-buffer primitives (`93`, `94`, `95`)
  - time/timer primitives (`98`, `99`, `100`)
- Doing the passive slice first reduces the number of moving parts in the next input pass.

## Step 10: Active Input Event Buffer, SDL Keyboard/Button Wiring, and the OOP-0 Sentinel Fix

With passive mouse-point support in place, the next honest gap was no longer “some input exists.” It was that the actual Blue Book event-buffer path still did not exist. The image could poll the pointing-device location, but it still could not consume the event-initiated stream behind `primInputSemaphore:`, `primSampleInterval:`, and `primInputWord`. That meant no buffered mouse movement, no mouse buttons, and no keyboard stream for the Smalltalk side to wait on and drain.

I implemented that path in the order the Blue Book describes it: first the interpreter-side buffer and primitives, then the host-side SDL translation. The first pass immediately exposed a real bug in my own code: I treated OOP `0` as an “unset” sentinel for the registered input semaphore, but newly allocated objects can legitimately land at OOP `0` in this image. That silently suppressed `asynchronousSignal:` even though input words were being queued correctly. The focused regression caught it immediately.

### Prompt Context

**User prompt (verbatim):** "ok, in the meantime, let's continue"

**Assistant interpretation:** Keep pushing the UI/runtime work forward without pausing, taking the next technically correct slice after passive pointer support.

**Inferred user intent:** Turn the visible but still mostly host-shell UI into a more faithful Smalltalk environment by filling in the remaining runtime plumbing, while keeping the diary and bug writeups detailed enough for later review.

### What I did
- Extended [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) with active input-buffer state:
  - registered input semaphore
  - sample interval in milliseconds
  - ring buffer for queued 16-bit event words
  - last-event and last-mouse-event timestamps
- Added Blue Book-style 16-bit boxing helper:
  - `positive16BitIntegerFor`
  - returns either `SmallInteger` or `LargePositiveInteger`
- Implemented primitive `93` (`primitiveInputSemaphore`)
- Implemented primitive `94` (`primitiveSampleInterval`)
- Implemented primitive `95` (`primitiveInputWord`)
- Added host-facing interpreter helpers:
  - `RecordMouseMotion`
  - `RecordMouseButton`
  - `RecordDecodedKey`
- Updated [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go):
  - start/stop SDL text input
  - feed mouse motion into the active movement stream
  - feed left/middle/right mouse buttons as parameter codes `128`, `129`, `130`
  - feed text input as decoded-keyboard ASCII keypresses
  - feed editing keys (`Backspace`, `Tab`, `Return`, `Escape`, `Delete`) on `KEYDOWN`
- Added focused regressions in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestPrimitiveInputSemaphoreStoresSemaphoreAndReturnsReceiver`
  - `TestPrimitiveSampleIntervalStoresMillisecondsAndReturnsReceiver`
  - `TestPrimitiveInputWordReturnsQueuedWord`
  - `TestRecordMouseMotionQueuesTimedCoordinatesAndSignalsSemaphore`
  - `TestRecordMouseMotionRespectsSampleInterval`
  - `TestRecordDecodedKeyQueuesOnAndOffWords`
- Ran formatting:

```bash
gofmt -w pkg/interpreter/interpreter.go pkg/interpreter/interpreter_test.go pkg/ui/ui.go
```

- Ran focused validation:

```bash
go test ./pkg/interpreter -run 'TestPrimitive(InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord|MousePointReturnsConfiguredPoint|CursorLocPutUpdatesCursorAndReturnsReceiver|CursorLocPutUpdatesMouseWhenLinked)|TestRecord(MouseMotionQueuesTimedCoordinatesAndSignalsSemaphore|MouseMotionRespectsSampleInterval|DecodedKeyQueuesOnAndOffWords)|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles'
SDL_VIDEODRIVER=dummy go test ./pkg/ui ./cmd/st80-ui
```

### Why
- The Blue Book splits passive pointer polling from the event-initiated input buffer. I had already done the passive half, so the next real missing semantics were `93`, `94`, and `95`.
- `primInputWord` needed the correct 16-bit boxing path, otherwise high-bit event words would have failed as soon as button/type codes crossed the `SmallInteger` range.
- Wiring SDL text input and editing keys separately matches the real host event model better than trying to synthesize everything out of raw keycodes alone.

### What worked
- The interpreter now has a real buffered event-word stream with per-word deferred semaphore signaling.
- Mouse motion now queues:
  - a time word
  - X location
  - Y location
- Mouse buttons now queue type-3 / type-4 words using the Blue Book button parameters.
- Text input now produces decoded-keyboard style on/off pairs.
- The sample interval now suppresses overly-frequent movement events.
- Focused regressions are green.
- SDL-side package builds/tests remain green under the dummy driver.

### What didn't work
- My first version of the queue-signaling path assumed `inputSemaphore != 0` meant “a real semaphore is present.” That was wrong in this object memory: a freshly allocated Semaphore can legitimately have OOP `0`.
- The initial focused run failed with:

```text
--- FAIL: TestRecordMouseMotionQueuesTimedCoordinatesAndSignalsSemaphore (0.00s)
    interpreter_test.go:441: expected 3 deferred semaphore signals, got 0
```

- That failure was entirely my own bug, not a spec ambiguity.
- This slice still does not implement:
  - clock/timer primitives `98`, `99`, `100`
  - full control/meta-key coverage beyond ASCII text and the main editing keys
  - visible host-side cursor overlay

### What I learned
- The Blue Book’s “buffered stream of 16-bit words” is straightforward once the queue and boxing helpers are explicit.
- OOP `0` is not a safe generic sentinel in this VM. The only reliable “no object” sentinel is `nil` (`2`), plus explicit validity/class checks where needed.
- SDL text input is the right path for decoded-keyboard semantics because it already incorporates shift/layout translation.

### What was tricky to build
- The tricky part was not the queue itself. It was respecting all the small semantic edges at once:
  - time words are per event group
  - `primitiveInputWord` must return unsigned 16-bit values via either `SmallInteger` or `LargePositiveInteger`
  - movement throttling applies to pointer movement, not all event kinds
  - decoded keyboard input is not the same as raw key up/down input
- The first-pass semaphore bug was subtle because the queue contents were correct. Only the deferred-signal side effect was wrong. Without the focused regression on `semaphoreIndex`, it would have been easy to miss.

### What warrants a second pair of eyes
- Review the decoded-keyboard host mapping in [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go), especially where printable text comes from `TextInputEvent` but editing keys come from `KeyboardEvent`.
- Review the choice to drop whole events when the ring buffer lacks capacity rather than partially enqueueing them. That is intentional, but worth confirming.
- Review the use of SDL event timestamps as the active-input clock source for buffered words.

### What should be done in the future
- Implement the remaining clock/timer primitives so the same host event/tick story is complete.
- Expand decoded-keyboard coverage for control/meta-key edge cases if the image expects more than text-plus-editing keys.
- Exercise the live UI on a real desktop session and confirm the image actually consumes the new buffered events as intended.

### Code review instructions
- Start in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `positive16BitIntegerFor`
  - `primitiveInputSemaphore`
  - `primitiveSampleInterval`
  - `primitiveInputWord`
  - `queueInputEvent`
  - `RecordMouseMotion`
  - `RecordMouseButton`
  - `RecordDecodedKey`
- Then review [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go):
  - `processEventsAndPresent`
  - `mapWindowToLogicalPoint`
  - `mouseButtonParameter`
  - `specialKeyParameter`
- Then read [06-input-event-buffer-oop-zero-sentinel-bug-writeup.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/06-input-event-buffer-oop-zero-sentinel-bug-writeup.md) for the first-pass regression and fix.
- Validate with:
  - `go test ./pkg/interpreter -run 'TestPrimitive(InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord|MousePointReturnsConfiguredPoint|CursorLocPutUpdatesCursorAndReturnsReceiver|CursorLocPutUpdatesMouseWhenLinked)|TestRecord(MouseMotionQueuesTimedCoordinatesAndSignalsSemaphore|MouseMotionRespectsSampleInterval|DecodedKeyQueuesOnAndOffWords)|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles'`
  - `SDL_VIDEODRIVER=dummy go test ./pkg/ui ./cmd/st80-ui`

### Technical details
- Blue Book reference used in this step:
  - input/output primitives on pages `648` to `652`
  - 16-bit positive integer boxing on pages `617` to `618`
- The event-word encoding implemented here is:
  - type `0`: delta-time
  - type `1`: mouse X
  - type `2`: mouse Y
  - type `3`: device on
  - type `4`: device off
  - type `5`: absolute time + two trailing words
- The specific first-pass bug fix was changing the input-semaphore signal guard from “non-zero and non-nil” to “non-nil”, because OOP `0` can still be a valid allocated object.

## Step 11: Host Clock And Timer Primitive Support

After the active input buffer landed, the remaining host-runtime gap was the clock side of the same subsystem. The Blue Book groups `primitiveTimeWordsInto`, `primitiveTickWordsInto`, and `primitiveSignalAtTick` with the I/O primitives for a reason: the image needs both incoming events and a notion of host time in order to wake waiting processes, poll delays, and maintain the rest of the interactive runtime.

I implemented the timer slice narrowly and kept it test-driven. That meant: add explicit host clock state inside the interpreter, store 32-bit values into byte-indexable objects in a consistent byte order, and drive scheduled semaphore signaling through the interpreter’s existing deferred-signal path. The focused tests also exposed one subtle fixture issue: when I allocate a fresh `Semaphore` for tests, its `ExcessSignals` field starts as `nil`, but `synchronousSignal` expects SmallInteger zero there. That was a test-fixture initialization problem, not a runtime-spec problem, and I recorded it because it is easy to trip over again later.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Continue filling in the remaining VM/host boundary without stopping after the active input work.

**Inferred user intent:** Reduce the remaining “stub” surface area in the UI/runtime path by implementing the next spec-backed subsystem instead of leaving the host loop half-wired.

### What I did
- Extended [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) with host clock state:
  - `timeNow`
  - `tickStart`
  - `timerSemaphore`
  - `timerTickDeadline`
  - `timerActive`
- Implemented:
  - `primitiveSecondClockInto` for primitive `98`
  - `primitiveMillisecondClockInto` for primitive `99`
  - `primitiveSignalAtMilliseconds` for primitive `100`
- Added supporting helpers:
  - `isByteIndexableWithLengthAtLeast`
  - `storeUint32LE`
  - `fetchUint32LE`
  - `currentSecondClock`
  - `currentMillisecondClock`
- Updated `checkProcessSwitch` so an armed timer deadline turns into a deferred semaphore signal at the scheduler boundary.
- Added direct tests in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - `TestPrimitiveSecondClockIntoStoresLittleEndianSeconds`
  - `TestPrimitiveMillisecondClockIntoStoresLittleEndianTicks`
  - `TestPrimitiveSignalAtMillisecondsSignalsImmediatelyWhenPastDue`
  - `TestPrimitiveSignalAtMillisecondsSchedulesFutureSignal`
- Added `newTestSemaphore` so timer and signal tests use a correctly initialized `Semaphore` object with `ExcessSignals` set to SmallInteger zero.
- Ran formatting:

```bash
gofmt -w pkg/interpreter/interpreter.go pkg/interpreter/interpreter_test.go
```

- Ran focused validation:

```bash
go test ./pkg/interpreter -run 'TestPrimitive(SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal|InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord)|TestRecord(MouseMotionQueuesTimedCoordinatesAndSignalsSemaphore|MouseMotionRespectsSampleInterval|DecodedKeyQueuesOnAndOffWords)|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles'
```

### Why
- The Blue Book explicitly specifies `98`, `99`, and `100` as part of the same I/O boundary as the input primitives.
- The UI/runtime bridge was still incomplete while these remained stubs.
- Timer signaling belongs in the same deferred scheduler path as other asynchronous semaphore signals, so it was best to integrate it there instead of inventing a separate wakeup mechanism.

### What worked
- Primitive `98` now writes seconds-since-1901 into the first four bytes of the target object.
- Primitive `99` now writes the host millisecond clock into the first four bytes of the target object.
- Primitive `100` now:
  - signals immediately if the requested deadline has already passed
  - arms a future deadline otherwise
  - clears any waiting timer if the first argument is not a valid `Semaphore`
- The future-timer path now wakes through `checkProcessSwitch`, which matches the interpreter’s existing asynchronous-signal model.
- Focused tests are green.

### What didn't work
- The first future-timer test failed because the freshly allocated `Semaphore` fixture had `ExcessSignals = nil` instead of SmallInteger zero.
- The exact failure was:

```text
--- FAIL: TestPrimitiveSignalAtMillisecondsSchedulesFutureSignal (0.00s)
    interpreter_test.go:613: expected scheduled timer to signal semaphore once, got excessSignals=2
```

- That was not a bug in the timer implementation. It was a fixture-construction bug in the test.
- I have not yet done a live desktop-session audit of delayed-process wakeups using the new timer primitives.

### What I learned
- The object-memory byte conventions in this VM are easiest to keep straight if I treat 16-bit and 32-bit positive integers consistently as little-endian byte sequences.
- The scheduler boundary remains the right place to turn host-side asynchronous conditions into Smalltalk-visible semaphore signals.
- Freshly instantiated pointer objects are not automatically valid fixtures for every class; class-specific fields sometimes need explicit initialization in tests.

### What was tricky to build
- The trickiest part was not writing bytes. It was choosing where the timer should actually fire. Firing it directly inside the primitive would skip the interpreter’s normal deferred-signal path. Firing it inside `checkProcessSwitch` keeps timer wakeups aligned with the same scheduling rules used for input-event semaphores.
- The second tricky part was separating runtime bugs from fixture bugs. The initial timer failure looked like a double-signal issue in the implementation, but the real problem was that the test `Semaphore` object started in an impossible state for `synchronousSignal`.

### What warrants a second pair of eyes
- Review the choice to encode the 32-bit clock words as little-endian byte sequences. It is internally consistent with the existing integer boxing helpers, but it is worth keeping under review when the image-side consumers are exercised more heavily.
- Review whether `currentSecondClock` should remain UTC-based or should eventually follow a more image-specific/local-time convention once more time behavior is exercised.
- Review whether the timer wakeup should be checked only in `checkProcessSwitch` or also somewhere else if later behavior shows missed opportunities.

### What should be done in the future
- Exercise the live image paths that consume these timer primitives and confirm delayed wakeups behave correctly in the running UI.
- Keep auditing byte-order assumptions anywhere the image exchanges raw multi-byte values with the host.
- If needed, widen tests beyond focused unit coverage into longer runtime scenarios involving `Delay` / scheduler wakeups.

### Code review instructions
- Start in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `primitiveSecondClockInto`
  - `primitiveMillisecondClockInto`
  - `primitiveSignalAtMilliseconds`
  - `storeUint32LE`
  - `fetchUint32LE`
  - the timer check at the top of `checkProcessSwitch`
- Then review [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):
  - the four new timer tests
  - `newTestSemaphore`
- Then read [07-timer-primitives-byte-order-and-semaphore-initialization-note.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/07-timer-primitives-byte-order-and-semaphore-initialization-note.md).

### Technical details
- Blue Book reference used in this step:
  - I/O primitive time/tick/timer behavior on pages `651` to `652`
- The runtime choices made here are:
  - seconds clock: current UTC time minus `1901-01-01T00:00:00Z`
  - millisecond clock: elapsed host time since interpreter creation
  - timer wakeup: checked in `checkProcessSwitch`
- The focused regression command for this step was:

```bash
go test ./pkg/interpreter -run 'TestPrimitive(SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal|InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord)|TestRecord(MouseMotionQueuesTimedCoordinatesAndSignalsSemaphore|MouseMotionRespectsSampleInterval|DecodedKeyQueuesOnAndOffWords)|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles'
```

## Step 12: Host Cursor Overlay Support

After input and timers, the remaining obvious UI mismatch was the cursor. The interpreter was already tracking:

- the designated cursor form via `beCursor`
- the current cursor location via `cursorLocPut:` / cursor-link behavior

but the host window still ignored all of that and rendered only the raw display bitmap. The Blue Book description is explicit that the cursor is ORed into the display on update, so the host window was still missing part of the visible Smalltalk UI even though the underlying VM state already existed.

I implemented this as a host overlay instead of trying to mutate the display form in object memory. The interpreter now exports a cursor snapshot in the same style as the display snapshot, and the UI layer overlays those 1-bit cursor words on top of the expanded ARGB framebuffer. That keeps the object memory honest and makes the host rendering path reflect the intended display semantics without turning cursor display into another in-memory `BitBlt`.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Keep closing the remaining visible UI/runtime gaps instead of stopping at partial interactivity.

**Inferred user intent:** Make the host window reflect more of the real Smalltalk UI state, not just the display buffer in isolation.

### What I did
- Added `CursorSnapshot` in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go).
- Added `CursorSnapshot()` on the interpreter so the UI can fetch:
  - cursor form OOP
  - cursor x/y
  - cursor width/height
  - cursor raster
  - copied cursor words
- Updated [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go):
  - fetch cursor snapshot alongside the display snapshot
  - overlay cursor bits after display expansion
  - clip overlay to the visible display bounds
- Updated [snapshot.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/snapshot.go) so direct non-SDL framebuffer snapshots use the same cursor overlay path as the live SDL UI.
- Added [ui_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui_test.go) with:
  - `TestCopyDisplayBitsOverlaysCursorBits`
- Ran formatting:

```bash
gofmt -w pkg/interpreter/interpreter.go pkg/ui/ui.go pkg/ui/snapshot.go pkg/ui/ui_test.go
```

- Ran focused validation:

```bash
go test ./pkg/ui ./pkg/interpreter -run 'TestCopyDisplayBitsOverlaysCursorBits|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles|TestPrimitive(SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal)'
SDL_VIDEODRIVER=dummy go run ./cmd/st80-ui -max-cycles 50000 -cycles-per-frame 500
```

### Why
- The host UI was still visibly incomplete while ignoring the designated Smalltalk cursor.
- The Blue Book semantics are simple here: cursor bits are ORed into the displayed screen image.
- Doing this as a host overlay is less invasive than mutating the display bitmap in object memory and avoids mixing transient presentation state into the VM’s persistent image memory.

### What worked
- The renderer now has access to the real cursor form and location.
- The display conversion path and snapshot path now share the same overlay behavior.
- The focused UI regression passes.
- The dummy SDL UI smoke run still exits cleanly.

### What didn't work
- The first cursor-overlay edit broke the non-SDL snapshot build because `copyDisplayBits` gained cursor parameters and [snapshot.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/snapshot.go) still called the old signature.
- The build failure was:

```text
pkg/ui/snapshot.go:61:42: not enough arguments in call to copyDisplayBits
    have ([]uint32, interpreter.DisplaySnapshot)
    want ([]uint32, interpreter.DisplaySnapshot, bool, interpreter.CursorSnapshot)
```

- That was a simple missed call-site update, not a design problem.
- I have not yet captured a real desktop screenshot proving a visible cursor shape/location in a live session.

### What I learned
- The cursor path fits naturally into the existing snapshot/export boundary. It did not require a separate rendering subsystem.
- Sharing the overlay logic between `st80-ui` and `st80-snapshot` is important; otherwise the debug path and the live path drift apart.
- Cursor display is another place where “host rendering behavior” should be explicit and testable, not implied.

### What was tricky to build
- The tricky part was choosing the layering. It is tempting to say “the cursor is part of the screen, so just modify the display words,” but that would make the exported framebuffer less faithful to the actual VM object memory. The cleaner split is:
  - display form snapshot from VM memory
  - cursor form snapshot from VM memory
  - OR-style composition in the host renderer
- The other small trap was making sure the direct snapshot path used exactly the same composition routine as the SDL path.

### What warrants a second pair of eyes
- Review the decision to render cursor bits as black ORed pixels. That matches the current 1-bit rendering convention, but should still be visually confirmed in a live session.
- Review whether the cursor origin semantics need an offset adjustment once real cursor motion is inspected visually.

### What should be done in the future
- Verify visible cursor behavior on a real desktop session.
- Check whether the image expects any cursor hotspot offset beyond the raw cursor location fields.
- If needed, add a richer cursor-specific diagnostic snapshot once live interaction is exercised more heavily.

### Code review instructions
- Start in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `CursorSnapshot`
  - `CursorSnapshot()`
- Then review [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go):
  - `copyDisplayBits`
  - `overlayCursorBits`
- Then review [snapshot.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/snapshot.go) and [ui_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui_test.go).

### Technical details
- Rendering rule used here:
  - each set cursor bit overwrites the target pixel to black
  - unset cursor bits leave the underlying display pixel unchanged
- Validation commands for this step:

```bash
go test ./pkg/ui ./pkg/interpreter -run 'TestCopyDisplayBitsOverlaysCursorBits|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles|TestPrimitive(SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal)'
SDL_VIDEODRIVER=dummy go run ./cmd/st80-ui -max-cycles 50000 -cycles-per-frame 500
```

## Step 13: Off-Screen Input Exercise Script And The No-Delta Result

At this point the host UI has real display rendering, active input primitives, timer primitives, and cursor overlay support. The next practical question was no longer “can the host deliver events?” in the abstract. It was “does the live UI visibly react when I inject events into the off-screen window?” That is a sharper question, and the answer matters because it tells me whether the next work should stay on the host side or move into image-level event-consumption debugging.

I added an off-screen exercise script that runs `st80-ui` under `Xvfb`, injects a small mouse/keyboard sequence via `xdotool`, and captures before/after/diff screenshots. The result was useful even though it was not the answer I wanted: the captured before/after images were visually identical. That means the next frontier is not “make Xvfb screenshots possible” or “can the host generate input events at all.” The next frontier is proving where the live path is dropping or ignoring the events.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Keep pushing forward until the UI path is not just present but demonstrably useful, and document each concrete debugging slice on the way.

**Inferred user intent:** Replace guesswork with concrete evidence about what still is and is not working in the live UI path.

### What I did
- Added ticket-local helper [exercise-ui-input-and-capture.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-ui-input-and-capture.sh).
- The script:
  - starts `Xvfb`
  - runs `st80-ui`
  - finds the `Smalltalk-80` window
  - captures a before screenshot
  - injects:
    - mouse move
    - left click
    - typed `a`
    - `Return`
  - captures an after screenshot
  - emits a diff image
- The first version tried:

```bash
xdotool windowactivate --sync "$WIN_ID"
```

which failed under plain `Xvfb` because there is no window manager.
- I removed the activation dependency and targeted the window directly with `xdotool`’s `--window` options.
- Ran the helper:

```bash
ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-ui-input-and-capture.sh
```

- Inspected:
  - `st80-ui-before.png`
  - `st80-ui-after.png`
  - `st80-ui-diff.png`

### Why
- I needed a cheap repeatable way to answer “does visible state change under injected input?” without requiring an interactive desktop session each time.
- A script in the ticket workspace is the right place for that helper because it is part of the debugging workflow, not just an ephemeral command line.
- The result constrains the next step: if the UI had visibly changed, the next work would be interaction polish; because it did not, the next work needs more runtime instrumentation.

### What worked
- The script reliably boots the UI under `Xvfb`.
- The script reliably captures before/after/diff images.
- The current screenshot clearly shows a live cursor-rendered UI state, so the cursor-overlay work is at least visible in the host window.
- The script is now reusable for future input debugging.

### What didn't work
- The first attempt failed with:

```text
Your windowmanager claims not to support _NET_ACTIVE_WINDOW, so the attempt to activate the window was aborted.
xdo_activate_window on window:2097160 reported an error
```

- After removing `windowactivate`, the script completed, but the resulting before/after screenshots were visually identical.
- The diff image was blank, which means the simple injected sequence did not produce any visible UI delta in this run.

### What I learned
- The host-side plumbing is far enough along that I can now ask narrower, evidence-based questions about live behavior.
- A no-delta result is still useful. It narrows the problem.
- The remaining issue may be one of:
  - the image not consuming these events in the exercised UI state
  - focus/targeting semantics under `Xvfb` not matching a real desktop session closely enough
  - missing parts of the keyboard/event mapping beyond the current text/editing-key subset
  - missing instrumentation to prove events are entering and/or being drained from the interpreter-side queue

### What was tricky to build
- The tricky part was not the capture. It was input injection under a root X server with no window manager. `windowactivate` is a natural first try, but it is the wrong assumption in plain `Xvfb`.
- The other tricky part is interpreting a blank diff correctly. It is tempting to over-read it as “input is broken,” but it is more precise to say “this particular injected sequence caused no visible screen delta.”

### What warrants a second pair of eyes
- Review whether the injected sequence is the right one for the currently visible Smalltalk UI state.
- Review whether I should instrument queue insertion/drain counts next instead of relying on visible-screen change as the only signal.
- Review whether a lightweight window manager under `Xvfb` would make the keyboard part of the exercise more representative.

### What should be done in the future
- Instrument live event consumption so I can tell:
  - whether the host recorded the event
  - whether the interpreter enqueued the event words
  - whether the image drained or ignored them
- Re-run the same exercise after that instrumentation exists.
- Later, repeat the exercise on a real desktop session to separate `Xvfb` limitations from real VM/image behavior.

### Code review instructions
- Review [exercise-ui-input-and-capture.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-ui-input-and-capture.sh).
- Then read [09-offscreen-input-exercise-note.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/09-offscreen-input-exercise-note.md).
- Optionally inspect the generated local artifacts in `various/ui-capture/` from the last run.

### Technical details
- Injected sequence in the current helper:
  - move mouse to `(120, 120)` in the window
  - click button 1
  - type `a`
  - press `Return`
- Observed result:
  - before/after screenshots looked the same
  - diff image was blank

## Step 14: Input-Debug Instrumentation And The No-Event Result

The off-screen exercise narrowed the problem to “no visible change,” but that still left an important ambiguity: was the host actually recording any events, queuing any words, or draining anything? Without that, a blank diff could still mean many different things. So the next slice was not another UI behavior guess. It was instrumentation.

I added coarse input-debug counters to the interpreter and an `-input-debug` flag to `st80-ui` so the live process could log when host events were recorded and when queue counters changed. Then I reran the same off-screen exercise. The result was stronger than before: the UI run log contained no input-debug lines at all. Combined with the unchanged images, that strongly suggests the current Xvfb/xdotool path is not delivering events into the SDL loop under this setup.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Continue replacing ambiguity with hard evidence, especially where the live UI still does not behave as expected.

**Inferred user intent:** Push past “it seems unchanged” and get specific about which stage of the live event path is flat.

### What I did
- Extended [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) with coarse input counters:
  - `inputWordsEnqueued`
  - `inputWordsDequeued`
  - `mouseMotionsRecorded`
  - `mouseButtonsRecorded`
  - `decodedKeysRecorded`
- Added `InputStats` plus `InputStats()` on the interpreter.
- Updated [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) so the UI loop can log input stats whenever they change.
- Added `-input-debug` in [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go).
- Updated [exercise-ui-input-and-capture.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-ui-input-and-capture.sh) to run `st80-ui` with `-input-debug`.
- Ran formatting:

```bash
gofmt -w pkg/interpreter/interpreter.go pkg/ui/ui.go cmd/st80-ui/main.go
```

- Revalidated focused build/test coverage:

```bash
go test ./pkg/ui ./pkg/interpreter ./cmd/st80-ui -run 'TestCopyDisplayBitsOverlaysCursorBits|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles|TestPrimitive(SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal)'
```

- Reran the off-screen exercise helper and inspected:
  - the before/after PNGs
  - the UI run log

### Why
- Without live counters, “no visible delta” still leaves too many explanations open.
- The cheapest useful instrumentation is not deep tracing. It is a few counters that answer:
  - did the host record anything?
  - did anything enter the queue?
  - did anything leave the queue?
- Putting the switch behind `-input-debug` keeps the normal UI path quiet.

### What worked
- The UI builds and runs with `-input-debug`.
- The focused test set is green after the instrumentation change.
- The instrumentation now exists for future live runs without further code edits.
- The rerun produced a more specific negative result:
  - before/after images remained visually identical
  - the UI run log contained only the image banner, with no input-debug lines

### What didn't work
- The rerun still did not produce visible UI change.
- More importantly, it did not produce any logged input counter changes either.
- That means the problem is upstream of queue consumption counters: under this off-screen setup, the SDL loop does not appear to be receiving the injected events at all.

### What I learned
- The current off-screen event-injection setup is likely the wrong layer to trust blindly. At least in this configuration, `xdotool --window` plus plain `Xvfb` is not enough evidence of event delivery into SDL.
- The new instrumentation is still valuable because it converts that conclusion from guesswork into a more concrete statement.
- The next debugging move is likely environment/event-delivery specific, not another VM primitive pass.

### What was tricky to build
- The tricky part here was restraint. It would be easy to add a much larger tracing system, but the right first step was a tiny amount of instrumentation that answers the highest-value question.
- The second tricky part was interpreting an empty debug log correctly: it does not prove the VM ignores events. It more specifically suggests the UI process never observed them.

### What warrants a second pair of eyes
- Review whether the logging point in the UI loop is the right place for this first round of instrumentation.
- Review whether `xdotool` under `Xvfb` needs a different focus/pointer strategy for SDL windows.
- Review whether the next environment experiment should add a lightweight window manager or use a different event-injection tool.

### What should be done in the future
- Figure out why the current Xvfb exercise is not delivering visible SDL events.
- Once that is understood, rerun the exact same exercise and compare the new input-debug counters.
- Only after that decide whether the next problem is image-side event consumption or environment-side delivery.

### Code review instructions
- Start in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `InputStats`
  - `InputStats()`
  - the counter increments in the input-recording and queue methods
- Then review [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) for the optional logging path.
- Then review [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go) and the updated [exercise-ui-input-and-capture.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-ui-input-and-capture.sh).

### Technical details
- Rerun observation:
  - PNG hashes differed
  - pixel-level compare reported zero changed pixels
  - the UI run log contained no input-debug lines
- That combination is consistent with metadata/capture variation, not real screen-content change.

## Step 15: OCR Verification Pass Against The Live VM And Test Surface

The intern OCR handoff is now in the useful state: it is not just a pile of text, it is structured enough to audit against the VM. So this step was a verification pass, not another implementation pass. I used the extracted class-layout and primitive-audit material as the source of expected field order and primitive surface, then compared that against the current interpreter, UI code, and image-side selector inventory.

This was worth doing now because the project has accumulated enough bug-fix history that “it seems consistent” is no longer a good standard. I wanted a concrete answer to two questions: does the OCR-backed Blue Book data actually agree with the current UI/display/input/timer implementation, and if not, where are the remaining mismatches? The pass found mostly good news and one real gap: primitive `97` is in the audit and still absent from the VM's I/O dispatch table.

### Prompt Context

**User prompt (verbatim):** "I think our intern now finished with their OCR, you can commit and do a big verification pass."

**Assistant interpretation:** Treat the OCR pack as a serious audit source, compare it against the live VM and current tests, record the verification results in the ticket, and commit the checkpoint.

**Inferred user intent:** Use the intern's OCR work to raise confidence in the interpreter/UI implementation and surface any remaining discrepancies systematically instead of continuing on assumptions.

### What I did
- Read the ticket-local OCR outputs, especially:
  - `reference/ocr-bluebook/02-class-layouts.csv`
  - `reference/ocr-bluebook/04-primitive-audit.csv`
  - `reference/ocr-bluebook/05-display-and-bitblt-audit.md`
  - `reference/ocr-bluebook/07-open-questions.md`
- Cross-checked the extracted field orders against `pkg/interpreter/interpreter.go`, focusing on:
  - `Point`
  - `Rectangle`
  - `Form`
  - `BitBlt`
- Cross-checked the OCR-audited UI/timer primitive surface against `dispatchInputOutputPrimitives`.
- Cross-checked the live image selector inventory in `data/method.oops` for the currently relevant input/timer/display selectors.
- Ran focused verification commands:

```bash
go test ./pkg/ui ./cmd/st80-ui
go test ./pkg/interpreter -run 'TestTrace2SendSelectorsMatch|TestTrace3DisplayStartupSendSelectorsMatch|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles|TestPrimitive(MousePointReturnsConfiguredPoint|CursorLocPutUpdatesCursorAndReturnsReceiver|CursorLocPutUpdatesMouseWhenLinked|InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord|SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal)'
rg -n 'primCursorLocPut|primMousePt|primSampleInterval|primInputWord|primInputSemaphore|secondClockInto|millisecondClockInto|signal:atMilliseconds:|beDisplay|beCursor' data/method.oops
rg -n 'case 9[0-9]|case 100|case 101|case 102|snapshotPrimitive|dispatchInputOutputPrimitives' pkg/interpreter/interpreter.go
```

- Created [10-blue-book-ocr-verification-pass.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/10-blue-book-ocr-verification-pass.md).
- Marked the OCR-pack task complete and added follow-up tasks for:
  - primitive `97`
  - the impractical blanket `go test ./pkg/...` path

### Why
- The OCR pack is only useful if it can survive contact with the code.
- The display/input/timer path is now large enough that a structured audit is higher value than another ad hoc probe.
- The project needs a written checkpoint that says what the OCR now proves, what it does not prove, and what concrete work remains.

### What worked
- The OCR layout tables matched the current interpreter constants for the display/UI-critical classes and structures.
- The active UI/timer primitive surface matched the audit for primitives `90` through `102` except for `97`.
- The image-side selector inventory confirmed that the relevant Smalltalk methods exist in the loaded image.
- Focused verification coverage is green:

```text
ok  	github.com/wesen/st80/pkg/ui	(cached)
?   	github.com/wesen/st80/cmd/st80-ui	[no test files]
ok  	github.com/wesen/st80/pkg/interpreter	0.044s
```

- The OCR task can now honestly be marked done. It is no longer just “instructions were written”; there is now an actual verification artifact built from the extracted material.

### What didn't work
- `go test ./pkg/...` is not currently a good blanket verifier. It launched `interpreter.test`, kept consuming CPU, and did not finish promptly enough to treat as a routine verification step.
- The pass found one concrete audited gap:
  - primitive `97` (`snapshotPrimitive`) is present in the OCR primitive audit but missing from `dispatchInputOutputPrimitives`
- The first frontmatter-validation attempt used the wrong path shape for `docmgr validate`:

```text
Error: open /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/10-blue-book-ocr-verification-pass.md: no such file or directory
```

  - The fix was to pass the path relative to the docs root (`2026/...`) instead of `ttmp/...`.

### What I learned
- The intern's OCR is good enough to use for real audits right now.
- The field-order bugs we fixed earlier were not accidental one-offs; this kind of OCR-backed table is exactly the right defense against future layout/argument-order mistakes.
- The current implementation gap is not broad UI drift. It is narrow and concrete.
- The default package test surface needs cleanup if we want “run the whole package tree” to mean something practical during iteration.

### What was tricky to build
- The tricky part was resisting the temptation to claim more certainty than the pass actually provides. The OCR pack can verify field order, selector expectations, and primitive presence/absence, but it does not automatically prove every method header or every runtime path.
- The other tricky part was validation hygiene. A slow or sprawling test suite can create fake confidence if I report it loosely. I killed the long-running `go test ./pkg/...` attempt and reran a deliberate focused verification set instead so the ticket reflects what actually passed.

### What warrants a second pair of eyes
- Review the new verification note for whether the “implemented vs missing” primitive table says enough for future audits.
- Review whether primitive `97` should be implemented now or deliberately deferred with a written rationale.
- Review whether the long-running interpreter diagnostics should move behind a build tag, a naming convention, or a separate integration-test command.

### What should be done in the future
- Implement or explicitly defer primitive `97` with a ticket-local writeup.
- Make `go test ./pkg/...` practical again by splitting out the heavy interpreter diagnostics from routine package verification.
- Reuse the OCR tables to audit more primitive families beyond the current UI/display/input/timer slice.

### Code review instructions
- Start with [10-blue-book-ocr-verification-pass.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/10-blue-book-ocr-verification-pass.md).
- Then inspect [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - the constant blocks near the top
  - `dispatchInputOutputPrimitives`
- Then inspect the OCR source files under `reference/ocr-bluebook/`.
- Validate with the focused commands listed above instead of relying on `go test ./pkg/...`.

### Technical details
- Concrete audit finding:
  - primitives `90`, `91`, `92`, `93`, `94`, `95`, `96`, `98`, `99`, `100`, `101`, `102` are present in the current I/O dispatch path
  - primitive `97` is the current omission
- Frontmatter hygiene after correcting the path shape:

```bash
docmgr validate frontmatter --doc 2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/10-blue-book-ocr-verification-pass.md --suggest-fixes
docmgr validate frontmatter --doc 2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/01-diary.md --suggest-fixes
```
- Concrete selector evidence from `data/method.oops` includes:
  - `<InputState>primInputSemaphore:`
  - `<InputState>primInputWord`
  - `<Time class>secondClockInto:`
  - `<Time class>millisecondClockInto:`
  - `<ProcessorScheduler>signal:atMilliseconds:`
  - `<DisplayScreen>beDisplay`
  - `<Cursor>beCursor`

## Step 16: Primitive 97 Snapshot Support And Image Writer

The OCR verification pass isolated primitive `97` as a real spec/code mismatch, so this step closed that gap directly. The missing piece was not just one more `case` in the primitive table. The interpreter had no way to serialize the current object memory back into the on-disk image format, so implementing `snapshotPrimitive` meant adding the inverse of `LoadImage`, deciding where snapshots should be written, and preserving the expected receiver/stack behavior.

This was a good next slice because it was concrete and bounded. The OCR pass had already done the hard part of making the gap precise. What remained was engineering discipline: do the smallest correct write path, test it as an actual round trip, and avoid cheating with a hardcoded output path or a lossy dump format that would only be useful for debugging.

### Prompt Context

**User prompt (verbatim):** "ok, continue then"

**Assistant interpretation:** Continue with the next highest-value concrete task from the verification pass without pausing for more planning.

**Inferred user intent:** Keep momentum, close the remaining audited VM gaps one by one, and preserve the same detailed ticket/diary/commit trail while doing it.

**Commit (code):** fec10f5 — "Implement snapshot primitive 97 support"

### What I did
- Confirmed that the current gap was real:
  - primitive `97` was present in the OCR primitive audit
  - primitive `97` was still missing from `dispatchInputOutputPrimitives`
- Read the existing image loader in `pkg/image/loader.go` and established the inverse file-format requirements.
- Measured the checked-in `data/VirtualImage` layout and confirmed there is a padding gap between object space and object table because the object table starts on a `512`-byte boundary.
- Added raw-word export helpers in `pkg/objectmemory/objectmemory.go`:
  - `ObjectTableWords()`
  - `ObjectSpaceWords()`
- Added `WriteImage(path, memory)` in `pkg/image/loader.go`.
- Added a round-trip serializer test in `pkg/image/loader_test.go`.
- Added `snapshotPath` plus `SetSnapshotPath(path string)` to the interpreter.
- Wired primitive `97` into `dispatchInputOutputPrimitives`.
- Implemented `primitiveSnapshot()` so it:
  - pops the receiver
  - writes the configured image path
  - pushes the receiver back on success
  - restores the stack and primitive-fails on error
- Threaded the loaded image path into the interpreter from:
  - `pkg/ui/ui.go`
  - `pkg/ui/snapshot.go`
  - `loadTestInterpreter` in `pkg/interpreter/interpreter_test.go`
- Added `TestPrimitiveSnapshotWritesImageAndReturnsReceiver`.
- Ran formatting and focused verification:

```bash
gofmt -w pkg/objectmemory/objectmemory.go pkg/image/loader.go pkg/image/loader_test.go pkg/interpreter/interpreter.go pkg/interpreter/interpreter_test.go pkg/ui/ui.go pkg/ui/snapshot.go
go test ./pkg/image ./pkg/interpreter ./pkg/ui ./cmd/st80-ui -run 'TestWriteImageRoundTripsObjectMemory|TestPrimitiveSnapshotWritesImageAndReturnsReceiver|TestTrace2SendSelectorsMatch|TestTrace3DisplayStartupSendSelectorsMatch|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles|TestPrimitive(MousePointReturnsConfiguredPoint|CursorLocPutUpdatesCursorAndReturnsReceiver|CursorLocPutUpdatesMouseWhenLinked|InputSemaphoreStoresSemaphoreAndReturnsReceiver|SampleIntervalStoresMillisecondsAndReturnsReceiver|InputWordReturnsQueuedWord|SecondClockIntoStoresLittleEndianSeconds|MillisecondClockIntoStoresLittleEndianTicks|SignalAtMillisecondsSignalsImmediatelyWhenPastDue|SignalAtMillisecondsSchedulesFutureSignal)'
```

### Why
- Primitive `97` was the cleanest audited mismatch left by the OCR pass.
- An image writer is useful beyond the primitive itself; it becomes infrastructure for later debugging and VM-state capture.
- Using the loaded image path as the configured snapshot target is better than inventing a silent fallback output path.

### What worked
- The new `WriteImage` path round-trips object memory through the same format that `LoadImage` reads.
- Primitive `97` now exists in the interpreter’s I/O dispatch table.
- The primitive returns the receiver on success and preserves normal primitive-failure behavior when no path is configured.
- The focused verification set stayed green after the changes:

```text
ok  	github.com/wesen/st80/pkg/image	0.007s
ok  	github.com/wesen/st80/pkg/interpreter	0.066s
ok  	github.com/wesen/st80/pkg/ui	0.011s [no tests to run]
?   	github.com/wesen/st80/cmd/st80-ui	[no test files]
```

### What didn't work
- There was no implementation blocker here, but there was one design trap I explicitly avoided: `filepath.Clean(\"\")` becomes `.`. If I had blindly cleaned an empty snapshot path, primitive `97` would have attempted to write to the current directory path instead of failing cleanly.
- I still did not use `go test ./pkg/...` as the verification command because the earlier verification pass already showed that it is not a practical routine check while the long-running interpreter diagnostics remain mixed into the default package tree.

### What I learned
- The image writer does need to preserve the object-table alignment rule. The checked-in image has a `384`-byte gap between object space and object table because the object table starts on a `512`-byte boundary.
- The clean way to support snapshot writes in this codebase is to configure the interpreter with the image path at construction time, not to make the primitive itself invent filesystem policy.
- The OCR verification pass is already paying off directly: it turned a missing primitive from a vague suspicion into a one-step implementation target.

### What was tricky to build
- The tricky part was not serialization itself. It was preserving the real file shape instead of only writing something the current loader would happen to accept. The object-table alignment gap is easy to miss if you only read the loader loosely.
- The second subtle point was stack behavior. Snapshot is a no-argument receiver primitive, so the implementation had to preserve the receiver on success and restore the popped stack slot on failure.

### What warrants a second pair of eyes
- Review whether reusing the loaded image path is the right long-term primitive-97 policy or whether later work should add an explicit “snapshot as” host feature.
- Review the object-table alignment rule in `WriteImage` and confirm that `512`-byte alignment is the right invariant for future images as well.
- Review whether the new raw-word export helpers on `ObjectMemory` are the right public surface or whether they should eventually move behind a more explicit serialization API.

### What should be done in the future
- Keep the `go test ./pkg/...` hygiene task open and split the long-running interpreter diagnostics out of the default package verification path.
- After that, return to the remaining UI blocker: why the off-screen `Xvfb` exercise is still not delivering observable SDL input events.

### Code review instructions
- Start with [11-snapshot-primitive-97-support-writeup.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/11-snapshot-primitive-97-support-writeup.md).
- Then inspect:
  - [loader.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/image/loader.go)
  - [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go)
  - [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go)
  - [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go)
- Validate with the focused test command listed above.

### Technical details
- Alignment fact observed from the checked-in image:
  - header = `512` bytes
  - object space = `517760` bytes
  - object table = `77472` bytes
  - padding gap before object table = `384` bytes
- New test coverage added:
  - `TestWriteImageRoundTripsObjectMemory`
  - `TestPrimitiveSnapshotWritesImageAndReturnsReceiver`

## Step 17: Direct Input Exercise Harness To Bypass SDL And X11

The primitive-97 slice closed the OCR audit gap, but it did not move the active UI blocker. That blocker is still “why does the off-screen `Xvfb` exercise produce no observable SDL events?” At this point the right next move was not more guessing about `xdotool` or window-manager focus. It was to remove SDL and X11 from the loop entirely and ask a simpler question: if I inject the same style of mouse/key input directly into the interpreter, does the image eventually react?

I added a direct input-exercise harness for exactly that reason. This gives me a fast iteration path that captures before/after framebuffer snapshots around an injected input sequence without relying on host event delivery. The long run produced a decisive result: with guaranteed delivery and enough cycles afterward, the raw display hash changed substantially. That means the image-side input path is alive, and the remaining problem is narrowed back to host-side SDL/X delivery.

### Prompt Context

**User prompt (verbatim):** (same as Step 16)

**Assistant interpretation:** Keep moving through the next bottleneck rather than stopping after the primitive-97 fix, and convert the off-screen UI ambiguity into a more precise diagnosis.

**Inferred user intent:** Continue with concrete engineering steps that reduce uncertainty and make the remaining UI problem smaller and more specific.

**Commit (code):** 89c742b — "Add direct input exercise snapshot tool"

### What I did
- Added [exercise.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/exercise.go) with:
  - `InputExerciseOptions`
  - `InputExerciseDiagnostic`
  - `ExerciseInputAndCapture`
- Added [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-exercise-snapshot/main.go) as a direct input-exercise CLI.
- Added ticket-local wrapper [exercise-direct-input-snapshot.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-direct-input-snapshot.sh).
- The harness now:
  - loads the image
  - runs `before` cycles
  - captures a baseline framebuffer snapshot
  - injects direct mouse/key events via interpreter calls
  - runs `after` cycles
  - captures an `after` framebuffer snapshot
  - reports changed pixels plus before/after display hashes
- Validated the new toolchain with:

```bash
go test ./pkg/ui ./cmd/st80-exercise-snapshot ./cmd/st80-snapshot
```

- Ran the short exercise:

```bash
ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-direct-input-snapshot.sh
```

which reported:

```text
beforeCycles=50000 afterCycles=50000 changedPixels=28 beforeHash=0162f0db51d0f337b9c13722d5b2dc815344aa4f9b2b9b4f79507aeb1e63586b afterHash=0162f0db51d0f337b9c13722d5b2dc815344aa4f9b2b9b4f79507aeb1e63586b
```

- Then ran the longer exercise:

```bash
BEFORE_CYCLES=50000 AFTER_CYCLES=500000 \
ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-direct-input-snapshot.sh
```

which reported:

```text
beforeCycles=50000 afterCycles=500000 changedPixels=10319 beforeHash=0162f0db51d0f337b9c13722d5b2dc815344aa4f9b2b9b4f79507aeb1e63586b afterHash=b680477060f52bcc2a95142d83f17e0f6405822fa4537df0b3dfb2a97b2ff13c beforeBlack=112228 afterBlack=111072
```

### Why
- The off-screen SDL path had stopped being a productive source of truth.
- A direct interpreter-side harness is a better diagnostic because it isolates the image/VM input path from the host-event delivery path.
- This creates a faster non-X iteration loop for future input experiments.

### What worked
- The new direct harness builds and runs cleanly.
- The short run already showed a tiny visible delta, consistent with cursor-only change.
- The longer run showed a real framebuffer change:
  - different display hash
  - `10319` changed pixels
  - different black-pixel count
- That is the key result of this step: the image does react to the injected input sequence once delivery is guaranteed and enough time passes.

### What didn't work
- The harness did not magically fix SDL/X delivery, and it was not meant to.
- The short `after` run (`50000` cycles) was not enough to prove image-side consumption because it only produced:
  - `28` changed pixels
  - identical display hashes
- That shorter result was ambiguous until I extended the `after` run.

### What I learned
- The active UI/input blocker is narrower than before. It is not “does the image respond to input at all?” It is “why are the injected Xvfb events not reaching SDL in the current host setup?”
- A fast direct harness is more valuable than repeatedly tweaking `xdotool` blindly.
- The image may need a fairly large number of post-input cycles before a meaningful framebuffer change shows up.

### What was tricky to build
- The tricky part was interpreting the first direct-input result correctly. `changedPixels=28` with identical display hashes was not enough. That could still be just cursor overlay movement rather than a real framebuffer update.
- The second subtle point was choosing where to inject. I kept the injection at the interpreter boundary using the same public input-recording methods the SDL host loop uses so the harness exercises the same VM-facing behavior without the SDL/X transport layer.

### What warrants a second pair of eyes
- Review whether the current direct input sequence is the right canonical one for future debugging.
- Review whether the longer post-input run should become the default in the script, since `50000` cycles is too ambiguous.
- Review whether the next host-side experiment should use:
  - `windowfocus` instead of `windowactivate`
  - a lightweight X window manager
  - a different event-injection tool

### What should be done in the future
- Return to the `Xvfb`/SDL delivery problem with this narrower hypothesis: image-side input works, host-side delivery is the remaining suspect.
- Consider increasing the default `AFTER_CYCLES` in the direct exercise wrapper so the first result is less likely to be ambiguous.

### Code review instructions
- Start with [12-direct-input-exercise-note.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/12-direct-input-exercise-note.md).
- Then inspect:
  - [exercise.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/exercise.go)
  - [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-exercise-snapshot/main.go)
  - [exercise-direct-input-snapshot.sh](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-direct-input-snapshot.sh)
- Validate with the two direct exercise commands listed above.

### Technical details
- Short run result:
  - `changedPixels=28`
  - same raw display hash
- Long run result:
  - `changedPixels=10319`
  - different raw display hash
  - `beforeBlack=112228`
  - `afterBlack=111072`

## Step 18: Raw SDL Event Debug And The Stronger Host-Side Negative Result

The direct interpreter-side harness was a useful narrowing step, but it still left one host-side question open: was SDL receiving raw input events under the off-screen setup and simply failing to map them, or was it not receiving them at all? That distinction matters. If `sdl.PollEvent()` is getting nothing, the next fix is environmental. If it is getting events that our mapping ignores, the next fix is in our host loop.

I added a raw SDL event-debug flag and reran the off-screen experiment under a stronger host setup: `Xvfb` plus `openbox`, explicit `windowfocus`, and the same `xdotool` input sequence. The result was still negative. The run log contained only the startup image banner and no raw event-debug lines at all. That is a much stronger result than the earlier “no input-debug lines” outcome, because it says the failure is upstream of our interpreter-side input mapping.

### Prompt Context

**User prompt (verbatim):** (same as Step 16)

**Assistant interpretation:** Keep pushing on the active UI blocker and reduce the host-side ambiguity enough that the next debugging step is obvious.

**Inferred user intent:** Continue until the remaining UI problem is much more concrete, rather than stopping after the direct-input harness result.

**Commit (code):** 342e7d3 — "Add SDL raw event debug logging"

### What I did
- Added `EventDebug` to [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) and [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go) so `st80-ui` can log raw SDL input events as they are polled.
- Logged these event classes:
  - quit
  - mouse motion
  - mouse button
  - text input
  - keyboard
- Validated the small code slice with:

```bash
go test ./pkg/ui ./cmd/st80-ui
```

- Ran an off-screen host-side experiment under:
  - `Xvfb`
  - `openbox`
  - `xdotool windowfocus`
  - `st80-ui -input-debug -event-debug`
- Injected the same sequence:
  - mouse move
  - left click
  - type `a`
  - `Return`
- Inspected the resulting run log.

### Why
- The direct interpreter-side harness had already shown that the image reacts once delivery is guaranteed.
- So the next highest-value question was whether SDL was seeing any raw events at all in the off-screen environment.
- A tiny raw-event logger is cheaper and more decisive than continuing to infer from unchanged screenshots.

### What worked
- The raw-event logger builds cleanly and can now be reused in future host-side debugging runs.
- The openbox-backed off-screen experiment ran to completion, located the window, focused it, injected the same sequence, and produced a reproducible run log.
- The result narrows the diagnosis further:
  - still no `input-debug` lines
  - still no `event-debug` lines

### What didn't work
- Even with a real window manager (`openbox`) and explicit `windowfocus`, the off-screen run log still contained only:

```text
Image: 596128 bytes, objectSpace=258880 words @512, objectTable=38736 words @518656
```

- So the stronger host-side setup still did not produce observable SDL input events in this environment.

### What I learned
- The current off-screen problem is now very likely before or at SDL event acquisition, not in the interpreter-side mapping logic.
- The direct harness and the raw-event logger complement each other well:
  - direct harness proves image-side reaction exists
  - raw-event logger proves the current off-screen SDL path still sees nothing useful

### What was tricky to build
- The tricky part here was not code complexity. It was experimental hygiene. The earlier no-input-debug result could still be argued away as “maybe SDL got events but they did not change counters.” The raw-event logger removes that escape hatch.
- The first attempt at the `openbox` experiment also hit a shell/glob issue in `zsh` while clearing an output directory. I reran it with `find ... -delete` to avoid turning a shell quirk into a false debugging trail.

### What warrants a second pair of eyes
- Review whether there is any SDL/X11 setup detail missing from the off-screen experiment beyond window manager and focus.
- Review whether the next host-side experiment should bypass `xdotool` entirely in favor of another X input injection mechanism.
- Review whether I should capture X-level diagnostics next, such as pointer/focus state or XTest behavior, before changing more VM code.

### What should be done in the future
- Keep the host-side diagnosis task open.
- Use the direct harness as the main fast iteration path for image-side input experiments.
- Treat the off-screen SDL/X delivery problem as a separate environment/debugging slice rather than a VM/input-semantics slice.

### Code review instructions
- Start with [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) and [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go) for the `-event-debug` addition.
- Then read the updated [09-offscreen-input-exercise-note.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/09-offscreen-input-exercise-note.md).
- Reproduce with an off-screen session that includes:
  - `openbox`
  - `windowfocus`
  - `-input-debug`
  - `-event-debug`

### Technical details
- Host-side run log still contained only the startup image line and no event-debug output.
- That means the current experiment did not reach even the raw SDL event logging branch for the injected sequence.

## Step 19: Cleaner SDL Focus Diagnostics And Smaller Default Poll Chunks

At this point the raw SDL event logger was useful, but the output still made live diagnosis harder than it needed to be. It printed focus status every poll, which buried the interesting transitions in noise, and it still did not print the SDL window's own ID at creation time. That made it too easy to stare at `windowID=1` or `windowID=2` without being certain whether that was actually the `Smalltalk-80` window.

I tightened that debug path instead of adding more one-off probes. The host UI now prints the created SDL window ID once, only logs mouse/keyboard focus when those IDs change, shows human-readable window-event names, and uses a smaller default `cycles-per-frame` so event polling is less coarse. That gives the next desktop run a much cleaner signal.

### Prompt Context

**User prompt (verbatim):** "alright do it"

**Assistant interpretation:** Apply the focused host-debug cleanup that makes the SDL event logs actually useful for the next manual run.

**Inferred user intent:** Remove low-value debug noise, make the event logs interpretable, and improve host responsiveness enough that manual testing is meaningful.

### What I did
- Changed the default `cycles-per-frame` from `50000` to `5000` in:
  - [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go)
  - [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go)
- When `-event-debug` is enabled, the UI now logs:
  - the created SDL window ID once at creation time
  - focus changes only, instead of printing focus every poll
  - human-readable SDL window event names
- Added helpers in [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) for:
  - resolving current mouse/keyboard focus IDs
  - formatting focus IDs cleanly
  - translating `SDL_WindowEventID` values to names
- Revalidated with:

```bash
gofmt -w pkg/ui/ui.go cmd/st80-ui/main.go
go test ./pkg/ui ./cmd/st80-ui
SDL_VIDEODRIVER=dummy go run ./cmd/st80-ui -event-debug -max-cycles 20000 -cycles-per-frame 5000
```

### Why
- The previous focus spam was making the right debugging information harder to see.
- Printing the created window ID once is the simplest way to tell whether later focus IDs actually refer to our SDL window.
- A smaller default chunk size should make host polling less coarse during manual interactive runs.

### What worked
- The build/test stayed green.
- The dummy-driver smoke run now prints a concise, interpretable startup sequence:

```text
Image: 596128 bytes, objectSpace=258880 words @512, objectTable=38736 words @518656
[event-debug cycle=5000] created-window windowID=1 title="Smalltalk-80" size=640x480
[event-debug cycle=5000] mouse-focus windowID=1
[event-debug cycle=5000] keyboard-focus windowID=1
```

- That means the debug stream now clearly identifies the SDL window itself and stops flooding the log with repeated unchanged focus state.

### What didn't work
- The first build attempt hit a small SDL type mismatch:

```text
cannot use e.Event (variable of uint32 type sdl.WindowEventID) as uint8 value in argument to windowEventName
```

- The fix was to make `windowEventName` accept `sdl.WindowEventID` directly.

### What I learned
- The right debugging improvement here was not “more logs,” it was better-shaped logs.
- Printing the window ID once is enough to make later focus lines meaningful.
- The default `50000` cycle chunk was too large for good interactive diagnostics even if it was acceptable for earlier headless progress.

### What was tricky to build
- The main subtlety was avoiding another noisy debug stream. Focus state is useful, but only on change.
- The second subtlety was SDL typing: `WindowEvent.Event` is not a plain `uint8` in this binding, so the helper had to match SDL's own event-ID type.

### What warrants a second pair of eyes
- Review whether `5000` is the right long-term default chunk size or whether it should go lower still for interactive use.
- Review whether we should also add a small frame delay/cap so manual runs stop pegging a CPU core unnecessarily.

### What should be done in the future
- Re-run the real desktop/manual test with the cleaner event-debug stream.
- If real mouse motion still does not produce `mouse-motion` lines, add one more narrow probe around SDL window events and pointer enter/leave behavior.

### Code review instructions
- Review [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) for:
  - created-window logging
  - focus-change-only logging
  - `windowEventName`
- Review [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go) for the default `cycles-per-frame` change.
- Validate with the commands listed above.

### Technical details
- New dummy-driver startup output:
  - created SDL window ID
  - mouse-focus ID
  - keyboard-focus ID
- The focus IDs now remain silent unless they actually change.

## Step 20: Replace SDL with Ebiten

### Prompt / goal
Replace the `go-sdl2` host backend with `github.com/hajimehoshi/ebiten/v2`, keep the Smalltalk interpreter-facing UI semantics intact, and remove the SDL-specific commands/dependency instead of leaving two parallel backends in the repo.

### Inferred user intent
Stop spending more time fighting SDL/X11 edge cases on this machine and move the graphical host window onto a different Go-native backend with a simpler update/input model.

### What I did
- Checked the current UI surface:
  - [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go)
  - [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go)
  - [ui_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui_test.go)
- Confirmed the full SDL footprint in code with:

```bash
rg -n "go-sdl2|veandco|sdl" . 
```

- Resolved the current Ebiten version with:

```bash
go list -m -json github.com/hajimehoshi/ebiten/v2@latest
```

- Result:
  - `github.com/hajimehoshi/ebiten/v2 v2.9.9`

- Pulled the dependency and API docs:

```bash
go get github.com/hajimehoshi/ebiten/v2@v2.9.9
go doc github.com/hajimehoshi/ebiten/v2 Game
go doc github.com/hajimehoshi/ebiten/v2 RunGame
go doc github.com/hajimehoshi/ebiten/v2 CursorPosition
go doc github.com/hajimehoshi/ebiten/v2 AppendInputChars
go doc github.com/hajimehoshi/ebiten/v2 IsFocused
go doc github.com/hajimehoshi/ebiten/v2 SetWindowSize
go doc github.com/hajimehoshi/ebiten/v2 Termination
go doc github.com/hajimehoshi/ebiten/v2/inpututil
```

- Replaced the SDL implementation in [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) with an Ebiten `Game`:
  - `Run` now loads the image, configures the window, and calls `ebiten.RunGame`
  - the interpreter still advances in `CyclesPerFrame` chunks
  - display snapshots still come from `interp.DisplaySnapshot()`
  - cursor overlay still uses the existing bitmap helper path
  - event/input handling now polls Ebiten state once per update tick
- Preserved the interpreter-facing host semantics:
  - mouse motion still calls `RecordMouseMotion`
  - button edges still call `RecordMouseButton`
  - printable text still flows through decoded-key recording
  - special editing keys still map through the same Smalltalk-side parameters
- Updated [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go) so `-event-debug` describes the new backend semantics rather than "raw SDL input events."
- Removed the standalone SDL diagnostics:
  - deleted [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/sdl-hello/main.go)
  - deleted [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/sdl-hello-raw/main.go)
- Added a new standalone Ebiten diagnostic command:
  - [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/ebiten-hello/main.go)
- Cleaned the module graph with:

```bash
gofmt -w pkg/ui/ui.go cmd/st80-ui/main.go cmd/ebiten-hello/main.go
go mod tidy
```

### Why
- The SDL backend had already reached the point where the remaining host-side debugging was dominating the UI work.
- Ebiten gives a simpler and more opinionated main loop:
  - fixed-timestep `Update`
  - direct keyboard/mouse polling
  - direct logical-size window layout
- That matches what this VM host actually needs: one bitmap, one update loop, and a small amount of input translation.

### What worked
- The SDL dependency is gone from code and module metadata:

```bash
rg -n "go-sdl2|veandco|github.com/veandco|\\bsdl\\b" go.mod go.sum pkg cmd
```

- That search now returns nothing.
- Targeted validation passed:

```bash
go test ./pkg/ui ./cmd/st80-ui ./cmd/ebiten-hello
```

- Output:

```text
ok  	github.com/wesen/st80/pkg/ui	0.016s
?   	github.com/wesen/st80/cmd/st80-ui	[no test files]
?   	github.com/wesen/st80/cmd/ebiten-hello	[no test files]
```

- The existing cursor-overlay test in [ui_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui_test.go) still passes without modification to the pixel-overlay logic.

### What did not work or remains open
- I started a wider `go test ./...` run after the targeted backend validation. It was still traversing the heavier interpreter suite at the time of this entry, so the targeted pass is the concrete validation result recorded here.
- The live desktop-session behavior under Ebiten still needs manual verification. Replacing SDL removes that backend, but it does not by itself prove that the machine-specific input issue is gone.

### What I learned
- The UI code was isolated enough that the backend swap was straightforward once the VM-facing interface was held constant.
- The right unit of replacement was the entire host loop, not a piecemeal renderer swap.
- Ebiten's `CursorPosition`, `AppendInputChars`, `IsFocused`, and `inpututil` helpers are enough for the current Smalltalk host integration without needing a lower-level event queue abstraction.

### What was tricky
- The old SDL path had raw event timestamps; the Ebiten path does not expose the same event objects, so the host timestamp is now derived from time since the UI process started.
- The old SDL path used window-size-aware coordinate mapping. Ebiten's cursor position is already reported in logical coordinates, which simplifies the mapping but means the semantics of `-event-debug` necessarily changed from "raw backend events" to "observed host input state changes."
- I had to decide whether to keep the SDL diagnostics around "just in case." I chose not to. Keeping them would have left the repo in an ambiguous half-migrated state.

### What warrants a second pair of eyes
- Review [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) for:
  - the new `hostGame` lifecycle
  - the logical-size resize behavior
  - the input translation from Ebiten polling to interpreter calls
- Review [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/ebiten-hello/main.go) to confirm it is the right minimal backend diagnostic to keep around.

### Future follow-up
- Run:

```bash
go run ./cmd/ebiten-hello
go run ./cmd/st80-ui -event-debug -input-debug
```

- Confirm whether:
  - mouse motion is now visible in the backend diagnostic
  - the VM host receives input and changes input-debug counters
  - the real desktop-session behavior is meaningfully better than it was under SDL

### Code review instructions
- Start with [go.mod](/home/manuel/code/wesen/2026-03-17--smalltalk/go.mod) to confirm the dependency migration.
- Review [ui.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go) for the host-loop replacement.
- Review [main.go](/home/manuel/code/wesen/2026-03-17--smalltalk/cmd/ebiten-hello/main.go) for the new diagnostic path.
- Confirm the deleted SDL commands are intentional and complete.

### Technical details
- New host backend:
  - `github.com/hajimehoshi/ebiten/v2 v2.9.9`
- Ebiten APIs used directly:
  - `RunGame`
  - `CursorPosition`
  - `AppendInputChars`
  - `IsFocused`
  - `SetWindowSize`
  - `SetWindowTitle`
  - `SetWindowResizingMode`
  - `Termination`
  - `inpututil.IsMouseButtonJustPressed`
  - `inpututil.IsMouseButtonJustReleased`
  - `inpututil.IsKeyJustPressed`
- SDL-only commands removed:
  - `cmd/sdl-hello`
  - `cmd/sdl-hello-raw`

## Step 21: Diagnose the first live Ebiten VM crash

### Prompt / goal
After the user confirmed that the Ebiten backend is "much better" and that live mouse input now reaches the app, investigate the new crash:

```text
panic: Recursive not understood error encountered
...
github.com/wesen/st80/pkg/interpreter.(*Interpreter).lookupMethodInClass
...
github.com/wesen/st80/pkg/interpreter.(*Interpreter).sendSpecialSelectorBytecode
```

### Inferred user intent
The backend migration was successful enough to surface a real VM/image bug. The next priority is to turn that crash from a vague panic into a debuggable failing send state.

### What I observed from the user report
- The Ebiten UI was receiving real motion:

```text
[event-debug cycle=1625000] mouse-motion x=90 y=182
[input-debug cycle=1630000] motions=49 buttons=4 keys=0 enqueued=8 dequeued=0 queue=8
[event-debug cycle=1635000] mouse-motion x=223 y=382
[input-debug cycle=1640000] motions=50 buttons=4 keys=0 enqueued=8 dequeued=0 queue=8
```

- Then the interpreter panicked inside `lookupMethodInClass`.
- The stack showed `lookupMethodInClass` recursing through `doesNotUnderstand:`.
- The user-visible class in the stack trace was `0x38`, which is suspicious because `0x0038` is `ClassSymbolPointer` in our object-memory constants.

### What I checked first
- Reviewed the panic path in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `lookupMethodInClass`
  - `findNewMethodInClass`
  - `sendSelector`
  - `sendSpecialSelectorBytecode`
- Reviewed the live input primitives again:
  - `primitiveMousePoint`
  - `primitiveCursorLocPut`
  - `primitiveInputSemaphore`
  - `primitiveSampleInterval`
  - `primitiveInputWord`
- Rechecked the existing primitive/input tests in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go), especially:
  - `TestPrimitiveMousePointReturnsConfiguredPoint`
  - `TestPrimitiveCursorLocPutUpdatesCursorAndReturnsReceiver`
  - `TestPrimitiveInputWordReturnsQueuedWord`
  - the direct event-queue tests for mouse motion and decoded keys

### Why I did not jump straight to a speculative fix
- The primitive implementations still looked plausible on inspection.
- The queue counters were a useful clue:
  - `enqueued=8`
  - `dequeued=0`
- That means the image crashed before draining the active input-word queue. So the immediate failure is less obviously "input buffer consumer broke" and more likely "some object returned to the image during mouse handling is not what the image expects."
- The stack also implicated a special-selector send, which could easily be something like `x`, `y`, `class`, or another fixed selector applied to a wrong-shape receiver.

### What I changed
- I replaced the bare recursive panic text in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):

```go
panic("Recursive not understood error encountered")
```

- with a detailed interpreter-side diagnostic that records:
  - current `messageSelector`
  - string form of the selector if available
  - class being searched
  - current receiver and receiver class
  - would-be send receiver and its class
  - active and home contexts
  - current method
  - instruction pointer
  - stack pointer
  - current bytecode
  - current argument count

- Added helper methods:
  - `recursiveNotUnderstoodDiagnostic`
  - `debugStringOf`

### Why
- The previous panic told us only that `doesNotUnderstand:` recursion happened.
- That is not enough to decide whether the real bug is:
  - a wrong receiver object
  - a wrong selector
  - a corrupted stack layout
  - a Point/cursor/mouse primitive returning the wrong object shape
  - a later send path unrelated to the queue itself
- The new panic should turn the next manual crash into a concrete failing state.

### Validation
- Reformatted the file with:

```bash
gofmt -w pkg/interpreter/interpreter.go
```

- Started a targeted validation run:

```bash
go test ./pkg/interpreter ./pkg/ui ./cmd/st80-ui ./cmd/ebiten-hello
```

- At the time of this diary entry, the heavy interpreter package was still in the known long-running test path, so the concrete fast validation result remains the earlier backend-targeted pass rather than a fully completed `pkg/interpreter` run.

### What I learned
- The backend migration did its job. The problem is now inside the VM/image interaction again.
- The queue counters matter: the image had not yet consumed the queued input words, so this is probably not the same bug as a malformed `primInputWord` dequeue.
- A diagnostic-first patch is the right move here because the next failure will likely tell us whether we are dealing with:
  - a bad `Point`
  - a bad receiver on a special selector like `x` / `y`
  - or a broader send-stack corruption

### What should happen next
- Re-run:

```bash
go run ./cmd/st80-ui -event-debug -input-debug
```

- Move the mouse until the crash happens again.
- Use the new expanded panic message to identify the selector/receiver/class combination that actually failed.
- Then fix the VM bug, not the symptom.

### Code review instructions
- Review [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) around:
  - `lookupMethodInClass`
  - `recursiveNotUnderstoodDiagnostic`
  - `debugStringOf`
- Confirm that the extra diagnostic information is side-effect free except when the panic path is hit.

### Technical details
- The user-triggered crash happened after live Ebiten mouse motion, around cycle `1,640,000`.
- The report strongly suggests the next bug is reachable only once real input is flowing into the image.

## Step 22: Hard-stop silent singleton corruption

### Prompt / goal
Use the richer panic data from the live Ebiten crash to identify the real corruption shape and remove the class of bug that allowed it to happen silently.

### What the richer panic revealed
The key user report was:

```text
panic: Recursive not understood error encountered: selector=0x002A("doesNotUnderstand:") lookupClass=0x6458 receiver=0x0002 class=0x6480("") sendReceiver=0x0006 sendClass=0x6458("") activeContext=0xFFFC homeContext=0xFFFC method=0x830C ip=8 sp=8 bytecode=208 argCount=1
```

I then resolved the important OOPs:

```bash
rg -n "16r6458|16r6480|16r830C" data/class.oops data/method.oops
```

Result:

- `0x6458` is not a class OOP. It is method OOP `<True>not`.
- `0x6480` is `UndefinedObject`.
- `0x830C` is `<Object>~~`.

That means the failing send path was:

- method `<Object>~~`
- receiver path produced `true` (`0x0006`)
- the method then sent `not`
- but `fetchClassOf(true)` returned `0x6458`, the method OOP for `<True>not`

So the actual bug is not "True forgot `not`."

The actual bug is:

> the singleton object for `true` had its class word overwritten with a method OOP

That is a silent heap-corruption problem.

### Why this immediately changed the debugging strategy
A singleton class word becoming a method OOP is exactly the kind of outcome you get from a bad field index write, especially an index of `0` passed through `index-1`, which lands on field `-1` and therefore on the class word.

I reviewed [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go) and found a major structural weakness:

- `FetchPointer`
- `StorePointer`
- `FetchWord`
- `StoreWord`
- `FetchByte`
- `StoreByte`

were only checking whether the computed address stayed inside the *entire object space*, not whether the access stayed inside the *target object's own bounds*.

That means a bad field index could silently scribble over a neighboring object's header or class word.

### What I changed in object memory
- Added per-object bounds checks in [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go) for:
  - `FetchPointer`
  - `StorePointer`
  - `FetchWord`
  - `StoreWord`
  - `FetchByte`
  - `StoreByte`
- The new panic messages now include:
  - target OOP
  - requested field/word/byte index
  - object-local length
  - location
  - value for stores

This means the next bad write should fail where it happens, not many cycles later when a corrupted singleton is used.

### What I changed in the reflective primitives
I also hardened the obvious user-visible entry points that could produce exactly this kind of class-word overwrite:

- `objectAt:`
- `objectAt:put:`
- `instVarAt:`
- `instVarAt:put:`

in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go).

New behavior:

- `objectAt:` / `objectAt:put:` now require `1 <= index <= fetchWordLengthOf(receiver)`
- `instVarAt:` / `instVarAt:put:` now require `1 <= index <= fixedFieldsOf(receiver class)`
- invalid indices now primitive-fail instead of falling through to a raw `index-1` store/fetch

### Why
- Even with stronger object-memory guards, the VM should not need a panic to reject obviously invalid reflective indices.
- Primitive failure is the right first line of defense.
- The object-memory bounds checks are the second line of defense in case some other internal path is still wrong.

### Tests added
In [objectmemory_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go):

- `TestStorePointerPanicsWhenFieldIndexIsNegative`
- `TestStorePointerPanicsWhenFieldIndexExceedsObjectLength`
- `TestStoreBytePanicsWhenByteIndexExceedsObjectLength`

In [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):

- `TestPrimitiveObjectAtPutFailsForZeroIndex`
- `TestPrimitiveInstVarAtPutFailsBeyondFixedFields`

### Validation
Ran:

```bash
gofmt -w pkg/objectmemory/objectmemory.go pkg/objectmemory/objectmemory_test.go
gofmt -w pkg/interpreter/interpreter.go pkg/interpreter/interpreter_test.go
go test ./pkg/interpreter ./pkg/objectmemory ./pkg/ui ./cmd/st80-ui ./cmd/ebiten-hello -run 'TestPrimitiveObjectAtPutFailsForZeroIndex|TestPrimitiveInstVarAtPutFailsBeyondFixedFields|TestPrimitiveMousePointReturnsConfiguredPoint|TestPrimitiveInputWordReturnsQueuedWord|TestCopyDisplayBitsOverlaysCursorBits|TestStorePointerPanicsWhenFieldIndexIsNegative|TestStorePointerPanicsWhenFieldIndexExceedsObjectLength|TestStoreBytePanicsWhenByteIndexExceedsObjectLength'
```

Result:

```text
ok  	github.com/wesen/st80/pkg/interpreter	0.027s
ok  	github.com/wesen/st80/pkg/objectmemory	0.005s
ok  	github.com/wesen/st80/pkg/ui	0.038s
?   	github.com/wesen/st80/cmd/st80-ui	[no test files]
?   	github.com/wesen/st80/cmd/ebiten-hello	[no test files]
```

### What I learned
- The richer panic was worth it immediately.
- The corruption signature is highly consistent with a bad reflective field index rather than a random allocator failure.
- The most important fix in this slice is not the primitive checks by themselves; it is removing silent adjacent-object corruption as a tolerated failure mode in object memory.

### What should happen next
- Re-run the live Ebiten UI and repeat the menu click.
- Expected outcomes:
  - either the old crash disappears because the bad index now primitive-fails harmlessly
  - or a new, earlier bounds panic identifies the exact store/fetch site that was previously corrupting `true`

### Code review instructions
- Review [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go) for the new per-object bounds checks.
- Review [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) for:
  - `checkObjectFieldBounds`
  - `checkInstanceVariableBounds`
  - the updated storage-management primitive cases
- Review the new regressions in:
  - [objectmemory_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go)
  - [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go)

### Technical details
- The key semantic clue was that `fetchClassOf(true)` returned method OOP `0x6458` (`<True>not`) instead of a class OOP.
- That points strongly at a class-word overwrite, which is exactly what a negative field index can do.
