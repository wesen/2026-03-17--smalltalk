---
Title: Context Body Reuse and Segment Wrap Allocation Bug Writeup
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
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go
      Note: Final allocator/body-reuse fix and segment-wrap guard (commit 6cb8881)
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory_test.go
      Note: Focused regression tests for exact-size body reuse, reserved mismatched slots, and segment exhaustion (commit 6cb8881)
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go
      Note: Context-shape hardening so undersized non-context objects are rejected cleanly (commit 6cb8881)
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go
      Note: The retained late-runtime trace that proved the bad block was already invalid immediately after blockCopy:
ExternalSources: []
Summary: Intern-facing writeup of the late-runtime allocation bug where recycled context OOP slots outlived their bodies, object-space growth eventually wrapped the 4-bit segment field, and blockCopy: began returning immediately invalid block objects.
LastUpdated: 2026-03-18T11:40:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Context Body Reuse and Segment Wrap Allocation Bug Writeup

## Goal

Explain the late-runtime allocator bug fixed by commit `6cb8881`, including the misleading symptom, the actual root cause, the failed first fix, and the narrower final repair that brought the interpreter back to stable long runs.

## Context

After the earlier block/value and `String>>at:put:` fixes, the interpreter no longer failed in display setup or in early context churn. The next runtime frontier moved much later, to about cycle `708768`.

At first glance, the crash still looked like a generic message-send problem:
- `Number>>to:do:` eventually sent `value:`
- the receiver of `value:` was invalid
- the system fell into recursive `doesNotUnderstand:`

That framing was misleading. A closer trace showed the `value:` receiver was already invalid immediately after the preceding `blockCopy:` in `IdentityDictionary>>keyAtValue:ifAbsent:`.

That changed the question from:
- "Why does the block get corrupted later?"

to:
- "Why is `primitiveBlockCopy` producing a broken block object after a long run?"

## Quick Reference

### Symptom

The late retained trace logged the freshly created would-be block object like this:

```text
createdBlock oop=0xAA7E class=0x0002() wordLen=49169 field0=0x0002 field1=0x0023 field2=0x0001 field3=0x0003 field4=0x0023 field5=0xAA7C
```

That is not a valid new `BlockContext`:
- class is wrong
- size is absurd
- the fields look like the OOP is reading from the wrong place in object space

### Immediate cause

`blockCopy:` itself was not filling the new context incorrectly. The newly allocated OOP was already pointing at the wrong object body location.

### Real root cause

The VM had started recycling method-context OOP slots, but it still appended every new object body to `objectSpace`.

That combination is dangerous in this object-memory format because an OTE stores:
- 4 bits of segment
- 16 bits of location within that segment

In this implementation, `HeapAddress` reconstructs the body address as:

```text
segment * 65536 + location
```

So once appends push object-space growth past segment `15`, a new body location cannot be represented exactly anymore. The segment bits wrap/truncate, and the OOP starts pointing at unrelated old memory.

That is why the late `blockCopy:` result looked instantly nonsensical.

### Why OOP-slot recycling alone was not enough

Reusing only the object-table entry helps delay OOP exhaustion, but it does not reclaim any object-space words. The object table keeps pointing into newer and newer appended bodies, and eventually the address format loses fidelity.

In other words:
- OOP-slot recycling solved one pressure source
- object-body growth remained unbounded
- the remaining bug surfaced when body addressing wrapped

## The Failed First Fix

The first repair attempt was conceptually on the right layer but still wrong in a critical way.

I first tried to reuse freed bodies without also constraining how the corresponding freed OOP slots could be reused. That produced an early startup regression around cycle `95`:

```text
panic at cycle=95 method=0x0362(<DisplayScreen class>displayExtent:): FetchPointer: OOP 0x02AC field 3: addr 259221 out of bounds (os=259220, loc=259216)
```

This exposed a second requirement:
- stale references to recycled context OOPs can still exist in context fields or dead stack slots
- if those OOPs are reassigned to a completely different, smaller object shape, later context-shape checks can walk into a valid but non-context object

So the allocator fix had to do more than "reuse some bodies." It also had to avoid reassigning tracked freed context OOPs to unrelated object shapes.

## Final Fix

The landed repair has four parts.

### 1. Track reusable bodies only for OOPs explicitly freed by this VM run

New image-loaded free entries are not assumed safe for body reuse. The allocator now records reusable bodies only when `FreeObject` is called on a real live object that this runtime is retiring.

That avoids treating arbitrary image-era free slots as if they carried a trustworthy reusable body.

### 2. Reuse only exact-size freed bodies

Before appending to `objectSpace`, the allocator scans tracked reusable bodies and reuses one only when:
- the OOP slot is free
- the freed body was explicitly tracked
- the requested body size exactly matches the retired body size

When that happens, the allocator rewrites:
- size
- class
- fields

and returns the same OOP without growing object space.

### 3. Reserve tracked freed context slots from mismatched reuse

If a free OOP slot has a tracked reusable body but the requested allocation size does not match, that slot is skipped rather than being reassigned to some unrelated new object.

That was the key correction after the failed first attempt. It prevents a recently freed context OOP from immediately turning into a tiny non-context object while stale references to the old context OOP still exist elsewhere.

### 4. Harden context-shape checks

`isMethodContext` and `isBlockContext` now require the candidate object to have at least `MethodIndex + 1` fields before they try to read field `3`.

Without that guard, any undersized but otherwise valid object could trigger an out-of-bounds fetch during context probing.

### 5. Fail loudly on true segment exhaustion

The allocator now panics explicitly if a fresh append would require a segment greater than `15`.

That is not a collector, but it is still an important correctness improvement:
- silent address wrap causes fake objects and misleading crashes
- an explicit exhaustion panic preserves the real failure mode

## Result

After the final fix:
- `go test ./...` passes
- `go run ./cmd/st80 data/VirtualImage 800000` completes cleanly
- `go run ./cmd/st80 data/VirtualImage 2000000` completes cleanly

That means the old late `blockCopy:` corruption frontier is gone, and the interpreter is back to the earlier two-million-cycle stability level while keeping the newer block/value/string fixes in place.

## Detailed Walkthrough

### Why the symptom was deceptive

The panic surfaced through `value:` and later `doesNotUnderstand:` behavior, which makes it easy to blame:
- send lookup
- block invocation
- context switching
- sender/home corruption

The decisive observation was simply timing:
- the block was already invalid immediately after `blockCopy:`
- therefore the bug had to be in allocation or addressing, not later execution

### Why the segment field matters

The object-table entry format is compact. That is efficient, but it makes the allocator unforgiving.

If the allocator appends forever without reclamation, eventually:
- `segment := fullLocation / 65536`
- `location := fullLocation % 65536`

stops being lossless, because only 4 bits of `segment` are stored.

Once that happens, the object table no longer names the actual object body that was just allocated.

### Why the first failed repair was still useful

The startup regression was not wasted work. It proved two things quickly:

1. The right layer really was object memory.
2. OOP identity and body identity cannot be repaired independently.

That failure made the final fix much narrower and better motivated.

## Minimal Review Checklist

If an intern reviews this bug later, they should verify:

1. `FreeObject` preserves enough metadata to describe a retired reusable body without making initial image free entries look reusable.
2. `instantiate(...)` only reuses tracked bodies on exact-size matches.
3. Tracked freed slots are skipped for mismatched allocations rather than being repurposed immediately.
4. Context-shape checks reject undersized objects before reading field `3`.
5. Validation crosses the old failure frontier, not just unit tests.

## Validation Commands

```bash
go test ./...
go run ./cmd/st80 data/VirtualImage 800000
go run ./cmd/st80 data/VirtualImage 2000000
```

Expected post-fix behavior:
- no invalid `blockCopy:` product around cycle `708768`
- no recursive `doesNotUnderstand:` at the old frontier
- stable long runs through at least two million cycles

## Related

- See `reference/01-diary.md` Step 5 for the trace that isolated the invalid block immediately after `blockCopy:`.
- See `reference/01-diary.md` Step 7 for the chronological implementation details of the final allocator fix.
