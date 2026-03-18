# Tasks

## TODO

- [x] Create the separate UI ticket after the interpreter/runtime reached a stable idle loop with a real BitBlt path
- [x] Expose a stepped interpreter API that a host window can drive incrementally
- [x] Export the designated Smalltalk display form as a host-friendly snapshot
- [x] Add an SDL host-window command that renders the display bitmap
- [x] Validate the new UI path under SDL's dummy video driver
- [x] Add a direct non-SDL framebuffer snapshot path for fast UI diagnostics
- [x] Investigate why the designated display framebuffer is `640x16` and all white
- [x] Feed keyboard and mouse events into the Smalltalk input primitives
- [x] Feed host mouse position into passive mouse-point / cursor-location primitives
- [x] Feed keyboard and button events into the Smalltalk input event-buffer primitives
- [ ] Implement host-side time/timer support for the remaining clock primitives
- [ ] Decide how to render or synthesize the Smalltalk cursor on the host side
- [ ] Verify the UI command visually in a real desktop session and record the result in this ticket
- [x] Investigate why the corrected 640x480 designated display remains all white and never receives visible drawing
- [ ] Build a Blue Book OCR extraction pack so we can audit class layouts, method argument order, and primitive expectations systematically
- [ ] Expand the decoded-keyboard host mapping to cover control/meta-key edge cases beyond ASCII text and the main editing keys
