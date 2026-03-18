#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/manuel/code/wesen/2026-03-17--smalltalk"
TICKET_DIR="$ROOT_DIR/ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop"
OUT_DIR="$TICKET_DIR/various/display-snapshots"
CYCLES="${CYCLES:-1000000}"
OUTPUT="${OUTPUT:-$OUT_DIR/display-${CYCLES}.png}"
IMAGE_PATH="${IMAGE_PATH:-data/VirtualImage}"

mkdir -p "$OUT_DIR"

cd "$ROOT_DIR"
go run ./cmd/st80-snapshot -image "$IMAGE_PATH" -cycles "$CYCLES" -output "$OUTPUT"
