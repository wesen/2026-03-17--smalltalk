# Changelog

## 2026-03-17

- Initial workspace created


## 2026-03-17

Project setup: created docmgr ticket ST80-001, diary document, CLAUDE.md, Claude Code hooks for diary/commit reminders


## 2026-03-17

Step 2: Object memory and image loader implemented (commit ab8f650). Reverse-engineered wolczko.com VirtualImage format. All 18,391 objects load correctly, guaranteed pointers validated.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/cmd/st80/main.go — Test harness for image loading
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/image/loader.go — Image file loader for wolczko.com format
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/objectmemory/objectmemory.go — Object memory with corrected OT bit layout

