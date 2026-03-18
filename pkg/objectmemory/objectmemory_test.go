package objectmemory

import "testing"

func TestInstantiateReusesFreedBodyForExactSize(t *testing.T) {
	om := New(nil, nil)

	first := om.InstantiateClass(ClassMethodContextPointer, 4, true)
	sizeBefore := om.ObjectSpaceSize()
	locationBefore := om.Location(first)
	segmentBefore := om.Segment(first)

	om.FreeObject(first)

	second := om.InstantiateClass(ClassBlockContextPointer, 4, true)

	if second != first {
		t.Fatalf("expected freed OOP 0x%04X to be reused, got 0x%04X", first, second)
	}
	if om.ObjectSpaceSize() != sizeBefore {
		t.Fatalf("expected object space to stay at %d words, got %d", sizeBefore, om.ObjectSpaceSize())
	}
	if om.Location(second) != locationBefore || om.Segment(second) != segmentBefore {
		t.Fatalf("expected reused body at seg=%d loc=%d, got seg=%d loc=%d",
			segmentBefore, locationBefore, om.Segment(second), om.Location(second))
	}
	if om.FetchClassOf(second) != ClassBlockContextPointer {
		t.Fatalf("expected reused object class 0x%04X, got 0x%04X", ClassBlockContextPointer, om.FetchClassOf(second))
	}
	for i := 0; i < om.FetchWordLengthOf(second); i++ {
		if got := om.FetchPointer(i, second); got != NilPointer {
			t.Fatalf("expected pointer field %d to be reset to nil, got 0x%04X", i, got)
		}
	}
}

func TestInstantiateAppendsWhenFreedBodySizeDoesNotMatch(t *testing.T) {
	om := New(nil, nil)

	first := om.InstantiateClass(ClassMethodContextPointer, 4, true)
	sizeBefore := om.ObjectSpaceSize()

	om.FreeObject(first)

	second := om.InstantiateClass(ClassMethodContextPointer, 5, true)

	if second == first {
		t.Fatalf("expected mismatched freed body slot 0x%04X to stay reserved, got immediate OOP reuse", first)
	}
	if om.ObjectSpaceSize() <= sizeBefore {
		t.Fatalf("expected object space to grow past %d words, got %d", sizeBefore, om.ObjectSpaceSize())
	}
	if om.FetchWordLengthOf(second) != 5 {
		t.Fatalf("expected new object word length 5, got %d", om.FetchWordLengthOf(second))
	}
}

func TestInstantiatePanicsWhenObjectSpaceWouldWrapPastSegmentLimit(t *testing.T) {
	om := New(nil, make([]uint16, (int(otSegmentMask)+1)*65536))

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected object-space exhaustion panic, got none")
		}
	}()

	om.InstantiateClass(ClassArrayPointer, 1, true)
}

func TestInstantiatePanicsWhenObjectTableWouldOverflow15BitOopSpace(t *testing.T) {
	om := New(make([]uint16, maxObjectTableEntries*2), nil)
	om.otEntryCount = maxObjectTableEntries

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected object-table exhaustion panic, got none")
		}
	}()

	om.InstantiateClass(ClassArrayPointer, 1, true)
}

func TestInstantiatePanicsWhenReservedSingletonIsMarkedFree(t *testing.T) {
	om := New(make([]uint16, MustBeBooleanSelector+2), nil)
	om.objectTable[otIndex(NilPointer)] = otFreeBit
	om.otEntryCount = len(om.objectTable) / 2

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected reserved-singleton-free panic, got none")
		}
	}()

	om.InstantiateClass(ClassArrayPointer, 1, true)
}

func TestStorePointerPanicsWhenFieldIndexIsNegative(t *testing.T) {
	om := New(nil, nil)
	oop := om.InstantiateClass(ClassArrayPointer, 2, true)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected StorePointer negative-index panic, got none")
		}
	}()

	om.StorePointer(-1, oop, NilPointer)
}

func TestStorePointerPanicsWhenFieldIndexExceedsObjectLength(t *testing.T) {
	om := New(nil, nil)
	oop := om.InstantiateClass(ClassArrayPointer, 2, true)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected StorePointer oversized-index panic, got none")
		}
	}()

	om.StorePointer(2, oop, NilPointer)
}

func TestStoreBytePanicsWhenByteIndexExceedsObjectLength(t *testing.T) {
	om := New(nil, nil)
	oop := om.InstantiateClassWithBytes(ClassStringPointer, 3)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected StoreByte oversized-index panic, got none")
		}
	}()

	om.StoreByte(3, oop, 0x41)
}
