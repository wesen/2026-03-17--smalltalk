# Smalltalk-80 VM in Go with SDL Display

## Project

Porting the Smalltalk-80 virtual machine (spec at https://www.wolczko.com/st80/) to Go with an SDL-based display.

## Ticket

- **Ticket ID:** ST80-001
- **Ticket path:** `ttmp/2026/03/17/ST80-001--smalltalk-80-vm-in-go-with-sdl-display/`
- **Diary:** `ttmp/2026/03/17/ST80-001--smalltalk-80-vm-in-go-with-sdl-display/reference/01-diary.md`

## Working Discipline

- **Diary:** Update the diary (see `/diary` skill) after completing each meaningful step. Include failures and exact errors.
- **Commits:** Commit code at appropriate intervals. Commit docs separately from code.
- **docmgr:** Use docmgr for task tracking, changelog, and file relations. See `/docmgr` skill.
- **Working loop:** Implement -> test -> commit code -> update diary/tasks/changelog -> commit docs.

## Commands

```bash
# Run tests
go test ./...

# Build
go build ./...

# Format
gofmt -w .
```
