package github

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Repo holds information about a cloned repository.
type Repo struct {
	Owner string
	Name  string
	Dir   string // local clone directory (temp dir, caller must clean up)
}

// CloneRepo clones the given repository into a temp directory and returns a Repo.
func CloneRepo(ref string) (*Repo, error) {
	owner, name, err := ParseRepoRef(ref)
	if err != nil {
		return nil, err
	}

	dir, err := cloneRepo(owner, name)
	if err != nil {
		return nil, fmt.Errorf("cloning repo: %w", err)
	}

	return &Repo{
		Owner: owner,
		Name:  name,
		Dir:   dir,
	}, nil
}

// Cleanup removes the temporary clone directory.
func (r *Repo) Cleanup() {
	if r.Dir != "" {
		_ = os.RemoveAll(r.Dir)
	}
}

// FullName returns "owner/name".
func (r *Repo) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

// DefaultBranchHEAD returns the HEAD SHA of the default branch via git ls-remote.
// This is used for cache invalidation so proposals are refreshed when the repo changes.
func DefaultBranchHEAD(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	cmd := exec.Command("git", "ls-remote", url, "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote: %w", err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return "", fmt.Errorf("git ls-remote returned no output for %s/%s", owner, repo)
	}
	return fields[0], nil
}

func cloneRepo(owner, name string) (string, error) {
	dir, err := os.MkdirTemp("", "planwerk-propose-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, name)
	clone := exec.Command("git", "clone", "--filter=blob:none", cloneURL, dir)
	clone.Stderr = os.Stderr
	if err := clone.Run(); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("git clone: %w", err)
	}

	return dir, nil
}

var (
	// https://github.com/owner/repo or https://github.com/owner/repo.git
	repoURLRe = regexp.MustCompile(`github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)
	// owner/repo (no # or / at end)
	repoShortRe = regexp.MustCompile(`^([^/#]+)/([^/#]+)$`)
)

// ParseRepoRef parses a repo reference in URL or short form (owner/repo).
func ParseRepoRef(ref string) (owner, repo string, err error) {
	ref = strings.TrimSpace(ref)

	if m := repoURLRe.FindStringSubmatch(ref); m != nil {
		owner, repo = m[1], m[2]
		if err := validateOwnerRepo(owner, repo); err != nil {
			return "", "", err
		}
		return owner, repo, nil
	}

	if m := repoShortRe.FindStringSubmatch(ref); m != nil {
		owner, repo = m[1], m[2]
		if err := validateOwnerRepo(owner, repo); err != nil {
			return "", "", err
		}
		return owner, repo, nil
	}

	return "", "", fmt.Errorf("invalid repo reference %q: expected URL (https://github.com/owner/repo) or short form (owner/repo)", ref)
}
