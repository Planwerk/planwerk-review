package claude

import (
	"sync"
	"testing"
	"time"
)

// TestClient_ConcurrentConfigIsolation builds Clients with distinct
// timeout/model/effort configuration from many goroutines at once and asserts
// each Client reads back only its own values. Configuration lives on the Client
// struct rather than in package-level state it replaced, so concurrent runners
// cannot leak config into one another. Run under -race to catch any reintroduced
// shared mutable state.
func TestClient_ConcurrentConfigIsolation(t *testing.T) {
	cases := []struct {
		model, effort string
		timeout       time.Duration
	}{
		{"opus", "xhigh", 5 * time.Minute},
		{"fable", "max", 30 * time.Minute},
		{"sonnet", "high", 10 * time.Minute},
	}

	var wg sync.WaitGroup
	for _, tc := range cases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				c := NewClient(
					WithModel(tc.model),
					WithEffort(tc.effort),
					WithTimeout(tc.timeout),
				)
				if c.model != tc.model || c.effort != tc.effort || c.timeout != tc.timeout {
					t.Errorf("client config leaked: got (%q, %q, %s), want (%q, %q, %s)",
						c.model, c.effort, c.timeout, tc.model, tc.effort, tc.timeout)
					return
				}
			}
		}()
	}
	wg.Wait()
}
