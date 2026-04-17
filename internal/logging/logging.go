// Package logging configures the process-wide slog default logger and
// provides a human-friendly console handler plus JSON output for CI logs.
//
// All user-facing stderr output in planwerk-review flows through slog so
// that log level filtering, structured attributes, and machine-readable
// output are uniformly available across every command.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Format identifies the log output format.
type Format string

const (
	// FormatText emits one human-readable line per record on stderr.
	FormatText Format = "text"
	// FormatJSON emits one JSON object per record, suitable for CI log
	// aggregation and machine parsing.
	FormatJSON Format = "json"
)

// Options configures Init.
type Options struct {
	// Writer is the destination for log records. When nil, os.Stderr is used.
	Writer io.Writer
	// Format selects the output format; empty defaults to FormatText.
	Format Format
	// Verbose lowers the minimum level from Info to Debug.
	Verbose bool
}

// ParseFormat resolves a CLI-supplied format string. Empty input maps to
// FormatText.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "", "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("unknown log format %q, supported: text, json", s)
	}
}

// Init installs a slog.Default logger matching opts. It must be called
// once during process startup before any logging occurs.
func Init(opts Options) error {
	w := opts.Writer
	if w == nil {
		w = os.Stderr
	}

	level := slog.LevelInfo
	if opts.Verbose {
		level = slog.LevelDebug
	}

	var handler slog.Handler
	switch opts.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	case FormatText, "":
		handler = newConsoleHandler(w, level)
	default:
		return fmt.Errorf("unknown log format %q, supported: text, json", opts.Format)
	}

	slog.SetDefault(slog.New(handler))
	return nil
}

// consoleHandler renders slog records in a minimal human-readable form:
// "<prefix><message> [key=value...]". INFO records carry no prefix so the
// default output mirrors the pre-slog stderr style.
type consoleHandler struct {
	mu    *sync.Mutex
	w     io.Writer
	level slog.Leveler
	attrs []slog.Attr
}

func newConsoleHandler(w io.Writer, level slog.Leveler) *consoleHandler {
	return &consoleHandler{mu: &sync.Mutex{}, w: w, level: level}
}

func (h *consoleHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *consoleHandler) Handle(_ context.Context, r slog.Record) error {
	var sb strings.Builder
	switch {
	case r.Level >= slog.LevelError:
		sb.WriteString("error: ")
	case r.Level >= slog.LevelWarn:
		sb.WriteString("warning: ")
	case r.Level <= slog.LevelDebug:
		sb.WriteString("debug: ")
	}
	sb.WriteString(r.Message)

	writeAttr := func(a slog.Attr) {
		if a.Equal(slog.Attr{}) {
			return
		}
		sb.WriteByte(' ')
		sb.WriteString(a.Key)
		sb.WriteByte('=')
		sb.WriteString(a.Value.String())
	}
	for _, a := range h.attrs {
		writeAttr(a)
	}
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(a)
		return true
	})
	sb.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, sb.String())
	return err
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &nh
}

func (h *consoleHandler) WithGroup(_ string) slog.Handler {
	// Console output does not use groups; return self to keep the handler
	// stateless and preserve existing attrs.
	return h
}
