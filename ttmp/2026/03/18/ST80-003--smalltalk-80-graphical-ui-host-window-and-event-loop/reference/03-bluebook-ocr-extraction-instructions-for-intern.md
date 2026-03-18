---
Title: Blue Book OCR Extraction Instructions For Intern
Ticket: ST80-003
Status: active
Topics:
    - bluebook
    - ocr
    - vm
    - graphics
    - audit
DocType: reference
Intent: implementation
Owners: []
RelatedFiles:
    - Path: smalltalk-Bluebook.pdf
      Note: Source document to OCR and structure
    - Path: pkg/interpreter/interpreter.go
      Note: Current Go VM implementation that will be audited against the extracted facts
    - Path: pkg/interpreter/interpreter_test.go
      Note: Existing regression and diagnostic coverage for VM/display bugs
Summary: Intern handoff for OCR and structured extraction of the Blue Book into audit-ready facts for the Smalltalk-80 VM and graphics pipeline.
LastUpdated: 2026-03-18T14:30:00-04:00
---

# Purpose

We need a structured, reviewable extraction from the Blue Book so we can audit the Go Smalltalk-80 VM against the book instead of relying on memory or one-off debugging.

This is not a generic OCR task. The goal is to extract implementation facts in a form that lets us answer questions like:

- What is the exact instance-variable order for a class?
- What is the exact argument order for a method or constructor?
- What primitive number is associated with a method?
- What fields are read or written by a primitive simulation?
- What are the exact index constants implied by the formal specification?
- What pages justify a given VM constant, object layout, or primitive behavior?

The current motivation is a real bug we just found: our `BitBlt` field-index constants were wrong, which made valid `copyBits` operations collapse into zero-area no-ops. We want to prevent that class of bug broadly.

# Constraints

- Use the Blue Book as the primary source.
- Do not use existing Smalltalk-80 implementations as reference material.
- Do not silently normalize the text into what you think it "should" say.
- Preserve ambiguity when the scan or wording is unclear.
- Every extracted fact must carry a page reference.

# Deliverables

Produce these artifacts under a new ticket-local subdirectory, for example:

`ttmp/.../reference/ocr-bluebook/`

Required files:

1. `00-ocr-notes.md`
   - OCR approach used
   - tools used
   - scan quality issues
   - pages that needed manual correction
   - unresolved OCR ambiguities

2. `01-page-index.csv`
   - one row per relevant page or page range
   - columns:
     - `page_start`
     - `page_end`
     - `topic`
     - `classes`
     - `methods`
     - `primitives`
     - `notes`

3. `02-class-layouts.csv`
   - one row per class we care about
   - columns:
     - `class_name`
     - `superclass`
     - `instance_variable_order`
     - `indexed_storage_kind`
     - `relevant_pages`
     - `confidence`
     - `notes`

4. `03-method-signatures.csv`
   - one row per relevant method
   - columns:
     - `class_name`
     - `selector`
     - `argument_count`
     - `argument_order`
     - `return_shape`
     - `primitive_number`
     - `relevant_pages`
     - `confidence`
     - `notes`

5. `04-primitive-audit.csv`
   - one row per relevant primitive
   - columns:
     - `primitive_number`
     - `selector`
     - `receiver_class`
     - `argument_shape`
     - `stack_effect`
     - `field_accesses`
     - `object_layout_dependencies`
     - `relevant_pages`
     - `confidence`
     - `notes`

6. `05-display-and-bitblt-audit.md`
   - narrative summary for display/rendering topics
   - exact field ordering for `BitBlt`
   - exact field ordering for `Form`
   - exact expectations for `DisplayScreen` and `Bitmap`/`DisplayBitmap`
   - clipping semantics
   - source/destination/halftone semantics
   - overlap semantics
   - any uncertainty called out explicitly

7. `06-object-memory-audit.md`
   - narrative summary for object memory and allocation topics
   - object header layout
   - object table entry layout
   - pointer vs word vs byte object distinctions
   - allocation/reclamation/compaction routines mentioned in the book
   - exact pages for each item

8. `07-open-questions.md`
   - questions that remained ambiguous after OCR/manual correction
   - each question should cite the pages involved
   - each question should say what decision the VM currently makes

# Priority topics

Do these first, in order:

1. `BitBlt`
2. `Form`
3. `DisplayScreen`
4. `Bitmap` / `DisplayBitmap`
5. `Point`
6. `Rectangle`
7. method headers / primitive header extensions
8. primitive dispatch and primitive numbering
9. object memory layout
10. allocation / reclamation / free lists / compaction

# What to extract for each topic

## Classes

For each class, extract:

- superclass
- named instance variables in declared order
- whether instances are pointer, word, or byte based
- whether instances are indexable
- whether the class is used by VM primitives directly
- pages where the class is introduced
- pages where the class is specified formally

Minimum classes to capture:

- `Point`
- `Rectangle`
- `Form`
- `DisplayScreen`
- `InfiniteForm`
- `OpaqueForm`
- `BitBlt`
- `Bitmap` / any bitmap-related storage class named in the book
- `CompiledMethod`
- `MethodContext`
- `BlockContext`
- `Message`
- `Behavior`
- any class directly involved in object memory or primitive semantics

## Methods

For each relevant method, extract:

- full selector
- receiver class
- ordered arguments
- whether it is an instance or class method
- whether it is a primitive method
- primitive number if applicable
- whether the method implies a field order or object layout
- exact page reference

Minimum methods to capture:

- `Point class>>x:y:`
- `Rectangle class>>origin:corner:`
- `Rectangle class>>origin:extent:`
- `Point>>corner:`
- `Point>>extent:`
- `Form>>extent:`
- `Form>>extent:offset:bits:`
- `Form>>fill:rule:mask:`
- `BitBlt class>>destForm:sourceForm:halftoneForm:combinationRule:destOrigin:sourceOrigin:extent:clipRect:`
- `BitBlt>>setDestForm:sourceForm:halftoneForm:combinationRule:destOrigin:sourceOrigin:extent:clipRect:`
- `BitBlt>>copyBits`
- `BitBlt>>copyBitsAgain`
- `DisplayScreen>>beDisplay`
- any method directly tied to primitives we implement in Go

## Primitives

For each primitive we currently implement or plan to implement, extract:

- primitive number
- selector(s)
- receiver expectations
- argument expectations
- accepted integer shape:
  - SmallInteger only
  - non-negative integer
  - large positive integer accepted
- object layout assumptions
- what fields are read
- what fields are written
- whether failure should fall back to Smalltalk code

Minimum primitives to capture:

- `71` (`new:` / `basicNew:` path)
- `96` (`copyBits`)
- `101` (`beCursor`)
- `102` (`beDisplay`)
- `18` (`makePoint`)
- the indexed access/storage primitives we use heavily

# Output quality rules

- Prefer exact names from the book, even if they are awkward.
- If OCR is uncertain, add `[uncertain]` in the relevant field and explain why in `00-ocr-notes.md`.
- If two pages disagree or seem to differ between descriptive and formal sections, record both and call it out.
- Do not flatten method argument order into prose only. Put it in explicit ordered fields.
- Do not summarize away instance-variable order. Preserve the exact sequence.

# Extraction format examples

## Example: class layout row

```csv
class_name,superclass,instance_variable_order,indexed_storage_kind,relevant_pages,confidence,notes
BitBlt,Object,"destForm|sourceForm|halftoneForm|combinationRule|destX|destY|width|height|sourceX|sourceY|clipX|clipY|clipWidth|clipHeight","named only","pp. X-Y",high,"Order must be preserved exactly; this drives primitive field indices"
```

## Example: method signature row

```csv
class_name,selector,argument_count,argument_order,return_shape,primitive_number,relevant_pages,confidence,notes
BitBlt class,"destForm:sourceForm:halftoneForm:combinationRule:destOrigin:sourceOrigin:extent:clipRect:",8,"destForm|sourceForm|halftoneForm|combinationRule|destOrigin|sourceOrigin|extent|clipRect","BitBlt instance","", "pp. X-Y",high,"Constructor-style class message; argument order must match field population logic"
```

## Example: primitive audit row

```csv
primitive_number,selector,receiver_class,argument_shape,stack_effect,field_accesses,object_layout_dependencies,relevant_pages,confidence,notes
96,copyBits,BitBlt,"receiver only","returns receiver or success/failure per method wrapper","destForm|sourceForm|halftoneForm|combinationRule|destX|destY|width|height|sourceX|sourceY|clipX|clipY|clipWidth|clipHeight","BitBlt field order; Form field order","pp. X-Y",high,"This primitive is extremely layout-sensitive"
```

# Recommended workflow

1. OCR the full PDF to searchable text.
2. Produce a rough page/topic index first.
3. Isolate the graphics chapter and formal-specification sections.
4. Extract class layouts before method summaries.
5. Extract method signatures before primitive summaries.
6. Cross-check descriptive sections against formal sections.
7. Mark ambiguities explicitly.
8. Only after the tables exist, write the narrative summaries.

# Specific traps to avoid

- Confusing descriptive prose with formal field order.
- Reordering arguments mentally into what "looks nicer."
- Losing whether a selector is a class method or instance method.
- Losing whether a method is primitive-backed.
- Failing to carry page references into the final tables.
- Inferring object-slot order from one example when the formal spec states it directly elsewhere.

# Final handoff checklist

Before handing the work back, confirm:

- every extracted class has page references
- every extracted method has ordered arguments
- every primitive has a number if the book gives one
- `BitBlt` has exact field order, not prose
- `Form` has exact field order, not prose
- object-memory layouts are captured from the formal specification
- ambiguities are documented rather than guessed away

# Review target

The intended reviewer is the VM implementer. The output should make it easy to compare the Blue Book against:

- field-index constants in `pkg/interpreter/interpreter.go`
- primitive dispatch tables in `pkg/interpreter/interpreter.go`
- class/layout assumptions embedded in the current display and object-memory code

The deliverable is successful if a reviewer can open your extracted files and answer "what exact ordering does the book require here?" in under one minute.
