package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wesen/st80/pkg/ui"
)

func main() {
	imagePath := flag.String("image", "data/VirtualImage", "path to the Smalltalk-80 image")
	cyclesPerFrame := flag.Uint64("cycles-per-frame", 50000, "interpreter cycles to run between window refreshes")
	maxCycles := flag.Uint64("max-cycles", 0, "maximum cycles to execute before exiting (0 means run until window close)")
	scale := flag.Int("scale", 2, "window scale factor")
	title := flag.String("title", "Smalltalk-80", "window title")
	inputDebug := flag.Bool("input-debug", false, "log coarse input queue/consumption counters as they change")
	flag.Parse()

	if err := ui.Run(ui.Options{
		ImagePath:      *imagePath,
		CyclesPerFrame: *cyclesPerFrame,
		MaxCycles:      *maxCycles,
		Scale:          int32(*scale),
		WindowTitle:    *title,
		InputDebug:     *inputDebug,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "st80-ui: %v\n", err)
		os.Exit(1)
	}
}
