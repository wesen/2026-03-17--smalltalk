---
Title: Diary
Ticket: ST80-002
Status: active
Topics:
    - vm
    - smalltalk
    - sdl
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go
      Note: Decode SmallInteger-backed method headers, header extensions, and class specs before bit extraction (commit dd8e4ba)
    - Path: /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go
      Note: Regression test for startup execution past the former context overflow (commit dd8e4ba)
ExternalSources: []
Summary: Continuation diary for finishing the Smalltalk-80 VM after the initial handoff ticket.
LastUpdated: 2026-03-17T23:58:00-04:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Capture the continuation work after the initial ST80-001 handoff: diagnose the next concrete VM blockers, record the exact commands and failures, and keep code/doc commits granular while the interpreter is pushed toward a complete Smalltalk-80 runtime.

## Step 1: Continuation Ticket, Startup Overflow Reproduction, and Header Decode Fix

I started this continuation by treating the handoff docs as potentially stale rather than authoritative. The prior notes claimed both that the VM could run for 1M+ cycles and that it still crashed around cycle 148, so the first priority was to reproduce the current behavior on the exact head commit before changing anything.

That verification immediately paid off. The startup crash was still real, but the deeper cause was not the original “pre-existing image context is too small” hypothesis. A focused in-package reproducer showed the crashing method reported `temporaryCount=14` and `largeContextFlag=0`, which is impossible under the Blue Book’s context-size rule. That contradiction pointed directly at a metadata-decoding bug: method headers and class instance-specification words were being bit-sliced as tagged SmallInteger OOPs instead of decoded 15-bit payloads.

### Prompt Context

**User prompt (verbatim):** "# AGENTS.md instructions for /home/manuel/code/wesen/2026-03-17--smalltalk

<INSTRUCTIONS>
## JavaScript REPL (Node)
- Use `js_repl` for Node-backed JavaScript with top-level await in a persistent kernel.
- `js_repl` is a freeform/custom tool. Direct `js_repl` calls must send raw JavaScript tool input (optionally with first-line `// codex-js-repl: timeout_ms=15000`). Do not wrap code in JSON (for example `{\"code\":\"...\"}`), quotes, or markdown code fences.
- Helpers: `codex.cwd`, `codex.homeDir`, `codex.tmpDir`, `codex.tool(name, args?)`, and `codex.emitImage(imageLike)`.
- `codex.tool` executes a normal tool call and resolves to the raw tool output object. Use it for shell and non-shell tools alike. Nested tool outputs stay inside JavaScript unless you emit them explicitly.
- `codex.emitImage(...)` adds one image to the outer `js_repl` function output each time you call it, so you can call it multiple times to emit multiple images. It accepts a data URL, a single `input_image` item, an object like `{ bytes, mimeType }`, or a raw tool response object with exactly one image and no text. It rejects mixed text-and-image content.
- `codex.tool(...)` and `codex.emitImage(...)` keep stable helper identities across cells. Saved references and persisted objects can reuse them in later cells, but async callbacks that fire after a cell finishes still fail because no exec is active.
- Request full-resolution image processing with `detail: \"original\"` only when the `view_image` tool schema includes a `detail` argument. The same availability applies to `codex.emitImage(...)`: if `view_image.detail` is present, you may also pass `detail: \"original\"` there. Use this when high-fidelity image perception or precise localization is needed, especially for CUA agents.
- Example of sharing an in-memory Playwright screenshot: `await codex.emitImage({ bytes: await page.screenshot({ type: \"jpeg\", quality: 85 }), mimeType: \"image/jpeg\", detail: \"original\" })`.
- Example of sharing a local image tool result: `await codex.emitImage(codex.tool(\"view_image\", { path: \"/absolute/path\", detail: \"original\" }))`.
- When encoding an image to send with `codex.emitImage(...)` or `view_image`, prefer JPEG at about 85 quality when lossy compression is acceptable; use PNG when transparency or lossless detail matters. Smaller uploads are faster and less likely to hit size limits.
- Top-level bindings persist across cells. If a cell throws, prior bindings remain available and bindings that finished initializing before the throw often remain usable in later cells. For code you plan to reuse across cells, prefer declaring or assigning it in direct top-level statements before operations that might throw. If you hit `SyntaxError: Identifier 'x' has already been declared`, first reuse the existing binding, reassign a previously declared `let`, or pick a new descriptive name. Use `{ ... }` only for a short temporary block when you specifically need local scratch names; do not wrap an entire cell in block scope if you want those names reusable later. Reset the kernel with `js_repl_reset` only when you need a clean state.
- Top-level static import declarations (for example `import x from \"./file.js\"`) are currently unsupported in `js_repl`; use dynamic imports with `await import(\"pkg\")`, `await import(\"./file.js\")`, or `await import(\"/abs/path/file.mjs\")` instead. Imported local files must be ESM `.js`/`.mjs` files and run in the same REPL VM context. Bare package imports always resolve from REPL-global search roots (`CODEX_JS_REPL_NODE_MODULE_DIRS`, then cwd), not relative to the imported file location. Local files may statically import only other local relative/absolute/`file://` `.js`/`.mjs` files; package and builtin imports from local files must stay dynamic. `import.meta.resolve()` returns importable strings such as `file://...`, bare package names, and `node:...` specifiers. Local file modules reload between execs, while top-level bindings persist until `js_repl_reset`.
- Avoid direct access to `process.stdout` / `process.stderr` / `process.stdin`; it can corrupt the JSON line protocol. Use `console.log`, `codex.tool(...)`, and `codex.emitImage(...)`.
</INSTRUCTIONS><environment_context>
  <cwd>/home/manuel/code/wesen/2026-03-17--smalltalk</cwd>
  <shell>zsh</shell>
  <current_date>2026-03-17</current_date>
  <timezone>America/New_York</timezone>
</environment_context>

We have been working on a smalltalk 80 interpreter based on the bluebook. You are not allowed to use existing implementations as reference, just the bluebook and the https://www.wolczko.com/st80/ resources (downloaded I think).

Your colleague has been working on it so far, and you can find the claude code logs in ~/.claude/hook-events.db (see the hook logging system in .claude), and you can keep track of their work in the diary and in git and in that database. They also left two documents on where they are at.

Create a new docmgr ticket and read the ttmp/2026/03/17/ST80-001--smalltalk-80-vm-in-go-with-sdl-display/reference/03-current-issues-and-research-needed.md and continue.

Keep a frequent diary, and commit frequqently."

**Assistant interpretation:** Create a fresh continuation ticket, mine the handoff materials and hook logs for current state, then continue the VM from the most immediate concrete blocker while documenting every meaningful step.

**Inferred user intent:** Preserve continuity across collaborators, avoid re-deriving already learned facts, and turn the next chunk of interpreter work into a traceable series of code and diary commits.

**Commit (code):** dd8e4ba — "Fix SmallInteger decoding for VM headers"

### What I did
- Created docmgr ticket `ST80-002` for the continuation work.
- Read the handoff docs:
  - `ttmp/2026/03/17/ST80-001--smalltalk-80-vm-in-go-with-sdl-display/reference/02-interpreter-architecture-guide.md`
  - `ttmp/2026/03/17/ST80-001--smalltalk-80-vm-in-go-with-sdl-display/reference/03-current-issues-and-research-needed.md`
- Queried `~/.claude/hook-events.db` to confirm the schema and recover recent local work context.
- Re-ran the current VM on head:
  - `go run ./cmd/st80 data/VirtualImage 2000`
  - `go run ./cmd/st80 data/VirtualImage 2000000`
- Confirmed the old startup failure still exists and captured the exact panic:

```text
Interpreter panic: StorePointer: OOP 0x418E field 38: addr 260316 out of bounds (os=260316, loc=260276)
```

- Added an in-package reproducer at `pkg/interpreter/interpreter_test.go` so the crash could be inspected with full internal state.
- Used `pdftotext` against `smalltalk-Bluebook.pdf` to verify the Blue Book’s exact wording around:
  - `temporaryCountOf:`
  - `largeContextFlagOf:`
  - `activateNewMethod`
- The test exposed the key contradiction:

```text
cycle=148 activeContext=0x418E method=0x021E receiver=0x31A2 bytecode=0 ip=121 sp=38
contextFields=38 storedIP=0x00CF storedSP=0x001D tempCount=14 largeContextFlag=0
```

- Fixed three metadata-decoding sites so they decode SmallInteger payloads before bit extraction:
  - `headerOf`
  - `headerExtensionOf`
  - `instanceSpecificationOf`
- Reworked the reproducer into a regression test that now asserts startup runs past the former crash.
- Validated with:
  - `go test ./...`
  - `go run ./cmd/st80 data/VirtualImage 3000`

### Why
- The handoff doc contained conflicting state, so reproducing the live failure was safer than coding against stale assumptions.
- The impossible `tempCount=14` with `largeContextFlag=0` combination was a stronger signal than the original overflow theory because it violated a direct Blue Book invariant.
- Decoding the tagged SmallInteger payload is required before any Blue Book bit slicing; otherwise every extracted field is shifted and corrupted.

### What worked
- The in-package reproducer made the crash inspectable without adding noisy ad hoc logging to the runtime path.
- Pulling the relevant Blue Book passages with `pdftotext` confirmed the spec detail that matters here:
  - the large-context flag is based on whether `maxStackDepth + temporaryCount > 12`
  - large `MethodContext`s provide room for 32 frame entries total
- Once the tagged SmallInteger issue was fixed, the startup overflow disappeared immediately.
- The interpreter now runs at least 3000 cycles cleanly and no longer dies at cycle 148.

### What didn't work
- The initial reading of the handoff issue list suggested the bug might still be a context-size policy problem for pre-existing image contexts. That was incomplete.
- Running the VM with the old code still crashed quickly:

```text
go run ./cmd/st80 data/VirtualImage 2000
```

```text
Interpreter panic: StorePointer: OOP 0x418E field 38: addr 260316 out of bounds (os=260316, loc=260276)
```

- After the header/spec decode fix, a new longer-run blocker appeared instead of full success:

```text
go run ./cmd/st80 data/VirtualImage 500000
```

```text
Interpreter panic: Recursive not understood error encountered
```

### What I learned
- Several interpreter metadata words are stored as tagged SmallIntegers, not raw 16-bit bitfields.
- The earlier startup overflow was downstream from corrupted header decoding, not merely from context allocation policy.
- Class instance-spec decoding was affected by the same bug, so the consequences were broader than temp counts alone.

### What was tricky to build
- The sharp edge was distinguishing “real VM invariant failure” from “bad instrumentation interpretation.” The overflow itself looked like a straightforward frame-capacity issue, but the decisive clue came from the reproduced method metadata: a method cannot simultaneously report 14 temporaries and claim it fits in a small context. The underlying cause was subtle because the code looked structurally correct and the bug lived in a tiny assumption about whether a fetched field represented a raw bitfield or an encoded SmallInteger OOP.

### What warrants a second pair of eyes
- The remaining long-run failure path into recursive `doesNotUnderstand:` still needs investigation; there may be another metadata or message-lookup bug farther along.
- `activateNewMethod` still unconditionally allocates large contexts. That is pragmatic for now, but it should eventually be brought back in line with the decoded large-context flag once the runtime is stable.

### What should be done in the future
- Diagnose the new `Recursive not understood error encountered` panic with the same level of instrumentation.
- Re-run against longer cycle counts and the local `trace2` / `trace3` resources after the next fix.
- Once the interpreter survives long enough to reach the scheduler idle path again, resume the missing primitive and I/O/display work.

### Code review instructions
- Start in `pkg/interpreter/interpreter.go` with `headerOf`, `headerExtensionOf`, and `instanceSpecificationOf`.
- Then review `pkg/interpreter/interpreter_test.go` to see how the startup regression is reproduced and guarded.
- Validate with:
  - `go test ./...`
  - `go run ./cmd/st80 data/VirtualImage 3000`
  - `go run ./cmd/st80 data/VirtualImage 500000` to observe the next blocker

### Technical details
- Commands used:

```bash
docmgr ticket create-ticket --ticket ST80-002 --title 'Smalltalk-80 interpreter continuation: context recovery and IO path' --topics vm,smalltalk,sdl,go
go run ./cmd/st80 data/VirtualImage 2000
go run ./cmd/st80 data/VirtualImage 2000000
go test ./pkg/interpreter -run TestReproduceStartupContextOverflow -v
pdftotext smalltalk-Bluebook.pdf - | sed -n '33248,33280p'
pdftotext smalltalk-Bluebook.pdf - | sed -n '34958,34982p'
go test ./...
git commit -m "Fix SmallInteger decoding for VM headers"
```

- Relevant Blue Book excerpts confirmed during this step:
  - method temporary count comes from bits 3..7 of the decoded header value
  - the large-context flag indicates whether `maximum stack depth + temporaries > 12`
  - large `MethodContext`s allocate `32 + TempFrameStart`, small ones `12 + TempFrameStart`

## Step 2: Method Cache Hash Fix and Restoration of Long-Run Execution

The next failure did not come from the interpreter dying in a completely new subsystem. It still looked like a message-send/runtime corruption bug, but once I tightened the trace it became clear that the selector/class lookup itself was fine and the corruption only appeared in the cached path. That distinction mattered because it turned a vague “message send is broken” problem into a very specific “cache entry layout is wrong” problem with a direct Blue Book citation.

This step also corrected a mistaken intermediate hypothesis of my own. I briefly suspected instruction-pointer decoding again because `<Object>doesNotUnderstand:>` was entering in ways that looked nonsensical. The method dump exposed one more subtle metadata issue (15-bit field numbering for SmallInteger payloads), but the decisive runtime corruption in the live startup sequence came from the method cache hash not reserving four words per entry. Once that hash was fixed, the startup path stopped jumping into bogus compiled methods, the later `doesNotUnderstand:` recursion vanished, and the VM ran cleanly through 2,000,000 cycles.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue past the first startup fix without pausing, keep the diary dense, and document each new bug as its own understandable unit instead of merging multiple root causes into one note.

**Inferred user intent:** Preserve a reviewable trail of runtime fixes while driving the VM toward a stable interpreter baseline before starting UI work.

**Commit (code):** 408f7b8 — "Fix method metadata and cache decoding"

### What I did
- Added deeper interpreter diagnostics in `pkg/interpreter/interpreter_test.go` to inspect:
  - the `doesNotUnderstand:` failure path
  - the first invalid `activeContext`
  - the first invalid `method` register
  - the startup cycles around the corruption point
  - direct method lookup for `Point>>y`
- Observed that the long-run failure had an invalid `activeContext` / `method` combination, but the first actionable corruption actually occurred earlier:

```text
method became invalid at cycle 129 after bytecode=207 in ctx=0x3BF6 home=0x3BF6 method=0x170C(<Form>extent:offset:bits:) ip=6 sp=9; new method=0x021E class=0x0038 activeContext=0x418E homeContext=0x418E isBlock=false
```

- Used the local Xerox resource tables to establish expected ground truth:
  - `Point>>x` = `0x8B6A`
  - `Point>>y` = `0x8BAC`
- Verified that direct uncached lookup for `Point>>y` was correct.
- Cleared the method cache just before the bad `y` send and saw the corruption disappear, which isolated the bug to cached lookup rather than dictionary lookup.
- Pulled the Blue Book cache algorithm with `pdftotext` and found the missing operation:
  - the hash must be shifted left by 2 so each cache entry occupies four consecutive array slots
- Fixed `findNewMethodInClass` so the Go implementation mirrors the Blue Book’s 1-based Smalltalk algorithm correctly in a 0-based Go array.
- Re-ran the VM successfully for:
  - `go run ./cmd/st80 data/VirtualImage 500000`
  - `go run ./cmd/st80 data/VirtualImage 2000000`
- Re-ran the full test suite:
  - `go test ./...`

### Why
- The bad method chosen at cycle 129 happened on a `y` send to a real `Point`, so the class dictionary itself was the first thing to verify.
- Once direct lookup proved correct, the only remaining place that could explain a wrong method for the same selector/class pair was the method cache.
- The Blue Book explicitly defines the cache entry layout as four sequential words per entry, so omitting the `bitShift: 2` term causes aliasing between logically unrelated entries.

### What worked
- The staged narrowing sequence worked well:
  - detect first invalid register
  - localize to a single send bytecode
  - verify direct lookup against the local Xerox tables
  - disable cache to see whether the corruption disappears
  - confirm the cache formula against the Blue Book
- Fixing the cache hash immediately removed both the early bad method activation and the later recursive `doesNotUnderstand:` path.
- After the fix, the interpreter ran through 2,000,000 cycles with no panic, which is the strongest runtime result of this session so far.

### What didn't work
- I briefly tried changing the SmallInteger bit extractor to 15-bit numbering globally before the cache bug was isolated. That improved one diagnostic path (`doesNotUnderstand:` metadata) but reintroduced the startup overflow, so it was not the standalone fix I needed.
- Several of the added tests were purely diagnostic and stopped reflecting the new ground truth once the cache fix landed. I kept them in the tree but marked them skipped so the stable suite remains signal, not archaeology.

### What I learned
- The cache bug was a literal Blue Book translation error:
  - Smalltalk hash formula: `(((selector bitAnd: class) bitAnd: 16rFF) bitShift: 2) + 1`
  - Go 0-based translation: `(((selector & class) & 0xFF) << 2)`
- Method lookup can be perfectly correct while cached dispatch is catastrophically wrong; treating those as one subsystem would have hidden the real fix.
- The earlier metadata work and the cache work were related only in symptoms. They were distinct bugs.

### What was tricky to build
- The hardest part was resisting the temptation to “explain everything” with one root cause. After the first header decode fix, it was natural to suspect another IP/header issue when `doesNotUnderstand:` still looked wrong. The cache diagnostics disproved that: the corruption appeared only in the live cached send path, and uncached lookup for the same selector/class pair was correct. That forced a second, separate bug model, which was the right call.

### What warrants a second pair of eyes
- The SmallInteger bit-extraction helper still deserves a future audit, because the Blue Book’s 15-bit payload numbering is easy to mistranslate and I temporarily explored a broader change there.
- The runtime now survives long runs, but the current frontier has shifted to “what exactly is the system doing during the apparent idle loop?” rather than “why is it crashing?”

### What should be done in the future
- Map the 500K / 1M / 1.5M cycle methods back to local method names to confirm the runtime is in the expected scheduler/input loop.
- Compare current execution against `trace2` / `trace3` again now that both the metadata and cache bugs are fixed.
- Start implementing the minimum I/O/display primitives needed to move from stable idle execution to visible UI behavior.
- After the runtime path is stable enough to support real graphics work, create the separate UI ticket requested by the user.

### Code review instructions
- Start with `findNewMethodInClass` in `pkg/interpreter/interpreter.go`.
- Compare the Go hash computation with the Blue Book cache formula quoted in the new cache writeup.
- Then inspect the regression tests in `pkg/interpreter/interpreter_test.go`:
  - `TestStartupRunsPastFormerContextOverflow`
  - `TestDetectFirstInvalidActiveContext`
  - `TestDetectFirstInvalidMethodRegister`
  - `TestLookupPointYMethod`
- Validate with:
  - `go test ./...`
  - `go run ./cmd/st80 data/VirtualImage 500000`
  - `go run ./cmd/st80 data/VirtualImage 2000000`

### Technical details
- Useful commands from this step:

```bash
go test ./pkg/interpreter -run TestDetectFirstInvalidMethodRegister -v
go test ./pkg/interpreter -run TestLookupPointYMethod -v
go test ./pkg/interpreter -run TestTraceAroundMethodCorruption -v
pdftotext smalltalk-Bluebook.pdf - | sed -n '34890,34940p'
go run ./cmd/st80 data/VirtualImage 500000
go run ./cmd/st80 data/VirtualImage 2000000
go test ./...
git commit -m "Fix method metadata and cache decoding"
```

- Key Blue Book cache excerpt distilled into Go:

```text
Smalltalk: (((messageSelector bitAnd: class) bitAnd: 16rFF) bitShift: 2) + 1
Go (0-based): ((int(messageSelector) & int(class)) & 0xFF) << 2
```
