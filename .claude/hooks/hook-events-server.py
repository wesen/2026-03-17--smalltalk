#!/usr/bin/env python3
"""Web server for browsing Claude Code hook event logs.

Usage: python3 hook-events-server.py [--port 8642] [--db ~/.claude/hook-events.db]

Classic Macintosh retro UI.
"""

import argparse
import glob
import html
import json
import os
import re
import sqlite3
import textwrap
from datetime import datetime
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs

DEFAULT_DB = os.environ.get(
    "CLAUDE_HOOK_EVENTS_DB",
    os.path.expanduser("~/.claude/hook-events.db"),
)
DEFAULT_PORT = 8642
PAGE_SIZE = 50

# ── CSS: Classic Macintosh 1984 ──────────────────────────────────────────────

MAC_CSS = """\
@font-face {
  font-family: 'Chicago';
  src: url('https://cdn.jsdelivr.net/gh/pbitutsky/Chicago-Font@master/ChicagoFLF.woff2') format('woff2');
}

:root {
  --bg: #fff;
  --fg: #000;
  --border: #000;
  --highlight: #000;
  --highlight-text: #fff;
  --window-title: #fff;
  --scrollbar: #ccc;
  --stripe1: #fff;
  --stripe2: #f0f0f0;
}

* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  background: repeating-conic-gradient(#c0c0c0 0% 25%, #a0a0a0 0% 50%) 0 0 / 4px 4px;
  font-family: 'Chicago', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  color: var(--fg);
  padding: 28px 12px 12px;
  min-height: 100vh;
}

/* ── Menu Bar ── */
.menu-bar {
  position: fixed;
  top: 0; left: 0; right: 0;
  height: 24px;
  background: var(--bg);
  border-bottom: 2px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 8px;
  z-index: 1000;
  gap: 16px;
}
.menu-bar .apple {
  font-size: 16px;
  font-weight: bold;
}
.menu-bar a {
  color: var(--fg);
  text-decoration: none;
  padding: 2px 8px;
}
.menu-bar a:hover {
  background: var(--highlight);
  color: var(--highlight-text);
}
.menu-bar .spacer { flex: 1; }
.menu-bar .clock {
  font-size: 11px;
}

/* ── Window Chrome ── */
.window {
  background: var(--bg);
  border: 2px solid var(--border);
  border-radius: 0;
  margin: 8px auto;
  max-width: 1100px;
  box-shadow: 2px 2px 0 var(--border);
}

.title-bar {
  background: var(--bg);
  border-bottom: 2px solid var(--border);
  padding: 3px 6px;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 22px;
  /* classic horizontal stripes */
  background-image: repeating-linear-gradient(
    0deg,
    transparent, transparent 1px,
    var(--border) 1px, var(--border) 2px,
    transparent 2px, transparent 3px
  );
}
.title-bar .close-box {
  width: 12px; height: 12px;
  border: 1px solid var(--border);
  background: var(--bg);
  flex-shrink: 0;
}
.title-bar .title {
  background: var(--bg);
  padding: 0 8px;
  font-weight: bold;
  font-size: 12px;
  white-space: nowrap;
}

.window-body {
  padding: 12px;
  overflow-x: auto;
}

/* ── Stats Grid ── */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 8px;
  margin-bottom: 12px;
}
.stat-box {
  border: 2px solid var(--border);
  padding: 8px;
  text-align: center;
}
.stat-box .stat-value {
  font-size: 28px;
  font-weight: bold;
  line-height: 1.2;
}
.stat-box .stat-label {
  font-size: 11px;
  margin-top: 2px;
}

/* ── Tables ── */
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
}
th {
  background: var(--highlight);
  color: var(--highlight-text);
  text-align: left;
  padding: 3px 6px;
  white-space: nowrap;
  border: 1px solid var(--border);
  position: sticky;
  top: 0;
}
td {
  padding: 3px 6px;
  border: 1px solid #999;
  vertical-align: top;
  max-width: 350px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
tr:nth-child(even) { background: var(--stripe2); }
tr:hover { background: #d0d0d0; }

/* ── Badges / Tags ── */
.badge {
  display: inline-block;
  padding: 1px 5px;
  border: 1px solid var(--border);
  font-size: 10px;
  margin: 1px 2px;
  background: var(--bg);
}
.badge-event { border-color: #666; }
.badge-tool {
  background: var(--highlight);
  color: var(--highlight-text);
}
.badge-session { border-style: dashed; }
.badge-fail {
  background: var(--fg);
  color: var(--bg);
  border-style: double;
  border-width: 3px;
}

/* ── Detail View ── */
.detail-grid {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 2px 12px;
  margin-bottom: 12px;
}
.detail-grid dt {
  font-weight: bold;
  text-align: right;
  padding: 2px 0;
}
.detail-grid dd {
  padding: 2px 0;
  word-break: break-all;
}
pre.json-block, pre.yaml-block {
  background: var(--bg);
  border: 2px inset #999;
  padding: 8px;
  overflow-x: auto;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 11px;
  max-height: 400px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

/* ── Pagination ── */
.pagination {
  display: flex;
  justify-content: center;
  gap: 4px;
  margin-top: 12px;
  flex-wrap: wrap;
}
.pagination a, .pagination span {
  display: inline-block;
  padding: 2px 8px;
  border: 2px outset #ccc;
  background: var(--bg);
  color: var(--fg);
  text-decoration: none;
  font-size: 11px;
}
.pagination a:active {
  border-style: inset;
}
.pagination .current {
  background: var(--highlight);
  color: var(--highlight-text);
  border-style: solid;
}

/* ── Filters ── */
.filter-bar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 10px;
  align-items: center;
}
.filter-bar label {
  font-weight: bold;
  font-size: 11px;
}
.filter-bar select, .filter-bar input[type=text] {
  font-family: 'Chicago', 'Monaco', monospace;
  font-size: 11px;
  border: 2px inset #999;
  padding: 2px 4px;
  background: var(--bg);
}
.filter-bar button {
  font-family: 'Chicago', 'Monaco', monospace;
  font-size: 11px;
  border: 2px outset #ccc;
  padding: 2px 12px;
  background: var(--bg);
  cursor: pointer;
}
.filter-bar button:active { border-style: inset; }

/* ── Bar Chart ── */
.bar-chart { margin: 8px 0; }
.bar-row {
  display: flex;
  align-items: center;
  margin: 2px 0;
  gap: 4px;
}
.bar-label {
  width: 120px;
  text-align: right;
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
}
.bar-fill {
  height: 14px;
  background: var(--highlight);
  border: 1px solid var(--border);
  min-width: 2px;
}
.bar-value {
  font-size: 10px;
  width: 50px;
  flex-shrink: 0;
}

/* ── Links ── */
a { color: var(--fg); }
a:visited { color: #333; }

/* ── Scrollbar Mac-style ── */
::-webkit-scrollbar { width: 16px; }
::-webkit-scrollbar-track {
  background: repeating-conic-gradient(#c0c0c0 0% 25%, #a0a0a0 0% 50%) 0 0 / 4px 4px;
  border-left: 1px solid var(--border);
}
::-webkit-scrollbar-thumb {
  background: var(--bg);
  border: 1px solid var(--border);
}

/* ── YAML Syntax Highlighting ── */
.y-key { font-weight: bold; }
.y-str { color: #333; }
.y-num { font-weight: bold; font-style: italic; }
.y-bool { font-style: italic; text-decoration: underline; text-decoration-style: dotted; }
.y-null { font-style: italic; color: #888; }
.y-scalar { font-weight: bold; color: #555; }
.y-dash { font-weight: bold; }
.y-colon { color: #666; }

/* ── Code Blocks with Copy ── */
.code-block-wrap {
  margin: 6px 0 12px;
  border: 2px solid var(--border);
}
.code-block-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 6px;
  border-bottom: 1px solid var(--border);
  background: var(--highlight);
  color: var(--highlight-text);
  font-size: 11px;
  font-weight: bold;
}
.code-block-title { }
.copy-btn {
  font-family: 'Chicago', 'Monaco', monospace;
  font-size: 10px;
  border: 1px solid var(--highlight-text);
  background: transparent;
  color: var(--highlight-text);
  padding: 1px 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 3px;
}
.copy-btn:hover {
  background: var(--highlight-text);
  color: var(--highlight);
}
.copy-btn svg { flex-shrink: 0; }
.copy-btn.copied {
  background: var(--highlight-text);
  color: var(--highlight);
}
pre.code-block {
  background: var(--bg);
  padding: 8px;
  overflow-x: auto;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 11px;
  max-height: 400px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
  border: none;
}
.copy-source { display: none; }

/* ── Conversation View ── */
.conv-timeline { margin: 8px 0; }
.conv-entry {
  border: 2px solid var(--border);
  margin: 4px 0;
}
.conv-entry-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 6px;
  font-size: 11px;
  cursor: pointer;
  user-select: none;
}
.conv-entry-header:hover { background: #e0e0e0; }
.conv-entry-header .arrow { font-size: 10px; width: 12px; flex-shrink: 0; }
.conv-entry[open] > .conv-entry-header .arrow { transform: rotate(90deg); }
.conv-entry-body {
  padding: 6px 8px;
  border-top: 1px solid #999;
  font-size: 11px;
  max-height: 600px;
  overflow-y: auto;
}
.conv-user > .conv-entry-header { background: #e8e8e8; }
.conv-assistant > .conv-entry-header { background: #d0d0d0; }
.conv-system > .conv-entry-header { background: #f0f0f0; font-style: italic; }
.conv-text {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 11px;
  line-height: 1.4;
  margin: 4px 0;
}
.conv-tool-block {
  border: 1px solid #999;
  margin: 4px 0;
  background: #f8f8f8;
}
.conv-tool-header {
  padding: 2px 6px;
  font-size: 11px;
  font-weight: bold;
  display: flex;
  align-items: center;
  gap: 6px;
  border-bottom: 1px solid #ccc;
}
.conv-tool-body {
  padding: 4px 6px;
  font-family: 'Monaco', monospace;
  font-size: 10px;
  max-height: 300px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
.conv-usage {
  display: flex;
  gap: 8px;
  font-size: 10px;
  color: #666;
  padding: 2px 0;
  flex-wrap: wrap;
}
.conv-thinking {
  color: #888;
  font-style: italic;
  border-left: 2px solid #ccc;
  padding-left: 6px;
  margin: 4px 0;
}
"""

# ── HTML helpers ─────────────────────────────────────────────────────────────


def esc(s):
    if s is None:
        return ""
    return html.escape(str(s))


def page_shell(title, body, active_tab=""):
    now = datetime.now().strftime("%I:%M %p")
    tabs = [
        ("Dashboard", "/"),
        ("Events", "/events"),
        ("Sessions", "/sessions"),
        ("Tools", "/tools"),
        ("Files", "/files"),
        ("Commands", "/commands"),
        ("Searches", "/searches"),
        ("Tokens", "/tokens"),
        ("Transcripts", "/transcripts"),
        ("SQL", "/sql"),
    ]
    menu_items = []
    for label, href in tabs:
        cls = ' style="background:#000;color:#fff"' if active_tab == label else ""
        menu_items.append(f'<a href="{href}"{cls}>{label}</a>')

    return textwrap.dedent(f"""\
    <!DOCTYPE html>
    <html lang="en">
    <head>
      <meta charset="utf-8">
      <meta name="viewport" content="width=device-width, initial-scale=1">
      <title>{esc(title)} — Hook Events</title>
      <style>{MAC_CSS}</style>
    </head>
    <body>
      <div class="menu-bar">
        <span class="apple">&#63743;</span>
        {''.join(menu_items)}
        <span class="spacer"></span>
        <span class="clock">{now}</span>
      </div>

      <div class="window">
        <div class="title-bar">
          <div class="close-box"></div>
          <span class="title">{esc(title)}</span>
        </div>
        <div class="window-body">
          {body}
        </div>
      </div>
    <script>
    function copyBlock(btn) {{
      var wrap = btn.closest('.code-block-wrap');
      var src = wrap.querySelector('.copy-source');
      navigator.clipboard.writeText(src.value).then(function() {{
        var orig = btn.innerHTML;
        btn.textContent = 'Copied!';
        btn.classList.add('copied');
        setTimeout(function() {{ btn.innerHTML = orig; btn.classList.remove('copied'); }}, 1500);
      }});
    }}
    </script>
    </body>
    </html>
    """)


def bar_chart(rows, label_col=0, value_col=1, max_width=400):
    if not rows:
        return "<p>No data.</p>"
    max_val = max(r[value_col] for r in rows) or 1
    out = ['<div class="bar-chart">']
    for r in rows:
        label = esc(str(r[label_col]) if r[label_col] else "(none)")
        val = r[value_col]
        w = max(2, int(val / max_val * max_width))
        out.append(
            f'<div class="bar-row">'
            f'<span class="bar-label">{label}</span>'
            f'<span class="bar-fill" style="width:{w}px"></span>'
            f'<span class="bar-value">{val}</span>'
            f"</div>"
        )
    out.append("</div>")
    return "\n".join(out)


def pagination_html(page, total_pages, base_url, params=None):
    if total_pages <= 1:
        return ""
    params = params or {}
    parts = ['<div class="pagination">']
    for p in range(1, total_pages + 1):
        qs_parts = [f"{k}={v}" for k, v in params.items()]
        qs_parts.append(f"page={p}")
        qs = "&".join(qs_parts)
        if p == page:
            parts.append(f'<span class="current">{p}</span>')
        else:
            parts.append(f'<a href="{base_url}?{qs}">{p}</a>')
    parts.append("</div>")
    return "\n".join(parts)


def format_json(s):
    if not s:
        return ""
    try:
        obj = json.loads(s)
        return json.dumps(obj, indent=2)
    except (json.JSONDecodeError, TypeError):
        return str(s)


# ── YAML formatting & highlighting ──────────────────────────────────────────


def _yaml_val(obj, indent=0):
    """Recursively convert a Python object to YAML-formatted text."""
    sp = "  " * indent
    if obj is None:
        return "null"
    if isinstance(obj, bool):
        return "true" if obj else "false"
    if isinstance(obj, (int, float)):
        return str(obj)
    if isinstance(obj, str):
        if "\n" in obj and len(obj) > 60:
            inner = "  " * (indent + 1)
            return "|\n" + "\n".join(inner + l for l in obj.splitlines())
        needs_quote = (
            not obj
            or obj in ("true", "false", "null", "yes", "no", "on", "off")
            or any(c in obj for c in ':{}[],"\'&*?|->!%@`#')
            or (obj and obj[0] in " -")
        )
        if needs_quote:
            return '"' + obj.replace("\\", "\\\\").replace('"', '\\"') + '"'
        return obj
    if isinstance(obj, list):
        if not obj:
            return "[]"
        lines = []
        for item in obj:
            rendered = _yaml_val(item, indent + 1)
            if isinstance(item, (dict, list)) and item:
                first, *rest = rendered.splitlines()
                lines.append(sp + "- " + first.lstrip())
                for r in rest:
                    lines.append(sp + "  " + r.lstrip())
            else:
                lines.append(sp + "- " + rendered)
        return "\n".join(lines)
    if isinstance(obj, dict):
        if not obj:
            return "{}"
        lines = []
        for k, v in obj.items():
            if isinstance(v, (dict, list)) and v:
                lines.append(sp + str(k) + ":")
                lines.append(_yaml_val(v, indent + 1))
            else:
                lines.append(sp + str(k) + ": " + _yaml_val(v, indent + 1))
        return "\n".join(lines)
    return repr(obj)


def format_yaml(s):
    """Convert a JSON string to YAML text."""
    if not s:
        return ""
    try:
        obj = json.loads(s)
    except (json.JSONDecodeError, TypeError):
        return str(s)
    return _yaml_val(obj)


def _hl_yaml_val(v):
    """Highlight a single YAML value token."""
    if not v:
        return ""
    if v in ("null", "~"):
        return f'<span class="y-null">{esc(v)}</span>'
    if v in ("true", "false"):
        return f'<span class="y-bool">{esc(v)}</span>'
    if v in ("|", ">", "|-", ">-"):
        return f'<span class="y-scalar">{esc(v)}</span>'
    if v in ("[]", "{}"):
        return f'<span class="y-null">{esc(v)}</span>'
    if re.match(r"^-?\d+(\.\d+)?$", v):
        return f'<span class="y-num">{esc(v)}</span>'
    return f'<span class="y-str">{esc(v)}</span>'


def highlight_yaml(yaml_text):
    """Apply syntax highlighting to YAML text, returning safe HTML."""
    lines = yaml_text.splitlines()
    result = []
    for line in lines:
        stripped = line.lstrip()
        indent = esc(line[: len(line) - len(stripped)])

        # Handle "- " list prefix
        dash = ""
        if stripped.startswith("- "):
            dash = '<span class="y-dash">-</span> '
            stripped = stripped[2:]

        # Try key: value
        km = re.match(r"^([^:]+?):(?: (.+))?$", stripped)
        if km and not stripped.startswith('"') and not stripped.startswith("'"):
            key, val = km.group(1), km.group(2)
            key_html = (
                f'<span class="y-key">{esc(key)}</span>'
                f'<span class="y-colon">:</span>'
            )
            if val:
                key_html += " " + _hl_yaml_val(val)
            result.append(indent + dash + key_html)
        elif stripped:
            result.append(indent + dash + _hl_yaml_val(stripped))
        else:
            result.append(indent + dash)
    return "\n".join(result)


COPY_SVG = '<svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor"><path d="M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 010 1.5h-1.5a.25.25 0 00-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 00.25-.25v-1.5a.75.75 0 011.5 0v1.5A1.75 1.75 0 019.25 16h-7.5A1.75 1.75 0 010 14.25z"/><path d="M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0114.25 11h-7.5A1.75 1.75 0 015 9.25zm1.75-.25a.25.25 0 00-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 00.25-.25v-7.5a.25.25 0 00-.25-.25z"/></svg>'


def render_code_block(title, highlighted_html, raw_text):
    """Wrap highlighted content in a titled code block with a copy button."""
    return (
        f'<div class="code-block-wrap">'
        f'<div class="code-block-header">'
        f'<span class="code-block-title">{esc(title)}</span>'
        f'<button class="copy-btn" onclick="copyBlock(this)" title="Copy to clipboard">'
        f"{COPY_SVG} Copy</button>"
        f"</div>"
        f'<pre class="code-block">{highlighted_html}</pre>'
        f'<textarea class="copy-source">{esc(raw_text)}</textarea>'
        f"</div>"
    )


def truncate(s, n=80):
    if not s:
        return ""
    s = str(s)
    return s[:n] + "..." if len(s) > n else s


# ── Database ─────────────────────────────────────────────────────────────────


def get_db(db_path):
    conn = sqlite3.connect(db_path, timeout=5)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    return conn


def query(conn, sql, params=()):
    return conn.execute(sql, params).fetchall()


def query_one(conn, sql, params=()):
    row = conn.execute(sql, params).fetchone()
    return row


# ── Transcript helpers ───────────────────────────────────────────────────────

TRANSCRIPTS_DIR = os.path.expanduser("~/.claude/projects")


def list_transcripts():
    """Scan ~/.claude/projects for all .jsonl transcripts, return metadata."""
    results = []
    for path in glob.glob(os.path.join(TRANSCRIPTS_DIR, "*", "*.jsonl")):
        proj = os.path.basename(os.path.dirname(path))
        sid = os.path.splitext(os.path.basename(path))[0]
        try:
            stat = os.stat(path)
            size = stat.st_size
            mtime = stat.st_mtime
        except OSError:
            size = 0
            mtime = 0
        results.append({
            "path": path,
            "project": proj,
            "session_id": sid,
            "size": size,
            "mtime": mtime,
        })
    return sorted(results, key=lambda r: r["mtime"], reverse=True)


def parse_transcript(path, summary_only=False):
    """Parse a JSONL transcript file. If summary_only, skip heavy content."""
    entries = []
    total_input = 0
    total_output = 0
    total_cache_create = 0
    total_cache_read = 0
    user_msgs = 0
    asst_msgs = 0
    tool_uses = []
    first_ts = last_ts = None
    first_user_msg = None
    model = None

    try:
        with open(path) as f:
            for line in f:
                obj = json.loads(line)
                ts = obj.get("timestamp")
                if ts:
                    if not first_ts:
                        first_ts = ts
                    last_ts = ts
                etype = obj.get("type")

                if etype == "user":
                    content = obj.get("message", {}).get("content", "")
                    if isinstance(content, str) and content:
                        user_msgs += 1
                        if not first_user_msg:
                            first_user_msg = content

                elif etype == "assistant":
                    asst_msgs += 1
                    msg = obj.get("message", {})
                    usage = msg.get("usage", {})
                    if not model:
                        model = msg.get("model")
                    total_input += usage.get("input_tokens", 0)
                    total_output += usage.get("output_tokens", 0)
                    total_cache_create += usage.get("cache_creation_input_tokens", 0)
                    total_cache_read += usage.get("cache_read_input_tokens", 0)

                    if not summary_only:
                        for c in msg.get("content", []):
                            if isinstance(c, dict) and c.get("type") == "tool_use":
                                tool_uses.append({
                                    "name": c.get("name"),
                                    "id": c.get("id"),
                                    "input_preview": truncate(
                                        json.dumps(c.get("input", {})), 120
                                    ),
                                    "timestamp": ts,
                                })

                if not summary_only:
                    entries.append(obj)
    except (OSError, json.JSONDecodeError):
        pass

    return {
        "first_ts": first_ts,
        "last_ts": last_ts,
        "user_msgs": user_msgs,
        "asst_msgs": asst_msgs,
        "total_input": total_input,
        "total_output": total_output,
        "total_cache_create": total_cache_create,
        "total_cache_read": total_cache_read,
        "total_tokens": total_input + total_output,
        "model": model,
        "first_user_msg": first_user_msg,
        "tool_uses": tool_uses,
        "entries": entries,
    }


# ── Request Handler ──────────────────────────────────────────────────────────


class HookEventsHandler(BaseHTTPRequestHandler):
    db_path = DEFAULT_DB

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/") or "/"
        params = {k: v[0] for k, v in parse_qs(parsed.query).items()}

        routes = {
            "/": self.page_dashboard,
            "/events": self.page_events,
            "/events/detail": self.page_event_detail,
            "/sessions": self.page_sessions,
            "/sessions/detail": self.page_session_detail,
            "/tools": self.page_tools,
            "/files": self.page_files,
            "/commands": self.page_commands,
            "/searches": self.page_searches,
            "/tokens": self.page_tokens,
            "/transcripts": self.page_transcripts,
            "/transcripts/detail": self.page_transcript_detail,
            "/sql": self.page_sql,
        }

        handler = routes.get(path)
        if handler:
            try:
                conn = get_db(self.db_path)
                body = handler(conn, params)
                conn.close()
                self.respond(200, body)
            except Exception as e:
                self.respond(500, page_shell("Error", f"<pre>{esc(str(e))}</pre>"))
        else:
            self.respond(404, page_shell("Not Found", "<p>404 — File not found.</p>"))

    def respond(self, code, body):
        self.send_response(code)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        data = body.encode("utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, format, *args):
        pass  # quiet

    # ── Dashboard ────────────────────────────────────────────────────────

    def page_dashboard(self, conn, params):
        total = query_one(conn, "SELECT COUNT(*) c FROM hook_events")["c"]
        sessions = query_one(
            conn, "SELECT COUNT(DISTINCT session_id) c FROM hook_events"
        )["c"]
        tools = query_one(
            conn,
            "SELECT COUNT(*) c FROM hook_events WHERE hook_event_name='PostToolUse'",
        )["c"]
        failures = query_one(
            conn,
            "SELECT COUNT(*) c FROM hook_events WHERE hook_event_name='PostToolUseFailure'",
        )["c"]
        distinct_tools = query_one(
            conn, "SELECT COUNT(DISTINCT tool_name) c FROM hook_events WHERE tool_name IS NOT NULL"
        )["c"]

        stats = f"""
        <div class="stats-grid">
          <div class="stat-box"><div class="stat-value">{total}</div><div class="stat-label">Total Events</div></div>
          <div class="stat-box"><div class="stat-value">{sessions}</div><div class="stat-label">Sessions</div></div>
          <div class="stat-box"><div class="stat-value">{tools}</div><div class="stat-label">Tool Uses</div></div>
          <div class="stat-box"><div class="stat-value">{failures}</div><div class="stat-label">Failures</div></div>
          <div class="stat-box"><div class="stat-value">{distinct_tools}</div><div class="stat-label">Distinct Tools</div></div>
        </div>
        """

        # Event type breakdown
        event_rows = query(
            conn,
            "SELECT hook_event_name, COUNT(*) c FROM hook_events GROUP BY 1 ORDER BY 2 DESC",
        )
        event_chart = bar_chart(
            [(r["hook_event_name"], r["c"]) for r in event_rows]
        )

        # Tool usage breakdown
        tool_rows = query(
            conn,
            "SELECT tool_name, COUNT(*) c FROM hook_events WHERE tool_name IS NOT NULL AND hook_event_name='PostToolUse' GROUP BY 1 ORDER BY 2 DESC LIMIT 15",
        )
        tool_chart = bar_chart([(r["tool_name"], r["c"]) for r in tool_rows])

        # Daily activity
        daily_rows = query(
            conn,
            "SELECT date(timestamp) d, COUNT(*) c FROM hook_events GROUP BY 1 ORDER BY 1 DESC LIMIT 14",
        )
        daily_chart = bar_chart([(r["d"], r["c"]) for r in daily_rows])

        # Recent events
        recent = query(
            conn,
            "SELECT id, timestamp, hook_event_name, tool_name, session_id FROM hook_events ORDER BY id DESC LIMIT 10",
        )
        recent_rows = ""
        for r in recent:
            evt_cls = "badge-fail" if r["hook_event_name"] == "PostToolUseFailure" else "badge-event"
            tool_badge = f'<span class="badge badge-tool">{esc(r["tool_name"])}</span>' if r["tool_name"] else ""
            recent_rows += f"""<tr>
              <td><a href="/events/detail?id={r['id']}">{r['id']}</a></td>
              <td>{esc(r['timestamp'][:19])}</td>
              <td><span class="badge {evt_cls}">{esc(r['hook_event_name'])}</span></td>
              <td>{tool_badge}</td>
              <td><a href="/sessions/detail?id={esc(r['session_id'])}"><span class="badge badge-session">{esc(truncate(r['session_id'], 16))}</span></a></td>
            </tr>"""

        body = f"""
        {stats}
        <table><tr><th colspan="5">Recent Events</th></tr>
        <tr><th>#</th><th>Time</th><th>Event</th><th>Tool</th><th>Session</th></tr>
        {recent_rows}
        </table>
        <br>
        <table><tr><th>Events by Type</th><th></th></tr></table>
        {event_chart}
        <br>
        <table><tr><th>Tool Usage (PostToolUse)</th><th></th></tr></table>
        {tool_chart}
        <br>
        <table><tr><th>Daily Activity</th><th></th></tr></table>
        {daily_chart}
        """
        return page_shell("Hook Events Dashboard", body, "Dashboard")

    # ── Events List ──────────────────────────────────────────────────────

    def page_events(self, conn, params):
        page = int(params.get("page", 1))
        event_filter = params.get("event", "")
        tool_filter = params.get("tool", "")
        session_filter = params.get("session", "")

        where_clauses = []
        where_params = []
        if event_filter:
            where_clauses.append("he.hook_event_name = ?")
            where_params.append(event_filter)
        if tool_filter:
            where_clauses.append("he.tool_name = ?")
            where_params.append(tool_filter)
        if session_filter:
            where_clauses.append("he.session_id = ?")
            where_params.append(session_filter)

        where = ("WHERE " + " AND ".join(where_clauses)) if where_clauses else ""

        total = query_one(
            conn, f"SELECT COUNT(*) c FROM hook_events he {where}", where_params
        )["c"]
        total_pages = max(1, (total + PAGE_SIZE - 1) // PAGE_SIZE)
        page = max(1, min(page, total_pages))
        offset = (page - 1) * PAGE_SIZE

        rows = query(
            conn,
            f"""SELECT he.id, he.timestamp, he.hook_event_name, he.tool_name,
                he.session_id, he.tool_input,
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
                ) AS ctx_tokens
            FROM hook_events he {where} ORDER BY he.id DESC LIMIT ? OFFSET ?""",
            where_params + [PAGE_SIZE, offset],
        )

        # Get distinct values for filter dropdowns
        event_types = query(conn, "SELECT DISTINCT hook_event_name FROM hook_events ORDER BY 1")
        tool_names = query(conn, "SELECT DISTINCT tool_name FROM hook_events WHERE tool_name IS NOT NULL ORDER BY 1")
        session_ids = query(conn, "SELECT DISTINCT session_id FROM hook_events ORDER BY 1")

        event_opts = "".join(
            f'<option value="{esc(r["hook_event_name"])}" {"selected" if r["hook_event_name"] == event_filter else ""}>{esc(r["hook_event_name"])}</option>'
            for r in event_types
        )
        tool_opts = "".join(
            f'<option value="{esc(r["tool_name"])}" {"selected" if r["tool_name"] == tool_filter else ""}>{esc(r["tool_name"])}</option>'
            for r in tool_names
        )
        session_opts = "".join(
            f'<option value="{esc(r["session_id"])}" {"selected" if r["session_id"] == session_filter else ""}>{esc(truncate(r["session_id"], 20))}</option>'
            for r in session_ids
        )

        filters = f"""
        <form class="filter-bar" method="get" action="/events">
          <label>Event:</label>
          <select name="event"><option value="">All</option>{event_opts}</select>
          <label>Tool:</label>
          <select name="tool"><option value="">All</option>{tool_opts}</select>
          <label>Session:</label>
          <select name="session"><option value="">All</option>{session_opts}</select>
          <button type="submit">Filter</button>
        </form>
        """

        trs = ""
        for r in rows:
            evt_cls = "badge-fail" if r["hook_event_name"] == "PostToolUseFailure" else "badge-event"
            tool_badge = f'<span class="badge badge-tool">{esc(r["tool_name"])}</span>' if r["tool_name"] else ""
            input_preview = esc(truncate(r["tool_input"], 60)) if r["tool_input"] else ""
            ctx_pct = r["ctx_pct"]
            ctx_tok = r["ctx_tokens"]
            if ctx_pct is not None:
                ctx_str = f'{ctx_pct}%'
                if ctx_tok is not None:
                    if ctx_tok >= 1_000_000:
                        ctx_str += f' ({ctx_tok/1_000_000:.1f}M)'
                    elif ctx_tok >= 1000:
                        ctx_str += f' ({ctx_tok/1000:.0f}k)'
            else:
                ctx_str = ""
            trs += f"""<tr>
              <td><a href="/events/detail?id={r['id']}">{r['id']}</a></td>
              <td>{esc(r['timestamp'][:19])}</td>
              <td><span class="badge {evt_cls}">{esc(r['hook_event_name'])}</span></td>
              <td>{tool_badge}</td>
              <td><a href="/sessions/detail?id={esc(r['session_id'])}"><span class="badge badge-session">{esc(truncate(r['session_id'], 16))}</span></a></td>
              <td>{ctx_str}</td>
              <td>{input_preview}</td>
            </tr>"""

        filter_params = {}
        if event_filter:
            filter_params["event"] = event_filter
        if tool_filter:
            filter_params["tool"] = tool_filter
        if session_filter:
            filter_params["session"] = session_filter
        pag = pagination_html(page, total_pages, "/events", filter_params)

        body = f"""
        {filters}
        <p style="font-size:11px;margin-bottom:4px">{total} events — page {page}/{total_pages}</p>
        <table>
        <tr><th>#</th><th>Time</th><th>Event</th><th>Tool</th><th>Session</th><th>Context</th><th>Input</th></tr>
        {trs}
        </table>
        {pag}
        """
        return page_shell("All Events", body, "Events")

    # ── Event Detail ─────────────────────────────────────────────────────

    def page_event_detail(self, conn, params):
        eid = params.get("id", "")
        if not eid:
            return page_shell("Event", "<p>No event ID.</p>")
        row = query_one(conn, "SELECT * FROM hook_events WHERE id = ?", (eid,))
        if not row:
            return page_shell("Event", f"<p>Event {esc(eid)} not found.</p>")

        # Find nearest context window snapshot (before, or failing that, after)
        ctx_snap = query_one(
            conn,
            """SELECT total_input_tokens, total_output_tokens,
                total_input_tokens + total_output_tokens AS total_tokens,
                used_percentage, remaining_percentage, context_window_size,
                total_cost_usd,
                current_input_tokens, current_output_tokens,
                cache_creation_tokens, cache_read_tokens
            FROM token_snapshots
            WHERE session_id = ? AND timestamp <= ?
            ORDER BY id DESC LIMIT 1""",
            (row["session_id"], row["timestamp"]),
        )
        if not ctx_snap:
            ctx_snap = query_one(
                conn,
                """SELECT total_input_tokens, total_output_tokens,
                    total_input_tokens + total_output_tokens AS total_tokens,
                    used_percentage, remaining_percentage, context_window_size,
                    total_cost_usd,
                    current_input_tokens, current_output_tokens,
                    cache_creation_tokens, cache_read_tokens
                FROM token_snapshots
                WHERE session_id = ? AND timestamp > ?
                ORDER BY id ASC LIMIT 1""",
                (row["session_id"], row["timestamp"]),
            )

        fields = [
            ("ID", str(row["id"])),
            ("Timestamp", row["timestamp"]),
            ("Session", f'<a href="/sessions/detail?id={esc(row["session_id"])}">{esc(row["session_id"])}</a>'),
            ("Event", f'<span class="badge badge-event">{esc(row["hook_event_name"])}</span>'),
            ("CWD", row["cwd"]),
            ("Permission Mode", row["permission_mode"]),
            ("Transcript", row["transcript_path"]),
            ("Agent ID", row["agent_id"]),
            ("Agent Type", row["agent_type"]),
            ("Tool", f'<span class="badge badge-tool">{esc(row["tool_name"])}</span>' if row["tool_name"] else None),
            ("Tool Use ID", row["tool_use_id"]),
        ]

        dl = '<dl class="detail-grid">'
        for label, val in fields:
            if val:
                dl += f"<dt>{label}</dt><dd>{val}</dd>"
        dl += "</dl>"

        # Context window info box
        ctx_html = ""
        if ctx_snap:
            def _fmtk(n):
                if not n: return "0"
                if n < 1000: return str(n)
                if n < 1_000_000: return f"{n/1000:.1f}k"
                return f"{n/1_000_000:.1f}M"
            pct = ctx_snap["used_percentage"] or 0
            win = ctx_snap["context_window_size"] or 0
            total_tok = ctx_snap["total_tokens"] or 0
            cost = ctx_snap["total_cost_usd"] or 0
            bar_w = 200
            fill_w = max(2, int(pct / 100 * bar_w))
            cache_cr = ctx_snap["cache_creation_tokens"] or 0
            cache_rd = ctx_snap["cache_read_tokens"] or 0
            ctx_html = f"""
            <div style="border:2px solid #000;padding:8px;margin-bottom:12px">
              <b>Context Window at this event</b>
              <div style="margin-top:6px;display:flex;gap:16px;flex-wrap:wrap;font-size:11px">
                <span><b>{_fmtk(ctx_snap['total_input_tokens'])}</b> in</span>
                <span><b>{_fmtk(ctx_snap['total_output_tokens'])}</b> out</span>
                <span><b>{_fmtk(total_tok)}</b> total</span>
                <span>${cost:.3f}</span>
                <span>window: {_fmtk(win)}</span>
                <span>cache: {_fmtk(cache_cr)} created, {_fmtk(cache_rd)} read</span>
              </div>
              <div style="margin-top:4px;display:flex;align-items:center;gap:6px;font-size:11px">
                <div style="width:{bar_w}px;height:14px;border:1px solid #000;background:#fff">
                  <div style="width:{fill_w}px;height:100%;background:#000"></div>
                </div>
                <span><b>{pct}%</b> used</span>
              </div>
            </div>
            """

        sections = ""
        if row["tool_input"]:
            yaml_text = format_yaml(row["tool_input"])
            sections += render_code_block("Tool Input", highlight_yaml(yaml_text), yaml_text)
        if row["tool_response"]:
            yaml_text = format_yaml(row["tool_response"])
            sections += render_code_block("Tool Response", highlight_yaml(yaml_text), yaml_text)
        if row["last_assistant_message"]:
            msg = row["last_assistant_message"]
            sections += render_code_block(
                "Last Assistant Message", esc(truncate(msg, 2000)), msg
            )

        raw_yaml = format_yaml(row["raw_json"])
        sections += render_code_block("Raw Event (YAML)", highlight_yaml(raw_yaml), raw_yaml)

        # Prev / Next / Transcript navigation
        prev_row = query_one(conn, "SELECT id FROM hook_events WHERE id < ? ORDER BY id DESC LIMIT 1", (eid,))
        next_row = query_one(conn, "SELECT id FROM hook_events WHERE id > ? ORDER BY id ASC LIMIT 1", (eid,))
        nav = '<div style="margin-top:8px;display:flex;gap:8px;flex-wrap:wrap">'
        if prev_row:
            nav += f'<a href="/events/detail?id={prev_row["id"]}" style="border:2px outset #ccc;padding:2px 10px">&larr; Prev</a>'
        if next_row:
            nav += f'<a href="/events/detail?id={next_row["id"]}" style="border:2px outset #ccc;padding:2px 10px">Next &rarr;</a>'
        # Link to transcript if it exists
        tp = row["transcript_path"]
        if tp and os.path.exists(tp):
            nav += f'<a href="/transcripts/detail?path={esc(tp)}" style="border:2px outset #ccc;padding:2px 10px">View Transcript &rarr;</a>'
        nav += "</div>"

        body = f"{dl}{ctx_html}{sections}{nav}"
        return page_shell(f"Event #{esc(eid)}", body, "Events")

    # ── Sessions ─────────────────────────────────────────────────────────

    def page_sessions(self, conn, params):
        rows = query(
            conn,
            """SELECT
                session_id,
                MIN(timestamp) AS started,
                MAX(timestamp) AS ended,
                ROUND((julianday(MAX(timestamp)) - julianday(MIN(timestamp))) * 86400) AS duration_secs,
                COUNT(*) AS total_events,
                COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END) AS tool_uses,
                COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END) AS tool_failures,
                GROUP_CONCAT(DISTINCT tool_name) AS tools_used
            FROM hook_events
            GROUP BY session_id
            ORDER BY started DESC""",
        )

        trs = ""
        for r in rows:
            dur = int(r["duration_secs"] or 0)
            dur_str = f"{dur // 60}m {dur % 60}s" if dur >= 60 else f"{dur}s"
            fail_badge = f' <span class="badge badge-fail">{r["tool_failures"]} fail</span>' if r["tool_failures"] else ""
            tools = ", ".join(t for t in (r["tools_used"] or "").split(",") if t)
            trs += f"""<tr>
              <td><a href="/sessions/detail?id={esc(r['session_id'])}">{esc(truncate(r['session_id'], 20))}</a></td>
              <td>{esc(r['started'][:19])}</td>
              <td>{dur_str}</td>
              <td>{r['total_events']}</td>
              <td>{r['tool_uses']}{fail_badge}</td>
              <td>{esc(truncate(tools, 50))}</td>
            </tr>"""

        body = f"""
        <table>
        <tr><th>Session</th><th>Started</th><th>Duration</th><th>Events</th><th>Tools</th><th>Tools Used</th></tr>
        {trs}
        </table>
        """
        return page_shell("Sessions", body, "Sessions")

    # ── Session Detail ───────────────────────────────────────────────────

    def page_session_detail(self, conn, params):
        sid = params.get("id", "")
        if not sid:
            return page_shell("Session", "<p>No session ID.</p>")

        summary = query_one(
            conn,
            """SELECT
                COUNT(*) AS total_events,
                MIN(timestamp) AS started,
                MAX(timestamp) AS ended,
                ROUND((julianday(MAX(timestamp)) - julianday(MIN(timestamp))) * 86400) AS duration_secs,
                COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END) AS tool_uses,
                COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END) AS tool_failures
            FROM hook_events WHERE session_id = ?""",
            (sid,),
        )

        dur = int(summary["duration_secs"] or 0)
        dur_str = f"{dur // 60}m {dur % 60}s"

        stats = f"""
        <div class="stats-grid">
          <div class="stat-box"><div class="stat-value">{summary['total_events']}</div><div class="stat-label">Events</div></div>
          <div class="stat-box"><div class="stat-value">{summary['tool_uses']}</div><div class="stat-label">Tool Uses</div></div>
          <div class="stat-box"><div class="stat-value">{summary['tool_failures']}</div><div class="stat-label">Failures</div></div>
          <div class="stat-box"><div class="stat-value">{dur_str}</div><div class="stat-label">Duration</div></div>
        </div>
        """

        # Tool breakdown for this session
        tool_rows = query(
            conn,
            "SELECT tool_name, COUNT(*) c FROM hook_events WHERE session_id=? AND tool_name IS NOT NULL AND hook_event_name='PostToolUse' GROUP BY 1 ORDER BY 2 DESC",
            (sid,),
        )
        tool_chart = bar_chart([(r["tool_name"], r["c"]) for r in tool_rows], max_width=300)

        # Event timeline
        events = query(
            conn,
            "SELECT id, timestamp, hook_event_name, tool_name, tool_input FROM hook_events WHERE session_id=? ORDER BY id",
            (sid,),
        )
        trs = ""
        for r in events:
            evt_cls = "badge-fail" if r["hook_event_name"] == "PostToolUseFailure" else "badge-event"
            tool_badge = f'<span class="badge badge-tool">{esc(r["tool_name"])}</span>' if r["tool_name"] else ""
            trs += f"""<tr>
              <td><a href="/events/detail?id={r['id']}">{r['id']}</a></td>
              <td>{esc(r['timestamp'][:19])}</td>
              <td><span class="badge {evt_cls}">{esc(r['hook_event_name'])}</span></td>
              <td>{tool_badge}</td>
              <td>{esc(truncate(r['tool_input'], 60))}</td>
            </tr>"""

        body = f"""
        <dl class="detail-grid">
          <dt>Session ID</dt><dd>{esc(sid)}</dd>
          <dt>Started</dt><dd>{esc(summary['started'])}</dd>
          <dt>Ended</dt><dd>{esc(summary['ended'])}</dd>
        </dl>
        {stats}
        <table><tr><th>Tool Usage</th><th></th></tr></table>
        {tool_chart}
        <br>
        <table>
        <tr><th>#</th><th>Time</th><th>Event</th><th>Tool</th><th>Input</th></tr>
        {trs}
        </table>
        """
        return page_shell(f"Session {esc(truncate(sid, 24))}", body, "Sessions")

    # ── Tools ────────────────────────────────────────────────────────────

    def page_tools(self, conn, params):
        rows = query(
            conn,
            """SELECT
                tool_name,
                COUNT(CASE WHEN hook_event_name = 'PostToolUse' THEN 1 END) AS uses,
                COUNT(CASE WHEN hook_event_name = 'PostToolUseFailure' THEN 1 END) AS failures,
                COUNT(DISTINCT session_id) AS sessions
            FROM hook_events
            WHERE tool_name IS NOT NULL
            GROUP BY tool_name
            ORDER BY uses DESC""",
        )

        chart = bar_chart([(r["tool_name"], r["uses"]) for r in rows])

        trs = ""
        for r in rows:
            fail_pct = (
                f'{r["failures"] / (r["uses"] + r["failures"]) * 100:.1f}%'
                if (r["uses"] + r["failures"]) > 0
                else "0%"
            )
            trs += f"""<tr>
              <td><a href="/events?tool={esc(r['tool_name'])}">{esc(r['tool_name'])}</a></td>
              <td>{r['uses']}</td>
              <td>{r['failures']}</td>
              <td>{fail_pct}</td>
              <td>{r['sessions']}</td>
            </tr>"""

        body = f"""
        {chart}
        <br>
        <table>
        <tr><th>Tool</th><th>Uses</th><th>Failures</th><th>Fail %</th><th>Sessions</th></tr>
        {trs}
        </table>
        """
        return page_shell("Tool Usage", body, "Tools")

    # ── Files ────────────────────────────────────────────────────────────

    def page_files(self, conn, params):
        rows = query(
            conn,
            """SELECT
                json_extract(tool_input, '$.file_path') AS file_path,
                tool_name,
                COUNT(*) AS touches
            FROM hook_events
            WHERE tool_name IN ('Read', 'Edit', 'Write')
              AND hook_event_name = 'PostToolUse'
              AND json_extract(tool_input, '$.file_path') IS NOT NULL
            GROUP BY file_path, tool_name
            ORDER BY touches DESC
            LIMIT 100""",
        )

        trs = ""
        for r in rows:
            trs += f"""<tr>
              <td>{esc(r['file_path'])}</td>
              <td><span class="badge badge-tool">{esc(r['tool_name'])}</span></td>
              <td>{r['touches']}</td>
            </tr>"""

        # Also aggregate by file only
        file_rows = query(
            conn,
            """SELECT
                json_extract(tool_input, '$.file_path') AS file_path,
                COUNT(*) AS total
            FROM hook_events
            WHERE tool_name IN ('Read', 'Edit', 'Write')
              AND hook_event_name = 'PostToolUse'
              AND json_extract(tool_input, '$.file_path') IS NOT NULL
            GROUP BY file_path
            ORDER BY total DESC
            LIMIT 20""",
        )
        chart = bar_chart([(r["file_path"].split("/")[-1] if r["file_path"] else "?", r["total"]) for r in file_rows])

        body = f"""
        <table><tr><th>Most Touched Files (by basename)</th><th></th></tr></table>
        {chart}
        <br>
        <table>
        <tr><th>File Path</th><th>Tool</th><th>Touches</th></tr>
        {trs}
        </table>
        """
        return page_shell("File Touch Frequency", body, "Files")

    # ── Commands ─────────────────────────────────────────────────────────

    def page_commands(self, conn, params):
        page = int(params.get("page", 1))
        total = query_one(
            conn,
            "SELECT COUNT(*) c FROM hook_events WHERE tool_name='Bash' AND hook_event_name='PostToolUse'",
        )["c"]
        total_pages = max(1, (total + PAGE_SIZE - 1) // PAGE_SIZE)
        page = max(1, min(page, total_pages))
        offset = (page - 1) * PAGE_SIZE

        rows = query(
            conn,
            """SELECT
                id, session_id, timestamp,
                json_extract(tool_input, '$.command') AS command,
                json_extract(tool_input, '$.description') AS description
            FROM hook_events
            WHERE tool_name = 'Bash' AND hook_event_name = 'PostToolUse'
            ORDER BY id DESC
            LIMIT ? OFFSET ?""",
            (PAGE_SIZE, offset),
        )

        trs = ""
        for r in rows:
            trs += f"""<tr>
              <td><a href="/events/detail?id={r['id']}">{r['id']}</a></td>
              <td>{esc(r['timestamp'][:19])}</td>
              <td><a href="/sessions/detail?id={esc(r['session_id'])}">{esc(truncate(r['session_id'], 16))}</a></td>
              <td style="font-family:monospace;white-space:pre-wrap;max-width:500px">{esc(r['command'])}</td>
              <td>{esc(r['description'] or '')}</td>
            </tr>"""

        pag = pagination_html(page, total_pages, "/commands")

        # Top commands
        top = query(
            conn,
            """SELECT json_extract(tool_input, '$.command') AS cmd, COUNT(*) c
            FROM hook_events WHERE tool_name='Bash' AND hook_event_name='PostToolUse'
            GROUP BY cmd ORDER BY c DESC LIMIT 15""",
        )
        chart = bar_chart([(truncate(r["cmd"], 30), r["c"]) for r in top])

        body = f"""
        <table><tr><th>Most Frequent Commands</th><th></th></tr></table>
        {chart}
        <br>
        <p style="font-size:11px">{total} commands — page {page}/{total_pages}</p>
        <table>
        <tr><th>#</th><th>Time</th><th>Session</th><th>Command</th><th>Description</th></tr>
        {trs}
        </table>
        {pag}
        """
        return page_shell("Bash Commands", body, "Commands")

    # ── Searches ─────────────────────────────────────────────────────────

    def page_searches(self, conn, params):
        rows = query(
            conn,
            """SELECT
                tool_name,
                json_extract(tool_input, '$.pattern') AS pattern,
                json_extract(tool_input, '$.path') AS search_path,
                COUNT(*) AS uses
            FROM hook_events
            WHERE tool_name IN ('Grep', 'Glob')
              AND hook_event_name = 'PostToolUse'
            GROUP BY tool_name, pattern, search_path
            ORDER BY uses DESC
            LIMIT 100""",
        )

        trs = ""
        for r in rows:
            trs += f"""<tr>
              <td><span class="badge badge-tool">{esc(r['tool_name'])}</span></td>
              <td style="font-family:monospace">{esc(r['pattern'])}</td>
              <td>{esc(r['search_path'])}</td>
              <td>{r['uses']}</td>
            </tr>"""

        body = f"""
        <table>
        <tr><th>Tool</th><th>Pattern</th><th>Path</th><th>Uses</th></tr>
        {trs}
        </table>
        """
        return page_shell("Search Patterns", body, "Searches")

    # ── Tokens ───────────────────────────────────────────────────────

    def page_tokens(self, conn, params):
        # Check if token_snapshots table exists
        has_table = query_one(
            conn,
            "SELECT COUNT(*) c FROM sqlite_master WHERE type='table' AND name='token_snapshots'",
        )
        if not has_table or has_table["c"] == 0:
            return page_shell(
                "Tokens",
                "<p>No token data yet. The status line logger creates the "
                "<code>token_snapshots</code> table after the first assistant turn.</p>",
                "Tokens",
            )

        # ── Summary stats ────────────────────────────────────────────
        latest = query_one(
            conn,
            """SELECT *, total_input_tokens + total_output_tokens AS total_tokens
            FROM token_snapshots ORDER BY id DESC LIMIT 1""",
        )

        totals = query_one(
            conn,
            """SELECT
                COUNT(DISTINCT session_id) AS sessions,
                COUNT(*) AS snapshots,
                MAX(total_cost_usd) AS max_cost,
                SUM(total_cost_usd) AS sum_cost
            FROM (
                SELECT session_id, MAX(total_cost_usd) AS total_cost_usd
                FROM token_snapshots GROUP BY session_id
            )""",
        )

        all_time_tokens = query_one(
            conn,
            """SELECT
                SUM(final_in) AS total_in,
                SUM(final_out) AS total_out,
                SUM(final_in + final_out) AS total
            FROM (
                SELECT session_id,
                    MAX(total_input_tokens) AS final_in,
                    MAX(total_output_tokens) AS final_out
                FROM token_snapshots GROUP BY session_id
            )""",
        )

        def fmtk(n):
            if not n:
                return "0"
            if n < 1000:
                return str(n)
            if n < 1_000_000:
                return f"{n/1000:.1f}k"
            return f"{n/1_000_000:.1f}M"

        stats = ""
        if latest:
            stats = f"""
            <div class="stats-grid">
              <div class="stat-box"><div class="stat-value">{fmtk(all_time_tokens['total_in'])}</div><div class="stat-label">All-Time Input</div></div>
              <div class="stat-box"><div class="stat-value">{fmtk(all_time_tokens['total_out'])}</div><div class="stat-label">All-Time Output</div></div>
              <div class="stat-box"><div class="stat-value">{fmtk(all_time_tokens['total'])}</div><div class="stat-label">All-Time Total</div></div>
              <div class="stat-box"><div class="stat-value">${totals['sum_cost']:.2f}</div><div class="stat-label">All-Time Cost</div></div>
              <div class="stat-box"><div class="stat-value">{totals['sessions']}</div><div class="stat-label">Sessions</div></div>
            </div>
            """

        # ── Per-session summary ──────────────────────────────────────
        session_rows = query(
            conn,
            """SELECT
                session_id,
                model_id,
                MIN(timestamp) AS started,
                MAX(total_input_tokens) AS final_in,
                MAX(total_output_tokens) AS final_out,
                MAX(total_input_tokens) + MAX(total_output_tokens) AS final_total,
                MAX(total_cost_usd) AS cost,
                MAX(used_percentage) AS peak_ctx,
                MAX(context_window_size) AS ctx_window,
                MAX(total_lines_added) AS lines_add,
                MAX(total_lines_removed) AS lines_rm,
                ROUND(MAX(total_duration_ms) / 1000.0) AS dur_s
            FROM token_snapshots
            GROUP BY session_id
            ORDER BY started DESC""",
        )

        session_trs = ""
        for r in session_rows:
            dur = int(r["dur_s"] or 0)
            dur_str = f"{dur // 60}m{dur % 60:02d}s" if dur >= 60 else f"{dur}s"
            win = fmtk(r["ctx_window"]) if r["ctx_window"] else "?"
            session_trs += f"""<tr>
              <td><a href="/sessions/detail?id={esc(r['session_id'])}">{esc(truncate(r['session_id'], 16))}</a></td>
              <td>{esc(r['started'][:19])}</td>
              <td>{fmtk(r['final_in'])}</td>
              <td>{fmtk(r['final_out'])}</td>
              <td><b>{fmtk(r['final_total'])}</b></td>
              <td>${r['cost']:.3f}</td>
              <td>{r['peak_ctx']}% of {win}</td>
              <td>+{r['lines_add'] or 0}/-{r['lines_rm'] or 0}</td>
              <td>{dur_str}</td>
            </tr>"""

        # ── Token-to-tool correlation ────────────────────────────────
        # For each session, compute tokens-per-tool-use
        correlation_rows = query(
            conn,
            """SELECT
                ts.session_id,
                MAX(ts.total_input_tokens) + MAX(ts.total_output_tokens) AS total_tokens,
                MAX(ts.total_cost_usd) AS cost,
                COUNT(DISTINCT he.id) AS tool_uses,
                CASE WHEN COUNT(DISTINCT he.id) > 0
                    THEN ROUND((MAX(ts.total_input_tokens) + MAX(ts.total_output_tokens)) * 1.0 / COUNT(DISTINCT he.id))
                    ELSE 0
                END AS tokens_per_tool,
                CASE WHEN COUNT(DISTINCT he.id) > 0
                    THEN ROUND(MAX(ts.total_cost_usd) * 1.0 / COUNT(DISTINCT he.id), 4)
                    ELSE 0
                END AS cost_per_tool
            FROM token_snapshots ts
            LEFT JOIN hook_events he
                ON he.session_id = ts.session_id
                AND he.hook_event_name = 'PostToolUse'
            GROUP BY ts.session_id
            ORDER BY total_tokens DESC""",
        )

        corr_trs = ""
        for r in correlation_rows:
            corr_trs += f"""<tr>
              <td><a href="/sessions/detail?id={esc(r['session_id'])}">{esc(truncate(r['session_id'], 16))}</a></td>
              <td>{fmtk(r['total_tokens'])}</td>
              <td>{r['tool_uses']}</td>
              <td>{fmtk(r['tokens_per_tool'])}</td>
              <td>${r['cost']:.3f}</td>
              <td>${r['cost_per_tool']:.4f}</td>
            </tr>"""

        # ── Per-tool token cost (which tools consume the most context?) ──
        tool_cost_rows = query(
            conn,
            """SELECT
                he.tool_name,
                COUNT(*) AS uses,
                COUNT(DISTINCT he.session_id) AS sessions,
                ROUND(AVG(length(COALESCE(he.tool_input, '')) + length(COALESCE(he.tool_response, '')))) AS avg_payload_bytes
            FROM hook_events he
            WHERE he.hook_event_name = 'PostToolUse'
              AND he.tool_name IS NOT NULL
            GROUP BY he.tool_name
            ORDER BY avg_payload_bytes DESC""",
        )

        tool_cost_trs = ""
        for r in tool_cost_rows:
            tool_cost_trs += f"""<tr>
              <td><span class="badge badge-tool">{esc(r['tool_name'])}</span></td>
              <td>{r['uses']}</td>
              <td>{r['sessions']}</td>
              <td>{fmtk(r['avg_payload_bytes'])}</td>
            </tr>"""

        tool_chart = bar_chart(
            [(r["tool_name"], r["avg_payload_bytes"]) for r in tool_cost_rows],
            max_width=300,
        )

        # ── Context growth per tool type ─────────────────────────────
        # For each tool, show average context size at time of use and
        # the average payload size as a proxy for context cost
        ctx_growth_rows = query(
            conn,
            """SELECT
                he.tool_name,
                COUNT(*) AS uses,
                ROUND(AVG(COALESCE(
                    (SELECT ts.total_input_tokens + ts.total_output_tokens
                     FROM token_snapshots ts WHERE ts.session_id = he.session_id
                     AND ts.timestamp <= he.timestamp ORDER BY ts.id DESC LIMIT 1),
                    (SELECT ts.total_input_tokens + ts.total_output_tokens
                     FROM token_snapshots ts WHERE ts.session_id = he.session_id
                     AND ts.timestamp > he.timestamp ORDER BY ts.id ASC LIMIT 1)
                ))) AS avg_ctx_at_use,
                MAX(COALESCE(
                    (SELECT ts.total_input_tokens + ts.total_output_tokens
                     FROM token_snapshots ts WHERE ts.session_id = he.session_id
                     AND ts.timestamp <= he.timestamp ORDER BY ts.id DESC LIMIT 1),
                    (SELECT ts.total_input_tokens + ts.total_output_tokens
                     FROM token_snapshots ts WHERE ts.session_id = he.session_id
                     AND ts.timestamp > he.timestamp ORDER BY ts.id ASC LIMIT 1)
                )) AS max_ctx_at_use,
                ROUND(AVG(COALESCE(
                    (SELECT ts.used_percentage FROM token_snapshots ts
                     WHERE ts.session_id = he.session_id AND ts.timestamp <= he.timestamp
                     ORDER BY ts.id DESC LIMIT 1),
                    (SELECT ts.used_percentage FROM token_snapshots ts
                     WHERE ts.session_id = he.session_id AND ts.timestamp > he.timestamp
                     ORDER BY ts.id ASC LIMIT 1)
                ))) AS avg_ctx_pct
            FROM hook_events he
            WHERE he.hook_event_name = 'PostToolUse'
              AND he.tool_name IS NOT NULL
            GROUP BY he.tool_name
            HAVING avg_ctx_at_use IS NOT NULL
            ORDER BY avg_ctx_at_use DESC""",
        )

        ctx_growth_trs = ""
        ctx_growth_chart_data = []
        for r in ctx_growth_rows:
            avg_ctx = int(r["avg_ctx_at_use"] or 0)
            max_ctx = int(r["max_ctx_at_use"] or 0)
            avg_pct = int(r["avg_ctx_pct"] or 0)
            ctx_growth_trs += f"""<tr>
              <td><span class="badge badge-tool">{esc(r['tool_name'])}</span></td>
              <td>{r['uses']}</td>
              <td>{fmtk(avg_ctx)}</td>
              <td>{fmtk(max_ctx)}</td>
              <td>{avg_pct}%</td>
            </tr>"""
            if avg_ctx > 0:
                ctx_growth_chart_data.append((r["tool_name"], avg_ctx))

        ctx_growth_chart = bar_chart(ctx_growth_chart_data, max_width=300) if ctx_growth_chart_data else ""

        # ── Token timeline (last 30 snapshots) ──────────────────────
        timeline = query(
            conn,
            """SELECT timestamp, session_id,
                total_input_tokens, total_output_tokens,
                total_input_tokens + total_output_tokens AS total,
                total_cost_usd, used_percentage
            FROM token_snapshots
            ORDER BY id DESC LIMIT 30""",
        )

        timeline_trs = ""
        for r in timeline:
            timeline_trs += f"""<tr>
              <td>{esc(r['timestamp'][:19])}</td>
              <td><a href="/sessions/detail?id={esc(r['session_id'])}">{esc(truncate(r['session_id'], 12))}</a></td>
              <td>{fmtk(r['total_input_tokens'])}</td>
              <td>{fmtk(r['total_output_tokens'])}</td>
              <td><b>{fmtk(r['total'])}</b></td>
              <td>${r['total_cost_usd']:.3f}</td>
              <td>{r['used_percentage']}%</td>
            </tr>"""

        body = f"""
        {stats}

        <table>
        <tr><th colspan="9">Sessions — Token Usage</th></tr>
        <tr><th>Session</th><th>Started</th><th>In</th><th>Out</th><th>Total</th><th>Cost</th><th>Peak Ctx</th><th>Lines</th><th>Duration</th></tr>
        {session_trs}
        </table>
        <br>

        <table>
        <tr><th colspan="6">Token-Tool Correlation (per session)</th></tr>
        <tr><th>Session</th><th>Tokens</th><th>Tool Uses</th><th>Tokens/Tool</th><th>Cost</th><th>Cost/Tool</th></tr>
        {corr_trs}
        </table>
        <br>

        <table><tr><th>Avg Context Size at Tool Call (tokens)</th><th></th></tr></table>
        {ctx_growth_chart}
        <br>
        <table>
        <tr><th colspan="5">Context Window at Tool Use</th></tr>
        <tr><th>Tool</th><th>Uses</th><th>Avg Context</th><th>Max Context</th><th>Avg Ctx %</th></tr>
        {ctx_growth_trs}
        </table>
        <br>

        <table><tr><th>Avg Payload Size by Tool (bytes)</th><th></th></tr></table>
        {tool_chart}
        <br>
        <table>
        <tr><th colspan="4">Tool Payload Weight</th></tr>
        <tr><th>Tool</th><th>Uses</th><th>Sessions</th><th>Avg Payload</th></tr>
        {tool_cost_trs}
        </table>
        <br>

        <table>
        <tr><th colspan="7">Recent Token Snapshots</th></tr>
        <tr><th>Time</th><th>Session</th><th>In</th><th>Out</th><th>Total</th><th>Cost</th><th>Ctx %</th></tr>
        {timeline_trs}
        </table>
        """
        return page_shell("Token Usage", body, "Tokens")

    # ── Transcripts ──────────────────────────────────────────────────────

    def page_transcripts(self, conn, params):
        project_filter = params.get("project", "")
        page = int(params.get("page", 1))
        transcripts = list_transcripts()

        # Collect unique projects
        projects = sorted(set(t["project"] for t in transcripts))

        if project_filter:
            transcripts = [t for t in transcripts if t["project"] == project_filter]

        proj_opts = "".join(
            f'<option value="{esc(p)}" {"selected" if p == project_filter else ""}>'
            f"{esc(p)}</option>"
            for p in projects
        )

        filters = f"""
        <form class="filter-bar" method="get" action="/transcripts">
          <label>Project:</label>
          <select name="project"><option value="">All ({len(projects)} projects)</option>{proj_opts}</select>
          <button type="submit">Filter</button>
        </form>
        """

        def fmtk(n):
            if not n: return "0"
            if n < 1000: return str(n)
            if n < 1_000_000: return f"{n/1000:.1f}k"
            return f"{n/1_000_000:.1f}M"

        per_page = 30
        total_pages = max(1, (len(transcripts) + per_page - 1) // per_page)
        page = max(1, min(page, total_pages))
        page_slice = transcripts[(page - 1) * per_page : page * per_page]

        trs = ""
        for t in page_slice:
            meta = parse_transcript(t["path"], summary_only=True)
            # Check if we have hook_events for this session
            he_count = query_one(
                conn,
                "SELECT COUNT(*) c FROM hook_events WHERE session_id = ?",
                (t["session_id"],),
            )
            hook_count = he_count["c"] if he_count else 0

            # Check token_snapshots
            ts_count = 0
            try:
                ts_row = query_one(
                    conn,
                    "SELECT COUNT(*) c FROM token_snapshots WHERE session_id = ?",
                    (t["session_id"],),
                )
                ts_count = ts_row["c"] if ts_row else 0
            except Exception:
                pass

            prompt_preview = esc(truncate(meta["first_user_msg"] or "", 50))
            started = (meta["first_ts"] or "")[:19]

            hook_badge = f'<span class="badge badge-tool">{hook_count} hooks</span>' if hook_count else ""
            tok_badge = f'<span class="badge badge-event">{ts_count} snaps</span>' if ts_count else ""

            trs += f"""<tr>
              <td><a href="/transcripts/detail?path={esc(t['path'])}">{esc(t['session_id'][:16])}...</a></td>
              <td>{esc(t['project'][:40])}</td>
              <td>{started}</td>
              <td>{meta['user_msgs']}</td>
              <td>{meta['asst_msgs']}</td>
              <td>{fmtk(meta['total_input'])}</td>
              <td>{fmtk(meta['total_output'])}</td>
              <td>{fmtk(meta['total_tokens'])}</td>
              <td>{hook_badge} {tok_badge}</td>
              <td>{prompt_preview}</td>
            </tr>"""

        stats = f"""
        <div class="stats-grid">
          <div class="stat-box"><div class="stat-value">{len(transcripts)}</div><div class="stat-label">Transcripts{' (filtered)' if project_filter else ''}</div></div>
          <div class="stat-box"><div class="stat-value">{len(projects)}</div><div class="stat-label">Projects</div></div>
        </div>
        """

        pag_params = {}
        if project_filter:
            pag_params["project"] = project_filter
        pag = pagination_html(page, total_pages, "/transcripts", pag_params)

        body = f"""
        {filters}
        {stats}
        <p style="font-size:11px;margin-bottom:4px">{len(transcripts)} transcripts — page {page}/{total_pages}</p>
        <table>
        <tr><th>Session</th><th>Project</th><th>Started</th><th>User</th><th>Asst</th><th>In</th><th>Out</th><th>Total</th><th>Hook Data</th><th>First Prompt</th></tr>
        {trs}
        </table>
        {pag}
        """
        return page_shell("Transcripts", body, "Transcripts")

    def page_transcript_detail(self, conn, params):
        path = params.get("path", "")
        if not path or not os.path.exists(path):
            return page_shell("Transcript", f"<p>Transcript not found: {esc(path)}</p>")

        sid = os.path.splitext(os.path.basename(path))[0]
        proj = os.path.basename(os.path.dirname(path))
        meta = parse_transcript(path, summary_only=False)

        def fmtk(n):
            if not n: return "0"
            if n < 1000: return str(n)
            if n < 1_000_000: return f"{n/1000:.1f}k"
            return f"{n/1_000_000:.1f}M"

        stats = f"""
        <div class="stats-grid">
          <div class="stat-box"><div class="stat-value">{meta['user_msgs']}</div><div class="stat-label">User Msgs</div></div>
          <div class="stat-box"><div class="stat-value">{meta['asst_msgs']}</div><div class="stat-label">Asst Msgs</div></div>
          <div class="stat-box"><div class="stat-value">{fmtk(meta['total_input'])}</div><div class="stat-label">Input Tokens</div></div>
          <div class="stat-box"><div class="stat-value">{fmtk(meta['total_output'])}</div><div class="stat-label">Output Tokens</div></div>
          <div class="stat-box"><div class="stat-value">{fmtk(meta['total_tokens'])}</div><div class="stat-label">Total Tokens</div></div>
          <div class="stat-box"><div class="stat-value">{fmtk(meta['total_cache_create'])}</div><div class="stat-label">Cache Created</div></div>
          <div class="stat-box"><div class="stat-value">{fmtk(meta['total_cache_read'])}</div><div class="stat-label">Cache Read</div></div>
        </div>
        """

        dl = f"""
        <dl class="detail-grid">
          <dt>Session ID</dt><dd><a href="/sessions/detail?id={esc(sid)}">{esc(sid)}</a></dd>
          <dt>Project</dt><dd>{esc(proj)}</dd>
          <dt>Model</dt><dd>{esc(meta['model'] or '?')}</dd>
          <dt>Started</dt><dd>{esc(meta['first_ts'] or '?')}</dd>
          <dt>Ended</dt><dd>{esc(meta['last_ts'] or '?')}</dd>
          <dt>File</dt><dd style="font-size:10px">{esc(path)}</dd>
        </dl>
        """

        # Hook events for this session, indexed by tool_use_id
        hook_events_by_tuid = {}
        try:
            he_rows = query(
                conn,
                "SELECT id, tool_use_id, tool_name, hook_event_name FROM hook_events WHERE session_id = ? AND tool_use_id IS NOT NULL",
                (sid,),
            )
            for r in he_rows:
                tuid = r["tool_use_id"]
                if tuid not in hook_events_by_tuid:
                    hook_events_by_tuid[tuid] = []
                hook_events_by_tuid[tuid].append(r)
        except Exception:
            pass

        he_count = query_one(
            conn, "SELECT COUNT(*) c FROM hook_events WHERE session_id = ?", (sid,),
        )
        hook_link = ""
        if he_count and he_count["c"] > 0:
            hook_link = f'<p><a href="/events?session={esc(sid)}" style="border:2px outset #ccc;padding:2px 10px;display:inline-block;margin-bottom:8px">{he_count["c"]} hook events for this session &rarr;</a></p>'

        # ── Build conversation timeline ──────────────────────────────
        conv_html = '<div class="conv-timeline">'
        turn_num = 0
        cumulative_in = 0
        cumulative_out = 0

        # Also build tool_result map: tool_use_id -> result content
        tool_results = {}
        for entry in meta["entries"]:
            if entry.get("type") == "user":
                content = entry.get("message", {}).get("content", "")
                if isinstance(content, list):
                    for c in content:
                        if isinstance(c, dict) and c.get("type") == "tool_result":
                            tuid = c.get("tool_use_id", "")
                            result_content = c.get("content", "")
                            if isinstance(result_content, list):
                                parts = []
                                for rc in result_content:
                                    if isinstance(rc, dict) and rc.get("type") == "text":
                                        parts.append(rc.get("text", ""))
                                result_content = "\n".join(parts)
                            tool_results[tuid] = str(result_content)

        for entry in meta["entries"]:
            etype = entry.get("type")
            ts = (entry.get("timestamp") or "")[:19]

            if etype == "user":
                content = entry.get("message", {}).get("content", "")
                if isinstance(content, str) and content.strip():
                    turn_num += 1
                    preview = esc(truncate(content, 80))
                    conv_html += f"""
                    <details class="conv-entry conv-user">
                      <summary class="conv-entry-header">
                        <span class="arrow">&#9654;</span>
                        <span class="badge badge-session">#{turn_num}</span>
                        <b>User</b>
                        <span style="color:#666">{ts}</span>
                        <span style="flex:1"></span>
                        <span style="color:#888;font-size:10px">{preview}</span>
                      </summary>
                      <div class="conv-entry-body">
                        <div class="conv-text">{esc(content)}</div>
                      </div>
                    </details>"""

            elif etype == "assistant":
                msg = entry.get("message", {})
                usage = msg.get("usage", {})
                inp = usage.get("input_tokens", 0)
                out = usage.get("output_tokens", 0)
                cache_cr = usage.get("cache_creation_input_tokens", 0)
                cache_rd = usage.get("cache_read_input_tokens", 0)
                cumulative_in += inp
                cumulative_out += out

                content = msg.get("content", [])
                # Build preview from content
                preview_parts = []
                for c in content:
                    if isinstance(c, dict):
                        if c.get("type") == "text" and c.get("text", "").strip():
                            preview_parts.append("text")
                        elif c.get("type") == "tool_use":
                            preview_parts.append(c.get("name", "tool"))
                        elif c.get("type") == "thinking":
                            preview_parts.append("thinking")
                preview = ", ".join(preview_parts) or "..."

                # Build inner content blocks
                inner = ""
                inner += f"""<div class="conv-usage">
                  <span>in:{fmtk(inp)}</span>
                  <span>out:{fmtk(out)}</span>
                  <span>cache_cr:{fmtk(cache_cr)}</span>
                  <span>cache_rd:{fmtk(cache_rd)}</span>
                  <span>cumul:{fmtk(cumulative_in + cumulative_out)}</span>
                </div>"""

                for c in content:
                    if not isinstance(c, dict):
                        continue
                    ctype = c.get("type")

                    if ctype == "thinking":
                        thinking_text = c.get("thinking", "")
                        if thinking_text.strip():
                            inner += f"""<details class="conv-tool-block">
                              <summary class="conv-tool-header">thinking ({len(thinking_text)} chars)</summary>
                              <div class="conv-tool-body conv-thinking">{esc(thinking_text)}</div>
                            </details>"""

                    elif ctype == "text":
                        text = c.get("text", "")
                        if text.strip():
                            inner += f'<div class="conv-text">{esc(text)}</div>'

                    elif ctype == "tool_use":
                        tool_name = c.get("name", "?")
                        tool_id = c.get("id", "")
                        tool_input = c.get("input", {})
                        input_yaml = _yaml_val(tool_input)
                        input_hl = highlight_yaml(input_yaml)

                        # Cross-link to hook events
                        hook_links = ""
                        hevts = hook_events_by_tuid.get(tool_id, [])
                        if hevts:
                            links = []
                            for he in hevts:
                                evt_label = he["hook_event_name"].replace("ToolUse", "")
                                links.append(
                                    f'<a href="/events/detail?id={he["id"]}" '
                                    f'style="font-size:10px">{evt_label} #{he["id"]}</a>'
                                )
                            hook_links = f' <span style="font-size:10px;color:#666">[{" | ".join(links)}]</span>'

                        # Tool result
                        result_html = ""
                        result_text = tool_results.get(tool_id)
                        if result_text:
                            result_preview = truncate(result_text, 200)
                            result_html = f"""<details style="margin-top:2px">
                              <summary style="font-size:10px;cursor:pointer;color:#666">Result ({len(result_text)} chars)</summary>
                              <div class="conv-tool-body">{esc(result_text)}</div>
                            </details>"""

                        inner += f"""<div class="conv-tool-block">
                          <div class="conv-tool-header">
                            <span class="badge badge-tool">{esc(tool_name)}</span>
                            {hook_links}
                            <span style="flex:1"></span>
                            <span style="font-size:9px;color:#999">{esc(tool_id[:20])}</span>
                          </div>
                          <details>
                            <summary style="padding:2px 6px;font-size:10px;cursor:pointer">Input ({len(input_yaml)} chars)</summary>
                            <div class="conv-tool-body">{input_hl}</div>
                          </details>
                          {result_html}
                        </div>"""

                turn_num += 1
                conv_html += f"""
                <details class="conv-entry conv-assistant" open>
                  <summary class="conv-entry-header">
                    <span class="arrow">&#9654;</span>
                    <span class="badge badge-session">#{turn_num}</span>
                    <b>Assistant</b>
                    <span style="color:#666">{ts}</span>
                    <span style="font-size:10px;color:#888">{esc(preview)}</span>
                    <span style="flex:1"></span>
                    <span style="font-size:10px">{fmtk(inp)}+{fmtk(out)}</span>
                  </summary>
                  <div class="conv-entry-body">{inner}</div>
                </details>"""

            elif etype == "system":
                subtype = entry.get("subtype", "")
                dur = entry.get("durationMs")
                dur_str = ""
                if dur:
                    s = dur // 1000
                    dur_str = f" ({s // 60}m{s % 60:02d}s)" if s >= 60 else f" ({s}s)"
                conv_html += f"""
                <div class="conv-entry conv-system" style="border-style:dashed">
                  <div class="conv-entry-header">
                    <span style="font-style:italic;color:#888">system: {esc(subtype)}{dur_str}</span>
                  </div>
                </div>"""

        conv_html += "</div>"

        # Tool usage summary chart
        tool_counts = {}
        for tu in meta["tool_uses"]:
            name = tu["name"]
            tool_counts[name] = tool_counts.get(name, 0) + 1
        tool_chart = bar_chart(
            sorted(tool_counts.items(), key=lambda x: -x[1])
        ) if tool_counts else ""

        body = f"""
        {dl}
        {stats}
        {hook_link}

        {f'<table><tr><th>Tool Usage</th><th></th></tr></table>{tool_chart}<br>' if tool_chart else ''}

        <table><tr><th>Conversation</th></tr></table>
        {conv_html}
        """
        return page_shell(f"Transcript {esc(sid[:16])}...", body, "Transcripts")

    # ── SQL Console ──────────────────────────────────────────────────────

    def page_sql(self, conn, params):
        sql_input = params.get("q", "").strip()
        result_html = ""

        if sql_input:
            # Only allow read-only queries
            normalized = sql_input.lstrip().upper()
            if not normalized.startswith("SELECT") and not normalized.startswith("WITH"):
                result_html = '<p style="border:2px solid #000;padding:8px">Only SELECT / WITH queries allowed.</p>'
            else:
                try:
                    rows = query(conn, sql_input)
                    if rows:
                        keys = rows[0].keys()
                        header = "".join(f"<th>{esc(k)}</th>" for k in keys)
                        trs = ""
                        for r in rows[:500]:
                            cells = "".join(
                                f"<td>{esc(truncate(str(r[k]), 120) if r[k] is not None else '')}</td>"
                                for k in keys
                            )
                            trs += f"<tr>{cells}</tr>"
                        result_html = f"""
                        <p style="font-size:11px">{len(rows)} rows{' (showing first 500)' if len(rows) > 500 else ''}</p>
                        <table><tr>{header}</tr>{trs}</table>
                        """
                    else:
                        result_html = "<p>No results.</p>"
                except Exception as e:
                    result_html = f'<pre style="border:2px solid #000;padding:8px">{esc(str(e))}</pre>'

        examples = """
        <p style="font-size:11px;margin-top:8px"><b>Example queries:</b></p>
        <ul style="font-size:11px;margin:4px 0 0 16px">
          <li><code>SELECT * FROM v_tool_usage</code></li>
          <li><code>SELECT * FROM v_session_summary</code></li>
          <li><code>SELECT * FROM v_daily_activity</code></li>
          <li><code>SELECT * FROM v_bash_commands LIMIT 20</code></li>
          <li><code>SELECT * FROM v_file_touch_frequency LIMIT 20</code></li>
          <li><code>SELECT hook_event_name, COUNT(*) c FROM hook_events GROUP BY 1 ORDER BY 2 DESC</code></li>
        </ul>
        """

        body = f"""
        <form method="get" action="/sql" style="margin-bottom:8px">
          <textarea name="q" rows="4" style="width:100%;font-family:Monaco,monospace;font-size:12px;border:2px inset #999;padding:4px;resize:vertical">{esc(sql_input)}</textarea>
          <br>
          <button type="submit" style="font-family:Chicago,Monaco,monospace;font-size:12px;border:2px outset #ccc;padding:3px 16px;margin-top:4px;cursor:pointer">Run Query</button>
        </form>
        {result_html}
        {examples if not sql_input else ''}
        """
        return page_shell("SQL Console", body, "SQL")


# ── Main ─────────────────────────────────────────────────────────────────────


def main():
    parser = argparse.ArgumentParser(description="Browse Claude Code hook event logs")
    parser.add_argument("--port", type=int, default=DEFAULT_PORT, help=f"Port (default {DEFAULT_PORT})")
    parser.add_argument("--db", default=DEFAULT_DB, help=f"SQLite DB path (default {DEFAULT_DB})")
    args = parser.parse_args()

    if not os.path.exists(args.db):
        print(f"Error: database not found at {args.db}")
        raise SystemExit(1)

    HookEventsHandler.db_path = args.db
    server = HTTPServer(("127.0.0.1", args.port), HookEventsHandler)
    print(f"Hook Events Browser → http://127.0.0.1:{args.port}")
    print(f"Database: {args.db}")
    print("Press Ctrl+C to stop.")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped.")


if __name__ == "__main__":
    main()
