---
Title: Smalltalk-80 object memory garbage collection and storage management
Ticket: ST80-004
Status: active
Topics:
    - go
    - smalltalk
    - vm
    - gc
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/interpreter/interpreter.go
      Note: Interpreter integration seam for collection and retry
    - Path: pkg/objectmemory/objectmemory.go
      Note: Primary implementation file for this ticket
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-18T16:46:13.921935531-04:00
WhatFor: Track the Blue Book Chapter 30 storage-management frontier after the live UI path exhausted the Smalltalk-80 object table.
WhenToUse: Use when working on object allocation, garbage collection, root discovery, free-chunk management, compaction, or any late-runtime corruption that now appears after OT exhaustion is removed.
---


# Smalltalk-80 object memory garbage collection and storage management

## Overview

This ticket isolates the Smalltalk-80 object-memory and garbage-collection work from the host-UI ticket. The trigger was a decisive live run failure: `primitiveMakePoint` exhausted the full 15-bit object-table space (`otEntryCount=32768`) under real mouse/UI allocation pressure.

The first implementation slice is now in place. The VM no longer dies at immediate object-table exhaustion; instead it performs a first-pass mark/sweep reclaim and retries allocation. That moved the frontier to a later `checkProcessSwitch` failure with an invalid `suspendedContext`, which is the next debugging target.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Diary**: [reference/01-diary.md](./reference/01-diary.md)
- **Intern Design Doc**: [reference/02-gc-plan-design-and-analysis-for-intern.md](./reference/02-gc-plan-design-and-analysis-for-intern.md)

## Status

Current status: **active**

Current implementation state:
- first-pass mark/sweep reclaim exists
- allocation retries once after GC
- compiled-method literal pointers are traced during marking
- full Blue Book free-chunk management / compaction is not implemented yet
- later scheduler corruption still remains after the OT frontier is removed

## Topics

- go
- smalltalk
- vm
- gc

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
