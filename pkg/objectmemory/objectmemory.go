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
// Bits 15-8: count (reference count)
// Bit 7: odd length flag
// Bit 6: pointer fields flag (1 = fields contain OOPs, 0 = raw data)
// Bit 5: free entry flag
// Bits 4-1: segment number
// Bit 0: unused
const (
	otCountShift   = 8
	otCountMask    = 0xFF00
	otOddLengthBit = 0x0080
	otPointerBit   = 0x0040
	otFreeBit      = 0x0020
	otSegmentShift = 1
	otSegmentMask  = 0x001E
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
}

// New creates an ObjectMemory from raw object table and object space data.
func New(objectTable []uint16, objectSpace []uint16) *ObjectMemory {
	return &ObjectMemory{
		objectTable:  objectTable,
		objectSpace:  objectSpace,
		otEntryCount: len(objectTable) / 2,
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

// Location returns the location (word offset in object space) of the object body.
func (om *ObjectMemory) Location(oop uint16) uint16 {
	return om.otEntryWord1(oop)
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
	loc := om.Location(oop)
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
	loc := om.Location(oop)
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
	loc := om.Location(ofObject)
	return om.objectSpace[int(loc)+2+fieldIndex]
}

// StorePointer stores an OOP at the given field index in the object.
func (om *ObjectMemory) StorePointer(fieldIndex int, ofObject uint16, withValue uint16) {
	loc := om.Location(ofObject)
	om.objectSpace[int(loc)+2+fieldIndex] = withValue
}

// FetchWord returns the raw 16-bit word at the given word index in the object.
func (om *ObjectMemory) FetchWord(wordIndex int, ofObject uint16) uint16 {
	loc := om.Location(ofObject)
	return om.objectSpace[int(loc)+2+wordIndex]
}

// StoreWord stores a raw 16-bit word at the given word index in the object.
func (om *ObjectMemory) StoreWord(wordIndex int, ofObject uint16, withValue uint16) {
	loc := om.Location(ofObject)
	om.objectSpace[int(loc)+2+wordIndex] = withValue
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

// ObjectSpaceSize returns the size of the object space in words.
func (om *ObjectMemory) ObjectSpaceSize() int {
	return len(om.objectSpace)
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
