---
Title: GC plan design and analysis for intern
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
    - /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go:Core object-memory allocation and reclaim implementation
    - /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go:Interpreter root discovery and allocation retry seam
    - /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/raw-ch30-object-memory.txt:Blue Book OCR source for allocation and garbage collection routines
    - /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/06-object-memory-audit.md:Summarized Blue Book storage-management audit
ExternalSources: []
Summary: "Detailed intern-facing design and analysis for the Smalltalk-80 Chapter 30 storage-management implementation."
LastUpdated: 2026-03-18T18:10:00-04:00
WhatFor: "Explain the current GC/storage-management frontier, the first implementation slice, and the next safe work order."
WhenToUse: "Use when implementing or reviewing object-memory allocation, marking, sweeping, root discovery, or later compaction work."
---

# GC plan design and analysis for intern

## Goal

Provide a detailed, intern-readable design and analysis document for the Smalltalk-80 object-memory / garbage-collection work, grounded in the Blue Book and the actual current Go implementation.

## Context

The project originally reached a stable enough interpreter/runtime that a real host UI could be attached. Once the Ebiten host path began exercising live UI behavior, the VM started allocating `Point` objects fast enough to reveal a true storage-management limit:

```text
panic: object table exhausted: otEntryCount=32768 class=0x001A bodySize=4
```

That panic matters because it is not incidental:
- the Blue Book object-pointer format allows only 15 bits of OT index space
- `0x001A` is `ClassPointPointer`
- `bodySize=4` is exactly a normal `Point` object with a 2-word header and 2 fields

So the VM had crossed from “interpreter bugs” into “real object-memory implementation gap.” The immediate task is no longer to guess at random primitives. It is to implement the missing storage-management layer from Blue Book Chapter 30 carefully enough that the live image can continue.

## Quick Reference

### Problem statement

Current project state before this ticket:
- image loading works
- object-table decoding works
- bytecode interpreter works far enough to boot and run the image
- long-run headless path reached a stable scheduler loop
- graphical host path exists and delivers input
- but allocation was still effectively append-only except for narrow context reuse

That model fails under real live allocation pressure.

### Blue Book sections that matter

Primary sources already available locally:
- [raw-ch30-object-memory.txt](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/raw-ch30-object-memory.txt)
- [06-object-memory-audit.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/06-object-memory-audit.md)
- [data/bluebook-spec-notes.md](/home/manuel/code/wesen/2026-03-17--smalltalk/data/bluebook-spec-notes.md)

Key Blue Book routines for this ticket:
- `allocateChunk:`
- `attemptToAllocateChunk:`
- `attemptToAllocateChunkInCurrentSegment:`
- `deallocate:`
- `markObjectsAccessibleFrom:`
- `rectifyCountsAndDeallocateGarbage`
- `reclaimInaccessibleObjects`
- the compiled-method special cases around `lastPointerOf:` / mixed pointer+byte layout

Blue Book caution from Chapter 29:
- if the VM uses GC, do not collect in the middle of a primitive routine if the interpreter/object graph is temporarily inconsistent

### Current implementation delta vs. Blue Book

What the code now has:
- explicit OT exhaustion trap
- explicit reserved-singleton reuse trap
- first-pass mark/sweep reclaim of unreachable objects
- exact-size body reuse for freed objects
- interpreter-side root discovery
- one-shot allocation retry after GC
- compiled-method literal-frame tracing during marking

What it still does not have:
- free-pointer list as a first-class Blue Book structure
- free-chunk lists by size
- non-append object-space allocation
- heap compaction
- `spaceOccupiedBy:`-style reclaim behavior
- dynamic reference-count rectification
- proof that all current allocation sites are safe GC points

### First-pass architecture that is now in code

#### 1. Object memory owns reclaim traversal

File:
- [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go)

Current first-pass object-memory responsibilities:
- detect allocation exhaustion cleanly via `allocationError`
- offer retry-friendly allocation wrappers:
  - `TryInstantiateClass`
  - `TryInstantiateClassWithWords`
  - `TryInstantiateClassWithBytes`
- reclaim unreachable objects with:
  - `ReclaimInaccessibleObjects`

Important current simplification:
- reclaim only frees OT entries and exact-size bodies through the existing `reusableBodies` mechanism
- it does **not** yet implement real free-chunk lists or compaction

#### 2. Interpreter owns root discovery and retry policy

File:
- [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go)

Current interpreter responsibilities:
- define conservative root discovery via:
  - `guaranteedRootPointers`
  - `garbageCollectionRoots`
- call object memory reclaim with:
  - `collectGarbage`
- retry allocation once with:
  - `instantiateWithGarbageCollection`

Current root categories included:
- guaranteed singleton/root OOPs
- key class/selector/table roots
- active/home context
- current method and receiver
- current selector and new method
- UI-designated objects like display/cursor
- input/timer semaphores
- pending asynchronous semaphore queue
- `newProcess` when a process switch is pending

#### 3. CompiledMethod marking is special

This is non-negotiable.

CompiledMethods are not uniform pointer objects. Their object body starts with:
- header
- literal frame (pointer-bearing)
- bytecodes (raw bytes)

If the collector treats them as “non-pointer objects” and ignores the literal frame, it will free live literals and likely crash later in method lookup/send/class resolution.

The current first-pass code handles this by:
- decoding the method header
- computing literal count
- tracing only the pointer prefix (`literalCount + 1`)

### Why the first pass is intentionally smaller than Chapter 30

The full Blue Book object memory is more ambitious than what we need for the first honest recovery slice. The project had already hit a specific limit:
- object-table space

The first thing to solve is:
- reclaim unreachable objects well enough that OT entries become reusable

That is why the current implementation deliberately stops short of:
- free-chunk list management
- compaction
- reference-count rectification

This is not because those are unimportant. It is because they are not the first blocker proven by the live run.

## Design Analysis

### Why OT exhaustion appeared specifically in the UI path

The UI host path feeds mouse position into the image. The image, in turn, uses `Point` objects in normal Smalltalk code. Once that path became live, the runtime started creating many more temporary `Point`s than the old headless/stable-idle path did.

Without a real collector, temporary allocation pressure looks like a leak even if the image would normally discard those objects quickly.

### Why the earlier context-only reclamation was insufficient

The previous tactical memory-management work was intentionally narrow:
- it reclaimed method-context slots under specific reachability conditions
- it allowed exact-size body reuse for explicitly freed context bodies

That helped with one concrete frontier but it was never general GC. It could not reclaim:
- `Point`s
- arrays
- message objects
- semaphores/process nodes
- arbitrary temporary image objects

The UI path proved exactly that limitation.

### Why the new off-screen failure matters

After the first-pass collector landed, this command:

```bash
go run ./cmd/st80-exercise-snapshot -before-cycles 50000 -after-cycles 500000 -click none -text '' -return=false
```

no longer died at immediate OT exhaustion. Instead it reached a later failure:

```text
panic: checkProcessSwitch: newProcess has invalid suspendedContext ... targetContext=0x0151<SmallInteger 168>
```

This is important because it means the first-pass collector is doing something real:
- it removed the earlier hard OT limit
- the system ran farther
- a deeper semantic frontier appeared

That later failure suggests one of three things:
1. the current GC root set is incomplete
2. GC is being triggered at a point where scheduler/process state is transiently inconsistent
3. a later independent corruption path was already present but previously masked by OT exhaustion

Update after the first diagnostic rerun:
- the current off-screen reproducer reaches that scheduler failure with `gcCount=0`
- so, for that reproducer at least, option 3 is the active interpretation right now

### Likely next debugging questions

#### Question 1: Is the current root set sufficient?

Review:
- all interpreter registers that can hold object pointers
- any temporary objects created during message send / perform / primitive transitions
- scheduler/process/semaphore objects that are temporarily held in locals rather than object memory

Candidate risk:
- a process or context may be reachable only through a transient interpreter state that is not yet in `garbageCollectionRoots`

#### Question 2: Are all allocation sites safe GC points?

The current implementation collects on allocation failure, which means collection may happen during:
- context creation
- block creation
- message-object creation for `doesNotUnderstand:`
- `Point` creation
- arbitrary `new` / `new:`

This is convenient, but it may not be safe everywhere if the interpreter has temporarily removed an object from one structure and not yet stored it into the next.

Blue Book Chapter 29 explicitly warns about scheduler/process routines and transient unrooted windows.

#### Question 3: When does heap-space management become the next blocker?

Right now the proven blocker was OT exhaustion, and the first pass addresses that by freeing OT entries and reusing exact-size bodies.

Later, heap fragmentation or pure heap-space pressure may become the dominant issue. When that happens, the next required pieces are:
- free-chunk lists
- `deallocate:` semantics based on `spaceOccupiedBy:`
- compaction

## Recommended Implementation Order From Here

### Phase 1: Stabilize the first-pass collector

Do this next:
1. audit and expand `garbageCollectionRoots`
2. add diagnostics around when GC ran immediately before later scheduler failures
3. identify whether `checkProcessSwitch` corruption is caused by collection timing or missing roots

Success condition:
- longer direct/off-screen runs survive past the previous `checkProcessSwitch` failure

### Phase 2: Make allocation semantics less ad hoc

Do this after Phase 1:
1. stop relying only on append-only object-space growth
2. introduce explicit free-chunk bookkeeping
3. align reclaim/deallocation with Blue Book `spaceOccupiedBy:` behavior

Success condition:
- reclaimed heap space can satisfy later allocations even when object sizes do not match exactly

### Phase 3: Decide on compaction

Only after the first two phases are stable:
1. measure whether fragmentation is now a real blocker
2. if yes, implement compaction

Success condition:
- the VM can continue through longer UI workloads without either OT exhaustion or heap fragmentation collapse

## Review Checklist for the Intern

Use this exact review order.

1. Read [raw-ch30-object-memory.txt](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/raw-ch30-object-memory.txt) around:
   - allocation and deallocation
   - garbage collection
   - compiled-method special handling
2. Read [06-object-memory-audit.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/reference/ocr-bluebook/06-object-memory-audit.md) for the condensed object-memory summary.
3. Read [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go) top to bottom.
4. Read [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) only around:
   - allocation wrappers
   - root discovery
   - scheduler/process switching
5. Run the focused tests:

```bash
go test ./pkg/objectmemory -run 'TestInstantiatePanicsWhenObjectTableWouldOverflow15BitOopSpace|TestInstantiatePanicsWhenReservedSingletonIsMarkedFree|TestReclaimInaccessibleObjectsFreesUnreachableObjectAndReusesBody|TestReclaimInaccessibleObjectsMarksCompiledMethodLiteralPointers|TestStorePointerPanicsWhenFieldIndexIsNegative|TestStorePointerPanicsWhenFieldIndexExceedsObjectLength|TestStoreBytePanicsWhenByteIndexExceedsObjectLength'
go test ./pkg/interpreter -run 'TestInstantiateClassWithPointersCollectsGarbageOnObjectTableExhaustion|TestPositiveIntegerValueOfLargePositiveInteger|TestPrimitiveMousePointReturnsConfiguredPoint|TestPrimitiveInputWordReturnsQueuedWord'
```

6. Reproduce the next frontier:

```bash
go run ./cmd/st80-exercise-snapshot -before-cycles 50000 -after-cycles 500000 -click none -text '' -return=false
```

## Usage Examples

### Example: understanding the current first-pass recovery path

Read these code paths in order:
- [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go)
  - `instantiateClassWithPointers`
  - `instantiateWithGarbageCollection`
  - `collectGarbage`
- [objectmemory.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go)
  - `TryInstantiateClass`
  - `ReclaimInaccessibleObjects`
  - `FreeObject`

Interpretation:
- allocation fails with an allocation error
- interpreter computes roots and runs reclaim
- method cache is flushed
- allocation is retried once

### Example: reasoning about a suspected missing root

If a later crash shows a process/context/message object with obviously invalid fields:
1. ask whether that object was reachable only through interpreter locals/registers
2. check whether it is present in `garbageCollectionRoots`
3. if not, add it there before assuming the sweep logic is wrong

## Related

- [01-diary.md](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-004--smalltalk-80-object-memory-garbage-collection-and-storage-management/reference/01-diary.md)
- [ST80-004 index](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-004--smalltalk-80-object-memory-garbage-collection-and-storage-management/index.md)
- [ST80-003 UI ticket](/home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/index.md)
