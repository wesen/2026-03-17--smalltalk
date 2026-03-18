# Display and BitBlt Audit — Blue Book Extraction

## BitBlt Field Order (CRITICAL)

The BitBlt class has exactly **14 named instance variables** in this order (pp. 349-351, 356):

| Index | Field Name       | Type / Meaning                                    |
|-------|------------------|---------------------------------------------------|
| 0     | destForm         | Form — destination                                |
| 1     | sourceForm       | Form or nil — source                              |
| 2     | halftoneForm     | Form or nil — halftone/mask pattern               |
| 3     | combinationRule  | SmallInteger 0-15 — combination rule              |
| 4     | destX            | SmallInteger — destination x                      |
| 5     | destY            | SmallInteger — destination y                      |
| 6     | width            | SmallInteger — transfer width                     |
| 7     | height           | SmallInteger — transfer height                    |
| 8     | sourceX          | SmallInteger — source x                           |
| 9     | sourceY          | SmallInteger — source y                           |
| 10    | clipX            | SmallInteger — clipping rectangle x               |
| 11    | clipY            | SmallInteger — clipping rectangle y               |
| 12    | clipWidth        | SmallInteger — clipping rectangle width           |
| 13    | clipHeight       | SmallInteger — clipping rectangle height          |

**Source:** Class hierarchy on p. 330 shows BitBlt as a direct subclass of Object. The parameter listing on pp. 349-350 gives the fields in this order. The BitBltSimulation class definition on p. 356 lists BitBlt as its superclass, confirming these are the inherited fields.

**Go VM mapping:**
```
BitBltDestFormIndex        = 0
BitBltSourceFormIndex      = 1
BitBltHalftoneFormIndex    = 2
BitBltCombinationRuleIndex = 3
BitBltDestXIndex           = 4
BitBltDestYIndex           = 5
BitBltWidthIndex           = 6
BitBltHeightIndex          = 7
BitBltSourceXIndex         = 8
BitBltSourceYIndex         = 9
BitBltClipXIndex           = 10
BitBltClipYIndex           = 11
BitBltClipWidthIndex       = 12
BitBltClipHeightIndex      = 13
```

## Form Field Order

Form has **4 named instance variables** (pp. 338-339):

| Index | Field Name | Type / Meaning                              |
|-------|------------|---------------------------------------------|
| 0     | bits       | Bitmap — word-indexable storage of pixels    |
| 1     | width      | SmallInteger — width in pixels               |
| 2     | height     | SmallInteger — height in pixels              |
| 3     | offset     | Point — display offset                       |

**Source:** Chapter 18 pp. 338-339 introduces Form with these four components. The class hierarchy shows Form as a subclass of DisplayMedium (which is a subclass of DisplayObject, which is a subclass of Object).

**Go VM mapping:**
```
FormBitsIndex   = 0
FormWidthIndex  = 1
FormHeightIndex = 2
FormOffsetIndex = 3
```

## DisplayScreen

- **Superclass:** Form (via DisplayMedium) — p. 330 class hierarchy, p. 651
- **Additional instance variables:** None documented in Part Four
- **Inherits:** bits, width, height, offset from Form
- **Key method:** `beDisplay` — primitive 102 (p. 651)
- **Behavior:** The instance of DisplayScreen that receives `beDisplay` becomes the screen that is updated approximately 60 times per second. The colors of pixels are determined by individual bits in the specially designated DisplayScreen instance (p. 651).

## Bitmap / DisplayBitmap

- **Bitmap** is a subclass of ArrayedCollection (p. 330 class hierarchy)
- **Bitmap** stores 16-bit words; it is word-indexable with no named instance variables
- **DisplayBitmap** is a subclass of Bitmap (p. 330)
- The `bits` field of a Form points to a Bitmap instance
- Bitmap's indexed storage provides raster storage: pixels packed left-to-right, top-to-bottom, 16 pixels per word (pp. 331-333)

## Cursor

- **Superclass:** Form (p. 330 class hierarchy)
- **Instances:** Always have width=16 and height=16 (p. 651)
- **Key method:** `beCursor` — primitive 101 (p. 651)
- **Behavior:** Every time the screen is updated, the cursor is ORed onto its pixels. The cursor location may be linked to the pointing device location or independent (p. 651).

## Point Field Order

| Index | Field Name | Type / Meaning |
|-------|------------|----------------|
| 0     | x          | SmallInteger   |
| 1     | y          | SmallInteger   |

**Source:** Formal spec p. 625: `initializePointIndices: XIndex ← 0. YIndex ← 1. ClassPointSize ← 2`

## Rectangle Field Order

| Index | Field Name | Type / Meaning      |
|-------|------------|---------------------|
| 0     | origin     | Point (top-left)    |
| 1     | corner     | Point (bottom-right) |

**Source:** Chapter 18 pp. 343-348. The formal spec does not provide explicit index constants for Rectangle, but the order is confirmed by the descriptive text and constructor signatures.

## Clipping Semantics (pp. 356-357)

The `clipRange` method in BitBltSimulation (pp. 356-357) defines clipping:

1. **Clip to clipping rectangle:** Adjust destX, destY, width, height, sourceX, sourceY so the destination region lies within [clipX, clipY, clipX+clipWidth, clipY+clipHeight].
2. **Clip to source form:** If sourceForm is not nil, further clip width and height so the source region lies within [0, 0, sourceForm width, sourceForm height].
3. **Early exit:** If width ≤ 0 or height ≤ 0 after clipping, return immediately (no-op).

The adjusted parameters are stored in local variables `sx, sy, dx, dy, w, h`.

## Source/Destination/Halftone Semantics

From Chapter 18 pp. 333-338, 349-351, 355-362:

- **destForm:** Required. The destination Form whose bits will be modified.
- **sourceForm:** Optional (may be nil). Provides source bits. If nil, halftone is used as source.
- **halftoneForm:** Optional (may be nil). If present, provides a repeating 16-wide pattern ANDed with the source before combination. The halftone is indexed by `(dy bitAnd: 15)` to select the appropriate row (p. 360).
- **combinationRule:** Integer 0-15. Defines the Boolean function combining source (after halftone) with destination:

| Rule | Function                              |
|------|---------------------------------------|
| 0    | 0 (all zeros)                         |
| 1    | source AND destination                |
| 2    | source AND (NOT destination)          |
| 3    | source                                |
| 4    | (NOT source) AND destination          |
| 5    | destination (no-op)                   |
| 6    | source XOR destination                |
| 7    | source OR destination                 |
| 8    | (NOT source) AND (NOT destination)    |
| 9    | (NOT source) XOR destination          |
| 10   | NOT destination                       |
| 11   | source OR (NOT destination)           |
| 12   | NOT source                            |
| 13   | (NOT source) OR destination           |
| 14   | (NOT source) OR (NOT destination)     |
| 15   | 1 (all ones)                          |

Source: p. 361, `merge:sourceWord with:destinationWord`

## Overlap Semantics (pp. 358-359)

When source and destination lie in the same bitmap, the copy direction must be adjusted to avoid destroying data:

- **Default:** hDir=1, vDir=1 (left-to-right, top-to-bottom)
- **If sourceForm == destForm AND dy > sy:** vDir ← -1; start at bottom row (`sy ← sy+h-1, dy ← dy+h-1`)
- **If sourceForm == destForm AND dy == sy AND dx > sx:** hDir ← -1; start at right column (`sx ← sx+w-1, dx ← dx+w-1`); also swap mask1/mask2 and invert skewMask

## BitBlt Constructor (p. 350)

The class-side constructor decomposes Point and Rectangle arguments into the 14 fields:

```
BitBlt class>>destForm:sourceForm:halftoneForm:combinationRule:destOrigin:sourceOrigin:extent:clipRect:
```

Maps: destOrigin.x → destX, destOrigin.y → destY, sourceOrigin.x → sourceX, sourceOrigin.y → sourceY, extent.x → width, extent.y → height, clipRect origin.x → clipX, clipRect origin.y → clipY, clipRect corner.x - clipRect origin.x → clipWidth, clipRect corner.y - clipRect origin.y → clipHeight.

## Primitive 96 — copyBits (p. 648, 651)

- **Selector:** copyBits (also copyBitsAgain)
- **Receiver:** BitBlt instance
- **Arguments:** none (receiver only)
- **Behavior:** "performs an operation on a bitmap specified by the receiver" (p. 651)
- **Field accesses:** All 14 BitBlt instance variables, plus Form fields (bits, width, height) of destForm, sourceForm (if not nil), and halftoneForm (if not nil)
- **Layout dependencies:** BitBlt field order, Form field order
- **The simulation in Chapter 18 (pp. 355-362) IS the specification** — the primitive duplicates exactly this behavior in machine code

## Uncertainty

- The book does not provide a formal `initializeBitBltIndices` routine analogous to `initializePointIndices`. The field order is established by the descriptive text on pp. 349-350, the constructor on p. 350, and the BitBltSimulation class definition on p. 356. All three sources agree.
- CharacterScanner's additional fields beyond BitBlt's 14 are not formally specified in Part Four. The CharacterScanner is described in Chapter 18 pp. 351-355 but only its Smalltalk code is given, not a formal field-index initialization.
