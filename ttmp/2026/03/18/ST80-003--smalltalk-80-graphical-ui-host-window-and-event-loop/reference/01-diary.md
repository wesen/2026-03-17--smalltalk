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
