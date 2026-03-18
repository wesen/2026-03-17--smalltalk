// Package interpreter implements the Smalltalk-80 bytecode interpreter
// as specified in Blue Book Chapters 27-28.
package interpreter

import (
	"fmt"

	om "github.com/wesen/st80/pkg/objectmemory"
)

// Context field indices (Blue Book p.581)
const (
	SenderIndex             = 0
	InstructionPointerIndex = 1
	StackPointerIndex       = 2
	MethodIndex             = 3
	// MethodContext field 4 is unused
	ReceiverIndex  = 5
	TempFrameStart = 6

	// BlockContext fields
	CallerIndex             = 0
	BlockArgumentCountIndex = 3
	InitialIPIndex          = 4
	HomeIndex               = 5
)

// Class field indices (Blue Book p.587)
const (
	SuperclassIndex            = 0
	MessageDictionaryIndex     = 1
	InstanceSpecificationIndex = 2
)

// MethodDictionary field indices
const (
	MethodArrayIndex = 1
	SelectorStart    = 2
)

// CompiledMethod field indices
const (
	HeaderIndex  = 0
	LiteralStart = 1
)

// Message field indices
const (
	MessageSelectorIndex  = 0
	MessageArgumentsIndex = 1
	MessageSize           = 2
)

// Association field index
const (
	ValueIndex = 1
)

// Form field indices.
const (
	FormBitsIndex   = 0
	FormWidthIndex  = 1
	FormHeightIndex = 2
	FormOffsetIndex = 3
)

// Point field indices.
const (
	PointXIndex = 0
	PointYIndex = 1
)

// Rectangle field indices.
const (
	RectangleOriginIndex = 0
	RectangleCornerIndex = 1
)

// BitBlt field indices. These match the Blue Book BitBlt state vector.
const (
	BitBltDestFormIndex        = 0
	BitBltSourceFormIndex      = 1
	BitBltHalftoneFormIndex    = 2
	BitBltCombinationRuleIndex = 3
	BitBltDestXIndex           = 4
	BitBltDestYIndex           = 5
	BitBltWidthIndex           = 6
	BitBltHeightIndex          = 7
	BitBltClipXIndex           = 8
	BitBltClipYIndex           = 9
	BitBltClipWidthIndex       = 10
	BitBltClipHeightIndex      = 11
	BitBltSourceXIndex         = 12
	BitBltSourceYIndex         = 13
)

// Method cache size (must be power of 2 * 4)
const methodCacheSize = 1024

// Interpreter is the Smalltalk-80 bytecode interpreter.
type Interpreter struct {
	memory *om.ObjectMemory

	// Context registers (Blue Book p.583)
	activeContext      uint16
	homeContext        uint16
	method             uint16
	receiver           uint16
	instructionPointer int
	stackPointer       int

	// Message sending registers (Blue Book p.587)
	messageSelector uint16
	argumentCount   int
	newMethod       uint16
	primitiveIndex  int

	// Current bytecode
	currentBytecode byte

	// Method cache: 4 entries per slot [selector, class, method, primIndex]
	methodCache [methodCacheSize]uint16

	// Primitive success flag
	success bool

	// Process scheduling registers (Blue Book p.642)
	newProcessWaiting bool
	newProcess        uint16
	semaphoreList     [256]uint16
	semaphoreIndex    int

	// Minimal VM-side display/input bookkeeping used by I/O primitives.
	displayScreen uint16
	cursorForm    uint16
	cursorLinked  bool

	// Cycle counter for tracing
	cycleCount uint64

	// BitBlt diagnostics for long-run primitive failures.
	lastCopyBitsFailure string
	lastCopyBitsBitBlt  uint16
	lastCopyBitsCycle   uint64
}

// DisplaySnapshot captures the current designated display form in host-friendly
// terms for the UI layer.
type DisplaySnapshot struct {
	FormPointer uint16
	Width       int
	Height      int
	Raster      int
	Words       []uint16
}

// Scheduling constants
const (
	ProcessListsIndex     = 0
	ActiveProcessIndex    = 1
	FirstLinkIndex        = 0
	LastLinkIndex         = 1
	ExcessSignalsIndex    = 2
	NextLinkIndex         = 0
	SuspendedContextIndex = 1
	PriorityIndex         = 2
	MyListIndex           = 3
)

// New creates a new Interpreter with the given object memory.
func New(memory *om.ObjectMemory) *Interpreter {
	return &Interpreter{
		memory:  memory,
		success: true,
	}
}

// ---- Bit extraction (Blue Book p.575) ----
//
// Method headers, header extensions, and class instance specifications are
// stored in SmallInteger payloads, so Blue Book bit indices refer to the 15-bit
// decoded integer value, not a raw 16-bit word.

func extractBits(firstBitIndex, lastBitIndex int, ofValue uint16) int {
	return int((ofValue >> (14 - lastBitIndex)) & ((1 << (lastBitIndex - firstBitIndex + 1)) - 1))
}

func highByteOf(val uint16) int {
	return int(val >> 8)
}

func lowByteOf(val uint16) int {
	return int(val & 0xFF)
}

// ---- Object memory helpers ----

func (interp *Interpreter) fetchPointer(fieldIndex int, ofObject uint16) uint16 {
	return interp.memory.FetchPointer(fieldIndex, ofObject)
}

func (interp *Interpreter) storePointer(fieldIndex int, ofObject uint16, withValue uint16) {
	interp.memory.StorePointer(fieldIndex, ofObject, withValue)
}

func (interp *Interpreter) fetchWordLengthOf(oop uint16) int {
	return interp.memory.FetchWordLengthOf(oop)
}

func (interp *Interpreter) fetchClassOf(oop uint16) uint16 {
	return interp.memory.FetchClassOf(oop)
}

func (interp *Interpreter) fetchByte(byteIndex int, ofObject uint16) byte {
	return interp.memory.FetchByte(byteIndex, ofObject)
}

func (interp *Interpreter) fetchWord(wordIndex int, ofObject uint16) uint16 {
	return interp.memory.FetchWord(wordIndex, ofObject)
}

func (interp *Interpreter) storeWord(wordIndex int, ofObject uint16, withValue uint16) {
	interp.memory.StoreWord(wordIndex, ofObject, withValue)
}

func (interp *Interpreter) instantiateClassWithPointers(classPointer uint16, instanceSize int) uint16 {
	return interp.memory.InstantiateClass(classPointer, instanceSize, true)
}

func (interp *Interpreter) instantiateClassWithWords(classPointer uint16, instanceSize int) uint16 {
	return interp.memory.InstantiateClassWithWords(classPointer, instanceSize)
}

func (interp *Interpreter) instantiateClassWithBytes(classPointer uint16, instanceSize int) uint16 {
	return interp.memory.InstantiateClassWithBytes(classPointer, instanceSize)
}

// ---- Context management (Blue Book p.582-585) ----

func (interp *Interpreter) isBlockContext(contextPointer uint16) bool {
	if om.IsSmallInteger(contextPointer) || !interp.memory.ValidOop(contextPointer) {
		return false
	}
	if interp.fetchWordLengthOf(contextPointer) <= MethodIndex {
		return false
	}
	methodOrArguments := interp.fetchPointer(MethodIndex, contextPointer)
	return om.IsSmallInteger(methodOrArguments)
}

func (interp *Interpreter) isMethodContext(contextPointer uint16) bool {
	if om.IsSmallInteger(contextPointer) || !interp.memory.ValidOop(contextPointer) {
		return false
	}
	if interp.fetchWordLengthOf(contextPointer) <= MethodIndex {
		return false
	}
	methodPointer := interp.fetchPointer(MethodIndex, contextPointer)
	if om.IsSmallInteger(methodPointer) || !interp.memory.ValidOop(methodPointer) {
		return false
	}
	return interp.fetchClassOf(methodPointer) == om.ClassCompiledMethodPointer
}

func (interp *Interpreter) fetchContextRegisters() {
	if interp.isBlockContext(interp.activeContext) {
		interp.homeContext = interp.fetchPointer(HomeIndex, interp.activeContext)
	} else {
		interp.homeContext = interp.activeContext
	}
	interp.receiver = interp.fetchPointer(ReceiverIndex, interp.homeContext)
	interp.method = interp.fetchPointer(MethodIndex, interp.homeContext)
	ipVal := interp.fetchPointer(InstructionPointerIndex, interp.activeContext)
	interp.instructionPointer = int(om.SmallIntegerValue(ipVal)) - 1
	spVal := interp.fetchPointer(StackPointerIndex, interp.activeContext)
	interp.stackPointer = int(om.SmallIntegerValue(spVal)) + TempFrameStart - 1
}

func (interp *Interpreter) storeContextRegisters() {
	interp.storePointer(InstructionPointerIndex, interp.activeContext,
		om.SmallIntegerOop(int16(interp.instructionPointer+1)))
	interp.storePointer(StackPointerIndex, interp.activeContext,
		om.SmallIntegerOop(int16(interp.stackPointer-TempFrameStart+1)))
}

func (interp *Interpreter) newActiveContext(aContext uint16) {
	interp.storeContextRegisters()
	interp.activeContext = aContext
	interp.fetchContextRegisters()
}

// ---- Stack operations (Blue Book p.585) ----

func (interp *Interpreter) push(object uint16) {
	interp.stackPointer++
	// No bounds check here — trust the image
	interp.storePointer(interp.stackPointer, interp.activeContext, object)
}

func (interp *Interpreter) popStack() uint16 {
	top := interp.fetchPointer(interp.stackPointer, interp.activeContext)
	interp.stackPointer--
	return top
}

func (interp *Interpreter) stackTop() uint16 {
	return interp.fetchPointer(interp.stackPointer, interp.activeContext)
}

func (interp *Interpreter) stackValue(offset int) uint16 {
	return interp.fetchPointer(interp.stackPointer-offset, interp.activeContext)
}

func (interp *Interpreter) pop(number int) {
	interp.stackPointer -= number
}

func (interp *Interpreter) unPop(number int) {
	interp.stackPointer += number
}

// ---- Bytecode fetch ----

func (interp *Interpreter) fetchBytecode() byte {
	b := interp.fetchByte(interp.instructionPointer, interp.method)
	interp.instructionPointer++
	return b
}

// ---- CompiledMethod access (Blue Book p.577-580) ----

func (interp *Interpreter) headerOf(methodPointer uint16) uint16 {
	// CompiledMethod headers are stored as SmallIntegers; decode the 15-bit
	// payload before applying the Blue Book bit layout.
	return uint16(om.SmallIntegerValue(interp.fetchPointer(HeaderIndex, methodPointer)))
}

func (interp *Interpreter) literal(offset int) uint16 {
	return interp.fetchPointer(offset+LiteralStart, interp.method)
}

func (interp *Interpreter) literalOfMethod(offset int, methodPointer uint16) uint16 {
	return interp.fetchPointer(offset+LiteralStart, methodPointer)
}

func (interp *Interpreter) temporaryCountOf(methodPointer uint16) int {
	header := interp.headerOf(methodPointer)
	return extractBits(3, 7, header)
}

func (interp *Interpreter) largeContextFlagOf(methodPointer uint16) int {
	header := interp.headerOf(methodPointer)
	return extractBits(8, 8, header)
}

func (interp *Interpreter) literalCountOfHeader(headerPointer uint16) int {
	return extractBits(9, 14, headerPointer)
}

func (interp *Interpreter) literalCountOf(methodPointer uint16) int {
	return interp.literalCountOfHeader(interp.headerOf(methodPointer))
}

func (interp *Interpreter) objectPointerCountOf(methodPointer uint16) int {
	return interp.literalCountOf(methodPointer) + LiteralStart
}

func (interp *Interpreter) initialInstructionPointerOfMethod(methodPointer uint16) int {
	return (interp.literalCountOf(methodPointer)+LiteralStart)*2 + 1
}

func (interp *Interpreter) flagValueOf(methodPointer uint16) int {
	header := interp.headerOf(methodPointer)
	return extractBits(0, 2, header)
}

func (interp *Interpreter) fieldIndexOf(methodPointer uint16) int {
	header := interp.headerOf(methodPointer)
	return extractBits(3, 7, header)
}

func (interp *Interpreter) headerExtensionOf(methodPointer uint16) uint16 {
	literalCount := interp.literalCountOf(methodPointer)
	return uint16(om.SmallIntegerValue(interp.literalOfMethod(literalCount-2, methodPointer)))
}

func (interp *Interpreter) argumentCountOf(methodPointer uint16) int {
	flagValue := interp.flagValueOf(methodPointer)
	if flagValue < 5 {
		return flagValue
	}
	if flagValue < 7 {
		return 0
	}
	return extractBits(2, 6, interp.headerExtensionOf(methodPointer))
}

func (interp *Interpreter) primitiveIndexOf(methodPointer uint16) int {
	flagValue := interp.flagValueOf(methodPointer)
	if flagValue == 7 {
		return extractBits(7, 14, interp.headerExtensionOf(methodPointer))
	}
	return 0
}

func (interp *Interpreter) methodClassOf(methodPointer uint16) uint16 {
	literalCount := interp.literalCountOf(methodPointer)
	association := interp.literalOfMethod(literalCount-1, methodPointer)
	return interp.fetchPointer(ValueIndex, association)
}

// ---- Temporary and literal access ----

func (interp *Interpreter) temporary(offset int) uint16 {
	return interp.fetchPointer(offset+TempFrameStart, interp.homeContext)
}

// ---- Class hierarchy (Blue Book p.586-589) ----

func (interp *Interpreter) superclassOf(classPointer uint16) uint16 {
	return interp.fetchPointer(SuperclassIndex, classPointer)
}

func (interp *Interpreter) hash(objectPointer uint16) int {
	return int(objectPointer >> 1)
}

func (interp *Interpreter) instanceSpecificationOf(classPointer uint16) uint16 {
	return uint16(om.SmallIntegerValue(interp.fetchPointer(InstanceSpecificationIndex, classPointer)))
}

func (interp *Interpreter) isPointers(classPointer uint16) bool {
	return extractBits(0, 0, interp.instanceSpecificationOf(classPointer)) == 1
}

func (interp *Interpreter) isWords(classPointer uint16) bool {
	return extractBits(1, 1, interp.instanceSpecificationOf(classPointer)) == 1
}

func (interp *Interpreter) isIndexable(classPointer uint16) bool {
	return extractBits(2, 2, interp.instanceSpecificationOf(classPointer)) == 1
}

func (interp *Interpreter) fixedFieldsOf(classPointer uint16) int {
	return extractBits(4, 14, interp.instanceSpecificationOf(classPointer))
}

// ---- Method lookup (Blue Book p.587-589) ----

func (interp *Interpreter) lookupMethodInDictionary(dictionary uint16) bool {
	length := interp.fetchWordLengthOf(dictionary)
	mask := length - SelectorStart - 1
	index := (mask & interp.hash(interp.messageSelector)) + SelectorStart
	wrapAround := false
	for {
		nextSelector := interp.fetchPointer(index, dictionary)
		if nextSelector == om.NilPointer {
			return false
		}
		if nextSelector == interp.messageSelector {
			methodArray := interp.fetchPointer(MethodArrayIndex, dictionary)
			interp.newMethod = interp.fetchPointer(index-SelectorStart, methodArray)
			interp.primitiveIndex = interp.primitiveIndexOf(interp.newMethod)
			return true
		}
		index++
		if index == length {
			if wrapAround {
				return false
			}
			wrapAround = true
			index = SelectorStart
		}
	}
}

func (interp *Interpreter) lookupMethodInClass(class uint16) {
	currentClass := class
	for currentClass != om.NilPointer {
		dictionary := interp.fetchPointer(MessageDictionaryIndex, currentClass)
		if interp.lookupMethodInDictionary(dictionary) {
			return
		}
		currentClass = interp.superclassOf(currentClass)
	}
	if interp.messageSelector == om.DoesNotUnderstandSelector {
		panic("Recursive not understood error encountered")
	}
	interp.createActualMessage()
	interp.messageSelector = om.DoesNotUnderstandSelector
	interp.lookupMethodInClass(class)
}

func (interp *Interpreter) createActualMessage() {
	argumentArray := interp.instantiateClassWithPointers(om.ClassArrayPointer, interp.argumentCount)
	message := interp.instantiateClassWithPointers(om.ClassMessagePointer, MessageSize)
	interp.storePointer(MessageSelectorIndex, message, interp.messageSelector)
	interp.storePointer(MessageArgumentsIndex, message, argumentArray)
	interp.transfer(interp.argumentCount,
		interp.stackPointer-interp.argumentCount+1, interp.activeContext,
		0, argumentArray)
	interp.pop(interp.argumentCount)
	interp.push(message)
	interp.argumentCount = 1
}

func (interp *Interpreter) transfer(count, firstFrom int, fromOop uint16, firstTo int, toOop uint16) {
	fromIndex := firstFrom
	toIndex := firstTo
	lastFrom := firstFrom + count
	for fromIndex < lastFrom {
		oop := interp.fetchPointer(fromIndex, fromOop)
		interp.storePointer(toIndex, toOop, oop)
		interp.storePointer(fromIndex, fromOop, om.NilPointer)
		fromIndex++
		toIndex++
	}
}

// ---- Method cache ----

func (interp *Interpreter) findNewMethodInClass(class uint16) {
	// Blue Book hash is (((selector bitAnd: class) bitAnd: 16rFF) bitShift: 2)
	// + 1 for 1-based Smalltalk Arrays. This Go array is 0-based, so omit the
	// final +1 but keep the << 2 to allocate 4 consecutive words per entry.
	h := ((int(interp.messageSelector) & int(class)) & 0xFF) << 2
	if h >= 0 && h+3 < methodCacheSize &&
		interp.methodCache[h] == interp.messageSelector &&
		interp.methodCache[h+1] == class {
		interp.newMethod = interp.methodCache[h+2]
		interp.primitiveIndex = int(interp.methodCache[h+3])
	} else {
		interp.lookupMethodInClass(class)
		if h >= 0 && h+3 < methodCacheSize {
			interp.methodCache[h] = interp.messageSelector
			interp.methodCache[h+1] = class
			interp.methodCache[h+2] = interp.newMethod
			interp.methodCache[h+3] = uint16(interp.primitiveIndex)
		}
	}
}

// ---- Sending messages (Blue Book p.604-607) ----

func (interp *Interpreter) sendSelector(selector uint16, count int) {
	interp.messageSelector = selector
	interp.argumentCount = count
	newReceiver := interp.stackValue(interp.argumentCount)
	interp.sendSelectorToClass(interp.fetchClassOf(newReceiver))
}

func (interp *Interpreter) sendSelectorToClass(classPointer uint16) {
	interp.findNewMethodInClass(classPointer)
	interp.executeNewMethod()
}

func (interp *Interpreter) executeNewMethod() {
	if interp.primitiveResponse() {
		return
	}
	interp.activateNewMethod()
}

func (interp *Interpreter) activateNewMethod() {
	// Always use large contexts to avoid overflow during debugging.
	// The Blue Book uses largeContextFlagOf to choose 12 vs 32, but the
	// flag in the image may not be set correctly for all methods.
	contextSize := TempFrameStart + 32
	newContext := interp.instantiateClassWithPointers(om.ClassMethodContextPointer, contextSize)
	interp.storePointer(SenderIndex, newContext, interp.activeContext)
	interp.storePointer(InstructionPointerIndex, newContext,
		om.SmallIntegerOop(int16(interp.initialInstructionPointerOfMethod(interp.newMethod))))
	interp.storePointer(StackPointerIndex, newContext,
		om.SmallIntegerOop(int16(interp.temporaryCountOf(interp.newMethod))))
	interp.storePointer(MethodIndex, newContext, interp.newMethod)

	// Transfer receiver + arguments from old context to new
	interp.transfer(interp.argumentCount+1,
		interp.stackPointer-interp.argumentCount, interp.activeContext,
		ReceiverIndex, newContext)
	interp.pop(interp.argumentCount + 1)
	interp.newActiveContext(newContext)
}

// ---- Return (Blue Book p.609-610) ----

func (interp *Interpreter) returnValueTo(resultPointer uint16, contextPointer uint16) {
	if contextPointer == om.NilPointer {
		interp.push(interp.activeContext)
		interp.push(resultPointer)
		interp.sendSelector(om.CannotReturnSelector, 1)
		return
	}
	sendersIP := interp.fetchPointer(InstructionPointerIndex, contextPointer)
	if sendersIP == om.NilPointer {
		interp.push(interp.activeContext)
		interp.push(resultPointer)
		interp.sendSelector(om.CannotReturnSelector, 1)
		return
	}
	interp.returnToActiveContext(contextPointer)
	interp.push(resultPointer)
}

func (interp *Interpreter) returnToActiveContext(aContext uint16) {
	oldContext := interp.activeContext
	interp.nilContextFields()
	interp.activeContext = aContext
	interp.fetchContextRegisters()
	interp.maybeRecycleContext(oldContext, interp.activeContext)
}

func (interp *Interpreter) nilContextFields() {
	interp.storePointer(SenderIndex, interp.activeContext, om.NilPointer)
	interp.storePointer(InstructionPointerIndex, interp.activeContext, om.NilPointer)
}

func (interp *Interpreter) contextReachableFrom(root uint16, target uint16, visited map[uint16]bool) bool {
	if root == om.NilPointer || om.IsSmallInteger(root) || visited[root] || !interp.memory.ValidOop(root) {
		return false
	}
	if root == target {
		return true
	}
	if !interp.isMethodContext(root) && !interp.isBlockContext(root) {
		return false
	}
	visited[root] = true

	fieldCount := interp.fetchWordLengthOf(root)
	for i := 0; i < fieldCount; i++ {
		field := interp.fetchPointer(i, root)
		if field == target {
			return true
		}
		if interp.contextReachableFrom(field, target, visited) {
			return true
		}
	}
	return false
}

func (interp *Interpreter) maybeRecycleContext(context uint16, activeRoot uint16) {
	if om.IsSmallInteger(context) || !interp.memory.ValidOop(context) {
		return
	}
	if !interp.isMethodContext(context) {
		return
	}
	if interp.contextReachableFrom(activeRoot, context, map[uint16]bool{}) {
		return
	}
	interp.memory.FreeObject(context)
}

func (interp *Interpreter) sender() uint16 {
	return interp.fetchPointer(SenderIndex, interp.homeContext)
}

func (interp *Interpreter) caller() uint16 {
	return interp.fetchPointer(SenderIndex, interp.activeContext)
}

// ---- Primitive support ----

func (interp *Interpreter) initPrimitive() {
	interp.success = true
}

func (interp *Interpreter) primitiveFail() {
	interp.success = false
}

func (interp *Interpreter) successCheck() bool {
	return interp.success
}

func (interp *Interpreter) popInteger() int {
	integerPointer := interp.popStack()
	if om.IsSmallInteger(integerPointer) {
		return int(om.SmallIntegerValue(integerPointer))
	}
	interp.success = false
	return 0
}

func (interp *Interpreter) pushInteger(integerValue int) {
	if integerValue >= -16384 && integerValue <= 16383 {
		interp.push(om.SmallIntegerOop(int16(integerValue)))
	} else {
		interp.success = false
	}
}

func (interp *Interpreter) characterCodeOf(character uint16) (byte, bool) {
	if !interp.memory.ValidOop(character) || interp.fetchClassOf(character) != om.ClassCharacterPointer {
		return 0, false
	}
	for i := 0; i < 256; i++ {
		if interp.fetchPointer(i, om.CharacterTablePointer) == character {
			return byte(i), true
		}
	}
	return 0, false
}

// ---- Primitive dispatch (Blue Book p.620-621) ----

func (interp *Interpreter) primitiveResponse() bool {
	flagValue := interp.flagValueOf(interp.newMethod)
	if flagValue == 5 {
		// Quick return self -- no-op, receiver already on stack
		return true
	}
	if flagValue == 6 {
		// Quick return instance variable
		thisReceiver := interp.popStack()
		fieldIndex := interp.fieldIndexOf(interp.newMethod)
		interp.push(interp.fetchPointer(fieldIndex, thisReceiver))
		return true
	}
	if flagValue == 7 || interp.primitiveIndex > 0 {
		interp.initPrimitive()
		interp.dispatchPrimitives()
		return interp.success
	}
	return false
}

func (interp *Interpreter) dispatchPrimitives() {
	if interp.primitiveIndex < 60 {
		interp.dispatchArithmeticPrimitives()
	} else if interp.primitiveIndex < 68 {
		interp.dispatchSubscriptAndStreamPrimitives()
	} else if interp.primitiveIndex < 80 {
		interp.dispatchStorageManagementPrimitives()
	} else if interp.primitiveIndex < 90 {
		interp.dispatchControlPrimitives()
	} else if interp.primitiveIndex < 110 {
		interp.dispatchInputOutputPrimitives()
	} else if interp.primitiveIndex < 128 {
		interp.dispatchSystemPrimitives()
	} else {
		interp.primitiveFail()
	}
}

// ---- Arithmetic primitives (Blue Book p.621-625) ----

func (interp *Interpreter) dispatchArithmeticPrimitives() {
	if interp.primitiveIndex < 20 {
		interp.dispatchIntegerPrimitives()
	} else if interp.primitiveIndex < 40 {
		interp.primitiveFail() // Large integer -- optional
	} else if interp.primitiveIndex < 60 {
		interp.dispatchFloatPrimitives()
	}
}

func (interp *Interpreter) dispatchIntegerPrimitives() {
	switch interp.primitiveIndex {
	case 1:
		interp.primitiveAdd()
	case 2:
		interp.primitiveSubtract()
	case 3:
		interp.primitiveLessThan()
	case 4:
		interp.primitiveGreaterThan()
	case 5:
		interp.primitiveLessOrEqual()
	case 6:
		interp.primitiveGreaterOrEqual()
	case 7:
		interp.primitiveEqual()
	case 8:
		interp.primitiveNotEqual()
	case 9:
		interp.primitiveMultiply()
	case 10:
		interp.primitiveDivide()
	case 11:
		interp.primitiveMod()
	case 12:
		interp.primitiveDiv()
	case 13:
		interp.primitiveQuo()
	case 14:
		interp.primitiveBitAnd()
	case 15:
		interp.primitiveBitOr()
	case 16:
		interp.primitiveBitXor()
	case 17:
		interp.primitiveBitShift()
	case 18:
		interp.primitiveMakePoint()
	default:
		interp.primitiveFail()
	}
}

func (interp *Interpreter) primitiveAdd() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		result := rcvr + arg
		if result >= -16384 && result <= 16383 {
			interp.pushInteger(result)
		} else {
			interp.success = false
		}
	}
	if !interp.success {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveSubtract() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		result := rcvr - arg
		if result >= -16384 && result <= 16383 {
			interp.pushInteger(result)
		} else {
			interp.success = false
		}
	}
	if !interp.success {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveMultiply() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		result := rcvr * arg
		if result >= -16384 && result <= 16383 {
			interp.pushInteger(result)
		} else {
			interp.success = false
		}
	}
	if !interp.success {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveDivide() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success && arg != 0 && rcvr%arg == 0 {
		interp.pushInteger(rcvr / arg)
	} else {
		interp.success = false
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveMod() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success && arg != 0 {
		result := rcvr % arg
		// Smalltalk mod rounds toward negative infinity
		if (result != 0) && ((result ^ arg) < 0) {
			result += arg
		}
		interp.pushInteger(result)
	} else {
		interp.success = false
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveDiv() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success && arg != 0 {
		result := rcvr / arg
		// Smalltalk div rounds toward negative infinity
		if (rcvr^arg) < 0 && rcvr%arg != 0 {
			result--
		}
		interp.pushInteger(result)
	} else {
		interp.success = false
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveQuo() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success && arg != 0 {
		interp.pushInteger(rcvr / arg) // Go division truncates toward zero
	} else {
		interp.success = false
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveLessThan() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		if rcvr < arg {
			interp.push(om.TruePointer)
		} else {
			interp.push(om.FalsePointer)
		}
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveGreaterThan() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		if rcvr > arg {
			interp.push(om.TruePointer)
		} else {
			interp.push(om.FalsePointer)
		}
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveLessOrEqual() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		if rcvr <= arg {
			interp.push(om.TruePointer)
		} else {
			interp.push(om.FalsePointer)
		}
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveGreaterOrEqual() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		if rcvr >= arg {
			interp.push(om.TruePointer)
		} else {
			interp.push(om.FalsePointer)
		}
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveEqual() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		if rcvr == arg {
			interp.push(om.TruePointer)
		} else {
			interp.push(om.FalsePointer)
		}
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveNotEqual() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		if rcvr != arg {
			interp.push(om.TruePointer)
		} else {
			interp.push(om.FalsePointer)
		}
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveBitAnd() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		interp.pushInteger(rcvr & arg)
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveBitOr() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		interp.pushInteger(rcvr | arg)
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveBitXor() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		interp.pushInteger(rcvr ^ arg)
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveBitShift() {
	arg := interp.popInteger()
	rcvr := interp.popInteger()
	if interp.success {
		var result int
		if arg >= 0 {
			result = rcvr << uint(arg)
		} else {
			result = rcvr >> uint(-arg)
		}
		if result >= -16384 && result <= 16383 {
			interp.pushInteger(result)
		} else {
			interp.success = false
			interp.unPop(2)
		}
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveMakePoint() {
	arg := interp.popStack()
	rcvr := interp.popStack()
	if om.IsSmallInteger(rcvr) && om.IsSmallInteger(arg) {
		pointResult := interp.instantiateClassWithPointers(om.ClassPointPointer, 2)
		interp.storePointer(0, pointResult, rcvr)
		interp.storePointer(1, pointResult, arg)
		interp.push(pointResult)
	} else {
		interp.unPop(2)
		interp.primitiveFail()
	}
}

// ---- Float primitives (stub) ----

func (interp *Interpreter) dispatchFloatPrimitives() {
	interp.primitiveFail() // Float primitives not yet implemented
}

// ---- Subscript and stream primitives (Blue Book p.627-628) ----

func (interp *Interpreter) dispatchSubscriptAndStreamPrimitives() {
	switch interp.primitiveIndex {
	case 60:
		interp.primitiveAt()
	case 61:
		interp.primitiveAtPut()
	case 62:
		interp.primitiveSize()
	case 63:
		interp.primitiveStringAt()
	case 64:
		interp.primitiveStringAtPut()
	default:
		interp.primitiveFail()
	}
}

func (interp *Interpreter) primitiveAt() {
	index := interp.popInteger()
	rcvr := interp.popStack()
	if !interp.success {
		interp.unPop(2)
		return
	}
	class := interp.fetchClassOf(rcvr)
	interp.checkIndexableBoundsOf(index, rcvr, class)
	if interp.success {
		result := interp.subscriptWith(rcvr, index, class)
		interp.push(result)
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveAtPut() {
	value := interp.popStack()
	index := interp.popInteger()
	rcvr := interp.popStack()
	if !interp.success {
		interp.unPop(3)
		return
	}
	class := interp.fetchClassOf(rcvr)
	interp.checkIndexableBoundsOf(index, rcvr, class)
	if interp.success {
		interp.subscriptStoring(rcvr, index, class, value)
		if interp.success {
			interp.push(value)
		} else {
			interp.unPop(3)
		}
	} else {
		interp.unPop(3)
	}
}

func (interp *Interpreter) primitiveSize() {
	rcvr := interp.popStack()
	class := interp.fetchClassOf(rcvr)
	length := interp.lengthOf(rcvr, class) - interp.fixedFieldsOf(class)
	if length >= -16384 && length <= 16383 {
		interp.pushInteger(length)
	} else {
		interp.unPop(1)
		interp.primitiveFail()
	}
}

func (interp *Interpreter) primitiveStringAt() {
	index := interp.popInteger()
	rcvr := interp.popStack()
	if !interp.success {
		interp.unPop(2)
		return
	}
	class := interp.fetchClassOf(rcvr)
	interp.checkIndexableBoundsOf(index, rcvr, class)
	if interp.success {
		character := interp.subscriptWith(rcvr, index, class)
		// Look up Character in CharacterTable
		charTable := om.CharacterTablePointer
		if om.IsSmallInteger(character) {
			charVal := int(om.SmallIntegerValue(character))
			if charVal >= 0 && charVal < 256 {
				interp.push(interp.fetchPointer(charVal, charTable))
				return
			}
		}
		interp.push(character)
	} else {
		interp.unPop(2)
	}
}

func (interp *Interpreter) primitiveStringAtPut() {
	value := interp.popStack()
	index := interp.popInteger()
	rcvr := interp.popStack()
	if !interp.success {
		interp.unPop(3)
		return
	}
	class := interp.fetchClassOf(rcvr)
	if interp.isPointers(class) || interp.isWords(class) {
		interp.unPop(3)
		interp.primitiveFail()
		return
	}
	interp.checkIndexableBoundsOf(index, rcvr, class)
	if !interp.success {
		interp.unPop(3)
		return
	}
	code, ok := interp.characterCodeOf(value)
	if !ok {
		interp.unPop(3)
		interp.primitiveFail()
		return
	}
	fixedFields := interp.fixedFieldsOf(class)
	interp.memory.StoreByte(index+fixedFields-1, rcvr, code)
	interp.push(value)
}

func (interp *Interpreter) checkIndexableBoundsOf(index int, array uint16, class uint16) {
	if index < 1 {
		interp.success = false
		return
	}
	if index+interp.fixedFieldsOf(class) > interp.lengthOf(array, class) {
		interp.success = false
	}
}

func (interp *Interpreter) lengthOf(array uint16, class uint16) int {
	if interp.isWords(class) {
		return interp.fetchWordLengthOf(array)
	}
	return interp.memory.FetchByteLengthOf(array)
}

func (interp *Interpreter) subscriptWith(array uint16, index int, class uint16) uint16 {
	fixedFields := interp.fixedFieldsOf(class)
	if interp.isWords(class) {
		if interp.isPointers(class) {
			return interp.fetchPointer(index+fixedFields-1, array)
		}
		value := interp.memory.FetchWord(index+fixedFields-1, array)
		return om.SmallIntegerOop(int16(value))
	}
	value := interp.fetchByte(index+fixedFields-1, array)
	return om.SmallIntegerOop(int16(value))
}

func (interp *Interpreter) subscriptStoring(array uint16, index int, class uint16, value uint16) {
	fixedFields := interp.fixedFieldsOf(class)
	if interp.isWords(class) {
		if interp.isPointers(class) {
			interp.storePointer(index+fixedFields-1, array, value)
		} else if om.IsSmallInteger(value) {
			interp.memory.StoreWord(index+fixedFields-1, array, uint16(om.SmallIntegerValue(value)))
		} else {
			interp.primitiveFail()
		}
	} else if om.IsSmallInteger(value) {
		interp.memory.StoreByte(index+fixedFields-1, array, byte(om.SmallIntegerValue(value)))
	} else {
		interp.primitiveFail()
	}
}

// ---- Storage management primitives ----

func (interp *Interpreter) dispatchStorageManagementPrimitives() {
	switch interp.primitiveIndex {
	case 68: // objectAt:
		index := interp.popInteger()
		rcvr := interp.popStack()
		if interp.success {
			interp.push(interp.fetchPointer(index-1, rcvr))
		} else {
			interp.unPop(2)
		}
	case 69: // objectAt:put:
		value := interp.popStack()
		index := interp.popInteger()
		rcvr := interp.popStack()
		if interp.success {
			interp.storePointer(index-1, rcvr, value)
			interp.push(value)
		} else {
			interp.unPop(3)
		}
	case 70: // basicNew, new
		class := interp.popStack()
		size := interp.fixedFieldsOf(class)
		if interp.isIndexable(class) {
			size = 0
		}
		var result uint16
		if interp.isPointers(class) {
			result = interp.instantiateClassWithPointers(class, size)
		} else {
			result = interp.instantiateClassWithWords(class, size)
		}
		interp.push(result)
	case 71: // basicNew:, new:
		sz := interp.popInteger()
		class := interp.popStack()
		if interp.success {
			size := sz + interp.fixedFieldsOf(class)
			var result uint16
			if interp.isPointers(class) {
				result = interp.instantiateClassWithPointers(class, size)
			} else if interp.isWords(class) {
				result = interp.instantiateClassWithWords(class, size)
			} else {
				result = interp.instantiateClassWithBytes(class, size)
			}
			interp.push(result)
		} else {
			interp.unPop(2)
		}
	case 72: // become:
		otherPointer := interp.popStack()
		thisReceiver := interp.popStack()
		if om.IsSmallInteger(otherPointer) || om.IsSmallInteger(thisReceiver) {
			interp.unPop(2)
			interp.primitiveFail()
			return
		}
		interp.memory.SwapPointersOf(thisReceiver, otherPointer)
		interp.push(thisReceiver)
	case 73: // instVarAt:
		index := interp.popInteger()
		rcvr := interp.popStack()
		if interp.success {
			interp.push(interp.fetchPointer(index-1, rcvr))
		} else {
			interp.unPop(2)
		}
	case 74: // instVarAt:put:
		value := interp.popStack()
		index := interp.popInteger()
		rcvr := interp.popStack()
		if interp.success {
			interp.storePointer(index-1, rcvr, value)
			interp.push(value)
		} else {
			interp.unPop(3)
		}
	case 75: // asOop, hash
		rcvr := interp.popStack()
		if om.IsSmallInteger(rcvr) {
			interp.unPop(1)
			interp.primitiveFail()
			return
		}
		interp.push(rcvr | 1)
	case 76: // asObject
		rcvr := interp.popStack()
		if !om.IsSmallInteger(rcvr) {
			interp.unPop(1)
			interp.primitiveFail()
			return
		}
		newOop := rcvr & 0xFFFE
		if !interp.memory.ValidOop(newOop) {
			interp.unPop(1)
			interp.primitiveFail()
			return
		}
		interp.push(newOop)
	case 77: // someInstance
		interp.primitiveFail()
	case 78: // nextInstance
		interp.primitiveFail()
	case 79: // newMethod
		interp.primitiveFail()
	default:
		interp.primitiveFail()
	}
}

// ---- Control primitives (Blue Book p.632-) ----

func (interp *Interpreter) dispatchControlPrimitives() {
	switch interp.primitiveIndex {
	case 80: // blockCopy:
		interp.primitiveBlockCopy()
	case 81: // value, value:, value:value:
		interp.primitiveValue()
	case 82: // valueWithArguments:
		interp.primitiveFail()
	case 83: // perform:with:with:with:, perform:with:with:, perform:with:, perform:
		interp.primitivePerform()
	case 84: // perform:withArguments:
		interp.primitivePerformWithArgs()
	case 85: // signal
		interp.primitiveSignal()
	case 86: // wait
		interp.primitiveWait()
	case 87: // resume
		interp.primitiveResume()
	case 88: // suspend
		interp.primitiveSuspend()
	case 89: // flushCache
		interp.methodCache = [methodCacheSize]uint16{}
	default:
		interp.primitiveFail()
	}
}

func (interp *Interpreter) primitiveBlockCopy() {
	blockArgumentCount := interp.popStack()
	ctx := interp.popStack()
	methodContext := ctx
	if interp.isBlockContext(ctx) {
		methodContext = interp.fetchPointer(HomeIndex, ctx)
	}
	contextSize := interp.fetchWordLengthOf(methodContext)
	newContext := interp.instantiateClassWithPointers(om.ClassBlockContextPointer, contextSize)
	// initialIP is set to instructionPointer + 3 (skip jump after blockCopy)
	initialIP := om.SmallIntegerOop(int16(interp.instructionPointer + 3))
	interp.storePointer(InitialIPIndex, newContext, initialIP)
	interp.storePointer(InstructionPointerIndex, newContext, initialIP)
	interp.storePointer(StackPointerIndex, newContext, om.SmallIntegerOop(0))
	interp.storePointer(BlockArgumentCountIndex, newContext, blockArgumentCount)
	interp.storePointer(HomeIndex, newContext, methodContext)
	interp.push(newContext)
}

func (interp *Interpreter) primitiveValue() {
	blockContext := interp.stackValue(interp.argumentCount)
	if !interp.isBlockContext(blockContext) {
		interp.primitiveFail()
		return
	}
	blockArgCount := int(om.SmallIntegerValue(interp.fetchPointer(BlockArgumentCountIndex, blockContext)))
	if interp.argumentCount != blockArgCount {
		interp.primitiveFail()
		return
	}
	// Transfer arguments to blockContext
	interp.transfer(interp.argumentCount,
		interp.stackPointer-interp.argumentCount+1, interp.activeContext,
		TempFrameStart, blockContext)
	interp.pop(interp.argumentCount + 1)
	initialIP := interp.fetchPointer(InitialIPIndex, blockContext)
	interp.storePointer(InstructionPointerIndex, blockContext, initialIP)
	interp.storePointer(StackPointerIndex, blockContext,
		om.SmallIntegerOop(int16(interp.argumentCount)))
	interp.storePointer(CallerIndex, blockContext, interp.activeContext)
	interp.newActiveContext(blockContext)
}

func (interp *Interpreter) primitivePerform() {
	performSelector := interp.messageSelector
	interp.messageSelector = interp.stackValue(interp.argumentCount - 1)
	newReceiver := interp.stackValue(interp.argumentCount)
	interp.lookupMethodInClass(interp.fetchClassOf(newReceiver))
	if interp.argumentCountOf(interp.newMethod) != interp.argumentCount-1 {
		interp.messageSelector = performSelector
		interp.primitiveFail()
		return
	}

	selectorIndex := interp.stackPointer - interp.argumentCount + 1
	interp.transfer(interp.argumentCount-1, selectorIndex+1, interp.activeContext, selectorIndex, interp.activeContext)
	interp.pop(1)
	interp.argumentCount--
	interp.executeNewMethod()
}

func (interp *Interpreter) primitivePerformWithArgs() {
	argumentArray := interp.popStack()
	if om.IsSmallInteger(argumentArray) || !interp.memory.ValidOop(argumentArray) {
		interp.unPop(1)
		interp.primitiveFail()
		return
	}
	arrayClass := interp.fetchClassOf(argumentArray)
	if arrayClass != om.ClassArrayPointer {
		interp.unPop(1)
		interp.primitiveFail()
		return
	}
	arraySize := interp.fetchWordLengthOf(argumentArray)
	if interp.stackPointer+arraySize >= interp.fetchWordLengthOf(interp.activeContext) {
		interp.unPop(1)
		interp.primitiveFail()
		return
	}

	performSelector := interp.messageSelector
	interp.messageSelector = interp.popStack()
	thisReceiver := interp.stackTop()
	interp.argumentCount = arraySize
	for index := 0; index < interp.argumentCount; index++ {
		interp.push(interp.fetchPointer(index, argumentArray))
	}

	interp.lookupMethodInClass(interp.fetchClassOf(thisReceiver))
	if interp.argumentCountOf(interp.newMethod) != interp.argumentCount {
		interp.pop(interp.argumentCount)
		interp.push(interp.messageSelector)
		interp.push(argumentArray)
		interp.argumentCount = 2
		interp.messageSelector = performSelector
		interp.primitiveFail()
		return
	}

	interp.executeNewMethod()
}

// ---- I/O primitives (stubs) ----

func (interp *Interpreter) dispatchInputOutputPrimitives() {
	switch interp.primitiveIndex {
	case 92: // cursorLink:
		interp.primitiveCursorLink()
	case 96: // copyBits
		interp.primitiveCopyBits()
	case 101: // beCursor
		interp.primitiveBeCursor()
	case 102: // beDisplay
		interp.primitiveBeDisplay()
	case 98: // secondClockInto:
		interp.primitiveFail()
	case 99: // millisecondClockInto:
		interp.primitiveFail()
	case 100: // signal:atMilliseconds:
		interp.primitiveFail()
	default:
		interp.primitiveFail()
	}
}

func (interp *Interpreter) primitiveBeDisplay() {
	screen := interp.popStack()
	interp.displayScreen = screen
	interp.push(screen)
}

func (interp *Interpreter) primitiveBeCursor() {
	cursor := interp.popStack()
	interp.cursorForm = cursor
	interp.push(cursor)
}

func (interp *Interpreter) primitiveCursorLink() {
	linkState := interp.popStack()
	rcvr := interp.popStack()
	switch linkState {
	case om.TruePointer:
		interp.cursorLinked = true
	case om.FalsePointer:
		interp.cursorLinked = false
	default:
		interp.unPop(2)
		interp.primitiveFail()
		return
	}
	interp.push(rcvr)
}

type formWords struct {
	bits   uint16
	width  int
	height int
}

func (interp *Interpreter) smallIntegerValueOf(oop uint16) (int, bool) {
	if !om.IsSmallInteger(oop) {
		return 0, false
	}
	return int(om.SmallIntegerValue(oop)), true
}

func (interp *Interpreter) pointValue(point uint16) (x int, y int, ok bool) {
	if om.IsSmallInteger(point) || !interp.memory.ValidOop(point) || interp.fetchClassOf(point) != om.ClassPointPointer {
		return 0, 0, false
	}
	x, ok = interp.smallIntegerValueOf(interp.fetchPointer(PointXIndex, point))
	if !ok {
		return 0, 0, false
	}
	y, ok = interp.smallIntegerValueOf(interp.fetchPointer(PointYIndex, point))
	return x, y, ok
}

func (interp *Interpreter) rectangleValue(rectangle uint16) (left int, top int, width int, height int, ok bool) {
	if om.IsSmallInteger(rectangle) || !interp.memory.ValidOop(rectangle) || interp.fetchClassOf(rectangle) != 0x0CB0 {
		return 0, 0, 0, 0, false
	}
	origin := interp.fetchPointer(RectangleOriginIndex, rectangle)
	corner := interp.fetchPointer(RectangleCornerIndex, rectangle)
	ox, oy, ok := interp.pointValue(origin)
	if !ok {
		return 0, 0, 0, 0, false
	}
	cx, cy, ok := interp.pointValue(corner)
	if !ok {
		return 0, 0, 0, 0, false
	}
	return ox, oy, cx - ox, cy - oy, true
}

func (interp *Interpreter) formWordsOf(form uint16) (formWords, bool) {
	if om.IsSmallInteger(form) || !interp.memory.ValidOop(form) {
		return formWords{}, false
	}
	bits := interp.fetchPointer(FormBitsIndex, form)
	width, ok := interp.smallIntegerValueOf(interp.fetchPointer(FormWidthIndex, form))
	if !ok {
		return formWords{}, false
	}
	height, ok := interp.smallIntegerValueOf(interp.fetchPointer(FormHeightIndex, form))
	if !ok {
		return formWords{}, false
	}
	if bits == om.NilPointer || !interp.memory.ValidOop(bits) {
		return formWords{}, false
	}
	bitsClass := interp.fetchClassOf(bits)
	if !interp.isWords(bitsClass) || interp.isPointers(bitsClass) {
		return formWords{}, false
	}
	return formWords{bits: bits, width: width, height: height}, true
}

func rotate16(value uint16, skew int) uint16 {
	if skew == 0 {
		return value
	}
	return uint16((uint32(value)<<uint(skew) | uint32(value)>>uint(16-skew)) & 0xFFFF)
}

func mergeWord(rule int, sourceWord uint16, destinationWord uint16) uint16 {
	allOnes := uint16(0xFFFF)
	switch rule {
	case 0:
		return 0
	case 1:
		return sourceWord & destinationWord
	case 2:
		return sourceWord &^ destinationWord
	case 3:
		return sourceWord
	case 4:
		return (^sourceWord) & destinationWord
	case 5:
		return destinationWord
	case 6:
		return sourceWord ^ destinationWord
	case 7:
		return sourceWord | destinationWord
	case 8:
		return (^sourceWord) & (^destinationWord)
	case 9:
		return (^sourceWord) ^ destinationWord
	case 10:
		return ^destinationWord
	case 11:
		return sourceWord | (^destinationWord)
	case 12:
		return ^sourceWord
	case 13:
		return (^sourceWord) | destinationWord
	case 14:
		return (^sourceWord) | (^destinationWord)
	case 15:
		return allOnes
	default:
		return destinationWord
	}
}

func (interp *Interpreter) copyBitsFailure(bitBlt uint16, format string, args ...any) bool {
	interp.lastCopyBitsBitBlt = bitBlt
	interp.lastCopyBitsCycle = interp.cycleCount
	interp.lastCopyBitsFailure = fmt.Sprintf(format, args...)
	return false
}

func (interp *Interpreter) doPrimitiveCopyBits(bitBlt uint16) bool {
	destForm := interp.fetchPointer(BitBltDestFormIndex, bitBlt)
	dest, ok := interp.formWordsOf(destForm)
	if !ok || dest.width <= 0 || dest.height <= 0 {
		return interp.copyBitsFailure(bitBlt, "invalid dest form oop=0x%04X ok=%v width=%d height=%d", destForm, ok, dest.width, dest.height)
	}

	sourceForm := interp.fetchPointer(BitBltSourceFormIndex, bitBlt)
	var source formWords
	hasSource := sourceForm != om.NilPointer
	if hasSource {
		source, ok = interp.formWordsOf(sourceForm)
		if !ok {
			return interp.copyBitsFailure(bitBlt, "invalid source form oop=0x%04X", sourceForm)
		}
	}

	halftoneForm := interp.fetchPointer(BitBltHalftoneFormIndex, bitBlt)
	var halftone formWords
	hasHalftone := halftoneForm != om.NilPointer
	if hasHalftone {
		halftone, ok = interp.formWordsOf(halftoneForm)
		if !ok {
			return interp.copyBitsFailure(bitBlt, "invalid halftone form oop=0x%04X", halftoneForm)
		}
	}

	intField := func(index int) (int, bool) {
		return interp.smallIntegerValueOf(interp.fetchPointer(index, bitBlt))
	}

	combinationRule, ok := intField(BitBltCombinationRuleIndex)
	if !ok || combinationRule < 0 || combinationRule > 15 {
		return interp.copyBitsFailure(bitBlt, "invalid combination rule oop=0x%04X decodedOk=%v value=%d", interp.fetchPointer(BitBltCombinationRuleIndex, bitBlt), ok, combinationRule)
	}
	destX, ok := intField(BitBltDestXIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid destX oop=0x%04X", interp.fetchPointer(BitBltDestXIndex, bitBlt))
	}
	destY, ok := intField(BitBltDestYIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid destY oop=0x%04X", interp.fetchPointer(BitBltDestYIndex, bitBlt))
	}
	width, ok := intField(BitBltWidthIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid width oop=0x%04X", interp.fetchPointer(BitBltWidthIndex, bitBlt))
	}
	height, ok := intField(BitBltHeightIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid height oop=0x%04X", interp.fetchPointer(BitBltHeightIndex, bitBlt))
	}
	clipX, ok := intField(BitBltClipXIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid clipX oop=0x%04X", interp.fetchPointer(BitBltClipXIndex, bitBlt))
	}
	clipY, ok := intField(BitBltClipYIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid clipY oop=0x%04X", interp.fetchPointer(BitBltClipYIndex, bitBlt))
	}
	clipWidth, ok := intField(BitBltClipWidthIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid clipWidth oop=0x%04X", interp.fetchPointer(BitBltClipWidthIndex, bitBlt))
	}
	clipHeight, ok := intField(BitBltClipHeightIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid clipHeight oop=0x%04X", interp.fetchPointer(BitBltClipHeightIndex, bitBlt))
	}
	sourceX, ok := intField(BitBltSourceXIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid sourceX oop=0x%04X", interp.fetchPointer(BitBltSourceXIndex, bitBlt))
	}
	sourceY, ok := intField(BitBltSourceYIndex)
	if !ok {
		return interp.copyBitsFailure(bitBlt, "invalid sourceY oop=0x%04X", interp.fetchPointer(BitBltSourceYIndex, bitBlt))
	}

	sx, sy := sourceX, sourceY
	dx, dy := destX, destY
	w, h := width, height
	if dx < clipX {
		sx += clipX - dx
		w -= clipX - dx
		dx = clipX
	}
	if dx+w > clipX+clipWidth {
		w -= (dx + w) - (clipX + clipWidth)
	}
	if dy < clipY {
		sy += clipY - dy
		h -= clipY - dy
		dy = clipY
	}
	if dy+h > clipY+clipHeight {
		h -= (dy + h) - (clipY + clipHeight)
	}
	if hasSource {
		if sx < 0 {
			dx -= sx
			w += sx
			sx = 0
		}
		if sx+w > source.width {
			w -= (sx + w) - source.width
		}
		if sy < 0 {
			dy -= sy
			h += sy
			sy = 0
		}
		if sy+h > source.height {
			h -= (sy + h) - source.height
		}
	}
	if w <= 0 || h <= 0 {
		return true
	}

	rightMasks := [...]uint16{0x0000, 0x0001, 0x0003, 0x0007, 0x000F, 0x001F, 0x003F, 0x007F, 0x00FF, 0x01FF, 0x03FF, 0x07FF, 0x0FFF, 0x1FFF, 0x3FFF, 0x7FFF, 0xFFFF}
	allOnes := uint16(0xFFFF)

	destRaster := (dest.width-1)/16 + 1
	sourceRaster := 0
	if hasSource {
		sourceRaster = (source.width-1)/16 + 1
	}
	skew := (sx - dx) & 15
	startBits := 16 - (dx & 15)
	mask1 := rightMasks[startBits]
	endBits := 15 - ((dx + w - 1) & 15)
	mask2 := ^rightMasks[endBits+1]
	var skewMask uint16
	if skew == 0 {
		skewMask = 0
	} else {
		skewMask = rightMasks[16-skew]
	}
	nWords := 1
	if w < startBits {
		mask1 &= mask2
		mask2 = 0
	} else {
		nWords = (w-startBits-1)/16 + 2
	}

	hDir, vDir := 1, 1
	if hasSource && source.bits == dest.bits && dy >= sy {
		if dy > sy {
			vDir = -1
			sy += h - 1
			dy += h - 1
		} else if dx > sx {
			hDir = -1
			sx += w - 1
			dx += w - 1
			skewMask = ^skewMask
			mask1, mask2 = mask2, mask1
		}
	}

	preload := hasSource && skew != 0 && skew <= (sx&15)
	if hDir < 0 {
		preload = !preload
	}
	sourceIndex := sy*sourceRaster + sx/16
	destIndex := dy*destRaster + dx/16
	preloadWords := 0
	if preload {
		preloadWords = 1
	}
	sourceDelta := sourceRaster*vDir - ((nWords + preloadWords) * hDir)
	destDelta := destRaster*vDir - (nWords * hDir)
	halftoneY := dy

	for i := 0; i < h; i++ {
		halftoneWord := allOnes
		if hasHalftone {
			row := halftoneY & 15
			if row >= interp.fetchWordLengthOf(halftone.bits) {
				return interp.copyBitsFailure(bitBlt, "halftone row out of range row=%d wordLen=%d halftoneBits=0x%04X", row, interp.fetchWordLengthOf(halftone.bits), halftone.bits)
			}
			halftoneWord = interp.fetchWord(row, halftone.bits)
			halftoneY += vDir
		}

		skewWord := halftoneWord
		prevWord := uint16(0)
		lineSourceIndex := sourceIndex
		lineDestIndex := destIndex
		if preload {
			if !hasSource || lineSourceIndex < 0 || lineSourceIndex >= interp.fetchWordLengthOf(source.bits) {
				return interp.copyBitsFailure(bitBlt, "preload source index out of range index=%d wordLen=%d", lineSourceIndex, interp.fetchWordLengthOf(source.bits))
			}
			prevWord = interp.fetchWord(lineSourceIndex, source.bits)
			lineSourceIndex += hDir
		}

		mergeMask := mask1
		for word := 0; word < nWords; word++ {
			if hasSource {
				if lineSourceIndex < 0 || lineSourceIndex >= interp.fetchWordLengthOf(source.bits) {
					return interp.copyBitsFailure(bitBlt, "source index out of range line=%d word=%d index=%d wordLen=%d sx=%d sy=%d dx=%d dy=%d w=%d h=%d nWords=%d preload=%v hDir=%d vDir=%d",
						i, word, lineSourceIndex, interp.fetchWordLengthOf(source.bits), sx, sy, dx, dy, w, h, nWords, preload, hDir, vDir)
				}
				prevWord &= skewMask
				thisWord := interp.fetchWord(lineSourceIndex, source.bits)
				skewWord = prevWord | (thisWord &^ skewMask)
				prevWord = thisWord
				skewWord = rotate16(skewWord, skew)
			}
			if lineDestIndex < 0 || lineDestIndex >= interp.fetchWordLengthOf(dest.bits) {
				return interp.copyBitsFailure(bitBlt, "dest index out of range line=%d word=%d index=%d wordLen=%d sx=%d sy=%d dx=%d dy=%d w=%d h=%d nWords=%d preload=%v hDir=%d vDir=%d",
					i, word, lineDestIndex, interp.fetchWordLengthOf(dest.bits), sx, sy, dx, dy, w, h, nWords, preload, hDir, vDir)
			}
			destWord := interp.fetchWord(lineDestIndex, dest.bits)
			merge := mergeWord(combinationRule, skewWord&halftoneWord, destWord)
			out := (mergeMask & merge) | (^mergeMask & destWord)
			interp.storeWord(lineDestIndex, dest.bits, out)
			lineSourceIndex += hDir
			lineDestIndex += hDir
			if word == nWords-2 {
				mergeMask = mask2
			} else {
				mergeMask = allOnes
			}
		}
		sourceIndex += sourceDelta
		destIndex += destDelta
	}

	return true
}

func (interp *Interpreter) primitiveCopyBits() {
	rcvr := interp.stackTop()
	if !interp.doPrimitiveCopyBits(rcvr) {
		interp.primitiveFail()
	}
}

// ---- Process scheduling (Blue Book p.641-647) ----

func (interp *Interpreter) schedulerPointer() uint16 {
	return interp.fetchPointer(ValueIndex, om.SchedulerAssociationPointer)
}

func (interp *Interpreter) activeProcessPointer() uint16 {
	if interp.newProcessWaiting {
		return interp.newProcess
	}
	return interp.fetchPointer(ActiveProcessIndex, interp.schedulerPointer())
}

func (interp *Interpreter) isEmptyList(aLinkedList uint16) bool {
	return interp.fetchPointer(FirstLinkIndex, aLinkedList) == om.NilPointer
}

func (interp *Interpreter) removeFirstLinkOfList(aLinkedList uint16) uint16 {
	firstLink := interp.fetchPointer(FirstLinkIndex, aLinkedList)
	lastLink := interp.fetchPointer(LastLinkIndex, aLinkedList)
	if firstLink == lastLink {
		interp.storePointer(FirstLinkIndex, aLinkedList, om.NilPointer)
		interp.storePointer(LastLinkIndex, aLinkedList, om.NilPointer)
	} else {
		nextLink := interp.fetchPointer(NextLinkIndex, firstLink)
		interp.storePointer(FirstLinkIndex, aLinkedList, nextLink)
	}
	interp.storePointer(NextLinkIndex, firstLink, om.NilPointer)
	return firstLink
}

func (interp *Interpreter) addLastLink(aLink uint16, toList uint16) {
	if interp.isEmptyList(toList) {
		interp.storePointer(FirstLinkIndex, toList, aLink)
	} else {
		lastLink := interp.fetchPointer(LastLinkIndex, toList)
		interp.storePointer(NextLinkIndex, lastLink, aLink)
	}
	interp.storePointer(LastLinkIndex, toList, aLink)
	interp.storePointer(MyListIndex, aLink, toList)
}

func (interp *Interpreter) wakeHighestPriority() uint16 {
	processLists := interp.fetchPointer(ProcessListsIndex, interp.schedulerPointer())
	priority := interp.fetchWordLengthOf(processLists)
	for {
		processList := interp.fetchPointer(priority-1, processLists)
		if !interp.isEmptyList(processList) {
			return interp.removeFirstLinkOfList(processList)
		}
		priority--
		if priority <= 0 {
			panic("wakeHighestPriority: no runnable process")
		}
	}
}

func (interp *Interpreter) sleep(aProcess uint16) {
	priority := int(om.SmallIntegerValue(interp.fetchPointer(PriorityIndex, aProcess)))
	processLists := interp.fetchPointer(ProcessListsIndex, interp.schedulerPointer())
	processList := interp.fetchPointer(priority-1, processLists)
	interp.addLastLink(aProcess, processList)
}

func (interp *Interpreter) transferTo(aProcess uint16) {
	interp.newProcessWaiting = true
	interp.newProcess = aProcess
}

func (interp *Interpreter) suspendActive() {
	interp.transferTo(interp.wakeHighestPriority())
}

func (interp *Interpreter) resume(aProcess uint16) {
	activeProcess := interp.activeProcessPointer()
	activePriority := int(om.SmallIntegerValue(interp.fetchPointer(PriorityIndex, activeProcess)))
	newPriority := int(om.SmallIntegerValue(interp.fetchPointer(PriorityIndex, aProcess)))
	if newPriority > activePriority {
		interp.sleep(activeProcess)
		interp.transferTo(aProcess)
	} else {
		interp.sleep(aProcess)
	}
}

func (interp *Interpreter) synchronousSignal(aSemaphore uint16) {
	if interp.isEmptyList(aSemaphore) {
		excessSignals := int(om.SmallIntegerValue(
			interp.fetchPointer(ExcessSignalsIndex, aSemaphore)))
		interp.storePointer(ExcessSignalsIndex, aSemaphore,
			om.SmallIntegerOop(int16(excessSignals+1)))
	} else {
		interp.resume(interp.removeFirstLinkOfList(aSemaphore))
	}
}

func (interp *Interpreter) checkProcessSwitch() {
	for interp.semaphoreIndex > 0 {
		interp.semaphoreIndex--
		interp.synchronousSignal(interp.semaphoreList[interp.semaphoreIndex])
	}
	if interp.newProcessWaiting {
		interp.newProcessWaiting = false
		activeProcess := interp.activeProcessPointer()
		interp.storePointer(SuspendedContextIndex, activeProcess, interp.activeContext)
		scheduler := interp.schedulerPointer()
		// newProcess was set by transferTo:
		interp.storePointer(ActiveProcessIndex, scheduler, interp.newProcess)
		interp.newActiveContext(
			interp.fetchPointer(SuspendedContextIndex, interp.newProcess))
	}
}

func (interp *Interpreter) asynchronousSignal(aSemaphore uint16) {
	if interp.semaphoreIndex < len(interp.semaphoreList) {
		interp.semaphoreList[interp.semaphoreIndex] = aSemaphore
		interp.semaphoreIndex++
	}
}

func (interp *Interpreter) primitiveSignal() {
	sem := interp.stackTop()
	interp.synchronousSignal(sem)
}

func (interp *Interpreter) primitiveWait() {
	thisReceiver := interp.stackTop()
	excessSignals := int(om.SmallIntegerValue(
		interp.fetchPointer(ExcessSignalsIndex, thisReceiver)))
	if excessSignals > 0 {
		interp.storePointer(ExcessSignalsIndex, thisReceiver,
			om.SmallIntegerOop(int16(excessSignals-1)))
	} else {
		interp.addLastLink(interp.activeProcessPointer(), thisReceiver)
		interp.suspendActive()
	}
}

func (interp *Interpreter) primitiveResume() {
	interp.resume(interp.stackTop())
}

func (interp *Interpreter) primitiveSuspend() {
	activeProcess := interp.activeProcessPointer()
	if interp.stackTop() == activeProcess {
		interp.popStack()
		interp.push(om.NilPointer)
		interp.suspendActive()
	} else {
		interp.primitiveFail()
	}
}

// ---- System primitives ----

func (interp *Interpreter) dispatchSystemPrimitives() {
	switch interp.primitiveIndex {
	case 110: // ==
		arg := interp.popStack()
		rcvr := interp.popStack()
		if rcvr == arg {
			interp.push(om.TruePointer)
		} else {
			interp.push(om.FalsePointer)
		}
	case 111: // class
		rcvr := interp.popStack()
		interp.push(interp.fetchClassOf(rcvr))
	case 112: // coreLeft
		interp.pushInteger(0)
	case 113: // quit
		panic("Smalltalk quit primitive invoked")
	case 114: // exitToDebugger
		panic("Smalltalk exitToDebugger primitive invoked")
	case 115: // oopsLeft
		interp.pushInteger(0)
	case 116: // signal:atOopsLeft:wordsLeft:
		interp.primitiveFail()
	default:
		interp.primitiveFail()
	}
}

// ---- Bytecode dispatch (Blue Book p.594-608) ----

func (interp *Interpreter) dispatchOnThisBytecode() {
	bc := int(interp.currentBytecode)
	switch {
	case bc <= 119:
		interp.stackBytecode()
	case bc <= 127:
		interp.returnBytecode()
	case bc <= 130:
		interp.stackBytecode()
	case bc <= 134:
		interp.sendBytecode()
	case bc <= 137:
		interp.stackBytecode()
	case bc <= 143:
		// unused
	case bc <= 175:
		interp.jumpBytecode()
	case bc <= 255:
		interp.sendBytecode()
	}
}

// ---- Stack bytecodes (Blue Book p.597-601) ----

func (interp *Interpreter) stackBytecode() {
	bc := int(interp.currentBytecode)
	switch {
	case bc <= 15: // Push Receiver Variable
		fieldIndex := bc & 0xF
		interp.push(interp.fetchPointer(fieldIndex, interp.receiver))
	case bc <= 31: // Push Temporary Variable
		fieldIndex := bc & 0xF
		interp.push(interp.temporary(fieldIndex))
	case bc <= 63: // Push Literal Constant
		fieldIndex := bc & 0x1F
		interp.push(interp.literal(fieldIndex))
	case bc <= 95: // Push Literal Variable (value of Association)
		fieldIndex := bc & 0x1F
		association := interp.literal(fieldIndex)
		interp.push(interp.fetchPointer(ValueIndex, association))
	case bc <= 103: // Pop and Store Receiver Variable
		fieldIndex := bc & 0x7
		interp.storePointer(fieldIndex, interp.receiver, interp.popStack())
	case bc <= 111: // Pop and Store Temporary Variable
		fieldIndex := bc & 0x7
		interp.storePointer(fieldIndex+TempFrameStart, interp.homeContext, interp.popStack())
	case bc == 112: // Push Receiver (self)
		interp.push(interp.receiver)
	case bc <= 119: // Push constant (true, false, nil, -1, 0, 1, 2)
		switch bc {
		case 113:
			interp.push(om.TruePointer)
		case 114:
			interp.push(om.FalsePointer)
		case 115:
			interp.push(om.NilPointer)
		case 116:
			interp.push(om.MinusOnePointer)
		case 117:
			interp.push(om.ZeroPointer)
		case 118:
			interp.push(om.OnePointer)
		case 119:
			interp.push(om.TwoPointer)
		}
	case bc == 128: // Extended push
		interp.extendedPushBytecode()
	case bc == 129: // Extended store
		interp.extendedStoreBytecode()
	case bc == 130: // Extended store and pop
		interp.extendedStoreBytecode()
		interp.popStack()
	case bc == 135: // Pop
		interp.popStack()
	case bc == 136: // Duplicate
		interp.push(interp.stackTop())
	case bc == 137: // Push active context
		interp.push(interp.activeContext)
	}
}

func (interp *Interpreter) extendedPushBytecode() {
	descriptor := interp.fetchBytecode()
	variableType := (descriptor >> 6) & 3
	variableIndex := int(descriptor & 0x3F)
	switch variableType {
	case 0:
		interp.push(interp.fetchPointer(variableIndex, interp.receiver))
	case 1:
		interp.push(interp.temporary(variableIndex))
	case 2:
		interp.push(interp.literal(variableIndex))
	case 3:
		association := interp.literal(variableIndex)
		interp.push(interp.fetchPointer(ValueIndex, association))
	}
}

func (interp *Interpreter) extendedStoreBytecode() {
	descriptor := interp.fetchBytecode()
	variableType := (descriptor >> 6) & 3
	variableIndex := int(descriptor & 0x3F)
	switch variableType {
	case 0:
		interp.storePointer(variableIndex, interp.receiver, interp.stackTop())
	case 1:
		interp.storePointer(variableIndex+TempFrameStart, interp.homeContext, interp.stackTop())
	case 2:
		// Illegal — can't store into literal constant
	case 3:
		association := interp.literal(variableIndex)
		interp.storePointer(ValueIndex, association, interp.stackTop())
	}
}

// ---- Jump bytecodes (Blue Book p.601-603) ----

func (interp *Interpreter) jumpBytecode() {
	bc := int(interp.currentBytecode)
	switch {
	case bc <= 151: // Short unconditional jump
		offset := (bc & 0x7) + 1
		interp.instructionPointer += offset
	case bc <= 159: // Short conditional jump (jump on false)
		offset := (bc & 0x7) + 1
		interp.jumpIf(om.FalsePointer, offset)
	case bc <= 167: // Long unconditional jump
		offset := ((bc & 0x7) - 4) * 256
		offset += int(interp.fetchBytecode())
		interp.instructionPointer += offset
	case bc <= 175: // Long conditional jump
		offset := (bc & 0x3) * 256
		offset += int(interp.fetchBytecode())
		if bc <= 171 {
			interp.jumpIf(om.TruePointer, offset)
		} else {
			interp.jumpIf(om.FalsePointer, offset)
		}
	}
}

func (interp *Interpreter) jumpIf(condition uint16, offset int) {
	boolean := interp.popStack()
	if boolean == condition {
		interp.instructionPointer += offset
	} else if boolean == om.TruePointer || boolean == om.FalsePointer {
		// Not the condition, don't jump (already popped)
	} else {
		interp.unPop(1)
		interp.sendSelector(om.MustBeBooleanSelector, 0)
	}
}

// ---- Send bytecodes (Blue Book p.604-608) ----

func (interp *Interpreter) sendBytecode() {
	bc := int(interp.currentBytecode)
	switch {
	case bc <= 134: // Extended send
		interp.extendedSendBytecode()
	case bc <= 207: // Special selectors
		interp.sendSpecialSelectorBytecode()
	case bc <= 223: // Send literal selector, 0 args
		selectorIndex := bc & 0xF
		selector := interp.literal(selectorIndex)
		interp.sendSelector(selector, 0)
	case bc <= 239: // Send literal selector, 1 arg
		selectorIndex := bc & 0xF
		selector := interp.literal(selectorIndex)
		interp.sendSelector(selector, 1)
	case bc <= 255: // Send literal selector, 2 args
		selectorIndex := bc & 0xF
		selector := interp.literal(selectorIndex)
		interp.sendSelector(selector, 2)
	}
}

func (interp *Interpreter) extendedSendBytecode() {
	bc := int(interp.currentBytecode)
	switch bc {
	case 131: // Single extended send
		descriptor := interp.fetchBytecode()
		selectorIndex := int(descriptor & 0x1F)
		argCount := int(descriptor >> 5)
		selector := interp.literal(selectorIndex)
		interp.sendSelector(selector, argCount)
	case 132: // Double extended send
		count := int(interp.fetchBytecode())
		selectorIndex := int(interp.fetchBytecode())
		selector := interp.literal(selectorIndex)
		interp.sendSelector(selector, count)
	case 133: // Single extended super send
		descriptor := interp.fetchBytecode()
		selectorIndex := int(descriptor & 0x1F)
		argCount := int(descriptor >> 5)
		interp.messageSelector = interp.literal(selectorIndex)
		interp.argumentCount = argCount
		methodClass := interp.methodClassOf(interp.method)
		interp.sendSelectorToClass(interp.superclassOf(methodClass))
	case 134: // Double extended super send
		interp.argumentCount = int(interp.fetchBytecode())
		selectorIndex := int(interp.fetchBytecode())
		interp.messageSelector = interp.literal(selectorIndex)
		methodClass := interp.methodClassOf(interp.method)
		interp.sendSelectorToClass(interp.superclassOf(methodClass))
	}
}

func (interp *Interpreter) sendSpecialSelectorBytecode() {
	bc := int(interp.currentBytecode)
	if bc >= 176 && bc <= 191 {
		// Arithmetic messages
		if interp.specialSelectorPrimitiveResponse() {
			return
		}
	}
	if bc >= 192 && bc <= 207 {
		// Common selectors
		if interp.commonSelectorPrimitive() {
			return
		}
	}
	// Fall through to regular send
	selectorIndex := (bc - 176) * 2
	selector := interp.fetchPointer(selectorIndex, om.SpecialSelectorsPointer)
	count := int(om.SmallIntegerValue(interp.fetchPointer(selectorIndex+1, om.SpecialSelectorsPointer)))
	interp.sendSelector(selector, count)
}

func (interp *Interpreter) specialSelectorPrimitiveResponse() bool {
	interp.initPrimitive()
	if !om.IsSmallInteger(interp.stackValue(1)) {
		return false
	}
	interp.arithmeticSelectorPrimitive()
	return interp.success
}

func (interp *Interpreter) arithmeticSelectorPrimitive() {
	bc := int(interp.currentBytecode)
	switch bc {
	case 176:
		interp.primitiveAdd()
	case 177:
		interp.primitiveSubtract()
	case 178:
		interp.primitiveLessThan()
	case 179:
		interp.primitiveGreaterThan()
	case 180:
		interp.primitiveLessOrEqual()
	case 181:
		interp.primitiveGreaterOrEqual()
	case 182:
		interp.primitiveEqual()
	case 183:
		interp.primitiveNotEqual()
	case 184:
		interp.primitiveMultiply()
	case 185:
		interp.primitiveDivide()
	case 186:
		interp.primitiveMod()
	case 187:
		interp.primitiveMakePoint()
	case 188:
		interp.primitiveBitShift()
	case 189:
		interp.primitiveDiv()
	case 190:
		interp.primitiveBitAnd()
	case 191:
		interp.primitiveBitOr()
	default:
		interp.primitiveFail()
	}
}

func (interp *Interpreter) commonSelectorPrimitive() bool {
	bc := int(interp.currentBytecode)
	interp.initPrimitive()
	argCount := int(om.SmallIntegerValue(interp.fetchPointer((bc-176)*2+1, om.SpecialSelectorsPointer)))
	interp.argumentCount = argCount
	switch bc {
	case 198: // ==
		arg := interp.popStack()
		rcvr := interp.popStack()
		if rcvr == arg {
			interp.push(om.TruePointer)
		} else {
			interp.push(om.FalsePointer)
		}
		return true
	case 199: // class
		rcvr := interp.popStack()
		interp.push(interp.fetchClassOf(rcvr))
		return true
	case 200: // blockCopy:
		if interp.isMethodContext(interp.stackValue(argCount)) || interp.isBlockContext(interp.stackValue(argCount)) {
			interp.primitiveBlockCopy()
			return interp.success
		}
	case 201, 202: // value, value:
		if interp.isBlockContext(interp.stackValue(argCount)) {
			interp.primitiveValue()
			return interp.success
		}
	}
	return false
}

// ---- Return bytecodes (Blue Book p.608-610) ----

func (interp *Interpreter) returnBytecode() {
	bc := int(interp.currentBytecode)
	switch bc {
	case 120:
		interp.returnValueTo(interp.receiver, interp.sender())
	case 121:
		interp.returnValueTo(om.TruePointer, interp.sender())
	case 122:
		interp.returnValueTo(om.FalsePointer, interp.sender())
	case 123:
		interp.returnValueTo(om.NilPointer, interp.sender())
	case 124:
		interp.returnValueTo(interp.popStack(), interp.sender())
	case 125:
		interp.returnValueTo(interp.popStack(), interp.caller())
	}
}

// ---- Main interpreter loop ----

func (interp *Interpreter) initializeActiveContext() {
	if interp.activeContext != 0 {
		return
	}
	scheduler := interp.fetchPointer(ValueIndex, om.SchedulerAssociationPointer)
	activeProcess := interp.fetchPointer(ActiveProcessIndex, scheduler)
	suspendedContext := interp.fetchPointer(SuspendedContextIndex, activeProcess)
	interp.activeContext = suspendedContext
	interp.fetchContextRegisters()
}

func (interp *Interpreter) stepCycle() {
	interp.checkProcessSwitch()
	interp.currentBytecode = interp.fetchBytecode()
	interp.dispatchOnThisBytecode()
	interp.cycleCount++
}

// RunSteps executes a bounded number of interpreter cycles without any CLI
// logging. It is intended for host/UI loops that want to interleave execution
// with rendering and event polling.
func (interp *Interpreter) RunSteps(steps uint64) error {
	interp.initializeActiveContext()
	for i := uint64(0); i < steps; i++ {
		interp.stepCycle()
	}
	return nil
}

// DisplaySnapshot returns a copy of the current designated display form, if
// the image has registered one via beDisplay.
func (interp *Interpreter) DisplaySnapshot() (DisplaySnapshot, bool) {
	interp.initializeActiveContext()
	if interp.displayScreen == 0 || interp.displayScreen == om.NilPointer || !interp.memory.ValidOop(interp.displayScreen) {
		return DisplaySnapshot{}, false
	}
	form, ok := interp.formWordsOf(interp.displayScreen)
	if !ok || form.width <= 0 || form.height <= 0 {
		return DisplaySnapshot{}, false
	}
	wordLen := interp.fetchWordLengthOf(form.bits)
	words := make([]uint16, wordLen)
	for i := 0; i < wordLen; i++ {
		words[i] = interp.fetchWord(i, form.bits)
	}
	return DisplaySnapshot{
		FormPointer: interp.displayScreen,
		Width:       form.width,
		Height:      form.height,
		Raster:      (form.width-1)/16 + 1,
		Words:       words,
	}, true
}

// CycleCount returns how many cycles have been executed through RunSteps/Run.
func (interp *Interpreter) CycleCount() uint64 {
	return interp.cycleCount
}

// Run starts the interpreter from the active process in the image.
func (interp *Interpreter) Run(maxCycles uint64) error {
	// Find the active process from the scheduler
	// Dump known-good objects to verify segment handling
	fmt.Println("--- Checking known objects after segment fix ---")
	fmt.Printf("nil (oop 2): ")
	interp.memory.DumpObject(om.NilPointer)
	fmt.Printf("SchedulerAssoc (oop 8): ")
	interp.memory.DumpObject(om.SchedulerAssociationPointer)
	fmt.Printf("SpecialSelectors (oop 48): class=0x%04X words=%d\n",
		interp.fetchClassOf(om.SpecialSelectorsPointer), interp.fetchWordLengthOf(om.SpecialSelectorsPointer))

	schedulerAssoc := om.SchedulerAssociationPointer
	fmt.Printf("SchedulerAssoc (oop %d): valid=%v\n", schedulerAssoc, interp.memory.ValidOop(schedulerAssoc))
	scheduler := interp.fetchPointer(ValueIndex, schedulerAssoc)
	fmt.Printf("Scheduler (oop 0x%04X):\n", scheduler)
	interp.memory.DumpObject(scheduler)
	// ProcessorScheduler has: quiescentProcessLists (0), activeProcess (1)
	activeProcess := interp.fetchPointer(1, scheduler)
	fmt.Printf("ActiveProcess (oop 0x%04X):\n", activeProcess)
	interp.memory.DumpObject(activeProcess)
	// Process inherits from Link (nextLink=0), then: suspendedContext (1), priority (2), myList (3)
	suspendedContext := interp.fetchPointer(1, activeProcess)
	fmt.Printf("SuspendedContext (oop 0x%04X): valid=%v\n",
		suspendedContext, interp.memory.ValidOop(suspendedContext))
	interp.initializeActiveContext()

	fmt.Printf("Starting interpreter: activeContext=0x%04X, method=0x%04X, receiver=0x%04X\n",
		interp.activeContext, interp.method, interp.receiver)

	interp.cycleCount = 0
	for maxCycles == 0 || interp.cycleCount < maxCycles {
		if interp.cycleCount < 20 {
			fmt.Printf("[cycle %d] ctx=0x%04X ip=%d sp=%d bc=%d method=0x%04X rcvr=0x%04X\n",
				interp.cycleCount, interp.activeContext, interp.instructionPointer,
				interp.stackPointer, interp.fetchByte(interp.instructionPointer, interp.method), interp.method, interp.receiver)
		}

		interp.stepCycle()

		if interp.cycleCount%500000 == 0 && interp.cycleCount > 0 {
			fmt.Printf("[cycle %d] ctx=0x%04X ip=%d sp=%d bc=%d method=0x%04X rcvr=0x%04X\n",
				interp.cycleCount, interp.activeContext, interp.instructionPointer,
				interp.stackPointer, interp.currentBytecode, interp.method, interp.receiver)
		}
	}

	fmt.Printf("Interpreter stopped after %d cycles\n", interp.cycleCount)
	return nil
}
