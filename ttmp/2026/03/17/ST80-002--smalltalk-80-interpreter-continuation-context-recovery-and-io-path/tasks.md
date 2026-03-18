# Tasks

## TODO

- [x] Diagnose the `Recursive not understood error encountered` failure after the header-decode fix
- [ ] Re-validate the interpreter against local `trace2` and `trace3` after the next runtime fix
- [x] Identify the current long-run methods / scheduler state to confirm the VM is now idling where expected
- [x] Trace the failing `LargePositiveInteger>>digitAt:put:` allocation/size path reached from `DisplayScreen class>>displayExtent:`
- [x] Diagnose the current late `value:` receiver corruption in `Behavior>>selectorAtMethod:setClass:` / `IdentityDictionary>>keyAtValue:ifAbsent:`
- [x] Revisit method-context recycling roots if the late `value:` failure is still caused by stale context/block OOP reuse
- [x] Design a safe method-context body reuse or equivalent memory-reclamation strategy; OOP-slot recycling alone does not stop late `blockCopy:` allocation corruption
- [ ] Restore correct small-vs-large context allocation once metadata decoding and runtime stability are trustworthy
- [ ] Decide whether the current context-only body reuse should remain a tactical stopgap or grow into broader object-memory reclamation
- [ ] Implement `BitBlt>>copyBits` and the remaining display/input primitives now exposed by the 2,000,000-cycle notifier/debugger path
- [ ] Implement the missing runtime pieces needed to reach a stable idle loop again (remaining primitives, I/O, display path)
- [ ] Create a separate UI ticket only after the interpreter/runtime is stable enough to support display work
