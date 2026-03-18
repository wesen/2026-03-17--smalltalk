---
Title: Interpreter Architecture Guide
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
      Note: Main interpreter implementation
    - Path: pkg/objectmemory/objectmemory.go
      Note: Object memory interface
    - Path: pkg/image/loader.go
      Note: Image file loader
    - Path: data/bluebook-spec-notes.md
      Note: Extracted Blue Book specifications
ExternalSources:
    - https://www.wolczko.com/st80/
Summary: Complete guide to how the Smalltalk-80 interpreter works, for someone building one from scratch
LastUpdated: 2026-03-17T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# How to Build a Smalltalk-80 Interpreter: A Guide

## Overview

A Smalltalk-80 VM has three layers: **Object Memory** (stores objects), **Image Loader** (reads the snapshot), and **Interpreter** (executes bytecodes). All Smalltalk behavior lives in the virtual image — a binary dump of objects. The VM loads this image and resumes execution.

## Layer 1: Object Memory

### Objects and OOPs

Every object has a 16-bit Object Oriented Pointer (OOP):
- **Bit 0 = 0**: Regular object — upper 15 bits index the Object Table
- **Bit 0 = 1**: SmallInteger — upper 15 bits are the signed value (-16384..16383)

### Object Table

Maps OOPs to heap locations. Each entry is 2 words:
- Word 0: COUNT(8) | OddLength(1) | Pointer(1) | Free(1) | unused(1) | SEGMENT(4)
- Word 1: LOCATION in segment

**Critical**: The Blue Book uses MSB-first bit numbering. "Bits 12-15" = standard bits 3-0. We got segment addressing wrong twice (first bits 4-1, then bits 4-0) before finding the correct interpretation (bits 3-0).

Heap address = segment × 65536 + location.

### Object Body

```
Word 0: SIZE (total words including size+class)
Word 1: CLASS OOP
Word 2+: fields (OOPs or raw data)
```

## Layer 2: Image Loading

The wolczko.com VirtualImage format:
- Bytes 0-3: Object space size in words (big-endian uint32)
- Bytes 4-7: Object table size in words (big-endian uint32)
- Bytes 8-511: Padding
- Object Space: starts at byte 512
- Object Table: last otSize×2 bytes of file

Validate with guaranteed pointers: OOP 2=nil, 4=false, 6=true, 8=SchedulerAssociation, 48=SpecialSelectors(Array/64), 50=CharacterTable(Array/256).

## Layer 3: The Interpreter

### State Registers

- **activeContext**: Current MethodContext or BlockContext
- **homeContext**: The MethodContext (for blocks, the block's home)
- **method**: CompiledMethod being executed
- **receiver**: The message receiver
- **instructionPointer**: Byte index into method bytecodes
- **stackPointer**: Field index of stack top in context

### Context Objects

**MethodContext**: sender(0), IP(1), SP(2), method(3), unused(4), receiver(5), temps+stack(6+)
**BlockContext**: caller(0), IP(1), SP(2), argCount(3), initialIP(4), home(5), stack(6+)

Distinguish them: field 3 is CompiledMethod (OOP, even) in MethodContext, SmallInteger (odd) in BlockContext.

### Main Loop

```
forever:
    checkProcessSwitch()
    bytecode = fetchByte()
    dispatch(bytecode)
```

### Bytecodes (256 total, 4 categories)

| Range | Category | Description |
|-------|----------|-------------|
| 0-15 | Stack | Push receiver variable |
| 16-31 | Stack | Push temporary variable |
| 32-63 | Stack | Push literal constant |
| 64-95 | Stack | Push literal variable (Association value) |
| 96-111 | Stack | Pop and store into receiver/temp variable |
| 112-119 | Stack | Push self/true/false/nil/-1/0/1/2 |
| 120-125 | Return | Return self/true/false/nil/stackTop from method/block |
| 128-130 | Stack | Extended push/store (2-byte) |
| 131-134 | Send | Extended send/super send (2-3 byte) |
| 135-137 | Stack | Pop/dup/push context |
| 144-175 | Jump | Unconditional and conditional jumps |
| 176-191 | Send | Arithmetic (+, -, <, >, *, /, etc.) |
| 192-207 | Send | Special (at:, ==, class, value, blockCopy:) |
| 208-255 | Send | Literal selector (0/1/2 args) |

### Message Sending

1. Get selector and argument count from bytecode
2. Find receiver on stack (below arguments)
3. Look up method: method cache → class hierarchy → doesNotUnderstand:
4. Execute: try primitive first, then create new MethodContext

### CompiledMethod Header

SmallInteger in field 0 encodes:
- Bits 0-2: Flag (0-4=argcount, 5=return self, 6=return ivar, 7=has extension)
- Bits 3-7: Temp count (or field index for flag=6)
- Bit 8: Large context flag
- Bits 9-14: Literal count

### Process Scheduling

OOP 8 → Association → ProcessorScheduler → activeProcess → suspendedContext.
`checkProcessSwitch()` signals buffered semaphores and switches processes via `transferTo:`.

### Primitives

~127 primitive methods: arithmetic (1-18), subscripting (60-67), allocation (68-79), control (80-89), I/O (90-109), system (110-127). Each succeeds (replaces stack) or fails (leaves stack, falls through to bytecodes).

## Practical Lessons

1. **Start with object memory + image loader.** Validate with guaranteed pointers before writing any interpreter code.
2. **Blue Book uses MSB-first bit numbering.** "Bit 0" = most significant. Convert carefully.
3. **Match the reference trace.** The trace2 file from wolczko.com shows the first ~472 message sends. Your bytecodes should match exactly.
4. **Always use large contexts initially.** Stack overflow from too-small contexts is hard to debug.
5. **Implement primitives incrementally.** Start with arithmetic and ==, add more as the interpreter demands them.
6. **Don't implement GC initially.** The image has enough free space. The OT free list provides new object table entries.
7. **The image format is undocumented.** Reverse-engineer from hex dumps + known object validation.
