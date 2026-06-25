package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/planwerk/planwerk-agent/internal/cli"
)

// runShipCmd executes the ship subcommand hermetically: it exercises only the
// RunE validation and argument wiring. The abort cases return before any
// gh/Claude call, so no real backend is touched.
func runShipCmd(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	var out, errBuf bytes.Buffer
	cmd := newShipCmd(&runtimeDeps{version: "test"})
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.Bytes(), err
}

func TestShipCmd_RequiresExactlyOneArg(t *testing.T) {
	if _, err := runShipCmd(t); err == nil {
		t.Fatalf("expected an error when no issue ref is given")
	}
	if _, err := runShipCmd(t, "a/b#1", "a/b#2"); err == nil {
		t.Fatalf("expected an error when two issue refs are given")
	}
}

func TestShipCmd_UnknownMergeMethod(t *testing.T) {
	_, err := runShipCmd(t, "--merge-method", "fast-forward", "acme/widgets#42")
	if err == nil || !strings.Contains(err.Error(), "unknown --merge-method") {
		t.Fatalf("expected an unknown-merge-method error, got %v", err)
	}
}

func TestShipCmd_RejectsNonPositiveInterval(t *testing.T) {
	_, err := runShipCmd(t, "--interval", "0", "acme/widgets#42")
	if err == nil || !strings.Contains(err.Error(), "--interval must be > 0") {
		t.Fatalf("expected an interval error, got %v", err)
	}
}

func TestShipCmd_RejectsNonPositiveMaxFixIterations(t *testing.T) {
	_, err := runShipCmd(t, "--max-fix-iterations", "0", "acme/widgets#42")
	if err == nil || !strings.Contains(err.Error(), "--max-fix-iterations must be > 0") {
		t.Fatalf("expected a max-fix-iterations error, got %v", err)
	}
}

func TestShipCmd_RejectsNegativeStartAt(t *testing.T) {
	_, err := runShipCmd(t, "--start-at", "-3", "acme/widgets#42")
	if err == nil || !strings.Contains(err.Error(), "--start-at") {
		t.Fatalf("expected a start-at error, got %v", err)
	}
}

// The three valid merge methods clear flag validation (they fail later, at the
// GitHub fetch, which a hermetic test does not reach — so we only assert the
// validation does not reject them).
func TestShipConfig_ToShipOptions(t *testing.T) {
	opts := cli.ShipConfig{
		IssueRef:    "acme/widgets#42",
		MergeMethod: "squash",
		NoMerge:     true,
		StartAt:     7,
	}.ToShipOptions()
	if opts.MergeMethod != "squash" || !opts.NoMerge || opts.StartAt != 7 {
		t.Fatalf("ToShipOptions mapped wrong: %+v", opts)
	}
}

func TestShipConfig_ToShipImplementOptions(t *testing.T) {
	opts := cli.ShipConfig{
		NoSimplify:  true,
		NoReview:    true,
		MaxPatterns: 12,
	}.ToShipImplementOptions("test")
	if !opts.NoSimplify || !opts.NoReview || opts.MaxPatterns != 12 {
		t.Fatalf("ToShipImplementOptions mapped wrong: %+v", opts)
	}
	if opts.DryRun {
		t.Fatalf("ship's implement options must not set DryRun")
	}
}
