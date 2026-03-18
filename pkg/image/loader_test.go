package image

import (
	"path/filepath"
	"reflect"
	"testing"

	om "github.com/wesen/st80/pkg/objectmemory"
)

func TestWriteImageRoundTripsObjectMemory(t *testing.T) {
	objectTable := []uint16{
		0x0040, 0x0000,
		0x0020, 0x0000,
	}
	objectSpace := []uint16{
		3, om.ClassArrayPointer, om.NilPointer,
	}
	memory := om.New(objectTable, objectSpace)

	path := filepath.Join(t.TempDir(), "roundtrip.image")
	if err := WriteImage(path, memory); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}

	reloaded, err := LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	if !reflect.DeepEqual(reloaded.ObjectTableWords(), memory.ObjectTableWords()) {
		t.Fatalf("object table mismatch after round trip")
	}
	if !reflect.DeepEqual(reloaded.ObjectSpaceWords(), memory.ObjectSpaceWords()) {
		t.Fatalf("object space mismatch after round trip")
	}
}
