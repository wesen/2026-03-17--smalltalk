package ui

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	stdimage "image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	stimage "github.com/wesen/st80/pkg/image"
	"github.com/wesen/st80/pkg/interpreter"
)

// SnapshotOptions control direct framebuffer capture without SDL.
type SnapshotOptions struct {
	ImagePath  string
	Cycles     uint64
	OutputPath string
}

// SnapshotDiagnostic describes the captured framebuffer contents.
type SnapshotDiagnostic struct {
	CycleCount  uint64
	Width       int
	Height      int
	Raster      int
	BlackPixels int
	WhitePixels int
	WordHash    string
	OutputPath  string
}

// CaptureSnapshot runs the interpreter headlessly for a bounded number of
// cycles, captures the designated display form, and optionally writes it to a
// PNG. This is intended for fast framebuffer diagnostics without SDL/Xvfb.
func CaptureSnapshot(opts SnapshotOptions) (SnapshotDiagnostic, error) {
	if opts.ImagePath == "" {
		opts.ImagePath = "data/VirtualImage"
	}

	om, err := stimage.LoadImage(opts.ImagePath)
	if err != nil {
		return SnapshotDiagnostic{}, fmt.Errorf("load image: %w", err)
	}

	interp := interpreter.New(om)
	if err := interp.RunSteps(opts.Cycles); err != nil {
		return SnapshotDiagnostic{}, err
	}

	snapshot, ok := interp.DisplaySnapshot()
	if !ok {
		return SnapshotDiagnostic{}, fmt.Errorf("no display snapshot available after %d cycles", opts.Cycles)
	}

	pixels := make([]uint32, snapshot.Width*snapshot.Height)
	black, white := copyDisplayBits(pixels, snapshot)
	hash := hashWords(snapshot.Words)

	if opts.OutputPath != "" {
		if err := writeSnapshotPNG(opts.OutputPath, pixels, snapshot.Width, snapshot.Height); err != nil {
			return SnapshotDiagnostic{}, err
		}
	}

	return SnapshotDiagnostic{
		CycleCount:  interp.CycleCount(),
		Width:       snapshot.Width,
		Height:      snapshot.Height,
		Raster:      snapshot.Raster,
		BlackPixels: black,
		WhitePixels: white,
		WordHash:    hash,
		OutputPath:  opts.OutputPath,
	}, nil
}

func hashWords(words []uint16) string {
	h := sha256.New()
	var buf [2]byte
	for _, word := range words {
		binary.LittleEndian.PutUint16(buf[:], word)
		_, _ = h.Write(buf[:])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeSnapshotPNG(path string, pixels []uint32, width int, height int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}

	img := stdimage.NewRGBA(stdimage.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := pixels[y*width+x]
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(pixel >> 16),
				G: uint8(pixel >> 8),
				B: uint8(pixel),
				A: uint8(pixel >> 24),
			})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("encode png %s: %w", path, err)
	}
	return nil
}
