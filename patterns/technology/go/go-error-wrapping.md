# Review Pattern: Go Error Wrapping

**Review-Area**: quality
**Detection-Hint**: Bare `return err` without fmt.Errorf wrapping, error messages starting with uppercase or ending with punctuation
**Severity**: WARNING
**Category**: technology
**Applies-When**: go
**Sources**: Effective Go (https://go.dev/doc/effective_go#errors), Go Code Review Comments (https://github.com/golang/go/wiki/CodeReviewComments#error-strings)

## What to check

1. Every error returned from a function should provide context via `fmt.Errorf("doing X: %w", err)`
2. Error messages must start lowercase and not end with punctuation (Go convention)
3. Use `%w` (not `%v`) to preserve the error chain for `errors.Is`/`errors.As`
4. Avoid `errors.New()` when wrapping an existing error — use `fmt.Errorf` with `%w`

## Why it matters

Without context wrapping, error messages in logs become "file not found" instead of
"loading user config: opening ~/.config/app.yaml: file not found", making production
debugging significantly harder.
