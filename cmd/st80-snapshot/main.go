package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wesen/st80/pkg/ui"
)

func main() {
	imagePath := flag.String("image", "data/VirtualImage", "path to the Smalltalk-80 image")
	cycles := flag.Uint64("cycles", 1000000, "interpreter cycles to run before taking the snapshot")
	output := flag.String("output", "", "PNG output path for the captured display framebuffer")
	flag.Parse()

	diag, err := ui.CaptureSnapshot(ui.SnapshotOptions{
		ImagePath:  *imagePath,
		Cycles:     *cycles,
		OutputPath: *output,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "st80-snapshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("cycles=%d width=%d height=%d raster=%d blackPixels=%d whitePixels=%d wordHash=%s",
		diag.CycleCount, diag.Width, diag.Height, diag.Raster, diag.BlackPixels, diag.WhitePixels, diag.WordHash)
	if diag.OutputPath != "" {
		fmt.Printf(" output=%s", diag.OutputPath)
	}
	fmt.Println()
}
