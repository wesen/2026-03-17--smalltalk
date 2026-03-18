#!/usr/bin/env python3
"""Log all Claude Code hook events to a SQLite database for stats analysis.

DB location: ~/.claude/hook-events.db (override with CLAUDE_HOOK_EVENTS_DB env var).
Receives hook JSON on stdin, extracts structured fields, stores raw JSON for full fidelity.
"""

import json
import os
import sqlite3
import sys
from datetime import datetime, timezone

DB_PATH = os.environ.get(
    "CLAUDE_HOOK_EVENTS_DB",
    os.path.expanduser("~/.claude/hook-events.db"),
)

SCHEMA = """
CREATE TABLE IF NOT EXISTS hook_events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp       TEXT    NOT NULL,
    -- Common fields (present on all events)
    session_id      TEXT,
    hook_event_name TEXT    NOT NULL,
    cwd             TEXT,
    permission_mode TEXT,
    transcript_path TEXT,
    agent_id        TEXT,
    agent_type      TEXT,
    -- Tool fields (PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest)
    tool_name       TEXT,
    tool_use_id     TEXT,
    tool_input      TEXT,   -- JSON object
    tool_response   TEXT,   -- JSON object (PostToolUse only)
    -- Stop fields
    stop_hook_active    INTEGER,
    last_assistant_message TEXT,
    -- Full payload for anything not extracted above
    raw_json        TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_he_session       ON hook_events(session_id);
CREATE INDEX IF NOT EXISTS idx_he_event         ON hook_events(hook_event_name);
CREATE INDEX IF NOT EXISTS idx_he_tool          ON hook_events(tool_name);
CREATE INDEX IF NOT EXISTS idx_he_ts            ON hook_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_he_session_event ON hook_events(session_id, hook_event_name);
CREATE INDEX IF NOT EXISTS idx_he_session_ts    ON hook_events(session_id, timestamp);

-- Handy views for common stats queries --

CREATE VIEW IF NOT EXISTS v_tool_usage AS
SELECT
    tool_name,
    COUNT(*)                       AS uses,
    COUNT(DISTINCT session_id)     AS sessions
FROM hook_events
WHERE tool_name IS NOT NULL
  AND hook_event_name = 'PostToolUse'
GROUP BY tool_name
ORDER BY uses DESC;

CREATE VIEW IF NOT EXISTS v_session_summary AS
SELECT
    session_id,
    MIN(timestamp)  AS started,
    MAX(timestamp)  AS ended,
    ROUND((julianday(MAX(timestamp)) - julianday(MIN(timestamp))) * 86400) AS duration_secs,
    COUNT(*)        AS total_events,
    COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END)        AS tool_uses,
    COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END) AS tool_failures,
    GROUP_CONCAT(DISTINCT tool_name) AS tools_used
FROM hook_events
GROUP BY session_id
ORDER BY started DESC;

CREATE VIEW IF NOT EXISTS v_daily_activity AS
SELECT
    date(timestamp)   AS day,
    COUNT(*)          AS events,
    COUNT(DISTINCT session_id) AS sessions,
    COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END) AS tool_uses
FROM hook_events
GROUP BY date(timestamp)
ORDER BY day DESC;

CREATE VIEW IF NOT EXISTS v_file_touch_frequency AS
SELECT
    json_extract(tool_input, '$.file_path') AS file_path,
    tool_name,
    COUNT(*) AS touches
FROM hook_events
WHERE tool_name IN ('Read', 'Edit', 'Write')
  AND hook_event_name = 'PostToolUse'
  AND json_extract(tool_input, '$.file_path') IS NOT NULL
GROUP BY file_path, tool_name
ORDER BY touches DESC;

CREATE VIEW IF NOT EXISTS v_bash_commands AS
SELECT
    session_id,
    timestamp,
    json_extract(tool_input, '$.command')     AS command,
    json_extract(tool_input, '$.description') AS description
FROM hook_events
WHERE tool_name = 'Bash'
  AND hook_event_name = 'PostToolUse'
ORDER BY timestamp DESC;

CREATE VIEW IF NOT EXISTS v_search_patterns AS
SELECT
    tool_name,
    json_extract(tool_input, '$.pattern') AS pattern,
    json_extract(tool_input, '$.path')    AS search_path,
    COUNT(*) AS uses
FROM hook_events
WHERE tool_name IN ('Grep', 'Glob')
  AND hook_event_name = 'PostToolUse'
GROUP BY tool_name, pattern, search_path
ORDER BY uses DESC;
"""


def init_db(conn):
    """Create tables, indexes, and views if they don't exist."""
    conn.executescript(SCHEMA)


def main():
    raw = sys.stdin.read()
    if not raw.strip():
        return

    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return  # silently skip malformed input

    try:
        conn = sqlite3.connect(DB_PATH, timeout=5)
        conn.execute("PRAGMA journal_mode=WAL")
        conn.execute("PRAGMA busy_timeout=3000")
        init_db(conn)
    except sqlite3.Error:
        return  # don't block Claude if DB is inaccessible

    def text(key):
        v = data.get(key)
        if v is None:
            return None
        if isinstance(v, (dict, list)):
            return json.dumps(v, separators=(",", ":"))
        return str(v)

    def boolean(key):
        v = data.get(key)
        if v is None:
            return None
        return 1 if v else 0

    try:
        conn.execute(
            """
            INSERT INTO hook_events (
                timestamp, session_id, hook_event_name, cwd, permission_mode,
                transcript_path, agent_id, agent_type, tool_name, tool_use_id,
                tool_input, tool_response, stop_hook_active, last_assistant_message,
                raw_json
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                datetime.now(timezone.utc).isoformat(),
                text("session_id"),
                text("hook_event_name") or "unknown",
                text("cwd"),
                text("permission_mode"),
                text("transcript_path"),
                text("agent_id"),
                text("agent_type"),
                text("tool_name"),
                text("tool_use_id"),
                text("tool_input"),
                text("tool_response"),
                boolean("stop_hook_active"),
                text("last_assistant_message"),
                raw,
            ),
        )
        conn.commit()
    except sqlite3.Error:
        pass  # never block Claude
    finally:
        conn.close()


if __name__ == "__main__":
    main()
