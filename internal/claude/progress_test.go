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

func TestStartProgress_NonTerminalEmitsSlogHeartbeat(t *testing.T) {
	prev := stderrIsTerminalFn
	stderrIsTerminalFn = func() bool { return false }
	t.Cleanup(func() { stderrIsTerminalFn = prev })

	// The returned stop must be callable and must not block or panic even
	// when no heartbeat has fired yet.
	stop := startProgress("review")
	stop()
}

func TestStartProgressLogged_EmitsSlogHeartbeat(t *testing.T) {
	type entry struct {
		msg  string
		args []any
	}
	var (
		mu      sync.Mutex
		entries []entry
	)
	prev := slogInfoFn
	slogInfoFn = func(msg string, args ...any) {
		mu.Lock()
		entries = append(entries, entry{msg: msg, args: args})
		mu.Unlock()
	}
	t.Cleanup(func() { slogInfoFn = prev })

	stop := startProgressLogged("review", 10*time.Millisecond)
	time.Sleep(35 * time.Millisecond)
	stop()

	mu.Lock()
	defer mu.Unlock()
	if len(entries) < 2 {
		t.Fatalf("expected at least two heartbeat log entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.msg != "claude still running" {
			t.Errorf("unexpected heartbeat message: %q", e.msg)
		}
		if len(e.args) < 4 {
			t.Errorf("expected label+elapsed attrs, got %v", e.args)
		}
	}
}

// Ensure startProgressTo accepts an io.Writer (compile-time check).
var _ io.Writer = (*safeBuffer)(nil)
