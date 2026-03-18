# Tasks

## TODO

- [x] Create a dedicated ticket for the object-memory / GC frontier exposed by live UI `Point` allocation
- [x] Write a detailed intern-facing plan / design / analysis document grounded in the Blue Book OCR
- [x] Implement a first-pass mark/sweep reclamation path and allocation retry on OT exhaustion
- [x] Add focused object-memory and interpreter regressions for the new reclamation path
- [ ] Audit the interpreter root set against the full Blue Book `rootObjectPointers` expectation
- [ ] Decide whether GC may run safely at every current allocation site or whether some primitives need deferred collection points
- [ ] Implement proper free-chunk lists / non-append heap allocation instead of exact-size body reuse only
- [ ] Implement Blue Book `spaceOccupiedBy:` / deallocation semantics for non-pointer objects and compiled methods more faithfully
- [ ] Decide whether heap compaction is needed immediately after OT reclamation or can wait until heap-space pressure is real
- [ ] Diagnose the new post-GC `checkProcessSwitch` invalid `suspendedContext` frontier reached in off-screen runs
- [x] Verify whether the later `checkProcessSwitch` invalid `suspendedContext` frontier actually occurs after GC or independently of it
- [ ] Validate that the Ebiten UI survives past the previous `object table exhausted` panic during real desktop use
