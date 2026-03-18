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
