---
Title: Host Cursor Overlay Note
Ticket: ST80-003
Status: active
Topics:
    - cursor
    - ui
    - vm
    - intern-review
DocType: reference
Intent: implementation
Owners: []
RelatedFiles:
    - Path: pkg/interpreter/interpreter.go
      Note: Cursor snapshot export for the designated Smalltalk cursor form
    - Path: pkg/ui/ui.go
      Note: Host-side cursor overlay during framebuffer expansion
    - Path: pkg/ui/snapshot.go
      Note: Snapshot path now shares the same cursor overlay logic
    - Path: pkg/ui/ui_test.go
      Note: Focused regression for cursor overlay composition
Summary: Notes on the chosen host-side cursor rendering strategy: export the cursor form/location from the interpreter and OR it into the displayed framebuffer without mutating object memory.
LastUpdated: 2026-03-18T18:35:00-04:00
---

# Why this exists

The Smalltalk image designates a cursor form separately from the display form. The Blue Book says that when the screen is updated, the cursor is ORed into the displayed pixels.

Before this slice, the host SDL window ignored that state entirely:

- the interpreter knew the cursor form
- the interpreter knew the cursor location
- the host rendered only the display bitmap

That meant the host UI was still visually incomplete even after the display, input, and timer work.

# Chosen strategy

The chosen implementation is:

1. export the raw display bitmap from the interpreter
2. export the designated cursor form and cursor location from the interpreter
3. compose them in the host renderer

This is intentionally not implemented by mutating the display object in object memory.

# Why host-side composition is the right split

Mutating the display object in object memory would blur two different ideas:

- the image’s stored display form
- the transient presentation-time cursor overlay

Keeping them separate has a few advantages:

- the display snapshot remains faithful to actual object memory
- the cursor remains presentation state
- the SDL renderer and the non-SDL snapshot path can share one composition rule
- debugging is easier because display contents and cursor contents can be inspected independently

# Composition rule

The host currently treats the cursor exactly like the Blue Book’s simple black-and-white OR model:

- if a cursor bit is `1`, the final displayed pixel becomes black
- if a cursor bit is `0`, the underlying display pixel is left unchanged

This matches the existing 1-bit display interpretation used everywhere else in the host renderer.

# Known limitation

This note does not prove the cursor hotspot/origin semantics are perfect. The current host path uses the raw cursor location fields as the top-left of the 16x16 cursor form and clips to the visible display bounds.

That is a sensible first implementation, but it still needs live visual confirmation in a real desktop session.

# Regression coverage

There is now a small focused regression in `pkg/ui/ui_test.go` that checks:

- a set cursor bit becomes a black pixel at the expected display coordinate
- neighboring pixels remain unchanged

This is intentionally narrow. It proves the overlay logic works without requiring the full image to drive a visible cursor during the test.

# Follow-up

The next validation step is not more code. It is to run the live UI on a real desktop session and verify:

- the cursor appears
- the cursor location tracks correctly
- the apparent hotspot is correct

If the hotspot is off, the next adjustment belongs in the host composition layer, not in object memory.
