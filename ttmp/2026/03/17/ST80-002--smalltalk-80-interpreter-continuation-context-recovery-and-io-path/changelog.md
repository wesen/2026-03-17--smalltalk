# Changelog

## 2026-03-17

- Initial workspace created


## 2026-03-17

Step 1: Fixed tagged-SmallInteger decoding for method headers, header extensions, and class instance specifications; this removes the startup context overflow and lets the VM run into the next runtime blocker (commit dd8e4ba).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Interpreter metadata decoding fix
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Regression coverage for the former startup crash


## 2026-03-17

Added a detailed ticket writeup of the tagged-SmallInteger header decode bug for later review, and recorded the next runtime tasks after commit dd8e4ba.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/reference/02-tagged-smallinteger-header-decode-bug-writeup.md — Intern-facing bug explanation and validation steps
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/tasks.md — Continuation task list after the startup fix


## 2026-03-17

Step 2: Fixed the Blue-Book method cache hash translation so cached selector/class lookups no longer alias across entries; the VM now runs 2,000,000 cycles cleanly (commit 408f7b8).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Method cache hash fix and comment
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Regression and retained diagnostics for cache corruption investigation
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/reference/03-method-cache-hash-collision-writeup.md — Intern-facing explanation of the cache bug and validation steps


## 2026-03-17

Step 3: Implemented become:, added typed pointer/word/byte allocation for new/new:, and moved the runtime frontier to a LargePositiveInteger digitAt:put: size/index mismatch during DisplayScreen setup (commit 6b32314).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — become:
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go — swapPointersOf plus typed allocation helpers
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/reference/01-diary.md — Step 3 investigation and outcomes


## 2026-03-18

Step 4: Fixed block/value register handling, implemented String at:put:, added guarded MethodContext slot recycling, and moved the runtime frontier from the display LargePositiveInteger crash to a later value:/block corruption around cycle 708768 (commit 1a02e02).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Block/value
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Retained skipped diagnostics documenting the investigation path (commit 1a02e02)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go — MethodContext OOP-slot recycling support (commit 1a02e02)


## 2026-03-18

Step 5: Added a focused late-runtime diagnostic showing that the bad value: receiver is already invalid immediately after blockCopy:, pointing the next investigation at object-space growth / allocation corruption rather than later loop execution (commit 85de9e9).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Late blockCopy/value corruption trace retained in skipped/manual form (commit 85de9e9)


## 2026-03-18

Step 6: Removed the downloaded Wolczko source artifacts from the ticket workspace at the user's request, restored a clean repo state, and recorded the new reference boundary in the diary.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/reference/01-diary.md — Cleanup/provenance record for the removed ticket-local `sources/` tree


## 2026-03-18

Step 7: Reused freed context bodies safely, reserved tracked recycled context OOPs for exact-shape reuse, hardened context-shape checks, and removed the late blockCopy:/value: corruption frontier; the interpreter now runs through 800,000 and 2,000,000 cycles cleanly again (commit 6cb8881).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go — Exact-size body reuse, tracked retired bodies, and explicit segment-wrap guard (commit 6cb8881)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go — Focused allocator regressions for safe reuse and segment exhaustion (commit 6cb8881)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Defensive context-shape checks for undersized non-context objects (commit 6cb8881)


## 2026-03-18

Step 8: Implemented `perform:` / `perform:withArguments:`, `beCursor`, `cursorLink:`, and correct `asOop` / `asObject` semantics; added a hard `trace2` regression plus a two-million-cycle state probe, and moved the long-run notifier/debugger frontier down into `BitBlt>>copyBits` (commit d0346da).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Control/input/system primitive fixes for the long-run notifier path (commit d0346da)
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — `trace2` selector regression and 2,000,000-cycle state probe (commit d0346da)


## 2026-03-18

Step 9: Added a temporary headless `primitiveCopyBits` success path; this removed the immediate `BitBlt>>copyBits` notifier/debugger chain and let the image settle into a stable low-priority ProcessorScheduler loop through 5,000,000 cycles, while making it explicit that real BitBlt/display semantics are still pending (commit c1384ff).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Tactical headless `primitiveCopyBits` implementation (commit c1384ff)
