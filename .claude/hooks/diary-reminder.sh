#!/bin/bash
# Debounced diary reminder: only fires if 2+ minutes since last reminder.
# Uses a stamp file to track last fire time.
# Reads INSTRUCTIONS.md and includes it in the reminder.

STAMP="/tmp/.claude-diary-reminder-stamp"
INTERVAL=120 # seconds
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
INSTRUCTIONS="$PROJECT_DIR/INSTRUCTIONS.md"

now=$(date +%s)

if [ -f "$STAMP" ]; then
  last=$(cat "$STAMP")
  elapsed=$(( now - last ))
  if [ "$elapsed" -lt "$INTERVAL" ]; then
    exit 0  # silent, no output
  fi
fi

echo "$now" > "$STAMP"

echo "REMINDER — check your working instructions:"
echo ""
if [ -f "$INSTRUCTIONS" ]; then
  cat "$INSTRUCTIONS"
else
  echo "(INSTRUCTIONS.md not found at $INSTRUCTIONS)"
fi
