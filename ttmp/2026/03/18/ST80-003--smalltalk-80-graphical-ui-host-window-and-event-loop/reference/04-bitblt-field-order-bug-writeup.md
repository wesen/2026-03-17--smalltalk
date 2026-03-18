---
Title: BitBlt Field Order Bug Writeup
Ticket: ST80-003
Status: active
Topics:
    - bitblt
    - vm
    - graphics
    - bug
    - intern-review
DocType: reference
Intent: implementation
Owners: []
RelatedFiles:
    - Path: pkg/interpreter/interpreter.go
      Note: BitBlt slot constants and primitiveCopyBits implementation
    - Path: pkg/interpreter/interpreter_test.go
      Note: Regression and diagnostic coverage around early display rendering
Summary: Detailed writeup of the BitBlt slot-order bug that left the UI blank and how it was diagnosed from the live image.
LastUpdated: 2026-03-18T14:45:00-04:00
---

# Bug summary

The VM had a real `copyBits` implementation, but the `BitBlt` field-index constants in [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go) did not match the slot order actually used by the image.

Specifically, the Go code assumed this tail order:

- `clipX`
- `clipY`
- `clipWidth`
- `clipHeight`
- `sourceX`
- `sourceY`

The live image was actually populating:

- `sourceX`
- `sourceY`
- `clipX`
- `clipY`
- `clipWidth`
- `clipHeight`

Because of that mismatch, `primitiveCopyBits` read `clipWidth=0` and `clipHeight=0` from the wrong slots. Every otherwise-valid render operation collapsed into a zero-area no-op and returned success without writing any pixels.

# Observable symptom

The user-visible result was:

- the SDL host window existed
- the designated display surface had been fixed to the correct `640x480` size
- the window still appeared blank white

At first glance that looked like either:

- a bad host-side pixel unpacking bug
- a broken `copyBits` merge loop
- or a missing render/input path in the image

The actual cause was earlier and simpler: the first `BitBlt` operations were being silently turned into no-ops by bad field decoding.

# Why this was tricky

This bug hid behind several facts that made the VM look healthier than it was:

1. The `copyBits` send itself was happening.
2. The `BitBlt>>copyBits` method really was a primitive method with primitive index `96`.
3. The primitive did not fail with a recorded error.
4. The image kept running.

That combination initially makes it feel like "rendering is working, but the output is wrong." In reality, `primitiveCopyBits` was returning early because clipping had already reduced the effective width and height to zero.

# Exact debugging path

The important checkpoints were:

1. We proved the designated display form itself was white, not just the SDL renderer.
   - direct framebuffer snapshots showed a corrected `640x480` display that never changed

2. We proved the image did reach rendering code.
   - execution was in `Form>>fill:rule:mask:` and then at a `copyBits` send

3. We proved `copyBits` was being sent to a real `BitBlt` instance.
   - the receiver at the send site was a `BitBlt`
   - method lookup resolved to `BitBlt>>copyBits`
   - primitive index resolved to `96`

4. We dumped the live `BitBlt` receiver fields at the first `copyBits` send.

The first relevant object looked like this at the send site:

```text
field[0]=destForm       = 0x0340
field[1]=sourceForm     = 0x0002
field[2]=halftoneForm   = 0x57B4
field[3]=combinationRule= 0x0007
field[4]=destX          = 0x0001
field[5]=destY          = 0x0001
field[6]=?              = 0x0501
field[7]=?              = 0x03C1
field[8]=?              = 0x0001
field[9]=?              = 0x0001
field[10]=?             = 0x0001
field[11]=?             = 0x0001
field[12]=?             = 0x0501
field[13]=?             = 0x03C1
```

Decoded as SmallIntegers:

- `0x0501` = `640`
- `0x03C1` = `480`
- `0x0001` = `0`
- `0x0007` = `3`

That was the smoking gun:

- if fields `6` and `7` are `width` and `height`, that is sensible
- if fields `12` and `13` are `clipWidth` and `clipHeight`, that is also sensible
- if fields `8` and `9` are `sourceX` and `sourceY`, that is sensible
- but the old Go constants were reading fields `10` and `11` as `clipWidth` and `clipHeight`

That meant the VM was effectively doing:

- `clipWidth = 0`
- `clipHeight = 0`

so clipping immediately shrank every copy region to nothing.

# Fix

In [interpreter.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go), the `BitBlt` tail indices were corrected from:

```text
clipX, clipY, clipWidth, clipHeight, sourceX, sourceY
```

to:

```text
sourceX, sourceY, clipX, clipY, clipWidth, clipHeight
```

Concretely:

- `BitBltSourceXIndex = 8`
- `BitBltSourceYIndex = 9`
- `BitBltClipXIndex = 10`
- `BitBltClipYIndex = 11`
- `BitBltClipWidthIndex = 12`
- `BitBltClipHeightIndex = 13`

# Validation after the fix

After the field-order fix:

- direct framebuffer snapshots stopped being all white
- at `5000` cycles, the snapshot had non-zero black pixels
- the SDL off-screen capture also stopped being blank white

The first post-fix direct snapshot reported:

```text
cycles=5000 width=640 height=480 blackPixels=12817 whitePixels=294383
```

That proves the previous "white UI" symptom was not an SDL presentation bug. The VM framebuffer was simply never being written because `BitBlt` field decoding was wrong.

# Regression protection

There is now a normal regression in [interpreter_test.go](/home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go):

- `TestDisplaySnapshotShowsRenderedPixelsAt5000Cycles`

This checks that:

- a display snapshot exists
- it is `640x480`
- it is not all white after early startup

# Why this matters beyond this bug

This is exactly the kind of bug that motivates a structured Blue Book audit:

- wrong instance-variable order
- wrong setter/constructor argument order
- wrong primitive field access order

These are not "algorithmic" mistakes in the usual sense. They are specification-alignment mistakes. Once the wrong order is assumed, the rest of the code can look perfectly reasonable and still be wrong.

# Review guidance for an intern

If you are reviewing this later, focus on:

1. the live-object evidence
   - look at the first dumped `BitBlt` object fields

2. the setter method shape
   - `BitBlt>>setDestForm:sourceForm:halftoneForm:combinationRule:destOrigin:sourceOrigin:extent:clipRect:`

3. the fixed constants in Go
   - verify the constant order now matches the actual object population order

4. the regression test
   - confirm that the test protects the user-visible symptom, not just the constant values

# Follow-up work

This fix restores drawing, but the rendered output is not yet visually correct enough to call the UI done. The next likely frontier is still in display semantics:

- BitBlt merge rules
- halftone behavior
- raster/bit unpack direction
- other display-related class/layout mismatches

That is why the Blue Book OCR extraction pack was added in the same ticket.
