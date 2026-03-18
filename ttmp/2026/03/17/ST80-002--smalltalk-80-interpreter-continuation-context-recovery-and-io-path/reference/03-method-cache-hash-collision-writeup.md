---
Title: Method Cache Hash Collision Writeup
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
      Note: Fixes the method cache hash so each cache entry uses four consecutive slots (commit 408f7b8)
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go
      Note: Diagnostic and regression tests that isolated the corruption to cached lookup (commit 408f7b8)
ExternalSources: []
Summary: Intern-facing writeup of the method cache hash bug that caused cached selector/class collisions and bogus method activations.
LastUpdated: 2026-03-17T23:59:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Method Cache Hash Collision Writeup

## Goal

Explain the second major runtime bug fixed in ST80-002: cached method lookup was corrupt even when direct dictionary lookup was correct, because the cache hash did not reserve four words per entry the way the Blue Book requires.

## Context

After fixing the tagged-SmallInteger metadata decoding bug, the interpreter got much farther but still failed later with a recursive `doesNotUnderstand:` path. That made the failure look like another generic message-send or context problem.

Closer tracing showed a narrower pattern:
- a direct lookup of `Point>>y` was correct
- the live startup run still activated a bogus method when sending `y` to a `Point`
- clearing the method cache right before that send made the corruption disappear

So the dictionary lookup path was sound. The cached path was not.

## Quick Reference

### Symptom

During startup, cycle 129 executed special selector bytecode `207` (`y`) while running:

```text
<Form>extent:offset:bits:
```

The receiver on the stack was a real `Point`, but the interpreter activated an invalid method:

```text
method became invalid at cycle 129 after bytecode=207 in ctx=0x3BF6 home=0x3BF6 method=0x170C(<Form>extent:offset:bits:) ip=6 sp=9; new method=0x021E class=0x0038 activeContext=0x418E homeContext=0x418E isBlock=false
```

### Expected ground truth

From the local Xerox method tables:
- `Point>>x` = `0x8B6A`
- `Point>>y` = `0x8BAC`

Direct uncached lookup confirmed the interpreter could find `Point>>y` correctly.

### Root Cause

The Blue Book’s method cache uses four sequential array slots per entry:
- selector
- class
- CompiledMethod
- primitive index

The hash formula in Smalltalk is:

```text
(((messageSelector bitAnd: class) bitAnd: 16rFF) bitShift: 2) + 1
```

The important part is `bitShift: 2`.

Our Go version had translated the selector/class mask but omitted the `<< 2`, effectively packing four-word entries into adjacent single slots and causing unrelated entries to alias.

Broken version:

```go
h := ((int(interp.messageSelector) & int(class)) & 0xFF) + 1
```

Fixed version:

```go
h := ((int(interp.messageSelector) & int(class)) & 0xFF) << 2
```

Because Go arrays are 0-based, the Smalltalk `+ 1` is intentionally omitted after adding the left shift.

### Why direct lookup still worked

`lookupMethodInClass(...)` traverses the class hierarchy and the target class’s `MethodDictionary` correctly. The failure only happened when `findNewMethodInClass(...)` hit a stale or aliased cache entry with the same selector/class lookup position but overlapping storage.

That distinction is the main lesson from this bug: direct lookup and cached lookup are separate failure surfaces.

### Result

After fixing the hash:
- the cycle-129 corruption disappeared
- the later `doesNotUnderstand:` recursion disappeared
- the interpreter ran through 500,000 cycles cleanly
- the interpreter ran through 2,000,000 cycles cleanly
- `go test ./...` passed

## Detailed Walkthrough

### How the bug was isolated

The debugging sequence that actually mattered was:

1. Detect the first invalid `method` register rather than waiting for a later panic.
2. See that the corruption happens on a `y` send to a `Point`.
3. Verify that the known-good Xerox tables expect `Point>>y`.
4. Run direct lookup for `Point>>y` and confirm it returns `0x8BAC`.
5. Clear the cache before the failing send and observe that the corruption disappears.
6. Compare `findNewMethodInClass(...)` with the Blue Book cache algorithm.

That sequence is worth keeping because it cleanly separates:
- lookup correctness
- cache correctness
- runtime consequences

### Why the bug is easy to mistranslate

The Blue Book algorithm is written against:
- 1-based Smalltalk Arrays
- groups of four words per cache entry

When porting to Go, two independent transformations are easy to mix up:
- remove the final `+ 1` because arrays are 0-based
- keep the `bitShift: 2` because the entry still spans 4 words

The port accidentally removed the `+ 1` correctly but also dropped the `<< 2`, which was not optional.

### Why the wrong behavior looked unrelated

A cache alias does not necessarily fail at the moment the bad entry is written. It fails later, when a different selector/class pair lands on the overlapping storage. That delayed failure makes the symptom look like:
- random bad method activation
- context corruption
- stack corruption
- recursive `doesNotUnderstand:`

Those are downstream effects. The actual bug is structural aliasing inside the cache array.

### Minimal Review Checklist

If an intern reviews cache code later, they should confirm:

1. Each logical cache entry has a fixed-width physical layout.
2. The hash computation lands only on legal entry boundaries.
3. The port preserves semantic operations from the Blue Book even when index bases change.
4. Direct lookup and cached lookup are tested separately.

## Usage Examples

### Reproduce the isolation path

Commands used during the investigation:

```bash
go test ./pkg/interpreter -run TestDetectFirstInvalidMethodRegister -v
go test ./pkg/interpreter -run TestLookupPointYMethod -v
go test ./pkg/interpreter -run TestTraceAroundMethodCorruption -v
pdftotext smalltalk-Bluebook.pdf - | sed -n '34890,34940p'
```

### Validate the fix

```bash
go test ./...
go run ./cmd/st80 data/VirtualImage 500000
go run ./cmd/st80 data/VirtualImage 2000000
```

Expected post-fix behavior:
- no early bad method activation on the `Point>>y` send
- no recursive `doesNotUnderstand:` panic
- stable long run

### Pattern to reuse elsewhere

When a cached path fails but the uncached path succeeds:
- do not treat it as a general lookup bug
- compare array layout assumptions against the original spec
- check for lost width/stride operations during porting

## Related

- See `reference/01-diary.md` Step 2 for the full chronological investigation.
- See `reference/02-tagged-smallinteger-header-decode-bug-writeup.md` for the earlier, separate metadata bug that this cache fix followed.
