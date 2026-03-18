// Package image loads Smalltalk-80 virtual image snapshot files.
//
// The virtual image file format (from wolczko.com/st80):
//
//	Bytes 0-3:   object space size in 16-bit words (big-endian uint32)
//	Bytes 4-7:   object table size in 16-bit words (big-endian uint32)
//	Bytes 8-511: padding (zeros)
//	Bytes 512+:  object space data (objectSpaceWords * 2 bytes, big-endian)
//	Then padding to align the object table
//	Then:        object table data (objectTableWords * 2 bytes, big-endian)
//	End of file.
//
// The object table is at the END of the file (last objectTableWords*2 bytes).
// The object space starts at offset 512.
package image

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/wesen/st80/pkg/objectmemory"
)

const headerSize = 512

func alignUp(value int, alignment int) int {
	if alignment <= 0 {
		return value
	}
	remainder := value % alignment
	if remainder == 0 {
		return value
	}
	return value + alignment - remainder
}

// LoadImage reads a Smalltalk-80 virtual image file and returns an ObjectMemory.
func LoadImage(path string) (*objectmemory.ObjectMemory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading image file: %w", err)
	}

	if len(data) < headerSize {
		return nil, fmt.Errorf("image file too small: %d bytes", len(data))
	}

	// Read header: two big-endian 32-bit values.
	objectSpaceWords := binary.BigEndian.Uint32(data[0:4])
	objectTableWords := binary.BigEndian.Uint32(data[4:8])

	objectSpaceBytes := int(objectSpaceWords) * 2
	objectTableBytes := int(objectTableWords) * 2

	// Object table is at the end of the file.
	otStart := len(data) - objectTableBytes
	// Object space starts after the header.
	osStart := headerSize

	if otStart < osStart+objectSpaceBytes {
		return nil, fmt.Errorf("invalid image: object table overlaps object space "+
			"(file=%d, osWords=%d, otWords=%d, otStart=%d, osEnd=%d)",
			len(data), objectSpaceWords, objectTableWords, otStart, osStart+objectSpaceBytes)
	}

	fmt.Printf("Image: %d bytes, objectSpace=%d words @%d, objectTable=%d words @%d\n",
		len(data), objectSpaceWords, osStart, objectTableWords, otStart)

	// Read object space: big-endian 16-bit words.
	objectSpace := make([]uint16, objectSpaceWords)
	for i := range objectSpace {
		offset := osStart + i*2
		objectSpace[i] = binary.BigEndian.Uint16(data[offset : offset+2])
	}

	// Read object table: big-endian 16-bit words.
	objectTable := make([]uint16, objectTableWords)
	for i := range objectTable {
		offset := otStart + i*2
		objectTable[i] = binary.BigEndian.Uint16(data[offset : offset+2])
	}

	return objectmemory.New(objectTable, objectSpace), nil
}

// WriteImage serializes the current object memory back into the Smalltalk-80
// virtual image format used by LoadImage.
func WriteImage(path string, memory *objectmemory.ObjectMemory) error {
	objectSpace := memory.ObjectSpaceWords()
	objectTable := memory.ObjectTableWords()

	objectSpaceWords := len(objectSpace)
	objectTableWords := len(objectTable)
	objectSpaceBytes := objectSpaceWords * 2
	objectTableBytes := objectTableWords * 2
	objectTableStart := alignUp(headerSize+objectSpaceBytes, headerSize)
	fileSize := objectTableStart + objectTableBytes

	data := make([]byte, fileSize)
	binary.BigEndian.PutUint32(data[0:4], uint32(objectSpaceWords))
	binary.BigEndian.PutUint32(data[4:8], uint32(objectTableWords))

	for i, word := range objectSpace {
		offset := headerSize + i*2
		binary.BigEndian.PutUint16(data[offset:offset+2], word)
	}
	for i, word := range objectTable {
		offset := objectTableStart + i*2
		binary.BigEndian.PutUint16(data[offset:offset+2], word)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write image file: %w", err)
	}
	return nil
}
