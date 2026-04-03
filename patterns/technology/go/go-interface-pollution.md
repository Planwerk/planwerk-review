# Review Pattern: Go Interface Pollution

**Review-Area**: architecture
**Detection-Hint**: Interfaces defined alongside their only implementation, interfaces with many methods, interfaces defined by the producer instead of the consumer
**Severity**: INFO
**Category**: technology
**Applies-When**: go
**Sources**: Effective Go (https://go.dev/doc/effective_go#interfaces), Go Code Review Comments (https://github.com/golang/go/wiki/CodeReviewComments#interfaces)

## What to check

1. Interfaces should be defined by the consumer, not the producer
2. Interfaces with a single implementation and no test mocks are likely premature
3. Prefer small interfaces (1-3 methods) — the bigger the interface, the weaker the abstraction
4. Don't export interfaces just for mocking — accept interfaces, return structs
5. Standard library interfaces (`io.Reader`, `io.Writer`, `fmt.Stringer`) should be preferred over custom ones when applicable

## Why it matters

Go proverb: "The bigger the interface, the weaker the abstraction." Premature interfaces add indirection without value and make code harder to navigate.
