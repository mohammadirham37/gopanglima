// Package logging builds structured loggers on top of log/slog.
//
// goPanglima uses *slog.Logger directly throughout the framework, so any
// slog handler can be plugged in. This package only provides a small
// constructor driven by configuration values such as LOG_LEVEL and
// LOG_FORMAT.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Format selects the slog handler used by New.
type Format string

const (
	// FormatText produces human-readable key=value lines.
	FormatText Format = "text"
	// FormatJSON produces one JSON object per line.
	FormatJSON Format = "json"
)

// Options configures New.
type Options struct {
	// Level is the minimum level: "debug", "info", "warn", or "error".
	// An empty value means "info".
	Level string
	// Format is FormatText or FormatJSON. An empty value means FormatText.
	Format Format
	// AddSource includes the source file and line in each record.
	AddSource bool
}

// New returns a logger that writes to w according to opts. It returns an
// error for unknown levels or formats rather than silently falling back.
func New(w io.Writer, opts Options) (*slog.Logger, error) {
	level, err := ParseLevel(opts.Level)
	if err != nil {
		return nil, err
	}
	hopts := &slog.HandlerOptions{Level: level, AddSource: opts.AddSource}
	switch Format(strings.ToLower(string(opts.Format))) {
	case "", FormatText:
		return slog.New(slog.NewTextHandler(w, hopts)), nil
	case FormatJSON:
		return slog.New(slog.NewJSONHandler(w, hopts)), nil
	default:
		return nil, fmt.Errorf("logging: unknown format %q (want %q or %q)", opts.Format, FormatText, FormatJSON)
	}
}

// ParseLevel converts a level name to a slog.Level. Matching is
// case-insensitive; "warning" is accepted as an alias for "warn". An empty
// string returns slog.LevelInfo.
func ParseLevel(name string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("logging: unknown level %q", name)
	}
}

// Discard returns a logger that drops every record. It is useful in tests.
func Discard() *slog.Logger {
	return slog.New(discardHandler{})
}

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardHandler) WithGroup(string) slog.Handler           { return d }
