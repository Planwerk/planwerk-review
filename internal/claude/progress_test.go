package claude

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// safeBuffer is a bytes.Buffer guarded by a mutex so the progress goroutine
// and the test goroutine can access it without a data race.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestStartProgressTo_EmitsLabeledElapsedUpdates(t *testing.T) {
	var buf safeBuffer
	stop := startProgressTo(&buf, "review", 10*time.Millisecond)
	// Wait for at least two ticks so we see periodic updates.
	time.Sleep(35 * time.Millisecond)
	stop()

	out := buf.String()
	if !strings.Contains(out, "[review]") {
		t.Errorf("output should contain the label, got: %q", out)
	}
	if !strings.Contains(out, "elapsed:") {
		t.Errorf("output should contain elapsed marker, got: %q", out)
	}
	if !strings.Contains(out, "still running") {
		t.Errorf("output should describe running state, got: %q", out)
	}
	if strings.Count(out, "\n") < 2 {
		t.Errorf("expected at least two updates on separate lines, got: %q", out)
	}
}

func TestStartProgressTo_StopIsSynchronousAndSilentAfter(t *testing.T) {
	var buf safeBuffer
	stop := startProgressTo(&buf, "review", 10*time.Millisecond)
	time.Sleep(25 * time.Millisecond)
	stop()
	afterStop := buf.String()

	// After stop returns, no more output may appear. Give the goroutine a
	// chance to misbehave, then assert that the buffer is unchanged.
	time.Sleep(30 * time.Millisecond)
	if buf.String() != afterStop {
		t.Errorf("progress goroutine wrote after stop() returned:\nbefore: %q\nafter:  %q", afterStop, buf.String())
	}
}

func TestStartProgress_NoOpWhenStderrNotTerminal(t *testing.T) {
	prev := stderrIsTerminalFn
	stderrIsTerminalFn = func() bool { return false }
	t.Cleanup(func() { stderrIsTerminalFn = prev })

	// The returned stop must be callable and must not block or panic.
	stop := startProgress("review")
	stop()
}

// Ensure startProgressTo accepts an io.Writer (compile-time check).
var _ io.Writer = (*safeBuffer)(nil)
