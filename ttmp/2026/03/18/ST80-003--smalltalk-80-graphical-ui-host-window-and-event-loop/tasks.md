# Tasks

## TODO

- [x] Create the separate UI ticket after the interpreter/runtime reached a stable idle loop with a real BitBlt path
- [x] Expose a stepped interpreter API that a host window can drive incrementally
- [x] Export the designated Smalltalk display form as a host-friendly snapshot
- [x] Add an SDL host-window command that renders the display bitmap
- [x] Validate the new UI path under SDL's dummy video driver
- [ ] Feed keyboard and mouse events into the Smalltalk input primitives
- [ ] Implement host-side time/timer support for the remaining clock primitives
- [ ] Decide how to render or synthesize the Smalltalk cursor on the host side
- [ ] Verify the UI command visually in a real desktop session and record the result in this ticket
