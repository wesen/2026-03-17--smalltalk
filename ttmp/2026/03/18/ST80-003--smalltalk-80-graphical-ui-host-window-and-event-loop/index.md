---
Title: Smalltalk-80 graphical UI host window and event loop
Ticket: ST80-003
Status: active
Topics:
    - vm
    - smalltalk
    - sdl
    - go
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-18T09:24:38.913946293-04:00
WhatFor: ""
WhenToUse: ""
---

# Smalltalk-80 graphical UI host window and event loop

## Overview

This ticket begins after the interpreter/runtime stabilized with a real BitBlt implementation. Its purpose is to expose the designated Smalltalk display form in a host SDL window, then expand that first visible UI milestone into full interactive input/time integration.

Current state: a working SDL host-window command exists in `cmd/st80-ui`, the interpreter can now run in stepped chunks and export display snapshots, and the full UI loop has been validated under SDL’s dummy video driver. Mouse, keyboard, cursor, and timer integration remain open.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- vm
- smalltalk
- sdl
- go

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
