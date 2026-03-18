#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/manuel/code/wesen/2026-03-17--smalltalk"
TICKET_DIR="$ROOT_DIR/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop"
OUT_DIR="$TICKET_DIR/various/direct-input-capture"

mkdir -p "$OUT_DIR"

BEFORE_PNG="$OUT_DIR/direct-before.png"
AFTER_PNG="$OUT_DIR/direct-after.png"

cd "$ROOT_DIR"
go run ./cmd/st80-exercise-snapshot \
  -before-cycles "${BEFORE_CYCLES:-50000}" \
  -after-cycles "${AFTER_CYCLES:-50000}" \
  -mouse-x "${MOUSE_X:-120}" \
  -mouse-y "${MOUSE_Y:-120}" \
  -click "${CLICK_BUTTON:-left}" \
  -text "${TYPE_TEXT:-a}" \
  -return="${PRESS_RETURN:-true}" \
  -before-output "$BEFORE_PNG" \
  -after-output "$AFTER_PNG"
