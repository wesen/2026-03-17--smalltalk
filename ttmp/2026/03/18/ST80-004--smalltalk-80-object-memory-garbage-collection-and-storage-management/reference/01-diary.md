---
Title: Diary
Ticket: ST80-004
Status: active
Topics:
    - go
    - smalltalk
    - vm
    - gc
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/st80-exercise-snapshot/main.go
      Note: Diary records the off-screen validation run that exposed the next frontier
    - Path: pkg/interpreter/interpreter.go
      Note: Diary step covers interpreter root discovery and allocation retry
    - Path: pkg/objectmemory/objectmemory.go
      Note: Diary step covers the new mark/sweep reclaim implementation
ExternalSources: []
Summary: Step-by-step diary for the storage-management / garbage-collection frontier opened by live UI object-table exhaustion.
LastUpdated: 2026-03-18T18:05:00-04:00
WhatFor: Chronological implementation diary for Blue Book Chapter 30 storage management work.
WhenToUse: Use when reviewing why GC/storage-management work started, what the first pass implemented, and what frontier remains.
---


# Diary

## Goal

Track the Smalltalk-80 object-memory / garbage-collection work after the live UI path proved that append-only allocation plus tactical context reuse was no longer sufficient.

## Step 1: Split Out GC Work and Land the First Mark/Sweep Recovery Slice

The live Ebiten/UI runs changed the nature of the project. Up to this point, most failures were still on the interpreter/primitive side. The decisive shift came when the VM stopped failing “mysteriously” and instead failed honestly with `object table exhausted: otEntryCount=32768 class=0x001A bodySize=4`. That is a storage-management frontier, not a UI frontier.

This step did two things in parallel. First, it created a dedicated ticket and a detailed plan/design/analysis document so the GC work is reviewable in isolation and can be handed to an intern without requiring them to reconstruct the whole project history. Second, it implemented the first safe slice: a mark/sweep reclamation path that runs on allocation exhaustion, traces compiled-method literal pointers correctly, and retries allocation once before failing again. That moved the runtime frontier forward to a later scheduler/process-switch corruption.

### Prompt Context

**User prompt (verbatim):** "ok, then create a new ticket, add a plan / design / analysis doc, and then continue. Make it very detailed doc for our intern"

**Assistant interpretation:** Create a new docmgr ticket specifically for the storage-management / GC frontier, document the design in detail for an intern, and continue implementation rather than stopping at planning.

**Inferred user intent:** The user wants the current object-table-exhaustion problem treated as a first-class VM subsystem task, with enough documentation for handoff and enough coding progress that the ticket is not just paperwork.

**Commit (code):** 94cba1d — "Add first-pass mark sweep GC recovery"

### What I did
- Created a new ticket workspace:
  - [ST80-004 index](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-004--smalltalk-80-object-memory-garbage-collection-and-storage-management/index.md)
- Added the new ticket diary and intern-facing design doc:
  - [01-diary.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-004--smalltalk-80-object-memory-garbage-collection-and-storage-management/reference/01-diary.md)
  - [02-gc-plan-design-and-analysis-for-intern.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-004--smalltalk-80-object-memory-garbage-collection-and-storage-management/reference/02-gc-plan-design-and-analysis-for-intern.md)
- Re-read the local Blue Book OCR for the specific storage-management routines:
  - `allocateChunk:`
  - `attemptToAllocateChunk:`
  - `markObjectsAccessibleFrom:`
  - `rectifyCountsAndDeallocateGarbage`
  - the Chapter 29 scheduling note about GC vs. primitive boundaries
- Implemented a first-pass GC/recovery slice in [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go):
  - `allocationError`
  - `GCStats`
  - `TryInstantiateClass`
  - `TryInstantiateClassWithWords`
  - `TryInstantiateClassWithBytes`
  - `ReclaimInaccessibleObjects`
  - compiled-method literal-frame tracing via `compiledMethodPointerCount`
- Implemented the interpreter-side seam in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `guaranteedRootPointers`
  - `garbageCollectionRoots`
  - `collectGarbage`
  - `instantiateWithGarbageCollection`
  - wrappers for pointer/word/byte allocation now retry once after GC
- Added focused regressions:
  - [objectmemory_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go)
  - [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go)
- Ran focused validation:
  - `go test ./pkg/objectmemory -run 'TestInstantiatePanicsWhenObjectTableWouldOverflow15BitOopSpace|TestInstantiatePanicsWhenReservedSingletonIsMarkedFree|TestReclaimInaccessibleObjectsFreesUnreachableObjectAndReusesBody|TestReclaimInaccessibleObjectsMarksCompiledMethodLiteralPointers|TestStorePointerPanicsWhenFieldIndexIsNegative|TestStorePointerPanicsWhenFieldIndexExceedsObjectLength|TestStoreBytePanicsWhenByteIndexExceedsObjectLength'`
  - `go test ./pkg/interpreter -run 'TestInstantiateClassWithPointersCollectsGarbageOnObjectTableExhaustion|TestPositiveIntegerValueOfLargePositiveInteger|TestPrimitiveMousePointReturnsConfiguredPoint|TestPrimitiveInputWordReturnsQueuedWord'`
  - `go test ./pkg/ui ./cmd/st80-ui ./cmd/ebiten-hello -run 'TestCopyDisplayBitsOverlaysCursorBits|TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles'`
- Ran one off-screen exercise command after the GC slice:
  - `go run ./cmd/st80-exercise-snapshot -before-cycles 50000 -after-cycles 500000 -click none -text '' -return=false`

### Why
- The user-facing panic was no longer speculative. It was the exact 15-bit object-table limit from the Blue Book object-pointer encoding.
- The current memory model before this step was:
  - append-only heap growth
  - append-only object-table growth
  - tactical context reclamation only
  - exact-size reuse only for explicitly freed bodies
- That model can survive the idle/headless path but not a live path that allocates large numbers of `Point` objects.
- A first-pass mark/sweep reclaim is the smallest real storage-management step that can honestly solve OT exhaustion without pretending compaction is already done.

### What worked
- The new `object table exhausted` trap proved the problem was real OT pressure, not another accidental singleton overwrite.
- The first-pass collector worked well enough that the integration regression now proves:
  - OT exhaustion occurs
  - GC runs
  - unreachable objects are freed
  - allocation succeeds on retry
- The collector traces compiled-method literal frames, so it does not incorrectly treat methods as pure byte objects during marking.
- Focused tests for object memory, interpreter allocation retry, and existing UI snapshot code all passed.

### What didn't work
- A longer off-screen run still does not stabilize the VM. After the new GC slice, this command:

```bash
go run ./cmd/st80-exercise-snapshot -before-cycles 50000 -after-cycles 500000 -click none -text '' -return=false
```

failed with:

```text
panic: checkProcessSwitch: newProcess has invalid suspendedContext newProcess=0x2B28<class=0x0016("") words=18> fields=[0=0x0002<class=0x6480("") words=0> 1=0x0151<SmallInteger 168> 2=0x0009<SmallInteger 4> 3=0x87C6<class=0x630C("") words=2>] activeProcess=0x6CA8<class=0x07A4("") words=4> activeFields=[0=0x0002<class=0x6480("") words=0> 1=0x9322<class=0x0016("") words=38> 2=0x0009<SmallInteger 4> 3=0x0002<class=0x6480("") words=0>] scheduler=0x87BE<class=0x6626("") words=2> targetContext=0x0151<SmallInteger 168>
```

- That means the first-pass collector solved the earlier frontier but exposed a later one:
  - either the root set is still incomplete
  - or GC is running at an allocation point that is not safe with current scheduler/process-list transient state
  - or the corruption is independent and simply occurs later now

### What I learned
- The Blue Book Chapter 30 storage-management work is no longer optional. The live VM has now definitively crossed the boundary where primitive/interpreter fixes alone can carry the project.
- Solving OT exhaustion first was still the right move, even though it did not “finish GC.” It converted an immediate hard limit into a later semantic failure that is far easier to localize.
- The current first-pass collector is strong enough to validate the direction:
  - mark reachable objects
  - preserve compiled-method literal pointers
  - free unreachable OT entries
  - reuse exact-size bodies
- The next debugging question is no longer “do we need GC?” It is “what exact root / safe-point / scheduler invariant is still wrong after the first GC pass?”

### What was tricky to build
- The first tricky part was separating “Blue Book-correct long-term design” from “smallest honest implementation slice.” Full Chapter 30 includes free-chunk lists, compaction, and more precise object-space management. Implementing all of that in one pass would have been slow and risky. The first slice had to solve the real failure mode without pretending the whole chapter was already implemented.
- The second tricky part was compiled methods. They are not homogeneous pointer objects, but they also are not safe to treat as pure byte arrays during marking. The collector had to trace the literal frame and ignore the bytecode tail.
- The third tricky part was the root set. The interpreter holds meaningful object pointers outside ordinary object memory:
  - active/home context
  - method and receiver
  - message selector and new method
  - pending semaphores
  - UI-facing designated objects like display and cursor
  The collector had to start from those or it would instantly free live objects.
- The fourth tricky part was test construction. A synthetic object-memory test that starts allocation at OOP `0` can accidentally collide with Blue Book-reserved OOPs (`2`, `4`, `6`, etc.), so the tests intentionally allocate past that range before asserting reuse behavior.

### What warrants a second pair of eyes
- [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go):
  - `ReclaimInaccessibleObjects`
  - `compiledMethodPointerCount`
  - the decision to reuse existing exact-size body recycling instead of implementing free-chunk lists immediately
- [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `garbageCollectionRoots`
  - whether the current root set is sufficient
  - whether allocation-triggered GC can happen at all current call sites without violating scheduler/process transient invariants
- The direct-exercise failure after the GC slice:
  - does it indicate an incomplete root set?
  - or does it indicate that collection is occurring during a scheduler transition that the current VM does not represent safely?

### What should be done in the future
- Compare `garbageCollectionRoots` directly against the Blue Book’s intended `rootObjectPointers` set and the interpreter’s transient registers.
- Determine whether GC must be deferred away from certain primitive/control-flow windows.
- Implement real free-chunk tracking and non-append heap allocation.
- Decide whether to land compaction immediately or only after the scheduler/root frontier is stable.
- Re-run the live Ebiten UI once the later scheduler corruption is understood.

### Code review instructions
- Start with [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go):
  - `allocationError`
  - `GCStats`
  - `ReclaimInaccessibleObjects`
  - `TryInstantiateClass*`
- Then read [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go):
  - `guaranteedRootPointers`
  - `garbageCollectionRoots`
  - `collectGarbage`
  - `instantiateWithGarbageCollection`
- Then review the new regressions in:
  - [objectmemory_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go)
  - [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go)
- Re-run the focused commands listed above.
- If you want the next frontier immediately, run:

```bash
go run ./cmd/st80-exercise-snapshot -before-cycles 50000 -after-cycles 500000 -click none -text '' -return=false
```

### Technical details
- Blue Book / OCR references used in this step:
  - [raw-ch30-object-memory.txt](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/raw-ch30-object-memory.txt)
    - `allocateChunk:`
    - `attemptToAllocateChunk:`
    - `markObjectsAccessibleFrom:`
    - `rectifyCountsAndDeallocateGarbage`
  - [06-object-memory-audit.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/06-object-memory-audit.md)
- This first pass is intentionally narrower than full Chapter 30:
  - it reclaims OT entries and exact-size bodies
  - it does not yet implement Blue Book free-chunk lists
  - it does not yet compact heap segments
  - it does not yet rectify dynamic reference counts
