# Tasks

## TODO

- [x] Create the separate UI ticket after the interpreter/runtime reached a stable idle loop with a real BitBlt path
- [x] Expose a stepped interpreter API that a host window can drive incrementally
- [x] Export the designated Smalltalk display form as a host-friendly snapshot
- [x] Add a graphical host-window command that renders the display bitmap
- [x] Validate the initial host-window path under a non-interactive driver
- [x] Add a direct framebuffer snapshot path that does not depend on the host window backend
- [x] Investigate why the designated display framebuffer is `640x16` and all white
- [x] Feed keyboard and mouse events into the Smalltalk input primitives
- [x] Feed host mouse position into passive mouse-point / cursor-location primitives
- [x] Feed keyboard and button events into the Smalltalk input event-buffer primitives
- [x] Implement host-side time/timer support for the remaining clock primitives
- [x] Decide how to render or synthesize the Smalltalk cursor on the host side
- [ ] Verify the Ebiten UI command visually in a real desktop session and record the result in this ticket
- [x] Investigate why the corrected 640x480 designated display remains all white and never receives visible drawing
- [x] Build a Blue Book OCR extraction pack so we can audit class layouts, method argument order, and primitive expectations systematically
- [ ] Expand the decoded-keyboard host mapping to cover control/meta-key edge cases beyond ASCII text and the main editing keys
- [ ] Audit live image behavior around the new timer primitives and delayed-process wakeups in a real desktop session
- [ ] Verify the host-rendered cursor shape/location visually in a real desktop session
- [ ] Verify live host input delivery and event semantics in the Ebiten loop during a real desktop session
- [x] Implement or explicitly defer Blue Book primitive `97` (`snapshotPrimitive`) after the OCR verification pass confirmed it is missing from the current I/O dispatch table
- [ ] Split long-running interpreter diagnostics out of the default `go test ./pkg/...` path so package-wide verification is practical
