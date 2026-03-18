package ui

import (
	"fmt"

	stimage "github.com/wesen/st80/pkg/image"
	"github.com/wesen/st80/pkg/interpreter"
)

// InputExerciseOptions controls a direct interpreter-side input exercise that
// captures before/after framebuffer snapshots without SDL or X11.
type InputExerciseOptions struct {
	ImagePath        string
	BeforeCycles     uint64
	AfterCycles      uint64
	MouseX           int
	MouseY           int
	ClickButton      string
	TypeText         string
	PressReturn      bool
	BeforeOutputPath string
	AfterOutputPath  string
}

// InputExerciseDiagnostic summarizes a direct interpreter-side input exercise.
type InputExerciseDiagnostic struct {
	Before        SnapshotDiagnostic
	After         SnapshotDiagnostic
	ChangedPixels int
}

// ExerciseInputAndCapture runs the image, captures a baseline snapshot, injects
// a small direct input sequence into the interpreter, runs for more cycles, and
// captures an after snapshot for comparison.
func ExerciseInputAndCapture(opts InputExerciseOptions) (InputExerciseDiagnostic, error) {
	if opts.ImagePath == "" {
		opts.ImagePath = "data/VirtualImage"
	}

	om, err := stimage.LoadImage(opts.ImagePath)
	if err != nil {
		return InputExerciseDiagnostic{}, fmt.Errorf("load image: %w", err)
	}

	interp := interpreter.New(om)
	interp.SetSnapshotPath(opts.ImagePath)
	if err := interp.RunSteps(opts.BeforeCycles); err != nil {
		return InputExerciseDiagnostic{}, err
	}

	before, beforePixels, err := captureSnapshotDiagnostic(interp, opts.BeforeOutputPath)
	if err != nil {
		return InputExerciseDiagnostic{}, err
	}

	var timestamp uint32 = 100
	if opts.MouseX >= 0 && opts.MouseY >= 0 {
		interp.RecordMouseMotion(opts.MouseX, opts.MouseY, timestamp)
		timestamp += 50
	}
	if parameter, ok := exerciseButtonParameter(opts.ClickButton); ok {
		interp.RecordMouseButton(parameter, true, opts.MouseX, opts.MouseY, timestamp)
		timestamp += 50
		interp.RecordMouseButton(parameter, false, opts.MouseX, opts.MouseY, timestamp)
		timestamp += 50
	}
	for _, r := range opts.TypeText {
		if r < 0 || r > 0x7F {
			continue
		}
		interp.RecordDecodedKey(uint16(r), timestamp)
		timestamp += 50
	}
	if opts.PressReturn {
		interp.RecordDecodedKey(13, timestamp)
	}

	if err := interp.RunSteps(opts.AfterCycles); err != nil {
		return InputExerciseDiagnostic{}, err
	}

	after, afterPixels, err := captureSnapshotDiagnostic(interp, opts.AfterOutputPath)
	if err != nil {
		return InputExerciseDiagnostic{}, err
	}

	return InputExerciseDiagnostic{
		Before:        before,
		After:         after,
		ChangedPixels: countChangedPixels(beforePixels, afterPixels),
	}, nil
}

func captureSnapshotDiagnostic(interp *interpreter.Interpreter, outputPath string) (SnapshotDiagnostic, []uint32, error) {
	snapshot, ok := interp.DisplaySnapshot()
	if !ok {
		return SnapshotDiagnostic{}, nil, fmt.Errorf("no display snapshot available after %d cycles", interp.CycleCount())
	}

	pixels := make([]uint32, snapshot.Width*snapshot.Height)
	cursor, hasCursor := interp.CursorSnapshot()
	black, white := copyDisplayBits(pixels, snapshot, hasCursor, cursor)
	hash := hashWords(snapshot.Words)

	if outputPath != "" {
		if err := writeSnapshotPNG(outputPath, pixels, snapshot.Width, snapshot.Height); err != nil {
			return SnapshotDiagnostic{}, nil, err
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
		OutputPath:  outputPath,
	}, pixels, nil
}

func exerciseButtonParameter(button string) (uint16, bool) {
	switch button {
	case "", "none":
		return 0, false
	case "left":
		return 128, true
	case "middle":
		return 129, true
	case "right":
		return 130, true
	default:
		return 0, false
	}
}

func countChangedPixels(before []uint32, after []uint32) int {
	if len(before) != len(after) {
		if len(before) > len(after) {
			return len(before)
		}
		return len(after)
	}
	changed := 0
	for i := range before {
		if before[i] != after[i] {
			changed++
		}
	}
	return changed
}
