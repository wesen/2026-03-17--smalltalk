# Changelog

## 2026-03-17

- Initial workspace created


## 2026-03-17

Step 1: Fixed tagged-SmallInteger decoding for method headers, header extensions, and class instance specifications; this removes the startup context overflow and lets the VM run into the next runtime blocker (commit dd8e4ba).

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter.go — Interpreter metadata decoding fix
- /home/manuel/code/wesen/2026-03-17--smalltalk/pkg/interpreter/interpreter_test.go — Regression coverage for the former startup crash


## 2026-03-17

Added a detailed ticket writeup of the tagged-SmallInteger header decode bug for later review, and recorded the next runtime tasks after commit dd8e4ba.

### Related Files

- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/reference/02-tagged-smallinteger-header-decode-bug-writeup.md — Intern-facing bug explanation and validation steps
- /home/manuel/code/wesen/2026-03-17--smalltalk/ttmp/2026/03/17/ST80-002--smalltalk-80-interpreter-continuation-context-recovery-and-io-path/tasks.md — Continuation task list after the startup fix

