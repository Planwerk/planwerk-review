package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/planwerk/planwerk-review/internal/report"
)

var cacheDir = defaultCacheDir()

func defaultCacheDir() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "planwerk-review")
	}
	return filepath.Join(os.TempDir(), "planwerk-review-cache")
}

// Key generates a cache key from the PR head SHA and review flags.
func Key(owner, repo string, number int, headSHA string, flags ...string) string {
	input := fmt.Sprintf("%s/%s#%d@%s", owner, repo, number, headSHA)
	for _, f := range flags {
		input += "+" + f
	}
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:16])
}

// Get retrieves a cached review result, if available.
func Get(key string) (*report.ReviewResult, bool) {
	path := filepath.Join(cacheDir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var result report.ReviewResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, false
	}
	return &result, true
}

// Put stores a review result in the cache.
func Put(key string, result *report.ReviewResult) error {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, key+".json"), data, 0o600)
}

// RepoKey generates a cache key for repository analysis.
// It includes the HEAD SHA so the cache invalidates when the repo changes.
func RepoKey(owner, repo, headSHA string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("propose:%s/%s@%s", owner, repo, headSHA)))
	return fmt.Sprintf("%x", h[:16])
}

// AuditKey generates a cache key for a full-codebase audit.
// It includes the default-branch HEAD SHA so the cache invalidates when the
// repo changes, plus any flags that alter the audit output.
func AuditKey(owner, repo, headSHA string, flags ...string) string {
	input := fmt.Sprintf("audit:%s/%s@%s", owner, repo, headSHA)
	for _, f := range flags {
		input += "+" + f
	}
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:16])
}

// GetRaw retrieves raw cached data.
func GetRaw(key string) ([]byte, bool) {
	path := filepath.Join(cacheDir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// PutRaw stores raw data in the cache.
func PutRaw(key string, data []byte) error {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, key+".json"), data, 0o600)
}

// Clear removes all cached review results.
func Clear() error {
	entries, err := os.ReadDir(cacheDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	removed := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			if err := os.Remove(filepath.Join(cacheDir, e.Name())); err != nil {
				return err
			}
			removed++
		}
	}
	fmt.Fprintf(os.Stderr, "Removed %d cached review(s).\n", removed)
	return nil
}
