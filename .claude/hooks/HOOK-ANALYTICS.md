# Claude Code Hook Analytics — Full Reference

A complete guide to the hook logging infrastructure, the SQLite database, the
web UI, and how to write your own queries against transcripts and hook events.

## Architecture Overview

```
Claude Code Session
    |
    |-- [18 hook events] --> log-to-sqlite.py --> hook_events table
    |
    |-- [status line]    --> statusline.py    --> token_snapshots table
    |
    |-- [transcript]     --> ~/.claude/projects/<project>/<session>.jsonl
    |
    +-- hook-events-server.py (web UI)
            reads: hook_events + token_snapshots + transcript files
            serves: http://127.0.0.1:8642
```

Three data sources, one database, one web UI:

| Source | What it captures | Storage |
|--------|-----------------|---------|
| `log-to-sqlite.py` | Every hook event (tool use, session lifecycle, etc.) | `hook_events` table |
| `statusline.py` | Token counts, cost, context window after each turn | `token_snapshots` table |
| Transcripts | Full conversation: prompts, responses, thinking, tool I/O | `.jsonl` files on disk |

All three share `session_id` as the join key.

## Quick Start

```bash
# Start the web UI
python3 .claude/hooks/hook-events-server.py

# Open http://127.0.0.1:8642

# Or query directly
sqlite3 ~/.claude/hook-events.db
```

## Database Location

Default: `~/.claude/hook-events.db`

Override: `export CLAUDE_HOOK_EVENTS_DB=/path/to/db.db`

Both `log-to-sqlite.py` and `statusline.py` respect this env var, and the web
server accepts `--db /path/to/db.db`.

---

## Tables

### `hook_events`

One row per hook event. Created by `log-to-sqlite.py`.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-increment primary key |
| `timestamp` | TEXT | ISO 8601 UTC when the event fired |
| `session_id` | TEXT | Groups events to a Claude Code session |
| `hook_event_name` | TEXT | Event type (see list below) |
| `cwd` | TEXT | Working directory |
| `permission_mode` | TEXT | `default`, `plan`, `bypassPermissions`, etc. |
| `transcript_path` | TEXT | Absolute path to the session's `.jsonl` transcript |
| `agent_id` | TEXT | Set when event fires inside a subagent |
| `agent_type` | TEXT | Subagent type (`Explore`, `Plan`, etc.) |
| `tool_name` | TEXT | `Bash`, `Read`, `Edit`, `Write`, `Grep`, `Glob`, `Agent`, etc. |
| `tool_use_id` | TEXT | Unique ID for this tool invocation (matches transcript `tool_use.id`) |
| `tool_input` | TEXT | JSON object — the tool's parameters |
| `tool_response` | TEXT | JSON object — the tool's output (PostToolUse only) |
| `stop_hook_active` | INTEGER | 0/1 (Stop event only) |
| `last_assistant_message` | TEXT | Claude's last response (Stop event only) |
| `raw_json` | TEXT | Full original hook payload as JSON |

**Hook event types stored:**

| Event | When it fires |
|-------|---------------|
| `SessionStart` | New session begins |
| `InstructionsLoaded` | CLAUDE.md loaded |
| `SessionEnd` | Session terminates |
| `UserPromptSubmit` | User sends a prompt |
| `PreToolUse` | Before tool execution |
| `PermissionRequest` | Permission prompt shown |
| `PostToolUse` | After successful tool execution |
| `PostToolUseFailure` | After failed tool execution |
| `Notification` | Agent notification |
| `SubagentStart` | Subagent spawned |
| `SubagentStop` | Subagent finished |
| `Stop` | Claude stops generating |
| `TaskCompleted` | Background task completes |
| `ConfigChange` | Settings modified |
| `PreCompact` | Before context compaction |
| `PostCompact` | After context compaction |
| `WorktreeCreate` | Worktree created |
| `WorktreeRemove` | Worktree removed |

### `token_snapshots`

One row per assistant turn (deduplicated — only when tokens change). Created by
`statusline.py`.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-increment primary key |
| `timestamp` | TEXT | ISO 8601 UTC |
| `session_id` | TEXT | Join key to `hook_events` |
| `model_id` | TEXT | e.g. `claude-opus-4-6[1m]` |
| `total_input_tokens` | INTEGER | Cumulative input tokens for session |
| `total_output_tokens` | INTEGER | Cumulative output tokens for session |
| `total_cost_usd` | REAL | Cumulative session cost in USD |
| `total_duration_ms` | INTEGER | Wall-clock time since session start |
| `total_api_duration_ms` | INTEGER | Time spent waiting for API |
| `context_window_size` | INTEGER | Max context window (200k or 1M) |
| `used_percentage` | INTEGER | % of context window used |
| `remaining_percentage` | INTEGER | % remaining |
| `current_input_tokens` | INTEGER | Input tokens in current context |
| `current_output_tokens` | INTEGER | Output tokens in last response |
| `cache_creation_tokens` | INTEGER | Tokens written to prompt cache |
| `cache_read_tokens` | INTEGER | Tokens read from prompt cache |
| `total_lines_added` | INTEGER | Cumulative lines of code added |
| `total_lines_removed` | INTEGER | Cumulative lines removed |
| `exceeds_200k` | INTEGER | 0/1, whether total tokens > 200k |
| `raw_json` | TEXT | Full status line JSON payload |

---

## Views

### From `log-to-sqlite.py`

**`v_tool_usage`** — Tool frequency ranking.

```sql
SELECT * FROM v_tool_usage;
-- tool_name | uses | sessions
-- Bash      |  142 |        8
-- Read      |   97 |        7
```

**`v_session_summary`** — Per-session overview.

```sql
SELECT * FROM v_session_summary;
-- session_id | started | ended | duration_secs | total_events | tool_uses | tool_failures | tools_used
```

**`v_daily_activity`** — Aggregated by day.

```sql
SELECT * FROM v_daily_activity;
-- day        | events | sessions | tool_uses
```

**`v_file_touch_frequency`** — Most read/edited/written files.

```sql
SELECT * FROM v_file_touch_frequency LIMIT 20;
-- file_path | tool_name | touches
```

**`v_bash_commands`** — Every shell command run.

```sql
SELECT * FROM v_bash_commands LIMIT 20;
-- session_id | timestamp | command | description
```

**`v_search_patterns`** — Grep/Glob patterns and frequency.

```sql
SELECT * FROM v_search_patterns;
-- tool_name | pattern | search_path | uses
```

### From `statusline.py`

**`v_token_timeline`** — Token counts over time.

```sql
SELECT * FROM v_token_timeline;
-- session_id | timestamp | total_input_tokens | total_output_tokens | total_tokens | total_cost_usd | used_percentage
```

**`v_session_token_summary`** — Final token counts per session.

```sql
SELECT * FROM v_session_token_summary;
-- session_id | model_id | started | ended | snapshots | final_input_tokens | final_output_tokens | final_total_tokens | final_cost_usd | duration_ms | api_duration_ms | lines_added | lines_removed | peak_context_pct
```

---

## Querying the Database

### Basics

```bash
# Interactive mode
sqlite3 ~/.claude/hook-events.db

# One-shot query
sqlite3 ~/.claude/hook-events.db "SELECT * FROM v_tool_usage"

# Pretty output
sqlite3 ~/.claude/hook-events.db ".headers on" ".mode column" "SELECT * FROM v_tool_usage"

# CSV export
sqlite3 ~/.claude/hook-events.db ".headers on" ".mode csv" "SELECT * FROM v_session_summary" > sessions.csv
```

### Extracting JSON fields with `json_extract()`

The `tool_input` and `tool_response` columns store JSON. SQLite's built-in
`json_extract()` function lets you reach into them:

```sql
-- What Bash commands were run?
SELECT
    timestamp,
    json_extract(tool_input, '$.command') AS cmd,
    json_extract(tool_input, '$.description') AS desc
FROM hook_events
WHERE tool_name = 'Bash' AND hook_event_name = 'PostToolUse'
ORDER BY timestamp DESC;

-- What files were edited?
SELECT
    json_extract(tool_input, '$.file_path') AS file,
    COUNT(*) AS edits
FROM hook_events
WHERE tool_name = 'Edit' AND hook_event_name = 'PostToolUse'
GROUP BY file
ORDER BY edits DESC;

-- What Grep patterns were searched?
SELECT
    json_extract(tool_input, '$.pattern') AS pattern,
    json_extract(tool_input, '$.path') AS search_path,
    COUNT(*) AS uses
FROM hook_events
WHERE tool_name = 'Grep' AND hook_event_name = 'PostToolUse'
GROUP BY pattern, search_path
ORDER BY uses DESC;

-- Agent subagent types spawned
SELECT
    json_extract(tool_input, '$.subagent_type') AS agent_type,
    json_extract(tool_input, '$.description') AS desc,
    COUNT(*) AS uses
FROM hook_events
WHERE tool_name = 'Agent' AND hook_event_name = 'PostToolUse'
GROUP BY agent_type
ORDER BY uses DESC;

-- Files written with content size
SELECT
    json_extract(tool_input, '$.file_path') AS file,
    length(json_extract(tool_input, '$.content')) AS bytes_written
FROM hook_events
WHERE tool_name = 'Write' AND hook_event_name = 'PostToolUse'
ORDER BY bytes_written DESC;
```

### Session analysis

```sql
-- Session duration and activity
SELECT
    session_id,
    MIN(timestamp) AS started,
    MAX(timestamp) AS ended,
    ROUND((julianday(MAX(timestamp)) - julianday(MIN(timestamp))) * 86400) AS secs,
    COUNT(*) AS events,
    COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END) AS tool_uses,
    COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END) AS failures
FROM hook_events
GROUP BY session_id
ORDER BY started DESC;

-- What tools did a specific session use?
SELECT tool_name, COUNT(*) AS uses
FROM hook_events
WHERE session_id = 'YOUR-SESSION-ID'
  AND hook_event_name = 'PostToolUse'
GROUP BY tool_name
ORDER BY uses DESC;

-- Timeline of a session
SELECT
    id, timestamp, hook_event_name, tool_name,
    json_extract(tool_input, '$.command') AS bash_cmd,
    json_extract(tool_input, '$.file_path') AS file_path,
    json_extract(tool_input, '$.pattern') AS grep_pattern
FROM hook_events
WHERE session_id = 'YOUR-SESSION-ID'
ORDER BY id;
```

### Failure analysis

```sql
-- All tool failures
SELECT
    timestamp, tool_name,
    json_extract(tool_input, '$.command') AS cmd,
    json_extract(tool_input, '$.file_path') AS file
FROM hook_events
WHERE hook_event_name = 'PostToolUseFailure'
ORDER BY timestamp DESC;

-- Failure rate by tool
SELECT
    tool_name,
    COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END) AS ok,
    COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END) AS fail,
    ROUND(100.0 * COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END)
        / COUNT(*), 1) AS fail_pct
FROM hook_events
WHERE hook_event_name IN ('PostToolUse', 'PostToolUseFailure')
GROUP BY tool_name
ORDER BY fail_pct DESC;
```

### Token and cost analysis

```sql
-- Cost per session
SELECT * FROM v_session_token_summary ORDER BY final_cost_usd DESC;

-- Token growth over time in a session
SELECT * FROM v_token_timeline WHERE session_id = 'YOUR-SESSION-ID';

-- Which sessions hit high context usage?
SELECT session_id, peak_context_pct, final_total_tokens, final_cost_usd
FROM v_session_token_summary
WHERE peak_context_pct > 50
ORDER BY peak_context_pct DESC;

-- Average cost per tool use
SELECT
    ts.session_id,
    MAX(ts.total_cost_usd) AS cost,
    COUNT(DISTINCT he.id) AS tool_uses,
    ROUND(MAX(ts.total_cost_usd) / NULLIF(COUNT(DISTINCT he.id), 0), 4) AS cost_per_tool
FROM token_snapshots ts
LEFT JOIN hook_events he
    ON he.session_id = ts.session_id
    AND he.hook_event_name = 'PostToolUse'
GROUP BY ts.session_id
ORDER BY cost_per_tool DESC;
```

### Cross-correlating hook events with token snapshots

Hook events and token snapshots share `session_id` and overlapping timestamps.
Because the status line fires slightly after hook events (by milliseconds), use
`COALESCE` with before/after fallback:

```sql
-- Context window state at each tool use
SELECT
    he.id,
    he.timestamp,
    he.tool_name,
    COALESCE(
        (SELECT ts.used_percentage FROM token_snapshots ts
         WHERE ts.session_id = he.session_id AND ts.timestamp <= he.timestamp
         ORDER BY ts.id DESC LIMIT 1),
        (SELECT ts.used_percentage FROM token_snapshots ts
         WHERE ts.session_id = he.session_id AND ts.timestamp > he.timestamp
         ORDER BY ts.id ASC LIMIT 1)
    ) AS ctx_pct,
    COALESCE(
        (SELECT ts.total_input_tokens + ts.total_output_tokens FROM token_snapshots ts
         WHERE ts.session_id = he.session_id AND ts.timestamp <= he.timestamp
         ORDER BY ts.id DESC LIMIT 1),
        (SELECT ts.total_input_tokens + ts.total_output_tokens FROM token_snapshots ts
         WHERE ts.session_id = he.session_id AND ts.timestamp > he.timestamp
         ORDER BY ts.id ASC LIMIT 1)
    ) AS total_tokens
FROM hook_events he
WHERE he.hook_event_name = 'PostToolUse'
ORDER BY he.id DESC
LIMIT 20;

-- Average context size when each tool is used
SELECT
    he.tool_name,
    COUNT(*) AS uses,
    ROUND(AVG(COALESCE(
        (SELECT ts.total_input_tokens + ts.total_output_tokens FROM token_snapshots ts
         WHERE ts.session_id = he.session_id AND ts.timestamp <= he.timestamp
         ORDER BY ts.id DESC LIMIT 1),
        (SELECT ts.total_input_tokens + ts.total_output_tokens FROM token_snapshots ts
         WHERE ts.session_id = he.session_id AND ts.timestamp > he.timestamp
         ORDER BY ts.id ASC LIMIT 1)
    ))) AS avg_ctx_tokens
FROM hook_events he
WHERE he.hook_event_name = 'PostToolUse' AND he.tool_name IS NOT NULL
GROUP BY he.tool_name
ORDER BY avg_ctx_tokens DESC;
```

---

## Querying Transcripts

Transcripts are `.jsonl` files in `~/.claude/projects/<project-slug>/`. Each
line is a JSON object. They are NOT in the SQLite database — you query them
with `jq`, `python`, or the web UI.

### Transcript location

```bash
# List all transcripts
find ~/.claude/projects -name '*.jsonl' | head -20

# Find transcripts for a project
ls ~/.claude/projects/-home-manuel-code-wesen-2026-03-17--smalltalk/*.jsonl

# The transcript_path column in hook_events points to the right file
sqlite3 ~/.claude/hook-events.db \
  "SELECT DISTINCT transcript_path FROM hook_events WHERE session_id = 'YOUR-SESSION-ID'"
```

### Transcript entry types

Each `.jsonl` line has a `type` field:

| Type | What it contains |
|------|-----------------|
| `user` | User prompt (`message.content` is the text, or a list with `tool_result` blocks) |
| `assistant` | Claude's response (`message.content` is a list of `text`, `thinking`, `tool_use` blocks; `message.usage` has token counts) |
| `system` | Metadata (subtypes: `turn_duration`, `stop_hook_summary`) |
| `progress` | Streaming progress (subagent messages, etc.) |
| `file-history-snapshot` | File backup snapshots |
| `queue-operation` | Internal queue ops |

### Querying with jq

```bash
TRANSCRIPT=~/.claude/projects/-home-manuel-code-wesen-2026-03-17--smalltalk/744a92c4.jsonl

# Count entries by type
jq -r '.type' "$TRANSCRIPT" | sort | uniq -c | sort -rn

# Extract all user prompts
jq -r 'select(.type == "user") | .message.content // empty' "$TRANSCRIPT" \
  | head -50

# Extract token usage per assistant turn
jq -r 'select(.type == "assistant") |
  "\(.timestamp)\t\(.message.usage.input_tokens)\t\(.message.usage.output_tokens)\t\(.message.usage.cache_creation_input_tokens // 0)\t\(.message.usage.cache_read_input_tokens // 0)"' \
  "$TRANSCRIPT" | column -t -s $'\t'

# List all tool_use calls
jq -r 'select(.type == "assistant") |
  .message.content[]? |
  select(.type == "tool_use") |
  "\(.name)\t\(.id)"' \
  "$TRANSCRIPT" | sort | uniq -c | sort -rn

# Get tool_use input for a specific tool
jq -r 'select(.type == "assistant") |
  .message.content[]? |
  select(.type == "tool_use" and .name == "Bash") |
  .input.command' \
  "$TRANSCRIPT"

# Extract all tool_results
jq -r 'select(.type == "user") |
  .message.content[]? |
  select(.type == "tool_result") |
  "\(.tool_use_id)\t\(.content | length) chars"' \
  "$TRANSCRIPT" 2>/dev/null

# Total token usage for a transcript
jq -s '[.[] | select(.type == "assistant") | .message.usage] |
  { total_input: (map(.input_tokens) | add),
    total_output: (map(.output_tokens) | add),
    total_cache_create: (map(.cache_creation_input_tokens // 0) | add),
    total_cache_read: (map(.cache_read_input_tokens // 0) | add) }' \
  "$TRANSCRIPT"

# Extract thinking blocks
jq -r 'select(.type == "assistant") |
  .message.content[]? |
  select(.type == "thinking" and .thinking != "") |
  .thinking[:200]' \
  "$TRANSCRIPT"

# Session duration from system entries
jq -r 'select(.type == "system" and .subtype == "turn_duration") |
  "\(.timestamp)\t\(.durationMs)ms"' \
  "$TRANSCRIPT"
```

### Querying with Python

```python
import json

def parse_transcript(path):
    """Parse a transcript and return structured data."""
    with open(path) as f:
        entries = [json.loads(line) for line in f]

    for entry in entries:
        if entry.get("type") == "assistant":
            msg = entry["message"]
            usage = msg.get("usage", {})
            print(f"Turn at {entry.get('timestamp', '?')[:19]}")
            print(f"  Tokens: in={usage.get('input_tokens',0)} out={usage.get('output_tokens',0)}")
            print(f"  Cache:  create={usage.get('cache_creation_input_tokens',0)} read={usage.get('cache_read_input_tokens',0)}")

            for block in msg.get("content", []):
                if block.get("type") == "text":
                    print(f"  Text: {block['text'][:100]}...")
                elif block.get("type") == "tool_use":
                    print(f"  Tool: {block['name']} (id={block['id'][:20]}...)")
                elif block.get("type") == "thinking":
                    print(f"  Thinking: {len(block.get('thinking',''))} chars")

parse_transcript("~/.claude/projects/YOUR-PROJECT/SESSION-ID.jsonl")
```

### Cross-linking transcripts to hook events

The `tool_use_id` field is the join key:

- In transcripts: `assistant.message.content[].id` (where `type == "tool_use"`)
- In hook_events: `tool_use_id` column

```bash
# Find the hook event for a specific tool_use from a transcript
TOOL_USE_ID="toolu_01Di3Jpa3nSxqiVzCkkJtXe2"

sqlite3 ~/.claude/hook-events.db ".headers on" ".mode column" \
  "SELECT id, hook_event_name, tool_name, timestamp
   FROM hook_events WHERE tool_use_id = '$TOOL_USE_ID'"

# Find all tool_use IDs in a transcript and check which have hook events
jq -r 'select(.type == "assistant") |
  .message.content[]? |
  select(.type == "tool_use") |
  .id' YOUR_TRANSCRIPT.jsonl | while read tuid; do
    count=$(sqlite3 ~/.claude/hook-events.db \
      "SELECT COUNT(*) FROM hook_events WHERE tool_use_id = '$tuid'")
    echo "$tuid -> $count hook events"
done
```

### Cross-linking transcripts to token snapshots

Both share `session_id`. The transcript filename IS the session ID:

```bash
# Get token snapshots for a transcript's session
SESSION_ID="744a92c4-cbf7-4a23-a1f5-25bf1d6413e2"

sqlite3 ~/.claude/hook-events.db ".headers on" ".mode column" \
  "SELECT timestamp, total_input_tokens, total_output_tokens,
          total_cost_usd, used_percentage
   FROM token_snapshots
   WHERE session_id = '$SESSION_ID'
   ORDER BY id"
```

---

## Web UI Reference

Start: `python3 .claude/hooks/hook-events-server.py [--port 8642] [--db PATH]`

### Pages

| Route | Tab | What it shows |
|-------|-----|--------------|
| `/` | Dashboard | Stats cards, event/tool/daily bar charts, recent events |
| `/events` | Events | Filterable paginated event list with context % column |
| `/events/detail?id=N` | — | Single event: metadata, context window box, YAML-highlighted tool input/response with copy buttons, prev/next/transcript links |
| `/sessions` | Sessions | Session list with duration, tool counts, failure counts |
| `/sessions/detail?id=ID` | — | Session stats, tool breakdown chart, event timeline |
| `/tools` | Tools | Tool usage bar chart, success/failure rates |
| `/files` | Files | Most-touched files by Read/Edit/Write |
| `/commands` | Commands | Bash command log with frequency chart |
| `/searches` | Searches | Grep/Glob patterns and frequency |
| `/tokens` | Tokens | Token/cost stats, session breakdown, token-tool correlation, context growth per tool, payload weight, recent snapshots |
| `/transcripts` | Transcripts | All transcripts across projects, with hook/snapshot cross-references |
| `/transcripts/detail?path=P` | — | Full conversation view: expandable user/assistant/system entries, thinking blocks, tool use with YAML input + result, cross-links to hook events, per-turn token usage |
| `/sql` | SQL | Interactive SQL console (SELECT/WITH only) |

### Filter parameters

| Page | Params |
|------|--------|
| `/events` | `?event=PostToolUse&tool=Bash&session=ID&page=N` |
| `/transcripts` | `?project=slug&page=N` |
| `/commands` | `?page=N` |
| `/sql` | `?q=SELECT+...` |

### Conversation view features (transcript detail)

- **User messages**: collapsible, shows full prompt text
- **Assistant messages**: open by default, shows:
  - Token usage bar (input, output, cache created, cache read, cumulative)
  - Text blocks as monospace content
  - Thinking blocks: collapsible, italic gray
  - Tool use blocks: tool name badge, cross-links to hook events `[Pre #id | Post #id]`, collapsible YAML-highlighted input, collapsible result
- **System entries**: dashed border, shows subtype and duration

---

## Recipes

### "What did I work on today?"

```sql
SELECT
    session_id,
    MIN(timestamp) AS started,
    COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END) AS tools,
    GROUP_CONCAT(DISTINCT tool_name) AS tools_used
FROM hook_events
WHERE date(timestamp) = date('now')
GROUP BY session_id
ORDER BY started;
```

### "How much did today cost?"

```sql
SELECT
    session_id,
    MAX(total_cost_usd) AS cost,
    MAX(total_input_tokens) + MAX(total_output_tokens) AS tokens
FROM token_snapshots
WHERE date(timestamp) = date('now')
GROUP BY session_id;
```

### "Which files change most across all sessions?"

```sql
SELECT * FROM v_file_touch_frequency LIMIT 20;
```

### "What commands do I run most?"

```sql
SELECT
    json_extract(tool_input, '$.command') AS cmd,
    COUNT(*) AS runs
FROM hook_events
WHERE tool_name = 'Bash' AND hook_event_name = 'PostToolUse'
GROUP BY cmd
ORDER BY runs DESC
LIMIT 20;
```

### "Show me sessions that hit >50% context"

```sql
SELECT session_id, model_id, peak_context_pct, final_total_tokens, final_cost_usd
FROM v_session_token_summary
WHERE peak_context_pct > 50
ORDER BY peak_context_pct DESC;
```

### "What's the average cost per tool call by tool type?"

```sql
SELECT
    he.tool_name,
    COUNT(*) AS uses,
    ROUND(SUM(ts_cost) / COUNT(*), 4) AS avg_cost_per_call
FROM hook_events he
JOIN (
    SELECT session_id, MAX(total_cost_usd) / NULLIF(COUNT(*), 0) AS ts_cost
    FROM token_snapshots
    GROUP BY session_id
) ts ON ts.session_id = he.session_id
WHERE he.hook_event_name = 'PostToolUse'
GROUP BY he.tool_name
ORDER BY avg_cost_per_call DESC;
```

### "Find all prompts mentioning a topic across all transcripts"

```bash
# Search all transcripts for a keyword in user prompts
grep -l '"content"' ~/.claude/projects/*/*.jsonl | while read f; do
    matches=$(jq -r 'select(.type == "user") | .message.content // empty' "$f" 2>/dev/null \
      | grep -i "sqlite" | head -1)
    if [ -n "$matches" ]; then
        echo "$(basename $f .jsonl): $matches"
    fi
done
```

### "Token usage per transcript (no database needed)"

```bash
for f in ~/.claude/projects/-home-manuel-code-wesen-2026-03-17--smalltalk/*.jsonl; do
    sid=$(basename "$f" .jsonl)
    stats=$(jq -s '[.[] | select(.type=="assistant") | .message.usage] |
      { in: (map(.input_tokens) | add // 0),
        out: (map(.output_tokens) | add // 0) }' "$f" 2>/dev/null)
    echo "$sid: $stats"
done
```

### "Reconstruct a full session timeline from all three sources"

```bash
SESSION="YOUR-SESSION-ID"
DB=~/.claude/hook-events.db
TRANSCRIPT=$(sqlite3 $DB "SELECT DISTINCT transcript_path FROM hook_events WHERE session_id='$SESSION' LIMIT 1")

echo "=== Hook Events ==="
sqlite3 $DB ".headers on" ".mode column" \
  "SELECT id, timestamp, hook_event_name, tool_name FROM hook_events
   WHERE session_id = '$SESSION' ORDER BY id"

echo ""
echo "=== Token Snapshots ==="
sqlite3 $DB ".headers on" ".mode column" \
  "SELECT timestamp, total_input_tokens, total_output_tokens, total_cost_usd, used_percentage
   FROM token_snapshots WHERE session_id = '$SESSION' ORDER BY id"

echo ""
echo "=== Transcript Summary ==="
jq -r 'select(.type == "assistant") |
  "\(.timestamp)\tin:\(.message.usage.input_tokens)\tout:\(.message.usage.output_tokens)\t\([.message.content[]? | select(.type=="tool_use") | .name] | join(","))"' \
  "$TRANSCRIPT" 2>/dev/null | column -t -s $'\t'
```

---

## Maintenance

```bash
# Database size
ls -lh ~/.claude/hook-events.db

# Row counts
sqlite3 ~/.claude/hook-events.db "
  SELECT 'hook_events' AS tbl, COUNT(*) FROM hook_events
  UNION ALL
  SELECT 'token_snapshots', COUNT(*) FROM token_snapshots"

# Purge events older than 30 days
sqlite3 ~/.claude/hook-events.db "
  DELETE FROM hook_events WHERE timestamp < datetime('now', '-30 days');
  DELETE FROM token_snapshots WHERE timestamp < datetime('now', '-30 days')"

# Reclaim space
sqlite3 ~/.claude/hook-events.db "VACUUM"

# Export everything as CSV
sqlite3 ~/.claude/hook-events.db ".headers on" ".mode csv" \
  "SELECT * FROM hook_events" > hook_events.csv
sqlite3 ~/.claude/hook-events.db ".headers on" ".mode csv" \
  "SELECT * FROM token_snapshots" > token_snapshots.csv
```

## Files

| File | Purpose |
|------|---------|
| `.claude/hooks/log-to-sqlite.py` | Hook script — logs all events to SQLite |
| `.claude/hooks/statusline.py` | Status line — displays + logs token counts |
| `.claude/hooks/hook-events-server.py` | Web UI server |
| `.claude/hooks/diary-reminder.sh` | Debounced diary reminder (90s) |
| `.claude/hooks/HOOK-EVENTS-DB.md` | Schema reference (original doc) |
| `.claude/hooks/HOOK-ANALYTICS.md` | This file |
| `.claude/settings.local.json` | Hook + status line registration |
| `~/.claude/hook-events.db` | SQLite database |
| `~/.claude/projects/*//*.jsonl` | Transcript files |
