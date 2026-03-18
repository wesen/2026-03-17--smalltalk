---
Title: Tagged SmallInteger Header Decode Bug Writeup
Ticket: ST80-002
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
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go
      Note: The actual fix for header, header-extension, and instance-spec decoding (commit dd8e4ba)
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go
      Note: Regression coverage proving startup now runs past the former overflow (commit dd8e4ba)
ExternalSources: []
Summary: Intern-facing writeup of the tagged SmallInteger decoding bug that caused the startup context overflow.
LastUpdated: 2026-03-17T23:33:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Tagged SmallInteger Header Decode Bug Writeup

## Goal

Explain one concrete VM bug and fix in enough detail that someone new to the codebase can review it later without reconstructing the entire debugging session.

## Context

The Smalltalk-80 VM stores several metadata words as `SmallInteger` objects inside the image:
- `CompiledMethod` field 0: method header
- header extension literal (for methods with flag value 7)
- class field 2: instance specification

In this interpreter, `SmallInteger` values are encoded as tagged OOPs:
- low bit `1` means “this is a SmallInteger”
- the remaining upper 15 bits carry the signed value

That means the bitfield described in the Blue Book is **not** stored directly in the raw 16-bit word fetched from object memory. The fetched value is a tagged pointer-like representation and must be decoded first.

The bug happened because we fetched those fields and immediately applied `extractBits(...)` to the encoded OOP. This shifted every logical bitfield by one position and contaminated anything derived from the header/spec values.

## Quick Reference

### Symptom

Startup crashed almost immediately:

```text
Interpreter panic: StorePointer: OOP 0x418E field 38: addr 260316 out of bounds (os=260316, loc=260276)
```

### First misleading interpretation

At first glance this looked like a simple context-capacity problem:
- the active context overflowed
- the VM was already using large contexts for new activations
- maybe an image-resident context had stale stack contents

That was a reasonable hypothesis, but it was not the real root cause.

### Decisive evidence

An in-package reproducer showed the crashing method state:

```text
cycle=148 activeContext=0x418E method=0x021E receiver=0x31A2 bytecode=0 ip=121 sp=38
contextFields=38 storedIP=0x00CF storedSP=0x001D tempCount=14 largeContextFlag=0
```

This is the key contradiction:
- `tempCount=14`
- `largeContextFlag=0`

Per the Blue Book, the large-context flag is set when:

```text
maximum stack depth + temporary count > 12
```

So `tempCount=14` with `largeContextFlag=0` is impossible. A method that already needs 14 temp-frame entries cannot fit in a small context even before operand stack growth is considered.

### Root Cause

We were extracting bits from tagged SmallInteger OOPs instead of decoded SmallInteger payloads.

Broken pattern:

```go
header := interp.fetchPointer(HeaderIndex, methodPointer)
return extractBits(3, 7, header)
```

Correct pattern:

```go
header := uint16(om.SmallIntegerValue(interp.fetchPointer(HeaderIndex, methodPointer)))
return extractBits(3, 7, header)
```

### Fix Sites

Three places needed the same correction:

1. `headerOf`
2. `headerExtensionOf`
3. `instanceSpecificationOf`

### Why `instanceSpecificationOf` mattered too

Even though the startup crash surfaced through method activation, the same decoding mistake also affected class layout queries:
- pointer-vs-word objects
- indexable flag
- fixed field counts

So the bug was broader than just temp counts and large-context flags.

### Result

After the fix:
- the startup overflow disappeared
- `go test ./...` passed
- `go run ./cmd/st80 data/VirtualImage 3000` completed
- the next real blocker moved deeper into runtime execution:

```text
Interpreter panic: Recursive not understood error encountered
```

That is progress. It means the old crash was masking later interpreter issues.

## Detailed Walkthrough

### Why the bug is easy to miss

The code looked structurally correct:
- fetch method header
- slice bits according to the Blue Book
- use the decoded temp count, literal count, primitive index, and flags

The subtle mistake was semantic, not structural: the fetched word was a `SmallInteger` object pointer, not the raw 15-bit header value the Blue Book’s bit numbering refers to.

Because the bug was only one level below the business logic, downstream failures looked unrelated:
- context overflow
- implausible stack growth
- suspicious large/small context behavior

### Why the reproduced contradiction matters

The contradiction was more valuable than the panic itself.

A raw overflow only says:
- “we stored past the end of the allocated frame”

But the decoded state says:
- “the metadata we believe about this method is internally inconsistent with the Blue Book”

That narrows the search dramatically from:
- context creation
- push/pop semantics
- image corruption
- sender chain bugs

down to:
- header extraction
- tagged integer decoding
- bitfield interpretation

### Why the fix is correct

The Blue Book describes method headers and class instance specifications as bitfields over the logical SmallInteger value. In this implementation, that logical value is recovered via:

```go
om.SmallIntegerValue(oop)
```

Only after that decode step is it valid to apply:

```go
extractBits(first, last, value)
```

This is also consistent with how the interpreter already handles other SmallInteger-backed fields such as:
- instruction pointers
- stack pointers
- block argument counts

The metadata path simply had not applied the same decode discipline.

### Minimal Review Checklist

If an intern reviews this bug later, they should verify:

1. Any Blue Book bitfield stored in an object field is first classified:
   - raw word?
   - object pointer?
   - tagged SmallInteger?
2. Any tagged SmallInteger is decoded before bit slicing.
3. Regression coverage actually crosses the former failure point rather than only unit-testing a helper in isolation.
4. The post-fix runtime advances to a different blocker, proving the old crash was genuinely removed.

## Usage Examples

### Reproduce the old failure state

Use the history in the diary or checkout a commit before `dd8e4ba`, then run:

```bash
go run ./cmd/st80 data/VirtualImage 2000
```

Expected old behavior:

```text
Interpreter panic: StorePointer: OOP 0x418E field 38: addr 260316 out of bounds (os=260316, loc=260276)
```

### Verify the fixed behavior

```bash
go test ./...
go run ./cmd/st80 data/VirtualImage 3000
go run ./cmd/st80 data/VirtualImage 500000
```

Expected new behavior:
- tests pass
- the 3000-cycle run completes
- the 500000-cycle run reaches the next blocker instead of the startup overflow

### Pattern to reuse elsewhere

If you see any future code that looks like this:

```go
bits := extractBits(a, b, interp.fetchPointer(field, object))
```

ask whether the fetched field is a tagged SmallInteger. If yes, decode it first.

## Related

- See `reference/01-diary.md` for the chronological debugging log of this fix.
- See the ST80-001 handoff doc `reference/03-current-issues-and-research-needed.md` for the earlier, partially incorrect overflow hypothesis that this step superseded.
