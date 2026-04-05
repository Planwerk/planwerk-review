package claude

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// progressInterval is how often elapsed-time updates are printed while a
// Claude CLI invocation is in flight.
const progressInterval = 15 * time.Second

// progressMu serializes writes to the progress sink so that concurrent
// Claude invocations do not interleave bytes within a single update line.
var progressMu sync.Mutex

// stderrIsTerminalFn is overridable in tests.
var stderrIsTerminalFn = stderrIsTerminal

// stderrIsTerminal reports whether os.Stderr refers to a character device
// (i.e., an interactive terminal). When stderr is redirected to a file or
// pipe, progress output is suppressed.
func stderrIsTerminal() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// startProgress prints a periodic elapsed-time message to stderr while a
// Claude call is running. It returns a stop function that synchronously
// halts the goroutine; callers should `defer stop()` immediately after
// starting. If stderr is not a terminal, startProgress prints nothing and
// returns a no-op stop function.
//
// The indicator prints one line per update (rather than using \r) so that
// concurrent Claude invocations produce readable, non-interleaved output.
func startProgress(label string) func() {
	if !stderrIsTerminalFn() {
		return func() {}
	}
	return startProgressTo(os.Stderr, label, progressInterval)
}

// startProgressTo is the testable core of startProgress. It writes updates
// to w at the given interval until the returned stop function is called.
func startProgressTo(w io.Writer, label string, interval time.Duration) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	start := time.Now()

	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				elapsed := time.Since(start).Round(time.Second)
				progressMu.Lock()
				_, _ = fmt.Fprintf(w, "  [%s] still running... (elapsed: %s)\n", label, elapsed)
				progressMu.Unlock()
			}
		}
	}()

	return func() {
		close(stop)
		<-done
	}
}
