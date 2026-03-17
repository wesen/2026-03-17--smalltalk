#!/bin/bash
# Debounced diary reminder: only fires if 2+ minutes since last reminder.
# Uses a stamp file to track last fire time.

STAMP="/tmp/.claude-diary-reminder-stamp"
INTERVAL=120 # seconds

now=$(date +%s)

if [ -f "$STAMP" ]; then
  last=$(cat "$STAMP")
  elapsed=$(( now - last ))
  if [ "$elapsed" -lt "$INTERVAL" ]; then
    exit 0  # silent, no output
  fi
fi

echo "$now" > "$STAMP"
echo "Don't forget to update your diary (see /diary skill) if you haven't already, otherwise continue. Don't forget to commit at appropriate intervals."
