# Object Memory Audit — Blue Book Extraction

## Overview

Chapter 30 (pp. 655-690) specifies the implementation of `RealObjectMemory`, which conforms to the `ObjectMemory` interface specified in Chapter 27 (pp. 570-574). The implementation assumes:

- 8 bits per byte (p. 656)
- 2 bytes per word (p. 656)
- Big-endian: more significant byte precedes less significant byte (p. 657)
- Word-addressed, word-indexed machine (p. 656)
- Address space partitioned into ≤16 segments of 64K (65,536) words each (p. 656)

## Object Pointer Encoding (p. 660, Figure 30.4)

An object pointer is 16 bits:

```
Bit 0 = 0:  [Object Table Index (15 bits)] [0]  → regular object
Bit 0 = 1:  [Immediate Signed Integer (15 bits)] [1]  → SmallInteger
```

- **Regular objects:** Bit 0 = 0. Bits 1-15 are an index into the object table. Up to 2^15 (32K) objects addressable.
- **SmallIntegers:** Bit 0 = 1. Bits 1-15 are a signed 15-bit integer. Range: -16384 to +16383 (±2^14).

```
isIntegerObject: objectPointer
    ↑(objectPointer bitAnd: 1) = 1
```

## Heap Object Layout (p. 657, Figure 30.1)

Each object in the heap has a **2-word header** followed by the body:

```
Word 0: size = N + 2    (total words including header)
Word 1: CLASS            (object pointer of the class)
Word 2: Field 0          \
Word 3: Field 1           } Body (N words)
  ...                    /
Word N+1: Field N-1      /
```

- Size field: unsigned 16-bit number, range 2 to 65,536 (p. 657)
- HeaderSize = 2 (constant, p. 658)

## Object Table Entry Layout (pp. 661-662, Figure 30.5)

Each object table entry occupies **2 words**:

### Word 1 (first word):

```
Bits 0-7:   COUNT     — reference count (8 bits)
Bit 8:      O         — odd length bit
Bit 9:      P         — pointer bit (1 = contains pointers)
Bit 10:     F         — free entry bit (1 = free)
Bits 11:    (unused in figure, but between F and SEGMENT)
Bits 12-15: SEGMENT   — heap segment index (4 bits)
```

### Word 2 (second word):

```
Bits 0-15:  LOCATION  — word offset within segment
```

### Accessor routines (p. 662):

| Routine              | Bits    | Meaning                              |
|----------------------|---------|--------------------------------------|
| countBitsOf:         | 0 to 7 | Reference count                      |
| oddBitOf:            | 8 to 8 | Odd-length flag                      |
| pointerBitOf:        | 9 to 9 | Pointer-fields flag                  |
| freeBitOf:           | 10 to 10 | Free entry flag                    |
| segmentBitsOf:       | 12 to 15 | Heap segment                       |
| locationBitsOf:      | word 2  | Heap location in segment            |

### Go VM mapping (from objectmemory.go):
```
otCountShift   = 8
otCountMask    = 0xFF00      // bits 15-8 (NOTE: Go uses reversed bit numbering)
otOddLengthBit = 0x0080      // bit 7
otPointerBit   = 0x0040      // bit 6
otFreeBit      = 0x0020      // bit 5
otSegmentMask  = 0x000F      // bits 3-0
```

**Note on bit numbering:** The Blue Book numbers bits with 0 at the most significant end (bit 0 = MSB, bit 15 = LSB). The Go implementation uses standard machine bit numbering (bit 0 = LSB). Therefore Blue Book bit 0-7 (count) = Go bits 15-8, Blue Book bit 8 (odd) = Go bit 7, Blue Book bit 9 (pointer) = Go bit 6, Blue Book bit 10 (free) = Go bit 5, Blue Book bits 12-15 (segment) = Go bits 3-0.

## Pointer vs Word vs Byte Object Distinctions

From Chapter 27 pp. 590-591 (instance specification):

The instance specification is a SmallInteger stored at `InstanceSpecificationIndex` (= 2) of a class. Its bit fields (Figure 27.8):

```
Bit 0:    isPointers  — 1 if fields contain object pointers
Bit 1:    isWords     — 1 if fields are 16-bit words (not bytes)
Bit 2:    isIndexable — 1 if instances have indexable fields
Bit 3:    (unused, always 0)
Bits 4-14: fixedFields — number of named instance variables
```

Constraints (p. 591):
- If isPointers=1 → fields contain object pointers, addressed in word quantities
- If isPointers=0 → fields contain numerical values, instances have indexable fields and no fixed fields
- isWords distinguishes 16-bit word values from 8-bit byte values

**Object creation** uses three different routines (p. 572):
- `instantiateClass:withPointers:instanceSize` — pointer objects
- `instantiateClass:withWords:instanceSize` — word objects (16-bit)
- `instantiateClass:withBytes:instanceByteSize` — byte objects (8-bit)

## Odd-Length Bit

The odd-length bit (bit 8 of OT word 1) is used for byte-indexable objects whose byte count is odd. Since the heap is word-addressed, an object with an odd number of bytes occupies one extra byte of storage. The odd-length bit indicates that the last byte of the last word is not part of the object's data (pp. 684-686).

## Free Space Management

### Free Pointer List (p. 664, Figure 30.6)
Free object table entries are linked through their location fields. The list is headed at `FreePointerList`. The `freeBitOf:` field is set to 1 for free entries.

### Free Chunk Lists (pp. 664-667, Figure 30.7)
Free heap chunks are organized by size:
- Chunks of size 0 through `BigSize-1` are on separate size-specific lists
- Chunks of size ≥ `BigSize` are on a single list (`LastFreeChunkList`)
- Lists are headed at `FirstFreeChunkList + size` in each heap segment
- Links between chunks use the class field of the chunk's header

### Key Constants (p. 664):
- **FreePointerList:** Head of free OT entry list
- **BigSize:** Smallest chunk not stored on a same-size list
- **FirstFreeChunkList:** Start of per-size free chunk lists
- **LastFreeChunkList:** Head of big-chunk list
- **NonPointer:** Any 16-bit value that cannot be an OT index (e.g., 2^16 - 1)

## Allocation Algorithm (pp. 667-670)

1. Get a free object table entry from `FreePointerList`
2. Find heap space:
   - For small objects (headerSize ≤ n < BigSize): try `removeFromFreeChunkList: size` in current segment
   - For large objects (n ≥ BigSize): search `LastFreeChunkList` for exact or subdivisible chunk
3. If no space in current segment: try next segment (after compacting it)
4. If no space in any segment: error "Out of memory"

## Deallocation Algorithm (p. 670)

1. Compute `spaceOccupiedBy: objectPointer`
2. Add chunk to appropriate free chunk list: `toFreeChunkList: (space min: BigSize) add: objectPointer`

## Compaction Algorithm (pp. 671-674)

Uses **pointer reversal** trick:
1. `abandonFreeChunksInSegment:` — find all free chunks, mark them with `NonPointer` class, recycle OT entries
2. `reverseHeapPointersAbove: lowWaterMark` — for each OT entry in the segment above lowWaterMark, swap the OT entry's location with the object's size header word
3. `sweepCurrentSegmentFrom: lowWaterMark` — sweep bottom-to-top, moving allocated objects down, skipping freed ones (identified by class = NonPointer), restoring size/location fields
4. Create single large free chunk from remaining space

## Garbage Collection (pp. 674-684)

Three approaches described:
1. **Simple reference counting** (pp. 675-677): count in OT bits 0-7; increment on store, decrement on overwrite; zero count → deallocate recursively
2. **Space-efficient reference counting** (pp. 677-682): uses count overflow table for counts > 128
3. **Marking collector** (pp. 682-684): mark phase traverses pointer objects; sweep phase reclaims unmarked objects

## CompiledMethod Special Handling (pp. 684-686)

CompiledMethods are **not homogeneous**: they contain both pointer fields (header + literal frame) and byte fields (bytecodes). The instance specification of CompiledMethod says non-pointer bytes, but this describes only the bytecode section. The storage manager must know:

- The number of pointer fields = `literalCountOfHeader: (header)` + `LiteralStart` (= literal count + 1)
- The remaining fields are bytes (bytecodes)
- The `pointerBitOf:` in the OT is 0 (non-pointer), but the GC and compactor must trace the pointer fields

The `lastPointerOf:` and `spaceOccupiedBy:` routines have special-case logic for CompiledMethods (p. 685-686).

## Guaranteed Object Pointers (p. 575-576)

```
"SmallIntegers"
MinusOnePointer ← 65535
ZeroPointer ← 1
OnePointer ← 3
TwoPointer ← 5

"UndefinedObject and Booleans"
NilPointer ← 2
FalsePointer ← 4
TruePointer ← 6

"Root"
SchedulerAssociationPointer ← 8

"Classes"
ClassSmallIntegerPointer ← 12     (from class.oops, not in Ch 27)
ClassStringPointer ← 14
ClassArrayPointer ← 16
ClassFloatPointer ← 20            (from class.oops)
ClassMethodContextPointer ← 22
ClassBlockContextPointer ← 24
ClassPointPointer ← 26
ClassLargePositiveIntegerPointer ← 28
ClassMessagePointer ← 32
ClassCompiledMethodPointer ← 34   (from class.oops)
ClassSemaphorePointer ← 38        (from class.oops)
ClassCharacterPointer ← 40

"Selectors"
DoesNotUnderstandSelector ← 42
CannotReturnSelector ← 44
MustBeBooleanSelector ← 52

"Tables"
SpecialSelectorsPointer ← 48
CharacterTablePointer ← 50
```
