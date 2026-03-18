#!/usr/bin/env python3
"""Claude Code status line: display token counts and log snapshots to SQLite.

Receives JSON on stdin with context_window and cost data after each assistant turn.
Displays a compact token/cost summary and appends a row to the token_snapshots table.
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
CREATE TABLE IF NOT EXISTS token_snapshots (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp             TEXT    NOT NULL,
    session_id            TEXT,
    model_id              TEXT,
    total_input_tokens    INTEGER,
    total_output_tokens   INTEGER,
    total_cost_usd        REAL,
    total_duration_ms     INTEGER,
    total_api_duration_ms INTEGER,
    context_window_size   INTEGER,
    used_percentage       INTEGER,
    remaining_percentage  INTEGER,
    current_input_tokens  INTEGER,
    current_output_tokens INTEGER,
    cache_creation_tokens INTEGER,
    cache_read_tokens     INTEGER,
    total_lines_added     INTEGER,
    total_lines_removed   INTEGER,
    exceeds_200k          INTEGER,
    raw_json              TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ts_session ON token_snapshots(session_id);
CREATE INDEX IF NOT EXISTS idx_ts_ts      ON token_snapshots(timestamp);

CREATE VIEW IF NOT EXISTS v_token_timeline AS
SELECT
    session_id,
    timestamp,
    total_input_tokens,
    total_output_tokens,
    total_input_tokens + total_output_tokens AS total_tokens,
    total_cost_usd,
    used_percentage,
    total_lines_added,
    total_lines_removed
FROM token_snapshots
ORDER BY timestamp;

CREATE VIEW IF NOT EXISTS v_session_token_summary AS
SELECT
    session_id,
    model_id,
    MIN(timestamp)                AS started,
    MAX(timestamp)                AS ended,
    COUNT(*)                      AS snapshots,
    MAX(total_input_tokens)       AS final_input_tokens,
    MAX(total_output_tokens)      AS final_output_tokens,
    MAX(total_input_tokens) + MAX(total_output_tokens) AS final_total_tokens,
    MAX(total_cost_usd)           AS final_cost_usd,
    MAX(total_duration_ms)        AS duration_ms,
    MAX(total_api_duration_ms)    AS api_duration_ms,
    MAX(total_lines_added)        AS lines_added,
    MAX(total_lines_removed)      AS lines_removed,
    MAX(used_percentage)          AS peak_context_pct
FROM token_snapshots
GROUP BY session_id
ORDER BY started DESC;
"""


def fmt_tokens(n):
    """Format token count: 1234 -> 1.2k, 12345 -> 12.3k, 123456 -> 123k."""
    if n is None:
        return "0"
    if n < 1000:
        return str(n)
    if n < 10000:
        return f"{n / 1000:.1f}k"
    if n < 1000000:
        return f"{n / 1000:.0f}k"
    return f"{n / 1000000:.1f}M"


def progress_bar(pct, width=10):
    """Render a text progress bar: ████░░░░░░"""
    if pct is None:
        pct = 0
    filled = round(pct / 100 * width)
    return "\u2588" * filled + "\u2591" * (width - filled)


def main():
    raw = sys.stdin.read()
    if not raw.strip():
        return

    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return

    # ── Extract fields ───────────────────────────────────────────────
    ctx = data.get("context_window") or {}
    cost = data.get("cost") or {}
    model = data.get("model") or {}
    cur = ctx.get("current_usage") or {}

    total_in = ctx.get("total_input_tokens") or 0
    total_out = ctx.get("total_output_tokens") or 0
    used_pct = ctx.get("used_percentage") or 0
    remaining_pct = ctx.get("remaining_percentage") or 0
    cost_usd = cost.get("total_cost_usd") or 0
    duration_ms = cost.get("total_duration_ms") or 0
    api_ms = cost.get("total_api_duration_ms") or 0
    lines_add = cost.get("total_lines_added") or 0
    lines_rm = cost.get("total_lines_removed") or 0

    # ── Log to SQLite (only when tokens actually changed) ──────────
    try:
        conn = sqlite3.connect(DB_PATH, timeout=3)
        conn.execute("PRAGMA journal_mode=WAL")
        conn.execute("PRAGMA busy_timeout=2000")
        conn.executescript(SCHEMA)

        sid = data.get("session_id")
        last = conn.execute(
            """SELECT total_input_tokens, total_output_tokens
            FROM token_snapshots
            WHERE session_id = ?
            ORDER BY id DESC LIMIT 1""",
            (sid,),
        ).fetchone()

        # Skip if token counts haven't changed since last snapshot
        if not last or last[0] != total_in or last[1] != total_out:
            conn.execute(
                """INSERT INTO token_snapshots (
                    timestamp, session_id, model_id,
                    total_input_tokens, total_output_tokens, total_cost_usd,
                    total_duration_ms, total_api_duration_ms,
                    context_window_size, used_percentage, remaining_percentage,
                    current_input_tokens, current_output_tokens,
                    cache_creation_tokens, cache_read_tokens,
                    total_lines_added, total_lines_removed,
                    exceeds_200k, raw_json
                ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)""",
                (
                    datetime.now(timezone.utc).isoformat(),
                    sid,
                    model.get("id"),
                    total_in,
                    total_out,
                    cost_usd,
                    duration_ms,
                    api_ms,
                    ctx.get("context_window_size"),
                    used_pct,
                    remaining_pct,
                    cur.get("input_tokens"),
                    cur.get("output_tokens"),
                    cur.get("cache_creation_input_tokens"),
                    cur.get("cache_read_input_tokens"),
                    lines_add,
                    lines_rm,
                    1 if data.get("exceeds_200k_tokens") else 0,
                    raw,
                ),
            )
            conn.commit()
        conn.close()
    except sqlite3.Error:
        pass  # never break the status line

    # ── Render status line ───────────────────────────────────────────
    total = total_in + total_out
    dur_s = duration_ms // 1000
    dur_str = f"{dur_s // 60}m{dur_s % 60:02d}s" if dur_s >= 60 else f"{dur_s}s"

    bar = progress_bar(used_pct, 12)

    # Color codes: green if <50%, yellow 50-80%, red >80%
    if used_pct > 80:
        pct_color = "\033[31m"  # red
    elif used_pct > 50:
        pct_color = "\033[33m"  # yellow
    else:
        pct_color = "\033[32m"  # green
    reset = "\033[0m"

    line1 = (
        f"\033[1m\u2191{fmt_tokens(total_in)}\033[0m "
        f"\033[1m\u2193{fmt_tokens(total_out)}\033[0m "
        f"\033[2m({fmt_tokens(total)})\033[0m"
        f"  \033[33m${cost_usd:.3f}\033[0m"
        f"  {pct_color}{bar} {used_pct}%{reset}"
    )

    line2 = (
        f"\033[2m+{lines_add}/-{lines_rm} lines"
        f"  {dur_str}"
        f"  api {api_ms}ms\033[0m"
    )

    print(line1)
    print(line2)


if __name__ == "__main__":
    main()
