# Review Pattern: Test Behavior, Not Implementation

**Review-Area**: testing
**Detection-Hint**: Tests asserting on call counts or interaction order, mocks standing in for in-process collaborators, assertions on private state, tests that break on refactors that preserve behavior
**Severity**: INFO
**Category**: design-principle
**Sources**: Growing Object-Oriented Software Guided by Tests (Freeman/Pryce), Test Driven Development: By Example (Kent Beck)

## What to check

1. Tests assert on observable behavior — return values, emitted events, persisted state, visible side effects — not on how the code computes them. A test that pins the implementation breaks on every refactor while missing real behavioral regressions.
2. Mock only at system boundaries (the ports & adapters seams to a database, network, filesystem, or clock). Replacing in-process collaborators with mocks couples the test to the call graph and exercises the wiring instead of the behavior.
3. Do NOT assert on call counts or interaction order for in-process collaborators ("was this method called exactly once, then that one"). Such assertions encode the current implementation, not the contract, and turn behavior-preserving refactors into red tests.
4. Prefer real objects and fakes over mocks when the collaborator is cheap and deterministic; reserve mocks for the slow, non-deterministic, or external dependencies a test genuinely cannot use directly.

## Why it matters

Tests earn their keep by catching regressions while surviving refactors. A suite that asserts on call counts and internal interactions does the reverse: it fails when behavior is preserved and passes when behavior quietly changes, so it slows every refactor and protects nothing. Mocking only at system boundaries keeps the unit under test and its in-process collaborators exercised together, so the test verifies what the code does rather than how it is wired.
