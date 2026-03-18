# Claude Code Hook Events Database

A SQLite-based logger that captures every Claude Code hook event for offline analytics.

## Overview

A single Python script (`log-to-sqlite.py`) is registered on all 18 hook event types.
Each invocation reads the hook JSON from stdin, extracts structured fields, and appends
a row to a shared SQLite database. The full raw JSON is always preserved so no data is
ever lost, while commonly-queried fields are broken out into their own columns for fast,
index-backed lookups.

## Files

| File | Purpose |
|------|---------|
| `.claude/hooks/log-to-sqlite.py` | Hook script — reads stdin JSON, writes to SQLite |
| `.claude/settings.local.json` | Registers the script on all hook events |
| `~/.claude/hook-events.db` | The SQLite database (created on first run) |

## Configuration

The database path defaults to `~/.claude/hook-events.db` (global, accumulates across
projects). Override it by setting the environment variable:

```bash
export CLAUDE_HOOK_EVENTS_DB=/path/to/custom.db
```

### Making it global (all projects)

1. Copy `log-to-sqlite.py` to `~/.claude/hooks/log-to-sqlite.py`
2. In `~/.claude/settings.json`, add the same hook entries from
   `.claude/settings.local.json` but change the command to:
   ```json
   "command": "python3 ~/.claude/hooks/log-to-sqlite.py"
   ```

## Registered Hook Events

The logger is registered on every available Claude Code hook event:

| Event | When it fires |
|-------|---------------|
| `SessionStart` | New Claude Code session begins |
| `InstructionsLoaded` | CLAUDE.md and other instructions loaded |
| `SessionEnd` | Session terminates |
| `UserPromptSubmit` | User sends a prompt |
| `PreToolUse` | Before a tool is executed |
| `PermissionRequest` | User is asked to approve a tool |
| `PostToolUse` | After a tool executes successfully |
| `PostToolUseFailure` | After a tool execution fails |
| `Notification` | Agent sends a notification |
| `SubagentStart` | A subagent is spawned |
| `SubagentStop` | A subagent finishes |
| `Stop` | Claude stops generating |
| `TaskCompleted` | A background task completes |
| `ConfigChange` | Settings are modified |
| `PreCompact` | Before context compaction |
| `PostCompact` | After context compaction |
| `WorktreeCreate` | An isolated worktree is created |
| `WorktreeRemove` | An isolated worktree is removed |

## Database Schema

### Table: `hook_events`

```sql
CREATE TABLE hook_events (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp              TEXT    NOT NULL,      -- ISO 8601 UTC
    -- Common (all events)
    session_id             TEXT,
    hook_event_name        TEXT    NOT NULL,
    cwd                    TEXT,
    permission_mode        TEXT,
    transcript_path        TEXT,
    agent_id               TEXT,                  -- set inside subagents
    agent_type             TEXT,                  -- set inside subagents
    -- Tool events (Pre/PostToolUse, PostToolUseFailure, PermissionRequest)
    tool_name              TEXT,
    tool_use_id            TEXT,
    tool_input             TEXT,                  -- JSON object
    tool_response          TEXT,                  -- JSON object (PostToolUse only)
    -- Stop event
    stop_hook_active       INTEGER,               -- 0/1
    last_assistant_message TEXT,
    -- Full payload
    raw_json               TEXT    NOT NULL
);
```

### Indexes

| Index | Columns | Use case |
|-------|---------|----------|
| `idx_he_session` | `session_id` | Filter by session |
| `idx_he_event` | `hook_event_name` | Filter by event type |
| `idx_he_tool` | `tool_name` | Filter by tool |
| `idx_he_ts` | `timestamp` | Time-range queries |
| `idx_he_session_event` | `session_id, hook_event_name` | Session + event combos |
| `idx_he_session_ts` | `session_id, timestamp` | Session timeline |

### Views

#### `v_tool_usage` — Tool frequency

```
tool_name  | uses | sessions
-----------+------+---------
Bash       |  142 |       8
Read       |   97 |       7
Edit       |   53 |       5
```

#### `v_session_summary` — Per-session overview

```
session_id | started | ended | duration_secs | total_events | tool_uses | tool_failures | tools_used
```

#### `v_daily_activity` — Aggregated by day

```
day        | events | sessions | tool_uses
-----------+--------+----------+----------
2026-03-17 |    312 |        3 |      198
```

#### `v_file_touch_frequency` — Most touched files

```
file_path              | tool_name | touches
-----------------------+-----------+--------
src/vm/interpreter.go  | Edit      |      12
src/vm/interpreter.go  | Read      |       8
```

#### `v_bash_commands` — Shell command log

```
session_id | timestamp | command          | description
-----------+-----------+------------------+-----------
abc-123    | ...       | go test ./...    | Run tests
```

#### `v_search_patterns` — Grep/Glob pattern frequency

```
tool_name | pattern        | search_path | uses
----------+----------------+-------------+-----
Grep      | func.*Main     | .           |    5
Glob      | **/*.go        | .           |    3
```

## Example Queries

### Basics

```sql
-- Everything from the last hour
SELECT * FROM hook_events
WHERE timestamp > datetime('now', '-1 hour');

-- Count events by type
SELECT hook_event_name, COUNT(*) FROM hook_events GROUP BY 1 ORDER BY 2 DESC;
```

### Tool analysis

```sql
-- Slowest sessions (by event count as proxy)
SELECT * FROM v_session_summary ORDER BY total_events DESC LIMIT 10;

-- Files edited more than 5 times
SELECT * FROM v_file_touch_frequency WHERE tool_name = 'Edit' AND touches > 5;

-- Bash commands containing 'test'
SELECT * FROM v_bash_commands WHERE command LIKE '%test%';
```

### JSON extraction (ad-hoc)

SQLite's JSON1 extension lets you query into the JSON columns:

```sql
-- Grep patterns used in a specific session
SELECT json_extract(tool_input, '$.pattern') AS pattern
FROM hook_events
WHERE tool_name = 'Grep' AND session_id = 'my-session-id';

-- Files written with their content length
SELECT
    json_extract(tool_input, '$.file_path') AS file,
    length(json_extract(tool_input, '$.content')) AS bytes
FROM hook_events
WHERE tool_name = 'Write' AND hook_event_name = 'PostToolUse';

-- Agent tool prompts
SELECT
    json_extract(tool_input, '$.description') AS desc,
    json_extract(tool_input, '$.subagent_type') AS agent_type
FROM hook_events
WHERE tool_name = 'Agent';
```

### Failure analysis

```sql
-- All tool failures
SELECT timestamp, tool_name, tool_input, tool_response
FROM hook_events
WHERE hook_event_name = 'PostToolUseFailure'
ORDER BY timestamp DESC;

-- Failure rate by tool
SELECT
    tool_name,
    COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END) AS successes,
    COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END) AS failures,
    ROUND(100.0 * COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END)
        / COUNT(*), 1) AS failure_pct
FROM hook_events
WHERE hook_event_name IN ('PostToolUse', 'PostToolUseFailure')
GROUP BY tool_name
ORDER BY failure_pct DESC;
```

## Design Decisions

- **Single table:** All events share a common shape. Tool-specific columns are NULL for
  non-tool events. This keeps queries simple (no JOINs) while SQLite stores NULLs
  efficiently.
- **raw_json column:** Full payload is always preserved. If new fields are added to
  hook events upstream, they're captured automatically and can be queried via
  `json_extract()` without a schema migration.
- **WAL mode + busy_timeout:** Enables concurrent reads while a write is in progress.
  The 3-second busy timeout handles rare lock contention without blocking Claude.
- **Silent failures:** All SQLite errors are caught and swallowed. The hook must never
  block or crash Claude — losing a log row is acceptable, hanging the session is not.
- **Python over shell:** Parameterized queries prevent SQL injection from tool inputs
  that may contain arbitrary strings (file paths, user code, etc.). JSON parsing is
  also more reliable than `jq` pipeline chains.
- **Global DB, local config:** The database lives in `~/.claude/` so data accumulates
  across projects and sessions. The hook registration is per-project in
  `settings.local.json` so it can be adopted incrementally.

## Maintenance

```bash
# Database size
ls -lh ~/.claude/hook-events.db

# Row count
sqlite3 ~/.claude/hook-events.db "SELECT COUNT(*) FROM hook_events"

# Purge events older than 30 days
sqlite3 ~/.claude/hook-events.db "DELETE FROM hook_events WHERE timestamp < datetime('now', '-30 days')"

# Reclaim space after purge
sqlite3 ~/.claude/hook-events.db "VACUUM"
```
