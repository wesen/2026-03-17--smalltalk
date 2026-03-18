# Changelog

## 2026-03-18

- Initial workspace created


## 2026-03-18

Step 1: Created the SDL UI ticket, added stepped interpreter/display-snapshot hooks, implemented a new `st80-ui` host-window command, and validated the full path with SDL's dummy video driver (commit 8e85254).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Stepped execution and display snapshot export for host UI work (commit 8e85254)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — SDL host window, bitmap conversion, and event/present loop (commit 8e85254)
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go — Windowed UI command-line entrypoint (commit 8e85254)


## 2026-03-18

Step 2: Added a ticket-local `Xvfb` screenshot script for `st80-ui`, ran it successfully, and recorded that the current visible UI is a blank white window rather than a populated Smalltalk desktop.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/capture-ui-screenshot.sh — Reproducible off-screen UI capture script
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png — First captured UI image


## 2026-03-18

Step 3: Added a direct non-SDL framebuffer snapshot command and ticket-local wrapper script, proving that the designated display surface itself is `640x16`, all white, and unchanged between one million and two million cycles (commit ee69a09).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/snapshot.go — Direct framebuffer capture and PNG writing without SDL/Xvfb (commit ee69a09)
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-snapshot/main.go — Command-line entrypoint for direct framebuffer snapshots (commit ee69a09)
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/dump-display-snapshot.sh — Fast ticket-local snapshot wrapper
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-1000000.png — First direct framebuffer PNG

## 2026-03-18

Step 4: fixed primitive 71 to accept LargePositiveInteger sizes, restored the full 640x480 designated display allocation path, and added a trace3 startup regression (commit acaa659).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Primitive 71 now accepts LargePositiveInteger size arguments for startup display allocation (commit acaa659)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Trace3 startup regression and detailed diagnostics for the display allocation bug (commit acaa659)
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-1000.png — Post-fix direct framebuffer snapshot showing the corrected 640x480 surface


## 2026-03-18

Step 5: broadened the same positive-integer decoding fix across clear size/index primitives and added direct decoder tests (commit d2d22d8).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added popPositiveInteger and widened positive size/index primitive decoding (commit d2d22d8)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Direct positive-integer decoder tests and retained startup regression coverage (commit d2d22d8)


## 2026-03-18

Step 6: fixed the `BitBlt` field-index order so `primitiveCopyBits` reads `sourceX/sourceY` and `clipX/clipY/clipWidth/clipHeight` from the correct slots, restoring non-white framebuffer output.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Corrected `BitBlt` slot constants for `primitiveCopyBits`
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Added a normal regression for non-white early display output
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/04-bitblt-field-order-bug-writeup.md — Detailed intern-facing explanation of the bug and fix
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-5000.png — First post-fix direct framebuffer snapshot showing non-white output
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png — Updated off-screen SDL capture after the `BitBlt` fix


## 2026-03-18

Step 7: prepared a structured OCR/extraction handoff for the Blue Book so an intern can build audit-ready class-layout, method-signature, and primitive-reference tables instead of raw OCR text.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/03-bluebook-ocr-extraction-instructions-for-intern.md — Intern-facing OCR and structured extraction workflow for the Blue Book
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/tasks.md — Added the systematic Blue Book OCR audit as an explicit follow-up task


## 2026-03-18

Step 8: fixed the BitBlt copy-loop row-advance bug so successful display blits now progress across the full framebuffer instead of stalling in the top 256 rows, producing a recognizable `System Browser` window.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Corrected `sourceIndex` / `destIndex` row progression in the BitBlt copy loop
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Strengthened the display regression and added diagnostics for display write ranges and BitBlt geometry
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/05-bitblt-copyloop-row-advance-bug-writeup.md — Detailed intern-facing explanation of the row-advance bug and fix
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-5000.png — Updated early snapshot showing a recognizable windowed scene
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/display-snapshots/display-50000.png — Later snapshot showing the visible `System Browser`
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/various/ui-capture/st80-ui.png — Refreshed off-screen SDL capture after the copy-loop fix


## 2026-03-18

Step 9: added passive mouse-point and cursor-location support so the SDL host can feed live mouse coordinates into primitives `90` and `91`.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added mouse/cursor bookkeeping plus primitive `90` and `91`
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Added direct primitive tests for passive mouse-point and cursor-location behavior
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — SDL event loop now maps host mouse coordinates into interpreter state
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/tasks.md — Split the broad input task into passive mouse support vs buffered keyboard/button follow-up


## 2026-03-18

Step 10: implemented the active input-event buffer primitives (`93`, `94`, `95`), wired SDL mouse/keyboard events into the Blue Book event-word stream, and fixed an OOP-0 sentinel mistake that initially suppressed deferred input semaphore signaling.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added buffered input state, 16-bit integer boxing, primitives `93`/`94`/`95`, and host input-event recording helpers
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Added regressions for input semaphore registration, sample interval handling, buffered word return, mouse-motion event words, and decoded-keypress encoding
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — SDL loop now feeds mouse motion, mouse buttons, text input, and editing keys into the interpreter input queue
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/06-input-event-buffer-oop-zero-sentinel-bug-writeup.md — Intern-facing writeup of the first-pass input semaphore signaling bug


## 2026-03-18

Step 11: implemented host clock/timer primitives `98`, `99`, and `100`, storing 32-bit little-endian time words into byte objects and wiring millisecond-deadline semaphore signaling through the interpreter scheduler.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added second-clock, millisecond-clock, and signal-at-milliseconds primitives plus host clock state
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Added direct tests for byte-order correctness and immediate/future timer signaling
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/07-timer-primitives-byte-order-and-semaphore-initialization-note.md — Intern-facing notes on the timer primitive semantics and the fresh-semaphore `ExcessSignals` initialization detail


## 2026-03-18

Step 12: added host-side cursor snapshot/overlay support so the SDL renderer can OR the designated Smalltalk cursor form into the presented framebuffer instead of ignoring cursor state entirely.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added `CursorSnapshot` export for the designated cursor form and location
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — The display conversion path now overlays cursor bits after expanding the framebuffer
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/snapshot.go — Snapshot rendering now uses the same cursor overlay path as the SDL host loop
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui_test.go — Added a focused regression for cursor overlay composition
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/08-host-cursor-overlay-note.md — Notes on the chosen OR-style host cursor rendering behavior


## 2026-03-18

Step 13: added an off-screen Xvfb/xdotool input-exercise script and recorded that a simple mouse/click/type sequence produced no visible before/after delta in the captured UI, pushing the next frontier toward live event-consumption instrumentation.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-ui-input-and-capture.sh — Ticket-local helper that injects mouse/keyboard events under Xvfb and captures before/after/diff screenshots
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/09-offscreen-input-exercise-note.md — Notes on the inconclusive off-screen input exercise and what it implies about the next debugging slice


## 2026-03-18

Step 14: added live input-debug counters plus a `-input-debug` UI flag, then reran the off-screen exercise and observed that the SDL/UI process still emitted no input-debug lines, narrowing the next issue to host-event delivery under the current Xvfb setup.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added coarse input counters and `InputStats` export
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — UI loop now optionally logs input queue/consumption counters as they change
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go — Added the `-input-debug` flag
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-ui-input-and-capture.sh — Exercise script now runs the UI with `-input-debug`
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/09-offscreen-input-exercise-note.md — Updated note with the no-debug-output result


## 2026-03-18

Step 15: verified the intern's Blue Book OCR pack against the live VM/UI code, confirmed that the class-layout and active UI/timer primitive mappings match the extraction, and recorded the two remaining audit findings: missing primitive `97` and the impractical blanket `go test ./pkg/...` path for verification.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/10-blue-book-ocr-verification-pass.md — OCR-backed verification results, commands, and concrete discrepancies
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/01-diary.md — Detailed diary entry for the OCR verification pass
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/tasks.md — Marked the OCR-pack task done and added follow-up tasks for primitive `97` and default test-suite hygiene


## 2026-03-18

Step 16: implemented Blue Book primitive `97` (`snapshotPrimitive`) by adding image serialization support, wiring the interpreter snapshot path, and verifying both raw image round-tripping and receiver-preserving primitive behavior (commit fec10f5).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/image/loader.go — Added `WriteImage` as the inverse of `LoadImage`, including object-table alignment handling (commit fec10f5)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/image/loader_test.go — Added raw object-memory round-trip coverage for snapshot serialization (commit fec10f5)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go — Added raw object-space and object-table export helpers for serialization (commit fec10f5)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added `SetSnapshotPath`, primitive `97` dispatch, and `primitiveSnapshot` (commit fec10f5)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Added direct primitive-97 coverage for snapshot write success and receiver preservation (commit fec10f5)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — Configures the interpreter snapshot path from the loaded image path (commit fec10f5)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/snapshot.go — Configures the interpreter snapshot path for headless snapshot runs (commit fec10f5)
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/11-snapshot-primitive-97-support-writeup.md — Detailed writeup of the primitive-97 bug and fix


## 2026-03-18

Step 17: added a direct interpreter-side input-exercise harness and used it to prove that the image does respond to the injected mouse/key sequence once delivery is guaranteed and enough post-input cycles are allowed, narrowing the remaining problem back to host-side SDL/X event delivery (commit 89c742b).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/exercise.go — Added a direct input-exercise API that captures before/after snapshots without SDL/X11 (commit 89c742b)
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-exercise-snapshot/main.go — Added a CLI for direct interpreter-side input injection and snapshot comparison (commit 89c742b)
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-direct-input-snapshot.sh — Ticket-local wrapper for the new direct input exercise tool (commit 89c742b)
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/12-direct-input-exercise-note.md — Recorded the direct-input results and the narrowed diagnosis


## 2026-03-18

Step 18: added raw SDL event-debug logging and confirmed that even an off-screen `Xvfb` run with `openbox`, explicit `windowfocus`, and the same `xdotool` sequence still produced no observable SDL input events, strengthening the diagnosis that the remaining blocker is host-side event delivery (commit 342e7d3).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — Added raw SDL event-debug logging around `sdl.PollEvent` handling (commit 342e7d3)
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go — Added the `-event-debug` flag for host-side input diagnostics (commit 342e7d3)
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/09-offscreen-input-exercise-note.md — Updated with the stronger openbox/raw-event-debug result


## 2026-03-18

Step 19: cleaned up the SDL debug stream by logging the created window ID once, logging focus only when it changes, translating window-event names, and reducing the default `cycles-per-frame` chunk size for more responsive manual runs.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — Created-window logging, focus-change-only logging, human-readable window-event names, and lower default polling chunk
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go — Lowered the default `cycles-per-frame` for interactive runs
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/01-diary.md — Detailed diary entry for the SDL debug cleanup


## 2026-03-18

Step 20: replaced the SDL host backend with Ebiten (`github.com/hajimehoshi/ebiten/v2 v2.9.9`), rewrote `st80-ui` around Ebiten's `Game` loop, removed the standalone SDL diagnostics, and added a new `ebiten-hello` diagnostic command for backend-level window/input checks.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/ui/ui.go — Replaced the SDL window, renderer, texture, and event loop with an Ebiten-backed host game that polls mouse/keyboard state each tick
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-ui/main.go — Updated the UI flag help so `-event-debug` describes the new host-event semantics instead of raw SDL events
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/ebiten-hello/main.go — New standalone Ebiten backend diagnostic window
- /home/manuel/code/wesen/2026-03-17--smalltalk/go.mod — Swapped the host-window dependency from `go-sdl2` to `ebiten/v2`
- /home/manuel/code/wesen/2026-03-17--smalltalk/go.sum — Module graph updated after `go mod tidy`
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/01-diary.md — Detailed migration diary entry


## 2026-03-18

Step 21: after the first successful live Ebiten input run exposed a VM-side crash instead of a backend-input blackout, hardened the recursive `doesNotUnderstand:` panic path with detailed selector/receiver/class/context diagnostics so the next failing manual run will identify the actual bad send state instead of only reporting the recursion symptom.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added recursive-DNU diagnostic formatting with selector, receiver, class, context, IP/SP, and bytecode detail
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/01-diary.md — Recorded the crash report and why the next patch is diagnostic-first


## 2026-03-18

Step 22: used the richer panic to identify that `true`'s class word had been silently corrupted, then hardened object-memory field/byte bounds and added primitive-level bounds checks for `objectAt:` / `objectAt:put:` / `instVarAt:` / `instVarAt:put:` so bad indices fail or trap at the actual offending access instead of silently overwriting neighboring singleton objects.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go — Added per-object bounds checks for pointer, word, and byte fetch/store accessors
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go — Added regressions for negative and oversized object-memory accesses
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added reflective primitive bounds checks for `objectAt:` / `objectAt:put:` / `instVarAt:` / `instVarAt:put:`
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Added regressions proving the reflective primitives no longer accept corrupting zero/oversized indices
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/01-diary.md — Recorded the new diagnosis and defensive fix path


## 2026-03-18

Step 23: after a new early `primitiveMakePoint` failure showed allocation apparently returning `nil`, added an explicit 15-bit object-table exhaustion guard plus a reserved-singleton-free trap and regressions so the allocator can no longer silently wrap `otEntryCount*2` or reuse corrupted singleton OT entries; this narrows the next live-run result toward either explicit OT exhaustion or a specific lower-level OT corruption path.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go — Added `maxObjectTableEntries`, explicit object-table exhaustion panic, and a reserved-singleton-free guard in `instantiate`
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go — Added regressions for 15-bit OOP-space exhaustion and reserved-singleton reuse
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/01-diary.md — Recorded why the `primitiveMakePoint` crash points at allocator wrap / OT exhaustion
