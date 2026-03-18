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
    - Path: pkg/interpreter/interpreter_test.go
      Note: Trace3 startup regression and startup diagnostics for the display allocation bug (commit acaa659)
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
