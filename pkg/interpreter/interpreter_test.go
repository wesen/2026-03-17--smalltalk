package interpreter

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wesen/st80/pkg/image"
	om "github.com/wesen/st80/pkg/objectmemory"
)

func loadOopNames(t *testing.T, relativePath string) map[uint16]string {
	t.Helper()

	path := filepath.Join("..", "..", relativePath)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", relativePath, err)
	}
	defer file.Close()

	names := map[uint16]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 || !strings.HasPrefix(fields[1], "16r") {
			continue
		}
		value, err := strconv.ParseUint(strings.TrimPrefix(fields[1], "16r"), 16, 16)
		if err != nil {
			t.Fatalf("parse oop %q in %s: %v", fields[1], relativePath, err)
		}
		names[uint16(value)] = strings.Join(fields[2:], " ")
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", relativePath, err)
	}
	return names
}

func symbolString(interp *Interpreter, oop uint16) string {
	if !interp.memory.ValidOop(oop) {
		return ""
	}
	class := interp.fetchClassOf(oop)
	if class != om.ClassSymbolPointer && class != om.ClassStringPointer {
		return ""
	}
	return interp.memory.FetchStringOf(oop)
}

func isMethodOrBlockContext(interp *Interpreter, oop uint16) bool {
	if !interp.memory.ValidOop(oop) {
		return false
	}
	class := interp.fetchClassOf(oop)
	return class == om.ClassMethodContextPointer || class == om.ClassBlockContextPointer
}

func isCompiledMethod(interp *Interpreter, oop uint16) bool {
	if !interp.memory.ValidOop(oop) {
		return false
	}
	return interp.fetchClassOf(oop) == om.ClassCompiledMethodPointer
}

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

func TestDiagnoseRecursiveNotUnderstood(t *testing.T) {
	t.Skip("diagnostic test retained for manual investigation")
	classNames := loadOopNames(t, "data/class.oops")
	methodNames := loadOopNames(t, "data/method.oops")
	interp := loadTestInterpreter(t)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected recursive doesNotUnderstand panic, got none")
		}

		receiverClass := interp.fetchClassOf(interp.receiver)
		t.Logf("panic: %v", r)
		t.Logf("cycle=%d activeContext=0x%04X method=0x%04X(%s) receiver=0x%04X receiverClass=0x%04X(%s)",
			interp.cycleCount, interp.activeContext, interp.method, methodNames[interp.method],
			interp.receiver, receiverClass, classNames[receiverClass])
		t.Logf("selector=0x%04X(%q) argumentCount=%d bytecode=%d ip=%d sp=%d",
			interp.messageSelector, symbolString(interp, interp.messageSelector), interp.argumentCount,
			interp.currentBytecode, interp.instructionPointer, interp.stackPointer)

		ctx := interp.activeContext
		for depth := 0; depth < 6 && ctx != om.NilPointer; depth++ {
			home := ctx
			if interp.isBlockContext(ctx) {
				home = interp.fetchPointer(HomeIndex, ctx)
			}
			method := interp.fetchPointer(MethodIndex, home)
			sender := interp.fetchPointer(SenderIndex, home)
			t.Logf("senderChain[%d]: ctx=0x%04X home=0x%04X method=0x%04X(%s) sender=0x%04X",
				depth, ctx, home, method, methodNames[method], sender)
			ctx = sender
		}
	}()

	for interp.cycleCount = 0; interp.cycleCount < 500000; interp.cycleCount++ {
		interp.checkProcessSwitch()
		interp.currentBytecode = interp.fetchBytecode()
		interp.dispatchOnThisBytecode()
	}

	t.Fatalf("expected panic before 500000 cycles")
}

func TestDetectFirstInvalidActiveContext(t *testing.T) {
	methodNames := loadOopNames(t, "data/method.oops")
	interp := loadTestInterpreter(t)

	for interp.cycleCount = 0; interp.cycleCount < 20000; interp.cycleCount++ {
		if !isMethodOrBlockContext(interp, interp.activeContext) {
			t.Fatalf("invalid activeContext before cycle %d: 0x%04X class=0x%04X method=0x%04X(%s) bytecode=%d ip=%d sp=%d",
				interp.cycleCount, interp.activeContext, interp.fetchClassOf(interp.activeContext),
				interp.method, methodNames[interp.method], interp.currentBytecode,
				interp.instructionPointer, interp.stackPointer)
		}

		previousContext := interp.activeContext
		previousMethod := interp.method
		previousIP := interp.instructionPointer
		previousSP := interp.stackPointer

		interp.checkProcessSwitch()
		if !isMethodOrBlockContext(interp, interp.activeContext) {
			t.Fatalf("activeContext became invalid during checkProcessSwitch at cycle %d from ctx=0x%04X method=0x%04X(%s) ip=%d sp=%d; new activeContext=0x%04X class=0x%04X",
				interp.cycleCount, previousContext, previousMethod, methodNames[previousMethod],
				previousIP, previousSP, interp.activeContext, interp.fetchClassOf(interp.activeContext))
		}
		interp.currentBytecode = interp.fetchBytecode()
		interp.dispatchOnThisBytecode()

		if !isMethodOrBlockContext(interp, interp.activeContext) {
			t.Fatalf("activeContext became invalid at cycle %d after executing bytecode=%d in ctx=0x%04X method=0x%04X(%s) ip=%d sp=%d; new activeContext=0x%04X class=0x%04X",
				interp.cycleCount, interp.currentBytecode, previousContext,
				previousMethod, methodNames[previousMethod], previousIP, previousSP,
				interp.activeContext, interp.fetchClassOf(interp.activeContext))
		}
	}
}

func TestDetectFirstInvalidMethodRegister(t *testing.T) {
	methodNames := loadOopNames(t, "data/method.oops")
	interp := loadTestInterpreter(t)

	for interp.cycleCount = 0; interp.cycleCount < 2000; interp.cycleCount++ {
		if !isCompiledMethod(interp, interp.method) {
			t.Fatalf("invalid method before cycle %d: method=0x%04X class=0x%04X activeContext=0x%04X homeContext=0x%04X",
				interp.cycleCount, interp.method, interp.fetchClassOf(interp.method),
				interp.activeContext, interp.homeContext)
		}

		previousContext := interp.activeContext
		previousHome := interp.homeContext
		previousMethod := interp.method
		previousIP := interp.instructionPointer
		previousSP := interp.stackPointer

		interp.checkProcessSwitch()
		interp.currentBytecode = interp.fetchBytecode()
		interp.dispatchOnThisBytecode()

		if !isCompiledMethod(interp, interp.method) {
			t.Fatalf("method became invalid at cycle %d after bytecode=%d in ctx=0x%04X home=0x%04X method=0x%04X(%s) ip=%d sp=%d; new method=0x%04X class=0x%04X activeContext=0x%04X homeContext=0x%04X isBlock=%v",
				interp.cycleCount, interp.currentBytecode, previousContext, previousHome,
				previousMethod, methodNames[previousMethod], previousIP, previousSP,
				interp.method, interp.fetchClassOf(interp.method), interp.activeContext,
				interp.homeContext, interp.isBlockContext(interp.activeContext))
		}
	}
}

func TestTraceAroundInvalidActiveContext(t *testing.T) {
	t.Skip("diagnostic test retained for manual investigation")
	methodNames := loadOopNames(t, "data/method.oops")
	interp := loadTestInterpreter(t)

	for interp.cycleCount = 0; interp.cycleCount < 9600; interp.cycleCount++ {
		if interp.cycleCount >= 9588 {
			t.Logf("before cycle=%d ctx=0x%04X class=0x%04X method=0x%04X(%s) ip=%d sp=%d stackTop=0x%04X",
				interp.cycleCount, interp.activeContext, interp.fetchClassOf(interp.activeContext),
				interp.method, methodNames[interp.method], interp.instructionPointer,
				interp.stackPointer, interp.stackTop())
		}

		interp.checkProcessSwitch()
		interp.currentBytecode = interp.fetchBytecode()

		if interp.cycleCount >= 9588 {
			t.Logf("fetched cycle=%d bytecode=%d selector=%q activeContext=0x%04X",
				interp.cycleCount, interp.currentBytecode,
				symbolString(interp, interp.messageSelector), interp.activeContext)
		}

		interp.dispatchOnThisBytecode()

		if interp.cycleCount >= 9588 {
			t.Logf("after cycle=%d ctx=0x%04X class=0x%04X method=0x%04X(%s) ip=%d sp=%d",
				interp.cycleCount, interp.activeContext, interp.fetchClassOf(interp.activeContext),
				interp.method, methodNames[interp.method], interp.instructionPointer, interp.stackPointer)
		}

		if !isMethodOrBlockContext(interp, interp.activeContext) {
			t.Fatalf("invalid activeContext after cycle %d: 0x%04X class=0x%04X",
				interp.cycleCount, interp.activeContext, interp.fetchClassOf(interp.activeContext))
		}
	}
}

func TestDumpDoesNotUnderstandMethod(t *testing.T) {
	t.Skip("diagnostic test retained for manual investigation")
	interp := loadTestInterpreter(t)
	method := uint16(0x7910)

	header := interp.headerOf(method)
	literalCount := interp.literalCountOf(method)
	tempCount := interp.temporaryCountOf(method)
	argCount := interp.argumentCountOf(method)
	initialIP := interp.initialInstructionPointerOfMethod(method)

	t.Logf("method=0x%04X header=0x%04X literalCount=%d tempCount=%d argCount=%d initialIP=%d byteLength=%d",
		method, header, literalCount, tempCount, argCount, initialIP, interp.memory.FetchByteLengthOf(method))

	for i := 0; i < literalCount; i++ {
		lit := interp.literalOfMethod(i, method)
		t.Logf("literal[%d]=0x%04X string=%q", i, lit, symbolString(interp, lit))
	}

	for i := 0; i < 20; i++ {
		t.Logf("byte[%d]=%d", i, interp.fetchByte(i, method))
	}
}

func TestDumpStartupCrashMethod(t *testing.T) {
	t.Skip("diagnostic test retained for manual investigation")
	interp := loadTestInterpreter(t)
	method := uint16(0x021E)

	header := interp.headerOf(method)
	literalCount := interp.literalCountOf(method)
	tempCount := interp.temporaryCountOf(method)
	argCount := interp.argumentCountOf(method)
	largeContext := interp.largeContextFlagOf(method)
	initialIP := interp.initialInstructionPointerOfMethod(method)

	t.Logf("method=0x%04X header=0x%04X literalCount=%d tempCount=%d argCount=%d large=%d initialIP=%d byteLength=%d",
		method, header, literalCount, tempCount, argCount, largeContext, initialIP, interp.memory.FetchByteLengthOf(method))

	for i := 0; i < literalCount; i++ {
		lit := interp.literalOfMethod(i, method)
		t.Logf("literal[%d]=0x%04X string=%q", i, lit, symbolString(interp, lit))
	}
}

func TestLookupPointYMethod(t *testing.T) {
	methodNames := loadOopNames(t, "data/method.oops")
	interp := loadTestInterpreter(t)

	selector := interp.fetchPointer((207-176)*2, om.SpecialSelectorsPointer)
	if got := symbolString(interp, selector); got != "y" {
		t.Fatalf("expected selector y, got %q (0x%04X)", got, selector)
	}

	interp.messageSelector = selector
	interp.lookupMethodInClass(om.ClassPointPointer)

	t.Logf("lookup Point>>y => method=0x%04X (%s) primitive=%d",
		interp.newMethod, methodNames[interp.newMethod], interp.primitiveIndex)

	if interp.newMethod != 0x8BAC {
		t.Fatalf("expected Point>>y = 0x8BAC, got 0x%04X", interp.newMethod)
	}
}

func TestTraceAroundMethodCorruption(t *testing.T) {
	t.Skip("diagnostic test retained for manual investigation")
	methodNames := loadOopNames(t, "data/method.oops")
	classNames := loadOopNames(t, "data/class.oops")
	interp := loadTestInterpreter(t)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	for interp.cycleCount = 0; interp.cycleCount < 140; interp.cycleCount++ {
		if interp.cycleCount >= 124 {
			receiver := interp.stackValue(0)
			receiverClass := interp.fetchClassOf(receiver)
			t.Logf("before cycle=%d ctx=0x%04X method=0x%04X(%s) ip=%d sp=%d nextReceiver=0x%04X class=0x%04X(%s)",
				interp.cycleCount, interp.activeContext, interp.method, methodNames[interp.method],
				interp.instructionPointer, interp.stackPointer, receiver, receiverClass, classNames[receiverClass])
		}

		interp.checkProcessSwitch()
		interp.currentBytecode = interp.fetchBytecode()

		if interp.cycleCount >= 124 {
			t.Logf("fetched cycle=%d bytecode=%d", interp.cycleCount, interp.currentBytecode)
		}

		interp.dispatchOnThisBytecode()

		if interp.cycleCount >= 124 {
			t.Logf("after cycle=%d ctx=0x%04X method=0x%04X(%s) ip=%d sp=%d",
				interp.cycleCount, interp.activeContext, interp.method, methodNames[interp.method],
				interp.instructionPointer, interp.stackPointer)
		}
	}
}

func TestMethodCorruptionWithoutCache(t *testing.T) {
	t.Skip("diagnostic test retained for manual investigation")
	methodNames := loadOopNames(t, "data/method.oops")
	interp := loadTestInterpreter(t)

	for interp.cycleCount = 0; interp.cycleCount < 131; interp.cycleCount++ {
		interp.checkProcessSwitch()
		interp.currentBytecode = interp.fetchBytecode()
		if interp.cycleCount == 129 {
			interp.methodCache = [methodCacheSize]uint16{}
		}
		interp.dispatchOnThisBytecode()
	}

	t.Logf("after cycle 129 ctx=0x%04X method=0x%04X(%s) ip=%d sp=%d",
		interp.activeContext, interp.method, methodNames[interp.method],
		interp.instructionPointer, interp.stackPointer)

	if interp.method != 0x8BAC {
		t.Fatalf("expected Point>>y after cycle 129 without cache, got 0x%04X", interp.method)
	}
}

func TestLogStateAtTwoMillionCycles(t *testing.T) {
	methodNames := loadOopNames(t, "data/method.oops")
	interp := loadTestInterpreter(t)

	for interp.cycleCount = 0; interp.cycleCount < 2000000; interp.cycleCount++ {
		interp.checkProcessSwitch()
		interp.currentBytecode = interp.fetchBytecode()
		interp.dispatchOnThisBytecode()
	}

	t.Logf("cycle=%d activeContext=0x%04X method=0x%04X(%s) receiver=0x%04X ip=%d sp=%d bytecode=%d",
		interp.cycleCount, interp.activeContext, interp.method, methodNames[interp.method],
		interp.receiver, interp.instructionPointer, interp.stackPointer, interp.currentBytecode)

	ctx := interp.activeContext
	for depth := 0; depth < 20 && ctx != om.NilPointer; depth++ {
		home := ctx
		if interp.isBlockContext(ctx) {
			home = interp.fetchPointer(HomeIndex, ctx)
		}
		method := interp.fetchPointer(MethodIndex, home)
		sender := interp.fetchPointer(SenderIndex, home)
		t.Logf("senderChain[%d]: ctx=0x%04X home=0x%04X method=0x%04X(%s) sender=0x%04X",
			depth, ctx, home, method, methodNames[method], sender)
		ctx = sender
	}
}

func TestFindFirstSubscriptError(t *testing.T) {
	methodNames := loadOopNames(t, "data/method.oops")
	interp := loadTestInterpreter(t)

	for interp.cycleCount = 0; interp.cycleCount < 500000; interp.cycleCount++ {
		interp.checkProcessSwitch()
		interp.currentBytecode = interp.fetchBytecode()
		interp.dispatchOnThisBytecode()

		if interp.method == 0x78EA || interp.method == 0x7916 {
			t.Logf("cycle=%d activeContext=0x%04X method=0x%04X(%s) receiver=0x%04X ip=%d sp=%d bytecode=%d",
				interp.cycleCount, interp.activeContext, interp.method, methodNames[interp.method],
				interp.receiver, interp.instructionPointer, interp.stackPointer, interp.currentBytecode)

			atPutContext := interp.fetchPointer(SenderIndex, interp.activeContext)
			if atPutContext != om.NilPointer {
				atPutReceiver := interp.fetchPointer(ReceiverIndex, atPutContext)
				atPutIndex := interp.fetchPointer(TempFrameStart, atPutContext)
				atPutValue := interp.fetchPointer(TempFrameStart+1, atPutContext)
				t.Logf("at:put: receiver=0x%04X class=0x%04X indexArg=0x%04X valueArg=0x%04X wordLen=%d byteLen=%d",
					atPutReceiver, interp.fetchClassOf(atPutReceiver), atPutIndex, atPutValue,
					interp.fetchWordLengthOf(atPutReceiver), interp.memory.FetchByteLengthOf(atPutReceiver))
				t.Logf("at:put: receiver pointerFields=%v oddLength=%v segment=%d location=%d",
					interp.memory.HasPointerFields(atPutReceiver), interp.memory.HasOddLength(atPutReceiver),
					interp.memory.Segment(atPutReceiver), interp.memory.Location(atPutReceiver))
			}

			ctx := interp.activeContext
			for depth := 0; depth < 12 && ctx != om.NilPointer; depth++ {
				home := ctx
				if interp.isBlockContext(ctx) {
					home = interp.fetchPointer(HomeIndex, ctx)
				}
				method := interp.fetchPointer(MethodIndex, home)
				sender := interp.fetchPointer(SenderIndex, home)
				t.Logf("senderChain[%d]: ctx=0x%04X home=0x%04X method=0x%04X(%s) sender=0x%04X",
					depth, ctx, home, method, methodNames[method], sender)
				ctx = sender
			}
			return
		}
	}

	t.Fatalf("did not encounter subscript error in first 500000 cycles")
}
