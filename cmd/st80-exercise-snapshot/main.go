package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wesen/st80/pkg/ui"
)

func main() {
	imagePath := flag.String("image", "data/VirtualImage", "path to the Smalltalk-80 image")
	beforeCycles := flag.Uint64("before-cycles", 50000, "interpreter cycles to run before injecting direct input")
	afterCycles := flag.Uint64("after-cycles", 50000, "interpreter cycles to run after injecting direct input")
	mouseX := flag.Int("mouse-x", 120, "logical x position for the injected mouse event")
	mouseY := flag.Int("mouse-y", 120, "logical y position for the injected mouse event")
	click := flag.String("click", "left", "mouse button to click: none|left|middle|right")
	text := flag.String("text", "a", "ASCII text to inject via decoded-key events")
	pressReturn := flag.Bool("return", true, "whether to inject a Return keypress")
	beforeOutput := flag.String("before-output", "", "PNG output path for the pre-input snapshot")
	afterOutput := flag.String("after-output", "", "PNG output path for the post-input snapshot")
	flag.Parse()

	diag, err := ui.ExerciseInputAndCapture(ui.InputExerciseOptions{
		ImagePath:        *imagePath,
		BeforeCycles:     *beforeCycles,
		AfterCycles:      *afterCycles,
		MouseX:           *mouseX,
		MouseY:           *mouseY,
		ClickButton:      *click,
		TypeText:         *text,
		PressReturn:      *pressReturn,
		BeforeOutputPath: *beforeOutput,
		AfterOutputPath:  *afterOutput,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "st80-exercise-snapshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("beforeCycles=%d afterCycles=%d changedPixels=%d beforeHash=%s afterHash=%s beforeBlack=%d afterBlack=%d",
		diag.Before.CycleCount,
		diag.After.CycleCount-diag.Before.CycleCount,
		diag.ChangedPixels,
		diag.Before.WordHash,
		diag.After.WordHash,
		diag.Before.BlackPixels,
		diag.After.BlackPixels,
	)
	if diag.Before.OutputPath != "" {
		fmt.Printf(" beforeOutput=%s", diag.Before.OutputPath)
	}
	if diag.After.OutputPath != "" {
		fmt.Printf(" afterOutput=%s", diag.After.OutputPath)
	}
	fmt.Println()
}
