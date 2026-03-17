---
Title: Smalltalk-80 VM in Go with SDL Display
Ticket: ST80-001
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
ExternalSources:
    - https://www.wolczko.com/st80/
Summary: "Port of the Smalltalk-80 virtual machine to Go with an SDL-based display"
LastUpdated: 2026-03-17T19:43:08.02134575-04:00
WhatFor: ""
WhenToUse: ""
---

# Smalltalk-80 VM in Go with SDL Display

## Overview

Port of the Smalltalk-80 virtual machine to Go, based on the specification and image files available at https://www.wolczko.com/st80/. The VM will include an SDL-based graphical display for the Smalltalk environment.

## Key Links

- **Source spec:** https://www.wolczko.com/st80/
- **Diary:** [reference/01-diary.md](./reference/01-diary.md)

## Status

Current status: **active** - Project setup phase

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
- reference/ - Prompt packs, API contracts, context summaries (includes diary)
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
