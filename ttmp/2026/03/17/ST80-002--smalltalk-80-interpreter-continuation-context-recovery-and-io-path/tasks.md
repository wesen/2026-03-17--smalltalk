# Tasks

## TODO

- [x] Diagnose the `Recursive not understood error encountered` failure after the header-decode fix
- [ ] Re-validate the interpreter against local `trace2` and `trace3` after the next runtime fix
- [ ] Identify the current long-run methods / scheduler state to confirm the VM is now idling where expected
- [ ] Trace the failing `LargePositiveInteger>>digitAt:put:` allocation/size path reached from `DisplayScreen class>>displayExtent:`
- [ ] Restore correct small-vs-large context allocation once metadata decoding and runtime stability are trustworthy
- [ ] Implement the missing runtime pieces needed to reach a stable idle loop again (remaining primitives, I/O, display path)
- [ ] Create a separate UI ticket only after the interpreter/runtime is stable enough to support display work
