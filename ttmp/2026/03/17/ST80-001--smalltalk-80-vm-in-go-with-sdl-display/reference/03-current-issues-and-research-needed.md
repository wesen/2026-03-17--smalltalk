---
Title: Current Issues and Research Needed
Ticket: ST80-001
Status: active
Topics:
    - vm
    - smalltalk
    - sdl
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/interpreter/interpreter.go
      Note: Interpreter with the bugs
    - Path: pkg/objectmemory/objectmemory.go
      Note: Object memory with bounds checking
    - Path: data/trace2
      Note: Reference trace (first 472 message sends)
    - Path: data/bluebook-spec-notes.md
      Note: Extracted specs for reference
ExternalSources:
    - https://www.wolczko.com/st80/
Summary: Analysis of current interpreter bugs and open research questions
LastUpdated: 2026-03-17T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Current Issues and Research Needed

## Status Summary

The interpreter loads the virtual image correctly and executes bytecodes matching the wolczko.com reference trace (trace2). It runs 1M+ cycles without crashing (when using large contexts). The system reaches the scheduler idle loop, where it repeatedly polls for events.

**What works:**
- Image loading with correct segment addressing
- All 256 bytecodes dispatched correctly
- Arithmetic primitives (1-18)
- Subscript primitives (60-67)
- Storage management (70-71: new/new:, 73-75: instVarAt/hash)
- Control primitives (80-81: blockCopy/value)
- System primitives (110-111: ==/class)
- Process scheduling (85-88: signal/wait/resume/suspend)
- Method lookup with cache
- Context creation and switching
- Reference trace validation (first 472 sends match exactly)

**What doesn't work (crashes or stubs):**
- Context stack overflow at cycle ~148 (see Issue 1 below)
- Float primitives (40-54) — stub, always fail
- LargePositiveInteger primitives (21-37) — stub, always fail
- Stream primitives (65-67) — stub
- perform: primitive (83-84) — stub
- become: primitive (72) — stub
- I/O primitives (90-105) — stub (display, cursor, input, BitBlt)
- Snapshot primitive (97) — stub
- No SDL display

## Issue 1: Context Stack Overflow at Cycle 148 (BLOCKER)

### Symptoms
At cycle 148, the interpreter panics:
```
StorePointer: OOP 0x418E field 38: addr 259938 out of bounds
```

The context at OOP 0x418E has 38 fields (size 40 including header). The stack pointer reaches field index 38, which is 1 past the end. This happens even with large contexts (32 stack slots).

### Analysis

The crashing method is OOP 0x021E, executing at IP=121 with bytecode 0 (push receiver variable 0). The context was loaded from the image (OOP 0x418E is in the original OT range), not newly allocated.

The stack should have at most TempFrameStart(6) + 32 = 38 fields for a large context. But field index 38 means we're writing to the 39th field (0-indexed), which overflows.

### Possible Root Causes

1. **Pre-existing context from image has accumulated stack state**: The image was snapshotted mid-execution. The context at 0x418E already had data in its stack area. When we resume execution, we start pushing without resetting the stack, and it overflows because the original context was sized for the original method's needs.

2. **The `temporaryCountOf` extraction is wrong**: If we extract too few temporaries from the method header, the initial SP is set too low, and subsequent pushes go higher than expected. However, the SP is stored in the context itself (field 2), so for pre-existing contexts, our extraction doesn't affect the starting SP.

3. **The `largeContextFlagOf` extraction is wrong**: The method at 0x021E might need a large context but the flag extraction returns 0. We're now always using large contexts, so this isn't the immediate cause. But it could explain why the image's contexts are too small.

4. **Our push doesn't match the Blue Book**: The Blue Book's `push:` increments SP then stores. Our code does the same. But maybe the SP value interpretation is off by 1.

### Research Needed

- **Dump OOP 0x418E's full state**: What method is it executing? What's the SP value? How many temps? What's the stack depth?
- **Check the method at OOP 0x021E**: What class/selector is this? How many literals, temps? What's the large context flag?
- **Compare our IP/SP interpretation with the Blue Book**: The Blue Book stores IP as a 1-based byte index relative to the method. Our code subtracts 1 to make it 0-based. Verify this matches.
- **Check if contexts in the image have correct sizes**: Walk the object table and check all MethodContext/BlockContext objects to see if any are undersized for their methods.

### Workaround

Currently using `contextSize = TempFrameStart + 32` for ALL new contexts. But the issue is with pre-existing contexts from the image, not newly allocated ones.

A brute-force fix would be to grow the context's object space allocation when a push would overflow. This requires modifying the object memory to support in-place growth or relocation.

## Issue 2: Idle Scheduler Loop

### Symptoms
After the startup sequence (~500 cycles), the interpreter enters an infinite loop. It cycles between the same few methods (visible at the 500K-cycle reporting interval).

### Analysis
This is expected behavior. The Smalltalk startup code calls `Processor postSnapshot` which re-initializes the system. After initialization, the scheduler enters its idle loop, waiting for input events to signal a Semaphore that would wake a process.

### What's Needed
To break out of the idle loop, we need:
1. **I/O primitives** — particularly `primitiveInputSemaphore` (93) to register the input Semaphore, and `primitiveInputWord` (95) to provide event data
2. **Timer primitives** — `primitiveSignalAtTick` (100) for the millisecond clock
3. **Display primitives** — `primitiveBeDisplay` (102) and `primitiveCopyBits` (96) for BitBlt
4. **SDL integration** — actual display window, mouse/keyboard input feeding events into the system

### Research Needed
- What sequence of I/O events does the image expect during startup?
- What does the input event word format look like? (Blue Book p.648: type in high 4 bits, parameter in low 12)
- How does the display initialization work? (Form/DisplayScreen objects)
- What's the minimum set of I/O primitives needed to get past the idle loop?

## Issue 3: Missing Primitives

### Float Primitives (40-54)
Floats are stored as IEEE 754 single-precision, split across 2 words of a non-pointer object. Need to implement:
- Conversion: SmallInteger → Float (40)
- Arithmetic: +, -, *, / (41-50)
- Comparison: <, >, <=, >=, =, ~= (43-48)
- truncated (51), fractionPart (52), exponent (53), timesTwoPower: (54)

### LargePositiveInteger Primitives (21-37)
These are optional (the Blue Book says so). Smalltalk methods exist as fallbacks. But they may be needed for performance during startup (the trace shows `digitMultiply:neg:` calls).

### perform: Primitives (83-84)
The perform: family removes the selector from the stack and sends the message. Non-trivial because it requires rewriting the stack layout mid-send.

### become: Primitive (72)
Swaps the identity of two objects. Requires exchanging their object table entries. Important for some system operations.

## Issue 4: Memory Management

### No Garbage Collection
Currently, new objects are appended to the end of the object space and free OT entries are reused. There is no GC. The image has 977 free OT entries and ~0 free heap space. After those entries are exhausted, allocation will fail.

### Research Needed
- How many objects does the startup sequence allocate?
- When will we run out of OT entries?
- Should we implement mark-sweep or reference counting?
- The Blue Book (Chapter 30) describes a compacting GC — is it necessary for basic operation?

## Issue 5: Trace Validation Beyond 472 Cycles

### Current State
We've verified the interpreter matches trace2 (472 message-send cycles) and trace3 (1979 cycles). Beyond that, we have no reference data.

### Research Needed
- Can we generate our own execution trace matching the trace2/trace3 format?
- Are there longer reference traces available?
- Can we cross-validate by comparing class/method names at each send against the method.oops file?

## Priority Order

1. **Fix context overflow** (Issue 1) — this is the immediate blocker
2. **Implement basic I/O stubs** (Issue 2) — to break out of idle loop
3. **Implement float primitives** (Issue 3) — needed for display coordinates
4. **Add SDL display** (Issue 2) — to see output
5. **Implement GC** (Issue 4) — for sustained operation
