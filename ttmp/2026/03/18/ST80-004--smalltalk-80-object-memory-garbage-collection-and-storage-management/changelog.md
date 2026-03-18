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
