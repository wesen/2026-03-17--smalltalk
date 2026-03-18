---
Title: BitBlt CopyLoop Row Advance Bug Writeup
Ticket: ST80-003
Status: active
Topics:
    - bitblt
    - bug
    - vm
    - graphics
    - intern-review
DocType: reference
Intent: implementation
Owners: []
RelatedFiles:
    - Path: pkg/interpreter/interpreter.go
      Note: BitBlt copy-loop implementation translated from the Blue Book simulation
    - Path: pkg/interpreter/interpreter_test.go
      Note: Regression and diagnostics for early display rendering and row-range coverage
Summary: Detailed writeup of the BitBlt copy-loop row-advance bug that truncated display writes to the top 256 rows.
LastUpdated: 2026-03-18T15:10:00-04:00
---

# Bug summary

After fixing the `BitBlt` field-order constants, the UI stopped being blank white, but the rendered image was still badly distorted. The framebuffer showed horizontal-band corruption and all non-white pixels were confined to rows `0..255`.

The root cause was a bad translation of the Blue Book `copyLoop` into Go.

In the Blue Book simulation, `sourceIndex` and `destIndex` are the running indices used directly inside the inner horizontal loop. After the inner loop, the code applies `sourceDelta` and `destDelta` to those already-advanced running indices.

In my Go port, I introduced per-line temporaries:

- `lineSourceIndex`
- `lineDestIndex`

and advanced only those inside the inner loop. After the row finished, I updated the base indices with:

```go
sourceIndex += sourceDelta
destIndex += destDelta
```

That was wrong, because `sourceDelta` and `destDelta` assume the base indices were already advanced across the row.

# Symptom

The visible result was:

- the display was no longer blank
- the UI showed a corrupted horizontal band
- the image never drew below row `255`

This was measurable, not just visual:

- row occupancy checks showed non-white pixels only in rows `0..255`
- display write instrumentation showed:

```text
writeIndexRange = 0..10241
```

Given a raster of `40` words per row, that is exactly the top `256` rows plus a tiny remainder.

# Why this happened

For a full-width copy on a `640`-pixel display:

- `destRaster = 40`
- `nWords = 40`
- `destDelta = destRaster - nWords = 0`

In the correct algorithm, that is fine, because `destIndex` has already been advanced by `40` words inside the row loop before `destDelta` is added.

In my incorrect Go version, `destIndex` itself never moved during the row loop. Only `lineDestIndex` moved. So after each row, adding `destDelta = 0` left the base index unchanged. The next row started from the wrong place.

The same issue applied to `sourceIndex`.

# Exact fix

The incorrect code was:

```go
sourceIndex += sourceDelta
destIndex += destDelta
```

The fix was:

```go
sourceIndex = lineSourceIndex + sourceDelta
destIndex = lineDestIndex + destDelta
```

That preserves the Blue Book meaning:

- advance across the row
- then apply the row-to-row delta

# Why this bug was subtle

The loop looked structurally faithful:

- outer loop over height
- inner loop over words
- line-local indices
- row deltas

But the semantics were different. The delta formulas are coupled to the fact that the running indices are mutated inside the inner loop. Once I split those into base indices plus line-local copies, I also had to change how the next-row starting positions were computed.

This is a classic bug when porting an algorithm from a mutating simulation into a language where you introduce "cleaner" temporaries.

# Validation

Before the fix:

- `50000`-cycle snapshot showed a distorted horizontal band
- trimmed image height was `256`
- display writes never exceeded word index `10241`

After the fix:

- `5000`-cycle snapshot showed a recognizable windowed desktop fragment
- `50000`-cycle snapshot showed a visible `System Browser`
- trimmed image height became effectively full-screen
- black-pixel count jumped from roughly `26k` to roughly `112k`

Representative post-fix snapshot:

```text
cycles=50000 width=640 height=480 blackPixels=112228 whitePixels=194972
```

# Regression protection

The existing display regression was strengthened so it now rejects the specific failure mode where rendering appears only above row `255`.

The regression now checks:

- display snapshot exists
- display size is `640x480`
- some pixels are non-white
- some non-white words exist below row `255`

# Review guidance for an intern

When reviewing this bug, compare:

1. the Blue Book `copyLoop`
2. the Go translation before the fix
3. the Go translation after the fix

The critical point is not the loop structure. The critical point is which variables the delta values are meant to update.

If you only compare "does it have an outer loop and an inner loop?" you will miss the bug. You have to compare the state transition of the running indices.

# Follow-up

This fix gets the UI from "blank/corrupt band" to "recognizable window." It does not prove every remaining BitBlt detail is perfect. The next reasonable audit points are:

- merge-rule edge cases
- overlap cases
- source/halftone combinations
- bit order and raster assumptions

But the copy-loop row progression itself is now aligned with the Blue Book simulation.
