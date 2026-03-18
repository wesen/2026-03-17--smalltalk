// Package objectmemory implements the Smalltalk-80 object memory as specified
// in the Blue Book (Smalltalk-80: The Language and its Implementation, Chapter 30).
//
// The object memory manages a table of objects, each identified by a 16-bit
// Object Oriented Pointer (OOP). SmallIntegers are encoded directly in the OOP
// (bit 0 = 1), while all other objects are referenced through the object table.
//
// Object table entries are 2 words (4 bytes) each:
//
//	Word 0: count(8) | odd-length(1) | free(1) | pointer-fields(1) | segment(4) | unused(1)
//	Word 1: location in segment (pointer to first word of object body in object space)
//
// Object body layout in object space:
//
//	Word 0: size (number of words including this size field)
//	Word 1: class OOP
//	Words 2..size-1: fields (either OOPs for pointer objects, or raw 16-bit values)
package objectmemory

import "fmt"

// Well-known object pointers from the Blue Book specification (p.576).
// initializeSmallIntegers
const (
	MinusOnePointer uint16 = 65535
	ZeroPointer     uint16 = 1
	OnePointer      uint16 = 3
	TwoPointer      uint16 = 5
)

// initializeGuaranteedPointers
const (
	NilPointer                       uint16 = 2
	FalsePointer                     uint16 = 4
	TruePointer                      uint16 = 6
	SchedulerAssociationPointer      uint16 = 8
	ClassStringPointer               uint16 = 14
	ClassArrayPointer                uint16 = 16
	ClassMethodContextPointer        uint16 = 22
	ClassBlockContextPointer         uint16 = 24
	ClassPointPointer                uint16 = 26
	ClassLargePositiveIntegerPointer uint16 = 28
	ClassMessagePointer              uint16 = 32
	ClassCharacterPointer            uint16 = 40
	DoesNotUnderstandSelector        uint16 = 42
	CannotReturnSelector             uint16 = 44
	SpecialSelectorsPointer          uint16 = 48
	CharacterTablePointer            uint16 = 50
	MustBeBooleanSelector            uint16 = 52
)

// From class.oops file (not in the Blue Book guaranteed pointers but needed)
const (
	ClassSmallIntegerPointer     uint16 = 12
	ClassFloatPointer            uint16 = 20
	ClassCompiledMethodPointer   uint16 = 34
	ClassSemaphorePointer        uint16 = 38
	ClassSymbolPointer           uint16 = 56
	ClassMethodDictionaryPointer uint16 = 76
	ClassAssociationPointer      uint16 = 132
	ClassObjectPointer           uint16 = 156
)

// Object table entry bit layout (word 0).
// Blue Book p.661-662, Figure 30.5 (bit 0 = MSB in Blue Book):
//
//	bits 0-7:   count (reference count)    → standard bits 15-8
//	bit 8:      odd length (O)             → standard bit 7
//	bit 9:      pointer fields (P)         → standard bit 6
//	bit 10:     free entry (F)             → standard bit 5
//	bit 11:     unused                     → standard bit 4
//	bits 12-15: segment                    → standard bits 3-0
const (
	otCountShift   = 8
	otCountMask    = 0xFF00
	otOddLengthBit = 0x0080
	otPointerBit   = 0x0040
	otFreeBit      = 0x0020
	otSegmentShift = 0
	otSegmentMask  = 0x000F
)

// ObjectMemory is the Smalltalk-80 object memory.
type ObjectMemory struct {
	// objectTable stores 2 words per entry, indexed by OOP/2.
	// Entry i corresponds to OOP = i*2.
	objectTable []uint16

	// objectSpace stores the actual object bodies.
	// Indexed by the location field from the object table entry.
	objectSpace []uint16

	// Number of object table entries.
	otEntryCount int

	// reusableBodies records bodies for OOPs explicitly freed during this run.
	// Free entries that already existed in the image are not assumed reusable.
	reusableBodies map[uint16]reusableBody
}

type reusableBody struct {
	segment  uint16
	location uint16
	size     int
}

// New creates an ObjectMemory from raw object table and object space data.
func New(objectTable []uint16, objectSpace []uint16) *ObjectMemory {
	return &ObjectMemory{
		objectTable:    objectTable,
		objectSpace:    objectSpace,
		otEntryCount:   len(objectTable) / 2,
		reusableBodies: map[uint16]reusableBody{},
	}
}

// IsSmallInteger returns true if the given OOP encodes a SmallInteger.
func IsSmallInteger(oop uint16) bool {
	return oop&1 == 1
}

// SmallIntegerValue extracts the integer value from a SmallInteger OOP.
// SmallIntegers encode the value as a signed 15-bit integer in the upper 15 bits.
func SmallIntegerValue(oop uint16) int16 {
	return int16(oop) >> 1
}

// SmallIntegerOop creates an OOP encoding the given SmallInteger value.
func SmallIntegerOop(value int16) uint16 {
	return uint16(value<<1) | 1
}

// otIndex returns the index into objectTable for the given OOP.
func otIndex(oop uint16) int {
	return int(oop & 0xFFFE) // Clear bit 0, use as word index
}

// otEntryWord0 returns word 0 of the object table entry for oop.
func (om *ObjectMemory) otEntryWord0(oop uint16) uint16 {
	return om.objectTable[otIndex(oop)]
}

// otEntryWord1 returns word 1 of the object table entry for oop.
func (om *ObjectMemory) otEntryWord1(oop uint16) uint16 {
	return om.objectTable[otIndex(oop)+1]
}

// setOtEntryWord0 sets word 0 of the object table entry for oop.
func (om *ObjectMemory) setOtEntryWord0(oop uint16, value uint16) {
	om.objectTable[otIndex(oop)] = value
}

// setOtEntryWord1 sets word 1 of the object table entry for oop.
func (om *ObjectMemory) setOtEntryWord1(oop uint16, value uint16) {
	om.objectTable[otIndex(oop)+1] = value
}

// FreeObject marks an object table entry reusable for future allocations.
// The object body remains in object space; only the OOP slot is recycled.
func (om *ObjectMemory) FreeObject(oop uint16) {
	if IsSmallInteger(oop) || !om.ValidOop(oop) || om.IsFree(oop) {
		return
	}
	w0 := om.otEntryWord0(oop)
	w1 := om.otEntryWord1(oop)
	loc := int(w0&otSegmentMask)*65536 + int(w1)
	if loc >= 0 && loc < len(om.objectSpace) {
		om.reusableBodies[oop] = reusableBody{
			segment:  w0 & otSegmentMask,
			location: w1,
			size:     int(om.objectSpace[loc]),
		}
	}
	om.setOtEntryWord0(oop, w0|otFreeBit)
}

// SwapPointersOf implements the Blue Book pointer swap used by become:.
// It swaps the object body location and shape bits while preserving each OOP's
// reference count and free-state metadata.
func (om *ObjectMemory) SwapPointersOf(firstPointer uint16, secondPointer uint16) {
	firstW0 := om.otEntryWord0(firstPointer)
	secondW0 := om.otEntryWord0(secondPointer)
	firstW1 := om.otEntryWord1(firstPointer)
	secondW1 := om.otEntryWord1(secondPointer)

	swapMask := uint16(otOddLengthBit | otPointerBit | otSegmentMask)
	keepMask := ^swapMask

	om.setOtEntryWord0(firstPointer, (firstW0&keepMask)|(secondW0&swapMask))
	om.setOtEntryWord1(firstPointer, secondW1)
	om.setOtEntryWord0(secondPointer, (secondW0&keepMask)|(firstW0&swapMask))
	om.setOtEntryWord1(secondPointer, firstW1)
}

// IsFree returns true if the object table entry for oop is marked free.
func (om *ObjectMemory) IsFree(oop uint16) bool {
	return om.otEntryWord0(oop)&otFreeBit != 0
}

// HasPointerFields returns true if the object contains OOP fields (not raw data).
func (om *ObjectMemory) HasPointerFields(oop uint16) bool {
	return om.otEntryWord0(oop)&otPointerBit != 0
}

// HasOddLength returns true if the last word of the object is only half-used
// (i.e., the object has an odd number of bytes).
func (om *ObjectMemory) HasOddLength(oop uint16) bool {
	return om.otEntryWord0(oop)&otOddLengthBit != 0
}

// Segment returns the segment number for the object.
func (om *ObjectMemory) Segment(oop uint16) int {
	return int((om.otEntryWord0(oop) & otSegmentMask) >> otSegmentShift)
}

// Location returns the raw location field from the object table entry.
func (om *ObjectMemory) Location(oop uint16) uint16 {
	return om.otEntryWord1(oop)
}

// HeapAddress returns the full word offset in object space, combining segment and location.
// Address = segment * 65536 + location
func (om *ObjectMemory) HeapAddress(oop uint16) int {
	return om.Segment(oop)*65536 + int(om.otEntryWord1(oop))
}

// CountBits returns the reference count bits from the object table entry.
func (om *ObjectMemory) CountBits(oop uint16) int {
	return int((om.otEntryWord0(oop) & otCountMask) >> otCountShift)
}

// FetchClassOf returns the class OOP of the given object.
// For SmallIntegers, returns ClassSmallInteger.
func (om *ObjectMemory) FetchClassOf(oop uint16) uint16 {
	if IsSmallInteger(oop) {
		return ClassSmallIntegerPointer
	}
	loc := om.HeapAddress(oop)
	// Class is at offset 1 from the object body start (word after size).
	// Object body: [size, class, field0, field1, ...]
	// But the location points to the size field.
	// Actually, in the Blue Book, the location points directly to the object body:
	// objectSpace[location] = size
	// objectSpace[location+1] = class
	return om.objectSpace[loc+1]
}

// FetchWordLengthOf returns the number of indexable words in the object.
// This is the size field minus the 2-word header (size word + class word).
func (om *ObjectMemory) FetchWordLengthOf(oop uint16) int {
	loc := om.HeapAddress(oop)
	size := int(om.objectSpace[loc])
	return size - 2 // subtract size and class words
}

// FetchByteLengthOf returns the number of indexable bytes in the object.
func (om *ObjectMemory) FetchByteLengthOf(oop uint16) int {
	wordLen := om.FetchWordLengthOf(oop)
	byteLen := wordLen * 2
	if om.HasOddLength(oop) {
		byteLen--
	}
	return byteLen
}

// FetchPointer returns the OOP stored at the given field index in the object.
// Field index 0 is the first field after the class.
func (om *ObjectMemory) FetchPointer(fieldIndex int, ofObject uint16) uint16 {
	loc := om.HeapAddress(ofObject)
	addr := loc + 2 + fieldIndex
	if addr < 0 || addr >= len(om.objectSpace) {
		panic(fmt.Sprintf("FetchPointer: OOP 0x%04X field %d: addr %d out of bounds (os=%d, loc=%d)",
			ofObject, fieldIndex, addr, len(om.objectSpace), loc))
	}
	return om.objectSpace[addr]
}

// StorePointer stores an OOP at the given field index in the object.
func (om *ObjectMemory) StorePointer(fieldIndex int, ofObject uint16, withValue uint16) {
	loc := om.HeapAddress(ofObject)
	addr := loc + 2 + fieldIndex
	if addr < 0 || addr >= len(om.objectSpace) {
		panic(fmt.Sprintf("StorePointer: OOP 0x%04X field %d: addr %d out of bounds (os=%d, loc=%d)",
			ofObject, fieldIndex, addr, len(om.objectSpace), loc))
	}
	om.objectSpace[addr] = withValue
}

// FetchWord returns the raw 16-bit word at the given word index in the object.
func (om *ObjectMemory) FetchWord(wordIndex int, ofObject uint16) uint16 {
	loc := om.HeapAddress(ofObject)
	return om.objectSpace[loc+2+wordIndex]
}

// StoreWord stores a raw 16-bit word at the given word index in the object.
func (om *ObjectMemory) StoreWord(wordIndex int, ofObject uint16, withValue uint16) {
	loc := om.HeapAddress(ofObject)
	om.objectSpace[loc+2+wordIndex] = withValue
}

// FetchByte returns the byte at the given byte index in the object.
func (om *ObjectMemory) FetchByte(byteIndex int, ofObject uint16) byte {
	wordIndex := byteIndex / 2
	w := om.FetchWord(wordIndex, ofObject)
	if byteIndex%2 == 0 {
		return byte(w >> 8) // high byte first (big-endian within word)
	}
	return byte(w & 0xFF)
}

// StoreByte stores a byte at the given byte index in the object.
func (om *ObjectMemory) StoreByte(byteIndex int, ofObject uint16, withValue byte) {
	wordIndex := byteIndex / 2
	w := om.FetchWord(wordIndex, ofObject)
	if byteIndex%2 == 0 {
		w = (w & 0x00FF) | (uint16(withValue) << 8)
	} else {
		w = (w & 0xFF00) | uint16(withValue)
	}
	om.StoreWord(wordIndex, ofObject, w)
}

// ObjectTableEntryCount returns the number of entries in the object table.
func (om *ObjectMemory) ObjectTableEntryCount() int {
	return om.otEntryCount
}

func (om *ObjectMemory) initializeBody(loc int, classPointer uint16, bodySize int, pointerFields bool) {
	om.objectSpace[loc] = uint16(bodySize)
	om.objectSpace[loc+1] = classPointer
	if pointerFields {
		for i := 2; i < bodySize; i++ {
			om.objectSpace[loc+i] = NilPointer
		}
		return
	}
	for i := 2; i < bodySize; i++ {
		om.objectSpace[loc+i] = 0
	}
}

func (om *ObjectMemory) instantiate(classPointer uint16, bodySize int, pointerFields bool, oddLength bool) uint16 {
	var flags uint16
	if pointerFields {
		flags = otPointerBit
	}
	if oddLength {
		flags |= otOddLengthBit
	}

	for i := 0; i < om.otEntryCount; i++ {
		oop := uint16(i * 2)
		if !om.IsFree(oop) {
			continue
		}
		reusable, ok := om.reusableBodies[oop]
		if !ok || reusable.size != bodySize {
			continue
		}
		loc := int(reusable.segment)*65536 + int(reusable.location)
		if loc < 0 || loc+bodySize > len(om.objectSpace) {
			delete(om.reusableBodies, oop)
			continue
		}
		om.initializeBody(loc, classPointer, bodySize, pointerFields)
		om.objectTable[otIndex(oop)] = flags | reusable.segment
		om.objectTable[otIndex(oop)+1] = reusable.location
		delete(om.reusableBodies, oop)
		return oop
	}

	fullLocation := len(om.objectSpace)
	// Compute segment and location within segment
	segment := fullLocation / 65536
	if segment > int(otSegmentMask) {
		panic(fmt.Sprintf("object space exhausted: need segment %d for class 0x%04X bodySize=%d (limit=%d words)",
			segment, classPointer, bodySize, (int(otSegmentMask)+1)*65536))
	}
	locationInSegment := fullLocation % 65536
	// Extend object space
	body := make([]uint16, bodySize)
	body[0] = uint16(bodySize)
	body[1] = classPointer
	// Fields default to NilPointer for pointer objects, 0 for word objects
	if pointerFields {
		for i := 2; i < bodySize; i++ {
			body[i] = NilPointer
		}
	}
	om.objectSpace = append(om.objectSpace, body...)

	flags |= uint16(segment<<otSegmentShift) & otSegmentMask

	// Find a free OT entry
	for i := 0; i < om.otEntryCount; i++ {
		oop := uint16(i * 2)
		if om.IsFree(oop) {
			if _, tracked := om.reusableBodies[oop]; tracked {
				continue
			}
			delete(om.reusableBodies, oop)
			om.objectTable[otIndex(oop)] = flags
			om.objectTable[otIndex(oop)+1] = uint16(locationInSegment)
			return oop
		}
	}
	// No free entry — extend the OT
	newOop := uint16(om.otEntryCount * 2)
	om.otEntryCount++
	om.objectTable = append(om.objectTable, flags, uint16(locationInSegment))
	return newOop
}

// InstantiateClass creates a new pointer or word object with the given word length.
func (om *ObjectMemory) InstantiateClass(classPointer uint16, instanceSize int, isPointers bool) uint16 {
	return om.instantiate(classPointer, instanceSize+2, isPointers, false)
}

// InstantiateClassWithWords creates a non-pointer object addressed in words.
func (om *ObjectMemory) InstantiateClassWithWords(classPointer uint16, wordLength int) uint16 {
	return om.instantiate(classPointer, wordLength+2, false, false)
}

// InstantiateClassWithBytes creates a non-pointer object addressed in bytes.
func (om *ObjectMemory) InstantiateClassWithBytes(classPointer uint16, byteLength int) uint16 {
	wordLength := byteLength / 2
	oddLength := byteLength%2 == 1
	if oddLength {
		wordLength++
	}
	return om.instantiate(classPointer, wordLength+2, false, oddLength)
}

// ObjectSpaceSize returns the size of the object space in words.
func (om *ObjectMemory) ObjectSpaceSize() int {
	return len(om.objectSpace)
}

// DumpObject prints detailed information about an object for debugging.
func (om *ObjectMemory) DumpObject(oop uint16) {
	if IsSmallInteger(oop) {
		fmt.Printf("  SmallInteger: value=%d\n", SmallIntegerValue(oop))
		return
	}
	idx := otIndex(oop)
	if idx+1 >= len(om.objectTable) {
		fmt.Printf("  OOP 0x%04X: OUT OF BOUNDS (idx=%d, otLen=%d)\n", oop, idx, len(om.objectTable))
		return
	}
	w0 := om.objectTable[idx]
	w1 := om.objectTable[idx+1]
	free := w0&otFreeBit != 0
	ptr := w0&otPointerBit != 0
	odd := w0&otOddLengthBit != 0
	seg := (w0 & otSegmentMask) >> otSegmentShift
	count := w0 >> otCountShift
	loc := int(seg)*65536 + int(w1)
	fmt.Printf("  OT[%d]: w0=0x%04X w1=0x%04X | count=%d odd=%v ptr=%v free=%v seg=%d addr=%d\n",
		idx/2, w0, w1, count, odd, ptr, free, seg, loc)
	if free {
		fmt.Printf("  (FREE entry)\n")
		return
	}
	if loc+1 >= len(om.objectSpace) {
		fmt.Printf("  Location %d OUT OF BOUNDS (osLen=%d)\n", loc, len(om.objectSpace))
		return
	}
	size := om.objectSpace[loc]
	class := om.objectSpace[loc+1]
	fmt.Printf("  OS[%d]: size=%d class=0x%04X\n", loc, size, class)
	limit := int(size) - 2
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		f := om.objectSpace[loc+2+i]
		fmt.Printf("    field[%d] = 0x%04X\n", i, f)
	}
}

// FetchStringOf extracts a string value from a String or Symbol object.
func (om *ObjectMemory) FetchStringOf(oop uint16) string {
	byteLen := om.FetchByteLengthOf(oop)
	bytes := make([]byte, byteLen)
	for i := 0; i < byteLen; i++ {
		bytes[i] = om.FetchByte(i, oop)
	}
	return string(bytes)
}

// ValidOop checks if an OOP refers to a valid, non-free object table entry.
func (om *ObjectMemory) ValidOop(oop uint16) bool {
	if IsSmallInteger(oop) {
		return true
	}
	idx := otIndex(oop)
	if idx+1 >= len(om.objectTable) {
		return false
	}
	return !om.IsFree(oop)
}

// Dump prints a summary of the object table for debugging.
func (om *ObjectMemory) Dump() {
	used := 0
	free := 0
	for i := 0; i < om.otEntryCount; i++ {
		oop := uint16(i * 2)
		if om.IsFree(oop) {
			free++
		} else {
			used++
		}
	}
	fmt.Printf("Object Memory: %d OT entries (%d used, %d free), %d words object space\n",
		om.otEntryCount, used, free, len(om.objectSpace))
}
