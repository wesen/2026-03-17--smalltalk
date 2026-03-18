package ui

import (
	"testing"

	"github.com/wesen/st80/pkg/interpreter"
)

func TestCopyDisplayBitsOverlaysCursorBits(t *testing.T) {
	display := interpreter.DisplaySnapshot{
		Width:  16,
		Height: 16,
		Raster: 1,
		Words:  make([]uint16, 16),
	}
	cursor := interpreter.CursorSnapshot{
		X:      2,
		Y:      3,
		Width:  16,
		Height: 16,
		Raster: 1,
		Words:  make([]uint16, 16),
	}
	cursor.Words[0] = 0x8000

	pixels := make([]uint32, display.Width*display.Height)
	copyDisplayBits(pixels, display, true, cursor)

	black := uint32(0xFF000000)
	white := uint32(0xFFFFFFFF)
	if got := pixels[3*display.Width+2]; got != black {
		t.Fatalf("expected cursor overlay pixel to be black, got 0x%08X", got)
	}
	if got := pixels[3*display.Width+1]; got != white {
		t.Fatalf("expected neighboring pixel to stay white, got 0x%08X", got)
	}
}
