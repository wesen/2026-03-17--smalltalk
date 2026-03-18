---
Title: Off-Screen Input Exercise Note
Ticket: ST80-003
Status: active
Topics:
    - input
    - ui
    - debugging
    - intern-review
DocType: reference
Intent: implementation
Owners: []
RelatedFiles:
    - Path: ttmp/2026/03/18/ST80-003--smalltalk-80-graphical-ui-host-window-and-event-loop/scripts/exercise-ui-input-and-capture.sh
      Note: Off-screen Xvfb/xdotool helper for before/after input captures
    - Path: pkg/ui/ui.go
      Note: Host UI event path being exercised by the script
    - Path: pkg/interpreter/interpreter.go
      Note: Interpreter-side buffered input, timer, and cursor state involved in the live UI path
Summary: Notes on the first off-screen live-input exercise, including the initial Xvfb activation pitfall and the eventual no-visual-delta result.
LastUpdated: 2026-03-18T19:05:00-04:00
---

# What this script was meant to answer

After display rendering, buffered input primitives, timer primitives, and cursor overlay were all implemented, the next useful question was:

- if I inject a small mouse/keyboard sequence into the off-screen UI window, does the visible UI change?

That is narrower and more actionable than “is input working?”

# What happened

The helper script:

1. launched `st80-ui` under `Xvfb`
2. captured a before screenshot
3. injected:
   - mouse move
   - left click
   - typed `a`
   - `Return`
4. captured an after screenshot
5. wrote a diff image

The first version failed because it tried to activate the window via `_NET_ACTIVE_WINDOW`, which is not available under plain `Xvfb` with no window manager.

After removing that dependency and targeting the window directly with `xdotool --window`, the script completed successfully.

The visible result was:

- before screenshot: recognizable `System Browser` scene with a visible cursor
- after screenshot: visually identical
- diff image: blank

After adding `-input-debug` instrumentation to the UI process and rerunning the same helper, the `st80-ui-run.log` still contained no input-debug lines at all.

That narrows the interpretation:

- not only was there no visible screen delta
- there was also no observed change in the host/input counters exposed by the UI process

# What this does and does not prove

This does prove:

- the off-screen capture path works
- the script can locate the UI window
- the helper can inject an input sequence without crashing the UI
- the current rerun did not produce any logged host/input counter changes inside the UI process

This does not yet prove:

- that the image consumed the injected events
- that keyboard focus under plain `Xvfb` matches a real desktop session
- that the chosen event sequence was meaningful for the exact visible Smalltalk state

So the correct conclusion is not “input is broken.” The correct conclusion is:

- this first injected sequence caused no visible screen delta

# Why this is still useful

The no-delta result narrows the next debugging step.

The next slice should probably instrument one or more of:

- host event receipt
- interpreter queue insertion
- interpreter queue drain / `primInputWord` usage
- image-side state changes after input

Without that, each future off-screen input attempt will still be partly guesswork.

# Follow-up

The best next debugging move after this note is:

- figure out why the current Xvfb/xdotool setup is not producing any observed SDL/UI-side events

Then rerun the exact same script and compare:

- were events recorded?
- were words queued?
- were words drained?
- did any visible UI state change?

# Updated follow-up result

I later revisited this with two stronger host-side experiments:

1. a direct interpreter-side input harness that bypasses SDL/X11 entirely
2. an off-screen `Xvfb` run with `openbox`, `xdotool windowfocus`, and raw SDL event logging (`-event-debug`)

The direct interpreter-side harness eventually produced a real framebuffer change after a longer post-input run. That means the image-side input path is alive when delivery is guaranteed.

The stronger off-screen host-side run still produced no `input-debug` lines and no `event-debug` lines beyond startup. So the current diagnosis is now much sharper than when this note was first written:

- the remaining blocker is not “does the image react to input at all?”
- the remaining blocker is “why does this off-screen Xvfb/xdotool path fail to deliver any usable input events to SDL?”
