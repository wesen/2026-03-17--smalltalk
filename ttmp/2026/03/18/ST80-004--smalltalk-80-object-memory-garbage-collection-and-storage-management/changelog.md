# Changelog

## 2026-03-18

- Initial workspace created

## 2026-03-18

Step 1: after the live Ebiten UI proved that the VM was exhausting the full 15-bit object-table space on `Point` allocation, created this dedicated storage-management ticket, mapped the relevant Blue Book Chapter 30 routines, implemented a first-pass mark/sweep reclamation path plus allocation retry, and moved the frontier forward from `object table exhausted` to a later scheduler/process-switch corruption path (commit 94cba1d).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go — Added allocation-error handling, mark/sweep reclaim, compiled-method literal tracing, and retry-friendly allocation entry points
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go — Added focused reclamation regressions for unreachable objects and compiled-method literal retention
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added interpreter-side GC root discovery, allocation retry, and GC bookkeeping
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Added an integration regression proving OT exhaustion now triggers GC and recovers
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/raw-ch30-object-memory.txt — Blue Book OCR source for allocation and garbage-collection routines


## 2026-03-18

Step 2: added GC-aware scheduler diagnostics and verified that the later `checkProcessSwitch` invalid-`suspendedContext` frontier occurs with `gcCount=0`, which means it is not caused by the new first-pass collector and remains a separate interpreter/scheduler bug.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Added GC counters to the scheduler invalid-context panic
- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80-exercise-snapshot/main.go — Reused for the off-screen validation run that proved the later failure is pre-GC
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-004--smalltalk-80-object-memory-garbage-collection-and-storage-management/reference/01-diary.md — Recorded the diagnostic result and its implication
