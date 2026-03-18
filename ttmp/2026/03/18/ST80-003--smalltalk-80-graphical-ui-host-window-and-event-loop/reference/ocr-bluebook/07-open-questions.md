# Open Questions — Blue Book Extraction

## Q1: BitBlt Field Index Initialization Routine

**Pages:** pp. 349-351, 356, 625
**Question:** The Blue Book provides explicit `initializePointIndices` (p. 625) and `initializeContextIndices` (p. 581) routines, but does NOT provide an `initializeBitBltIndices` routine. The BitBlt field order is established only by:
1. The descriptive parameter list on pp. 349-350
2. The constructor method on p. 350
3. The BitBltSimulation class definition on p. 356 (which inherits from BitBlt)

**Current VM decision:** Uses indices 0-13 matching the descriptive order. This is almost certainly correct since all three sources agree and the bug fix in the recent BitBlt field order issue confirms it.

**Risk:** Low — all sources are consistent.

---

## Q2: CharacterScanner Instance Variables Beyond BitBlt

**Pages:** pp. 351-355, 615
**Question:** CharacterScanner is a subclass of BitBlt (p. 330 class hierarchy). It adds instance variables for text scanning, but Part Four does not provide a formal field-index initialization for CharacterScanner. The Smalltalk code in Chapter 18 (pp. 351-355) uses named variables but does not state their numeric indices relative to BitBlt's 14 fields.

**Current VM decision:** Not sure what indices are used. This matters for primitive 103 (scanCharacters) if implemented.

**Risk:** Medium — matters if/when primitive 103 is implemented. The Smalltalk code can serve as the fallback.

---

## Q3: DisplayScreen Additional Instance Variables

**Pages:** pp. 390-396, 651
**Question:** The class hierarchy shows DisplayScreen as a subclass of Form (via DisplayMedium). The book does not explicitly state whether DisplayScreen adds any instance variables beyond Form's four (bits, width, height, offset). The `beDisplay` primitive (102) only needs to identify the receiver and read its Form fields.

**Current VM decision:** Treats DisplayScreen as having only Form's 4 fields. This appears correct.

**Risk:** Low.

---

## Q4: Form `offset` Field — Point or Two Integers?

**Pages:** pp. 338-339
**Question:** The descriptive text on p. 338 states that a Form has an "offset" and the context implies it is a Point. However, the formal specification in Part Four does not explicitly state the type of field index 3. The Go VM stores it as a pointer to a Point object.

**Current VM decision:** offset (index 3) is a Point object pointer. This matches the descriptive text and all usage patterns.

**Risk:** Low.

---

## Q5: Primitive 96 (copyBits) — Exact Specification Location

**Pages:** pp. 648, 651, 355-362
**Question:** The I/O primitives section (p. 648) states primitive 96 "performs an operation on a bitmap specified by the receiver" but does not give the formal pseudocode routine, unlike other primitives. Instead, it refers to Chapter 18's BitBltSimulation as the specification. This means the simulation code on pp. 355-362 IS the formal spec for copyBits, even though it's written as Smalltalk methods on a different class (BitBltSimulation, not BitBlt).

**Current VM decision:** Implements copyBits following the BitBltSimulation code. This is correct per the book's intent.

**Risk:** Low — the simulation IS the specification.

---

## Q6: Halftone Form Height Assumption

**Pages:** pp. 356, 360
**Question:** The BitBltSimulation code indexes the halftone with `(1 + (dy bitAnd: 15))` (p. 360), implying the halftone form is always exactly 16 words tall. The book does not explicitly state this constraint, but the indexing arithmetic hardcodes it.

**Current VM decision:** The Go VM should assume halftone forms are 16 pixels tall (16 words). This matches the simulation.

**Risk:** Low — the simulation code is unambiguous.

---

## Q7: OT Entry Bit 11 — Unused?

**Pages:** pp. 661-662, Figure 30.5
**Question:** Figure 30.5 shows the OT entry first word as: COUNT(0-7) | O(8) | P(9) | F(10) | SEGMENT(12-15). Bit 11 is not labeled. The accessor routines jump from freeBitOf (bit 10) to segmentBitsOf (bits 12-15), leaving bit 11 unaccounted for.

**Current VM decision:** The Go VM's segment mask is `0x000F` (bits 3-0 in LSB-first notation = bits 12-15 in MSB-first), which matches. Bit 11 appears to be unused/reserved.

**Risk:** Low — bit 11 is simply not used.

---

## Q8: Positive16BitValueOf — LargePositiveInteger Byte Order

**Pages:** pp. 617-618
**Question:** The `positive16BitValueOf:` routine (p. 618) reads a 2-byte LargePositiveInteger as: `value ← memory fetchByte: 1 ofObject: integerPointer. value ← value*256 + (memory fetchByte: 0 ofObject: integerPointer)`. This means byte 0 is the low byte and byte 1 is the high byte — little-endian digit storage for LargePositiveInteger.

Similarly, `positive16BitIntegerFor:` (p. 617) stores: byte 0 = lowByteOf, byte 1 = highByteOf.

**Current VM decision:** The Go VM should store LargePositiveInteger digits in little-endian byte order. This was the subject of a recent bug fix.

**Risk:** Low — now that the byte order is established.

---

## Q9: Instance Specification Bit 3 — Undefined

**Pages:** p. 590, Figure 27.8
**Question:** Figure 27.8 shows the instance specification as: pointers(0) | words(1) | indexable(2) | [gap] | number of fixed fields(4-14). Bit 3 is not labeled. The `fixedFieldsOf:` routine extracts bits 4-14, skipping bit 3.

**Current VM decision:** Bit 3 is not used. This matches the book.

**Risk:** None.

---

## Q10: Object Table Size and Maximum Objects

**Pages:** pp. 659-661
**Question:** The book states "a common arrangement is for each object table entry to occupy two words and for the entire table to occupy 64K words or less, yielding a maximum capacity of 32K objects" (p. 660). The exact constants `ObjectTableSegment`, `ObjectTableStart`, and `ObjectTableSize` are listed as implementation-dependent (p. 661).

**Current VM decision:** Implementation-dependent — depends on the image being loaded.

**Risk:** None — these are configuration constants.
