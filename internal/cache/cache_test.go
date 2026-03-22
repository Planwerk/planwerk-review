package cache

import "testing"

func TestKey(t *testing.T) {
	k1 := Key("owner", "repo", 1, "abc123")
	k2 := Key("owner", "repo", 1, "abc123")
	k3 := Key("owner", "repo", 1, "def456")

	if k1 != k2 {
		t.Error("same inputs should produce same key")
	}
	if k1 == k3 {
		t.Error("different SHA should produce different key")
	}
	if len(k1) != 32 {
		t.Errorf("key length = %d, want 32 hex chars", len(k1))
	}
}
