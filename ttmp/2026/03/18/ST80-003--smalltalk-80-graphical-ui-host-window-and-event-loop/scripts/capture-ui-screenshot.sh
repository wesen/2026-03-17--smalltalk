#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/manuel/code/wesen/2026-03-17--smalltalk"
TICKET_DIR="$ROOT_DIR/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop"
OUT_DIR="$TICKET_DIR/various/ui-capture"
DISPLAY_NUM="${DISPLAY_NUM:-:99}"
MAX_CYCLES="${MAX_CYCLES:-0}"
CYCLES_PER_FRAME="${CYCLES_PER_FRAME:-50000}"
SCALE="${SCALE:-2}"
WINDOW_TITLE="${WINDOW_TITLE:-Smalltalk-80}"

mkdir -p "$OUT_DIR"

PNG_PATH="$OUT_DIR/st80-ui.png"
XWD_PATH="$OUT_DIR/st80-ui.xwd"
TREE_PATH="$OUT_DIR/xwininfo-tree.txt"
RUN_LOG="$OUT_DIR/st80-ui-run.log"
XVFB_LOG="$OUT_DIR/xvfb.log"

rm -f "$PNG_PATH" "$XWD_PATH" "$TREE_PATH" "$RUN_LOG" "$XVFB_LOG"

Xvfb "$DISPLAY_NUM" -screen 0 1280x1024x24 >"$XVFB_LOG" 2>&1 &
XVFB_PID=$!
UI_PID=""

cleanup() {
  if [[ -n "$UI_PID" ]]; then
    kill "$UI_PID" >/dev/null 2>&1 || true
    wait "$UI_PID" >/dev/null 2>&1 || true
  fi
  kill "$XVFB_PID" >/dev/null 2>&1 || true
  wait "$XVFB_PID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

sleep 1

(
  cd "$ROOT_DIR"
  DISPLAY="$DISPLAY_NUM" go run ./cmd/st80-ui \
    -max-cycles "$MAX_CYCLES" \
    -cycles-per-frame "$CYCLES_PER_FRAME" \
    -scale "$SCALE" \
    -title "$WINDOW_TITLE"
) >"$RUN_LOG" 2>&1 &
UI_PID=$!

WIN_ID=""
for _ in $(seq 1 40); do
  if ! kill -0 "$UI_PID" >/dev/null 2>&1; then
    echo "UI process exited before window capture" >&2
    exit 1
  fi
  DISPLAY="$DISPLAY_NUM" xwininfo -root -tree >"$TREE_PATH"
  WIN_ID="$(DISPLAY="$DISPLAY_NUM" xwininfo -root -tree | awk -v title="$WINDOW_TITLE" '$0 ~ title {print $1; exit}')"
  if [[ -n "$WIN_ID" ]]; then
    break
  fi
  sleep 0.25
done

if [[ -z "$WIN_ID" ]]; then
  echo "UI window titled '$WINDOW_TITLE' not found" >&2
  exit 1
fi

sleep 1

DISPLAY="$DISPLAY_NUM" xwd -silent -id "$WIN_ID" -out "$XWD_PATH"
convert "$XWD_PATH" "$PNG_PATH"

echo "$PNG_PATH"
