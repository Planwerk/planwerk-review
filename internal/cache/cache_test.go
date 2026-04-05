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

func TestRepoKey(t *testing.T) {
	k1 := RepoKey("owner", "repo", "abc123")
	k2 := RepoKey("owner", "repo", "abc123")
	k3 := RepoKey("owner", "repo", "def456")
	k4 := RepoKey("other", "repo", "abc123")

	if k1 != k2 {
		t.Error("same inputs should produce same key")
	}
	if k1 == k3 {
		t.Error("different HEAD SHA should produce different key")
	}
	if k1 == k4 {
		t.Error("different owner should produce different key")
	}
	if len(k1) != 32 {
		t.Errorf("key length = %d, want 32 hex chars", len(k1))
	}
}

func TestAuditKey(t *testing.T) {
	k1 := AuditKey("owner", "repo", "abc123")
	k2 := AuditKey("owner", "repo", "abc123")
	k3 := AuditKey("owner", "repo", "def456")
	k4 := AuditKey("owner", "repo", "abc123", "min=critical")

	if k1 != k2 {
		t.Error("same inputs should produce same key")
	}
	if k1 == k3 {
		t.Error("different HEAD SHA should produce different key")
	}
	if k1 == k4 {
		t.Error("different flags should produce different key")
	}
	if len(k1) != 32 {
		t.Errorf("key length = %d, want 32 hex chars", len(k1))
	}
}

func TestAuditKey_DistinctFromRepoKey(t *testing.T) {
	// AuditKey and RepoKey must namespace differently so audit and propose
	// caches cannot collide for the same repo+SHA.
	auditK := AuditKey("owner", "repo", "abc123")
	proposeK := RepoKey("owner", "repo", "abc123")
	if auditK == proposeK {
		t.Error("AuditKey and RepoKey must use distinct namespaces")
	}
}
