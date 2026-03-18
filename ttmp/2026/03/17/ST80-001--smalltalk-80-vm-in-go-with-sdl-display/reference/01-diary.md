---
Title: Diary
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
    - Path: cmd/st80/main.go
      Note: Test harness
    - Path: data/VirtualImage
      Note: Smalltalk-80 virtual image from wolczko.com
    - Path: pkg/image/loader.go
      Note: Image loader
    - Path: pkg/objectmemory/objectmemory.go
      Note: Object memory implementation
    - Path: smalltalk-Bluebook.pdf
      Note: Blue Book PDF reference
ExternalSources:
    - https://www.wolczko.com/st80/
Summary: Implementation diary for porting the Smalltalk-80 VM to Go with SDL display
LastUpdated: 2026-03-17T19:43:20.368656267-04:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Capture the step-by-step implementation journey of porting the Smalltalk-80 virtual machine (from wolczko.com/st80) to Go with an SDL-based display. Record decisions, failures, learnings, and review instructions.

## Step 1: Project Setup and Infrastructure

Set up the docmgr ticket workspace, CLAUDE.md project conventions, and Claude Code hooks to ensure consistent diary-keeping and commit discipline throughout the implementation.

### Prompt Context

**User prompt (verbatim):** "Implement a go VM with SDL view for https://www.wolczko.com/st80/" (followed by) "Create a docmgr ticket for the port, and keep a frequent diary as you work. Store in claude.md and make a hook https://code.claude.com/docs/en/hooks so that after each taskcompleted it adds "Don't forget to update your diary (see skills) if you haven't already, otherwise continue. Don't forget to commit at appropriate intervals." and to UserPromptSubmit. Also PostCompact : "read docmgr and diary skills, and going forward don't forget to keep a diary and commit as you go. Don't forget to commit at appropriate intervals.""

**Assistant interpretation:** Create the project scaffolding (docmgr ticket, diary doc, CLAUDE.md, hooks) before starting the actual VM implementation.

**Inferred user intent:** Ensure disciplined documentation and version control throughout a complex multi-step implementation project.

### What I did
- Initialized docmgr in the project (`docmgr init --seed-vocabulary`)
- Created `.ttmp.yaml` config
- Created ticket `ST80-001` with topics vm, smalltalk, sdl, go
- Created diary document at `ttmp/.../reference/01-diary.md`
- Created CLAUDE.md with project conventions
- Created Claude Code hooks for TaskCompleted, UserPromptSubmit, PostCompact

### Why
- The VM port is a complex multi-step project that benefits from structured documentation
- Hooks ensure diary and commit discipline without manual reminders

### What worked
- docmgr init and ticket creation went smoothly after finding the right commands

### What didn't work
- Initial `docmgr doc add` failed because the ticket didn't exist yet (needed `docmgr ticket create-ticket` first)
- `docmgr init /path` failed with "Too many arguments" (needed `--root` flag instead)

### What I learned
- docmgr requires `ticket create-ticket` before `doc add`
- Ticket directories are created with a date-based path template: `YYYY/MM/DD/TICKET--slug`

### What was tricky to build
- N/A for this setup step

### What warrants a second pair of eyes
- Hook configuration in `.claude/settings.local.json` - verify the event names and message format are correct

### What should be done in the future
- Begin researching the Smalltalk-80 VM specification from wolczko.com/st80
- Plan the VM architecture (object memory, interpreter, bytecode dispatch, SDL display)

### Code review instructions
- Check CLAUDE.md for project conventions
- Check `.claude/settings.local.json` for hook configuration
- Verify diary format matches the diary skill spec

### Technical details
- Ticket path: `ttmp/2026/03/17/ST80-001--smalltalk-80-vm-in-go-with-sdl-display/`
- Diary path: `ttmp/.../reference/01-diary.md`

## Step 2: Image Loader and Object Memory — Reverse-Engineering the Virtual Image Format

This was the hardest step so far — figuring out the binary format of the Smalltalk-80 virtual image file from wolczko.com, since it's not documented anywhere. Through iterative hex-dump analysis and hypothesis testing, I determined the exact file layout and object table bit encoding.

### Prompt Context

**User prompt (verbatim):** (same as Step 1 — "Implement a go VM with SDL view for https://www.wolczko.com/st80/")

**Assistant interpretation:** Implement the foundational layers: object memory data structures and image file loading.

**Inferred user intent:** Get the VM to the point where it can load the virtual image and inspect objects, as the prerequisite for building the interpreter.

**Commit (code):** ab8f650 — "Implement object memory and image loader with correct format"

### What I did
- Created `pkg/objectmemory/objectmemory.go` — full Blue Book object memory interface
- Created `pkg/image/loader.go` — image file parser
- Created `cmd/st80/main.go` — diagnostic/test program
- Extracted the Smalltalk-80 virtual image from wolczko.com into `data/`
- Read the Blue Book PDF (chapters 26-27) for guaranteed pointer values and object structure
- Wrote multiple diagnostic programs to probe the image file format
- Fixed the OT bit layout (pointer/free bits were swapped)
- Verified all guaranteed pointers load correctly (nil, false, true, SchedulerAssociation, SpecialSelectors, CharacterTable, class objects)

### Why
- The object memory is the foundation of the entire VM — everything else depends on being able to load and access objects
- The image format isn't documented; it had to be reverse-engineered empirically

### What worked
- Class.oops file from wolczko.com provided ground truth for validating OOP assignments
- Interpreting data at file offset 0x200 as `[size, class, fields...]` immediately showed sensible results (Association with class=0x0084, Array with class=0x0010)
- Once the SchedulerAssociation (OOP 8) validated as an Association pointing to location 6, the entire format clicked

### What didn't work
- **Initial OT bit layout was wrong**: I assumed bit 6=free, bit 5=pointer. The actual layout is bit 6=pointer, bit 5=free. This caused all guaranteed pointers to show as "FREE".
- **Tried multiple wrong file layouts**: Hypothesis A (header=896, OS first, OT last) seemed wrong because OOP 2 appeared free — but the real issue was the bit layout, not the file layout.
- **Header size confusion**: The file has 596,128 bytes. The computed "header" of 896 bytes was actually 512 bytes of true header + 384 bytes of padding between OS and OT.
- **WebFetch failures**: wolczko.com had certificate issues, mirandabanda.org had self-signed cert, esug.org returned 404. Had to rely on the Blue Book PDF instead.

### What I learned
- **Virtual image file format**: `[512-byte header][objectSpace][384-byte gap][objectTable]`
  - Header bytes 0-3: OS size in words (big-endian uint32)
  - Header bytes 4-7: OT size in words (big-endian uint32)
  - Object space starts at offset 512
  - Object table is at `fileSize - otSize*2`
- **OT entry format** (2 words per entry):
  - Word 0: count(8) | oddLength(1) | pointerFields(1) | free(1) | segment(4) | unused(1)
  - Word 1: location (word offset into object space)
- **Object body format**: `[size, class, field0, field1, ...]` where size includes the size and class words
- **Guaranteed pointers** (Blue Book p.576): NilPointer=2, FalsePointer=4, TruePointer=6, SchedulerAssociationPointer=8, etc.
- **SmallInteger encoding**: bit 0=1, value = OOP >> 1 (signed). ZeroPointer=1, OnePointer=3, MinusOnePointer=65535.

### What was tricky to build
The image format reverse-engineering was the main challenge. The symptoms: all well-known OOPs showed as FREE despite 11,761 objects appearing valid. The root cause was the swapped pointer/free bits in the OT entry word 0. I discovered this by:
1. Noticing that data at offset 0x200 gave sensible object space data (Association, Array classes matched class.oops)
2. Checking the OT at the file end and seeing OOP 8's w1 value (0x0006) pointed to exactly where the Association object sat in the OS
3. Realizing that swapping bits 5 and 6 made all guaranteed pointers valid

### What warrants a second pair of eyes
- The 384-byte gap between OS and OT — is this always present, or is it image-specific?
- Whether the `segment` field in OT entries matters for this image (all objects appear to be in segment 0)
- The `count` field values (reference counts) — some are very high (128, 136, 213) which may need investigation

### What should be done in the future
- Implement process scheduling (primitives 85-88)
- Implement remaining storage management primitives
- Implement SDL display and I/O primitives
- Boot the full Smalltalk-80 environment

### Technical details
- Image: 596,128 bytes, 258,880 words OS, 38,736 words OT (19,368 entries)
- 18,391 used objects, 977 free entries
- SpecialSelectors: Array of 64 elements (32 selector/argcount pairs)
- CharacterTable: Array of 256 Character instances
- SchedulerAssociation: key=0x6536 (Symbol #Processor), value=0x87BE (ProcessorScheduler)

## Step 3: Interpreter Skeleton and Segment Fix

Built the full bytecode interpreter skeleton and fixed two critical addressing bugs that prevented startup.

### Prompt Context

**User prompt (verbatim):** "continue. don't stop until the interpreter is done and successfully runs programs and all"

**Assistant interpretation:** Implement the complete interpreter and keep going until it runs.

**Inferred user intent:** Get a working Smalltalk-80 VM that can execute the image.

**Commit (code):** a3136a0 — "Add interpreter skeleton and object allocation"
**Commit (code):** 85dfccc — "Fix segment addressing: use bits 3-0"

### What I did
- Implemented full bytecode dispatch (stack, jump, send, return bytecodes)
- Implemented arithmetic primitives 1-18, subscript primitives 60-67
- Implemented storage management primitives 68-79 (stubs for become, someInstance, nextInstance)
- Implemented control primitive stubs (80-89)
- Implemented system primitives 110-116
- Added object allocation (InstantiateClass)
- Fixed segment addressing: the Blue Book uses MSB-first bit numbering, so segmentBitsOf reads bits 12-15 which are bits 3-0 in standard LSB numbering
- Fixed Process field layout: suspendedContext is at index 1 (not 2)
- Extracted comprehensive Blue Book spec notes to data/bluebook-spec-notes.md

### What didn't work
- **Segment addressing wrong twice**: First used bits 4-1 (otSegmentShift=1, mask=0x001E), then bits 4-0 (mask=0x001F), finally correct with bits 3-0 (mask=0x000F). The Blue Book p.662 says "bits 12 to 15" which maps to standard bits 3-0.
- **Process field order wrong**: Assumed Link(0:nextLink) → Process(1:myList, 2:suspendedContext, 3:priority). Actual: Link(0:nextLink) → Process(1:suspendedContext, 2:priority, 3:myList). Found from Blue Book p.642 initializeSchedulerIndices.

### What I learned
- Blue Book uses MSB-first bit numbering throughout (bit 0 = most significant). Must always convert.
- The wolczko.com image uses 4 segments (0-3) with 65536 words each covering 258,880 words.
- Interpreter successfully runs 100 cycles with context switching, message sends, and returns.

### What was tricky to build
The segment addressing required three attempts. The confusion came from the Blue Book's MSB-first bit numbering convention. "bits 12 to 15" means the 4 LEAST significant bits in standard notation, not what you'd expect from reading "12 to 15". The empirical verification method (trying different formulas against known objects) was the key to finding the correct interpretation.

### What warrants a second pair of eyes
- The method cache hash function — using `(selector & class) & 0xFF + 1` which might not match the Blue Book exactly
- Block copy primitive — the initialIP calculation may be off

### What should be done in the future
- Run for more cycles and fix crashes as they arise
- Implement process scheduling primitives (85-88)
- Implement I/O primitives for SDL display
- Add the trace2/trace3 validation (compare execution trace against wolczko.com reference traces)
