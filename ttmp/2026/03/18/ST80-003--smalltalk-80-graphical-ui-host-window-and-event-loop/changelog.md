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
