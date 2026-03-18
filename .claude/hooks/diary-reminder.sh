#!/bin/bash
# Debounced diary reminder: only fires if 90s+ since last reminder.
# Uses per-session stamp files to track last fire time.
# Reads hook event JSON from stdin to extract session_id.
# Reads INSTRUCTIONS.md and includes it in the reminder.

INTERVAL=90 # seconds
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
INSTRUCTIONS="$PROJECT_DIR/INSTRUCTIONS.md"
LOG="/tmp/claude-hook.log"

# Read event JSON from stdin and extract session_id
INPUT=$(cat)
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // empty' 2>/dev/null)

if [ -z "$SESSION_ID" ]; then
  SESSION_ID="unknown"
fi

# Per-session stamp file
STAMP="/tmp/.claude-diary-reminder-stamp-${SESSION_ID}"

now=$(date +%s)
echo "$(date) session=$SESSION_ID" >>"$LOG"

if [ -f "$STAMP" ]; then
  last=$(cat "$STAMP")
  elapsed=$((now - last))
  echo "session=$SESSION_ID last=$last elapsed=$elapsed" >>"$LOG"
  if [ "$elapsed" -lt "$INTERVAL" ]; then
    exit 0 # silent, no output
  fi
fi

echo "session=$SESSION_ID now=$now STAMP=$STAMP" >>"$LOG"
echo "$now" >"$STAMP"

echo "REMINDER — check your working instructions:"
echo ""
if [ -f "$INSTRUCTIONS" ]; then
  cat "$INSTRUCTIONS"
  echo "session=$SESSION_ID FULL INSTRUCTIONS" >>"$LOG"
else
  echo "session=$SESSION_ID NOT FOUND INSTRUCTIONS" >>"$LOG"
  echo "(INSTRUCTIONS.md not found at $INSTRUCTIONS)"
fi
