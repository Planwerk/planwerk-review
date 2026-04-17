package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	cases := []struct {
		in      string
		want    Format
		wantErr bool
	}{
		{"", FormatText, false},
		{"text", FormatText, false},
		{"TEXT", FormatText, false},
		{"json", FormatJSON, false},
		{"JSON", FormatJSON, false},
		{"yaml", "", true},
	}
	for _, tc := range cases {
		got, err := ParseFormat(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseFormat(%q) expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseFormat(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestConsoleHandler_PrefixesByLevel(t *testing.T) {
	var buf bytes.Buffer
	h := newConsoleHandler(&buf, slog.LevelDebug)
	log := slog.New(h)

	log.Debug("hello debug")
	log.Info("hello info")
	log.Warn("hello warn")
	log.Error("hello error")

	out := buf.String()
	want := []string{
		"debug: hello debug\n",
		"hello info\n",
		"warning: hello warn\n",
		"error: hello error\n",
	}
	for _, line := range want {
		if !strings.Contains(out, line) {
			t.Errorf("output missing %q; got:\n%s", line, out)
		}
	}
}

func TestConsoleHandler_RendersAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := newConsoleHandler(&buf, slog.LevelInfo)
	log := slog.New(h)

	log.Info("loaded patterns", "count", 42)
	out := buf.String()
	if !strings.Contains(out, "loaded patterns count=42") {
		t.Errorf("expected attr rendered inline, got: %q", out)
	}
}

func TestConsoleHandler_RespectsLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	h := newConsoleHandler(&buf, slog.LevelInfo)
	log := slog.New(h)

	log.Debug("should be suppressed")
	log.Info("should appear")

	out := buf.String()
	if strings.Contains(out, "suppressed") {
		t.Errorf("debug record should be filtered at info level, got: %q", out)
	}
	if !strings.Contains(out, "should appear") {
		t.Errorf("info record should be emitted, got: %q", out)
	}
}

func TestInit_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := Init(Options{Writer: &buf, Format: FormatJSON}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	slog.Info("structured", "key", "value")

	var rec map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &rec); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", buf.String(), err)
	}
	if rec["msg"] != "structured" {
		t.Errorf("expected msg=structured, got %v", rec["msg"])
	}
	if rec["key"] != "value" {
		t.Errorf("expected key=value attribute, got %v", rec["key"])
	}
}

func TestInit_VerboseEnablesDebug(t *testing.T) {
	var buf bytes.Buffer
	if err := Init(Options{Writer: &buf, Format: FormatText, Verbose: true}); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	slog.Debug("visible when verbose")
	if !strings.Contains(buf.String(), "visible when verbose") {
		t.Errorf("debug record missing with verbose=true: %q", buf.String())
	}
}

func TestInit_RejectsUnknownFormat(t *testing.T) {
	if err := Init(Options{Format: Format("yaml")}); err == nil {
		t.Error("expected error for unknown format")
	}
}
