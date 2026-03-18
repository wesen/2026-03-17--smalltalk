package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/wesen/st80/pkg/image"
	"github.com/wesen/st80/pkg/interpreter"
)

func main() {
	imagePath := "data/VirtualImage"
	maxCycles := uint64(1000000)

	if len(os.Args) > 1 {
		imagePath = os.Args[1]
	}
	if len(os.Args) > 2 {
		n, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err == nil {
			maxCycles = n
		}
	}

	fmt.Printf("Loading image: %s\n", imagePath)
	om, err := image.LoadImage(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading image: %v\n", err)
		os.Exit(1)
	}

	om.Dump()

	interp := interpreter.New(om)
	fmt.Printf("\nRunning interpreter for up to %d cycles...\n", maxCycles)

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\nInterpreter panic: %v\n", r)
		}
	}()

	if err := interp.Run(maxCycles); err != nil {
		fmt.Fprintf(os.Stderr, "Interpreter error: %v\n", err)
		os.Exit(1)
	}
}
