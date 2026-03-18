// Package interpreter implements the Smalltalk-80 bytecode interpreter
// as specified in Blue Book Chapters 27-28.
package interpreter

import (
	"fmt"

	om "github.com/wesen/st80/pkg/objectmemory"
)

// Context field indices (Blue Book p.581)
const (
	SenderIndex           = 0
	InstructionPointerIndex = 1
	StackPointerIndex     = 2
	MethodIndex           = 3
	// MethodContext field 4 is unused
	ReceiverIndex         = 5
	TempFrameStart        = 6

	// BlockContext fields
	CallerIndex              = 0
	BlockArgumentCountIndex  = 3
	InitialIPIndex           = 4
	HomeIndex                = 5
)

// Class field indices (Blue Book p.587)
const (
	SuperclassIndex              = 0
	MessageDictionaryIndex       = 1
	InstanceSpecificationIndex   = 2
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
	MessageSelectorIndex   = 0
	MessageArgumentsIndex  = 1
	MessageSize            = 2
)

// Association field index
const (
	ValueIndex = 1
)

// Method cache size (must be power of 2 * 4)
const methodCacheSize = 1024

// Interpreter is the Smalltalk-80 bytecode interpreter.
type Interpreter struct {
	memory *om.ObjectMemory

	// Context registers (Blue Book p.583)
	activeContext    uint16
	homeContext      uint16
	method          uint16
	receiver        uint16
	instructionPointer int
	stackPointer    int

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

	// Cycle counter for tracing
	cycleCount uint64
}

// New creates a new Interpreter with the given object memory.
func New(memory *om.ObjectMemory) *Interpreter {
	return &Interpreter{
		memory:  memory,
		success: true,
	}
}

// ---- Bit extraction (Blue Book p.575) ----

func extractBits(firstBitIndex, lastBitIndex int, ofValue uint16) int {
	return int((ofValue >> (15 - lastBitIndex)) & ((1 << (lastBitIndex - firstBitIndex + 1)) - 1))
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

func (interp *Interpreter) instantiateClassWithPointers(classPointer uint16, instanceSize int) uint16 {
	return interp.memory.InstantiateClass(classPointer, instanceSize, true)
}

// ---- Context management (Blue Book p.582-585) ----

func (interp *Interpreter) isBlockContext(contextPointer uint16) bool {
	methodOrArguments := interp.fetchPointer(MethodIndex, contextPointer)
	return om.IsSmallInteger(methodOrArguments)
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
	return interp.fetchPointer(HeaderIndex, methodPointer)
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
	return (interp.literalCountOf(methodPointer) + LiteralStart) * 2 + 1
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
	return interp.literalOfMethod(literalCount-2, methodPointer)
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
	return interp.fetchPointer(InstanceSpecificationIndex, classPointer)
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
	h := ((int(interp.messageSelector) & int(class)) & 0xFF) + 1
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
	contextSize := TempFrameStart
	if interp.largeContextFlagOf(interp.newMethod) == 1 {
		contextSize += 32
	} else {
		contextSize += 12
	}
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
	interp.nilContextFields()
	interp.activeContext = aContext
	interp.fetchContextRegisters()
}

func (interp *Interpreter) nilContextFields() {
	interp.storePointer(SenderIndex, interp.activeContext, om.NilPointer)
	interp.storePointer(InstructionPointerIndex, interp.activeContext, om.NilPointer)
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
		if (rcvr ^ arg) < 0 && rcvr%arg != 0 {
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
	interp.primitiveFail() // Complex, defer
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
		result := interp.instantiateClassWithPointers(class, size)
		interp.push(result)
	case 71: // basicNew:, new:
		sz := interp.popInteger()
		class := interp.popStack()
		if interp.success {
			size := sz + interp.fixedFieldsOf(class)
			result := interp.instantiateClassWithPointers(class, size)
			interp.push(result)
		} else {
			interp.unPop(2)
		}
	case 72: // become:
		interp.primitiveFail() // Complex
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
		interp.pushInteger(int(rcvr >> 1))
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
		interp.primitiveFail() // Semaphore
	case 86: // wait
		interp.primitiveFail() // Semaphore
	case 87: // resume
		interp.primitiveFail() // Process
	case 88: // suspend
		interp.primitiveFail() // Process
	case 89: // flushCache
		interp.methodCache = [methodCacheSize]uint16{}
	default:
		interp.primitiveFail()
	}
}

func (interp *Interpreter) primitiveBlockCopy() {
	blockArgumentCount := interp.popStack()
	ctx := interp.popStack()
	contextSize := TempFrameStart + 12 // small context
	newContext := interp.instantiateClassWithPointers(om.ClassBlockContextPointer, contextSize)
	// initialIP is set to instructionPointer + 3 (skip jump after blockCopy)
	initialIP := om.SmallIntegerOop(int16(interp.instructionPointer + 3))
	interp.storePointer(InitialIPIndex, newContext, initialIP)
	interp.storePointer(InstructionPointerIndex, newContext, initialIP)
	interp.storePointer(StackPointerIndex, newContext, om.SmallIntegerOop(0))
	interp.storePointer(BlockArgumentCountIndex, newContext, blockArgumentCount)
	if interp.isBlockContext(ctx) {
		interp.storePointer(HomeIndex, newContext, interp.fetchPointer(HomeIndex, ctx))
	} else {
		interp.storePointer(HomeIndex, newContext, ctx)
	}
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
	interp.activeContext = blockContext
	interp.fetchContextRegisters()
}

func (interp *Interpreter) primitivePerform() {
	performSelector := interp.messageSelector
	_ = performSelector
	interp.primitiveFail() // Complex — defer
}

func (interp *Interpreter) primitivePerformWithArgs() {
	interp.primitiveFail() // Complex — defer
}

// ---- I/O primitives (stubs) ----

func (interp *Interpreter) dispatchInputOutputPrimitives() {
	switch interp.primitiveIndex {
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
	receiverClass := interp.fetchClassOf(interp.stackValue(argCount))
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
		if receiverClass == om.ClassMethodContextPointer || receiverClass == om.ClassBlockContextPointer {
			interp.primitiveBlockCopy()
			return interp.success
		}
	case 201, 202: // value, value:
		if receiverClass == om.ClassBlockContextPointer {
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

// Run starts the interpreter from the active process in the image.
func (interp *Interpreter) Run(maxCycles uint64) error {
	// Find the active process from the scheduler
	schedulerAssoc := om.SchedulerAssociationPointer
	scheduler := interp.fetchPointer(ValueIndex, schedulerAssoc)
	// ProcessorScheduler has: quiescentProcessLists (0), activeProcess (1)
	activeProcess := interp.fetchPointer(1, scheduler)
	// Process has: nextLink (0), myList (1), suspendedContext (2), priority (3)
	interp.activeContext = interp.fetchPointer(2, activeProcess)
	interp.fetchContextRegisters()

	fmt.Printf("Starting interpreter: activeContext=0x%04X, method=0x%04X, receiver=0x%04X\n",
		interp.activeContext, interp.method, interp.receiver)

	for interp.cycleCount = 0; maxCycles == 0 || interp.cycleCount < maxCycles; interp.cycleCount++ {
		interp.currentBytecode = interp.fetchBytecode()
		interp.dispatchOnThisBytecode()

		if interp.cycleCount < 100 || interp.cycleCount%100000 == 0 {
			if interp.cycleCount%100000 == 0 && interp.cycleCount > 0 {
				fmt.Printf("[cycle %d] ip=%d, sp=%d, bc=%d\n",
					interp.cycleCount, interp.instructionPointer, interp.stackPointer, interp.currentBytecode)
			}
		}
	}

	fmt.Printf("Interpreter stopped after %d cycles\n", interp.cycleCount)
	return nil
}
