package patterns

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stubPushWiki swaps the package-level pushWiki seam with a test fake and returns
// a restore function, mirroring stubFetch.
func stubPushWiki(fn func(dir string, relPaths []string, commitMsg, token string) error) func() {
	old := pushWiki
	pushWiki = fn
	return func() { pushWiki = old }
}

func TestCloneWikiAuthenticated_DerivesWikiURLAndHead(t *testing.T) {
	var gotScheme, gotURL, gotRef string
	restore := stubFetch(func(p parsedURI, dest string) error {
		gotScheme, gotURL, gotRef = p.scheme, p.cloneURL, p.ref
		mustWrite(t, filepath.Join(dest, "review_patterns", "rule.md"), wikiPatternMarkdown)
		gitInitWithCommit(t, dest)
		return nil
	})
	defer restore()

	dir, head, cleanup, err := CloneWikiAuthenticated("acme/widgets", "main")
	if err != nil {
		t.Fatalf("CloneWikiAuthenticated: %v", err)
	}
	defer cleanup()

	if gotScheme != schemeWiki {
		t.Errorf("clone scheme = %q, want wiki", gotScheme)
	}
	if want := "https://github.com/acme/widgets.wiki.git"; gotURL != want {
		t.Errorf("clone URL = %q, want %q", gotURL, want)
	}
	if gotRef != "main" {
		t.Errorf("clone ref = %q, want main", gotRef)
	}
	if dir == "" {
		t.Error("clone dir must be non-empty")
	}
	if head == "" {
		t.Error("HEAD commit should be resolved from the fresh clone")
	}

	// cleanup removes the temp workspace.
	cleanup()
	if _, err := os.Stat(filepath.Dir(dir)); !os.IsNotExist(err) {
		t.Errorf("cleanup should remove the temp workspace, stat err = %v", err)
	}
}

func TestCloneWikiAuthenticated_FetchFailureCleansUp(t *testing.T) {
	restore := stubFetch(func(parsedURI, string) error {
		return os.ErrPermission
	})
	defer restore()

	_, _, cleanup, err := CloneWikiAuthenticated("acme/widgets", "")
	if err == nil {
		t.Fatal("expected an error when the clone fails")
	}
	if cleanup == nil {
		t.Fatal("cleanup must never be nil, even on error")
	}
	cleanup() // must be safe to call
}

func TestPushWikiDeletions_ForwardsToSeam(t *testing.T) {
	var gotDir, gotMsg string
	var gotPaths []string
	restore := stubPushWiki(func(dir string, relPaths []string, commitMsg, token string) error {
		gotDir, gotPaths, gotMsg = dir, relPaths, commitMsg
		return nil
	})
	defer restore()

	paths := []string{"review_patterns/stale.md", "memory/old.md"}
	if err := PushWikiDeletions("/tmp/wiki", paths, "prune stale entries"); err != nil {
		t.Fatalf("PushWikiDeletions: %v", err)
	}
	if gotDir != "/tmp/wiki" || gotMsg != "prune stale entries" {
		t.Errorf("forwarded dir=%q msg=%q, want /tmp/wiki / prune stale entries", gotDir, gotMsg)
	}
	if strings.Join(gotPaths, ",") != strings.Join(paths, ",") {
		t.Errorf("forwarded paths = %v, want %v", gotPaths, paths)
	}
}

func TestPushWikiDeletions_EmptyPathsIsError(t *testing.T) {
	restore := stubPushWiki(func(string, []string, string, string) error {
		t.Fatal("the seam must not run when there is nothing to delete")
		return nil
	})
	defer restore()

	if err := PushWikiDeletions("/tmp/wiki", nil, "noop"); err == nil {
		t.Fatal("expected an error for an empty deletion list")
	}
}

func TestWikiPushEnv_TokenStaysInEnvNotCleartext(t *testing.T) {
	const token = "ghs_supersecrettoken"
	env := wikiPushEnv(token)

	var countSet, keySet, valueSet bool
	for _, e := range env {
		switch e {
		case "GIT_CONFIG_COUNT=1":
			countSet = true
		case "GIT_CONFIG_KEY_0=http.extraHeader":
			keySet = true
		case "GIT_CONFIG_VALUE_0=" + wikiAuthHeader(token):
			valueSet = true
		}
		if strings.Contains(e, token) {
			t.Errorf("raw token leaked into env entry: %q", e)
		}
	}
	if !countSet || !keySet || !valueSet {
		t.Errorf("token must be injected via GIT_CONFIG_*; count=%v key=%v value=%v", countSet, keySet, valueSet)
	}

	if wikiPushEnv("") != nil {
		t.Error("no token should leave the env nil (inherited), the anonymous push path")
	}
}

// TestPushWiki_RemovesCommitsAndPushes exercises the real git rm/commit/push
// against a local bare remote (no network), proving the deletion reaches the
// remote's HEAD tree.
func TestPushWiki_RemovesCommitsAndPushes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	bare := t.TempDir()
	git(t, bare, "init", "--bare", "-q")

	// Seed the bare remote with two pages via a throwaway clone.
	seed := t.TempDir()
	git(t, seed, "clone", "-q", bare, ".")
	git(t, seed, "config", "user.email", "t@example.com")
	git(t, seed, "config", "user.name", "tester")
	mustWrite(t, filepath.Join(seed, "review_patterns", "stale.md"), "stale\n")
	mustWrite(t, filepath.Join(seed, "review_patterns", "keep.md"), "keep\n")
	git(t, seed, "add", "-A")
	git(t, seed, "commit", "-q", "-m", "seed")
	git(t, seed, "push", "-q", "origin", "HEAD")

	// Fresh clone the deletion is applied to.
	clone := t.TempDir()
	git(t, clone, "clone", "-q", bare, ".")

	if err := pushWiki(clone, []string{"review_patterns/stale.md"}, "prune stale", ""); err != nil {
		t.Fatalf("pushWiki: %v", err)
	}

	tree := gitOut(t, bare, "ls-tree", "-r", "--name-only", "HEAD")
	if strings.Contains(tree, "stale.md") {
		t.Errorf("stale.md should be gone from the remote HEAD tree:\n%s", tree)
	}
	if !strings.Contains(tree, "keep.md") {
		t.Errorf("keep.md should still be present in the remote HEAD tree:\n%s", tree)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir}, args...)
	if out, err := exec.Command("git", full...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", full...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}
