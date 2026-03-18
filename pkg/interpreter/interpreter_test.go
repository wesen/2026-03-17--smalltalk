package interpreter

import (
	"path/filepath"
	"testing"

	"github.com/wesen/st80/pkg/image"
	om "github.com/wesen/st80/pkg/objectmemory"
)

func loadTestInterpreter(t *testing.T) *Interpreter {
	t.Helper()

	imagePath := filepath.Join("..", "..", "data", "VirtualImage")
	memory, err := image.LoadImage(imagePath)
	if err != nil {
		t.Fatalf("load image: %v", err)
	}

	interp := New(memory)
	scheduler := interp.fetchPointer(ValueIndex, om.SchedulerAssociationPointer)
	activeProcess := interp.fetchPointer(ActiveProcessIndex, scheduler)
	suspendedContext := interp.fetchPointer(SuspendedContextIndex, activeProcess)
	interp.activeContext = suspendedContext
	interp.fetchContextRegisters()
	return interp
}

func TestStartupRunsPastFormerContextOverflow(t *testing.T) {
	interp := loadTestInterpreter(t)

	defer func() {
		r := recover()
		if r != nil {
			contextSize := interp.fetchWordLengthOf(interp.activeContext)
			tempCount := interp.temporaryCountOf(interp.method)
			largeContext := interp.largeContextFlagOf(interp.method)
			storedSP := interp.fetchPointer(StackPointerIndex, interp.activeContext)
			storedIP := interp.fetchPointer(InstructionPointerIndex, interp.activeContext)

			t.Fatalf("unexpected panic after %d cycles: %v\nactiveContext=0x%04X method=0x%04X receiver=0x%04X bytecode=%d ip=%d sp=%d\ncontextFields=%d storedIP=0x%04X storedSP=0x%04X tempCount=%d largeContextFlag=%d",
				interp.cycleCount, r, interp.activeContext, interp.method, interp.receiver,
				interp.currentBytecode, interp.instructionPointer, interp.stackPointer,
				contextSize, storedIP, storedSP, tempCount, largeContext)
		}
	}()

	for interp.cycleCount = 0; interp.cycleCount < 2000; interp.cycleCount++ {
		interp.checkProcessSwitch()
		interp.currentBytecode = interp.fetchBytecode()
		interp.dispatchOnThisBytecode()
	}

	if interp.cycleCount != 2000 {
		t.Fatalf("expected to complete 2000 cycles, got %d", interp.cycleCount)
	}
}
