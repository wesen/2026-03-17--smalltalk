# Intern OCR Diary — Blue Book Extraction

## 2026-03-18

### Step 1: Project Setup and PDF Assessment
- Located Blue Book PDF at `smalltalk-Bluebook.pdf` (742 pages, 33MB)
- Determined page offset: PDF page = printed page + 22
- Confirmed `pdftotext` available (poppler-utils); no tesseract/ocrmypdf needed
- PDF has embedded text layer — good quality, no OCR engine required
- Created output directory: `reference/ocr-bluebook/`
- Set up task list with 8 deliverables

### Step 2: Chapter Identification and Page Mapping
Key chapters identified from Table of Contents:
- **Ch 18** Graphics Kernel: pp. 329-363 (PDF 351-385) — Form, Bitmap, BitBlt, Point, Rectangle
- **Ch 20** Display Objects: pp. 381-415 (PDF 403-437) — DisplayScreen, DisplayMedium
- **Ch 27** VM Spec: pp. 567-592 (PDF 589-614) — Objects used by interpreter
- **Ch 29** Primitives: pp. 611-654 (PDF 633-676) — All primitive specifications
- **Ch 30** Object Memory: pp. 655-690 (PDF 677-712) — Heap, object table, allocation

### Step 3: Raw Text Extraction
Saved raw pdftotext output for all 5 key chapters. These serve as intermediate products for future reference.

### Step 4: Visual Verification of Priority Topics
Read each priority topic visually through PDF reader:

**BitBlt (Priority 1):**
- Ch 18 pp. 349-351: Named instance variables listed in parameter table
- Ch 18 p. 356: BitBltSimulation class definition with full instance variable list
- Ch 18 pp. 355-362: Complete BitBltSimulation code (copyBits, clipRange, computeMasks, checkOverlap, calculateOffsets, copyLoop, merge:with:)
- Field order: destForm|sourceForm|halftoneForm|combinationRule|destX|destY|width|height|sourceX|sourceY|clipX|clipY|clipWidth|clipHeight
- 16 combination rules (0-15) specified on p. 361

**Form (Priority 2):**
- Ch 18 pp. 338-339: Form described with bits, width, height, offset
- Instance variables: bits|width|height|offset (4 named fields)
- Bitmap is word-indexable storage for Form's bits field

**DisplayScreen (Priority 3):**
- Ch 29 p. 651: DisplayScreen is a subclass of Form
- beDisplay message invokes primitiveBeDisplay (primitive 102)
- Screen updated ~60 times/second from last beDisplay recipient

**Point (Priority 5):**
- Ch 29 p. 625: initializePointIndices: XIndex ← 0, YIndex ← 1, ClassPointSize ← 2
- Instance variables: x|y

**Rectangle (Priority 6):**
- Ch 18 pp. 343-348: origin and corner (two Points)
- Instance variables: origin|corner

### Step 5: Formal Specification Extraction (Ch 27)
Extracted all formal constants from Chapter 27:
- Guaranteed pointers (p. 575-576): MinusOnePointer=65535, ZeroPointer=1, OnePointer=3, TwoPointer=5, NilPointer=2, FalsePointer=4, TruePointer=6, etc.
- CompiledMethod indices (p. 577): HeaderIndex=0, LiteralStart=1
- Context indices (p. 581): SenderIndex=0, InstructionPointerIndex=1, StackPointerIndex=2, MethodIndex=3, (unused=4), ReceiverIndex=5, TempFrameStart=6
- BlockContext: CallerIndex=0, BlockArgumentCountIndex=3, InitialIPIndex=4, HomeIndex=5
- Class indices (p. 587): SuperclassIndex=0, MessageDictionaryIndex=1, InstanceSpecificationIndex=2
- Method dictionary: MethodArrayIndex=1, SelectorStart=2
- Message: MessageSelectorIndex=0, MessageArgumentsIndex=1, MessageSize=2
- Instance specification bit layout (p. 590-591, Fig 27.8): bit 0=pointers, bit 1=words, bit 2=indexable, bits 4-14=fixed fields

### Step 6: Primitive Table Extraction (Ch 29)
Extracted complete primitive dispatch table from pp. 612-615 and all dispatch routines.
Key primitives verified:
- Prim 18 (makePoint): Number @, pp. 613, 625
- Prim 70 (new): Behavior basicNew, p. 634
- Prim 71 (new:): Behavior new:, pp. 634-635
- Prim 96 (copyBits): BitBlt copyBits/copyBitsAgain, p. 648
- Prim 101 (beCursor): Cursor beCursor, p. 648
- Prim 102 (beDisplay): DisplayScreen beDisplay, p. 648
- I/O primitive dispatch (p. 647-648): primitives 90-105
- Scheduler indices (p. 642): ProcessListsIndex=0, ActiveProcessIndex=1, FirstLinkIndex=0, LastLinkIndex=1, ExcessSignalsIndex=2, etc.

### Step 7: Object Memory Extraction (Ch 30)
Extracted from pp. 655-674:
- Heap storage: objects in contiguous heap words, 2-word header (size + class)
- Object table entry (p. 661, Fig 30.5): COUNT|O|P|F|SEGMENT / LOCATION
  - Count bits: 0-7, Odd length bit: 8, Pointer bit: 9, Free bit: 10, Segment: 12-15
  - Second word: location (16 bits)
- Object pointer encoding (p. 660, Fig 30.4): bit 0=0 → object table index, bit 0=1 → immediate signed integer
- SmallInteger range: -16384 to 16383 (15-bit signed)
- Allocation/deallocation algorithms, compaction with pointer reversal

### Step 8: Cross-check Against Go VM
Used agent to explore Go VM constants in pkg/interpreter/interpreter.go and pkg/objectmemory/objectmemory.go.
All field indices and pointer values match the Blue Book specification.

### Step 9: Writing Deliverables
Now writing all 8 deliverable files based on extracted data.
