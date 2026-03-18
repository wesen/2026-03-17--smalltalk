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
