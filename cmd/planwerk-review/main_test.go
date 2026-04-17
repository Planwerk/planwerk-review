package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveBuildInfoUsesLdflagsVersion(t *testing.T) {
	bi := resolveBuildInfo("v1.2.3")
	if bi.Version != "v1.2.3" {
		t.Fatalf("Version = %q, want v1.2.3", bi.Version)
	}
	if bi.IsDev {
		t.Fatalf("IsDev = true, want false for tagged version")
	}
}

func TestResolveBuildInfoFallsBackWhenLdflagsDev(t *testing.T) {
	bi := resolveBuildInfo(devVersion)
	// When tests run under `go test`, debug.ReadBuildInfo is available but
	// Main.Version is "(devel)" which is filtered out, so Version remains
	// "dev". In binaries installed via `go install <pkg>@v1.2.3`, the
	// fallback promotes Main.Version to the resolved version.
	if bi.Version == "" {
		t.Fatalf("Version must not be empty after fallback")
	}
	if bi.GoVersion == "" {
		t.Fatalf("GoVersion must be populated from debug.ReadBuildInfo")
	}
}

func TestWriteVersionDefault(t *testing.T) {
	var buf bytes.Buffer
	writeVersion(&buf, buildInfo{Version: "v1.2.3"}, false)
	out := buf.String()
	if !strings.Contains(out, "planwerk-review version v1.2.3") {
		t.Fatalf("missing version line: %q", out)
	}
	if strings.Contains(out, "commit:") || strings.Contains(out, "built:") || strings.Contains(out, "go:") {
		t.Fatalf("non-verbose output must not include build metadata: %q", out)
	}
	if strings.Contains(out, "warning:") {
		t.Fatalf("non-dev build must not warn: %q", out)
	}
}

func TestWriteVersionVerbose(t *testing.T) {
	var buf bytes.Buffer
	writeVersion(&buf, buildInfo{
		Version:   "v1.2.3",
		Commit:    "abc123",
		BuildDate: "2026-04-17T11:07:47Z",
		GoVersion: "go1.26.1",
	}, true)
	out := buf.String()
	for _, want := range []string{
		"planwerk-review version v1.2.3",
		"commit: abc123",
		"built: 2026-04-17T11:07:47Z",
		"go: go1.26.1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("verbose output missing %q:\n%s", want, out)
		}
	}
}

func TestWriteVersionDevWarning(t *testing.T) {
	var buf bytes.Buffer
	writeVersion(&buf, buildInfo{Version: devVersion, IsDev: true}, false)
	if !strings.Contains(buf.String(), "warning:") {
		t.Fatalf("dev build must emit warning: %q", buf.String())
	}
}
