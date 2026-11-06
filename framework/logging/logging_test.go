package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"": slog.LevelInfo, "INFO": slog.LevelInfo, "debug": slog.LevelDebug,
		"warn": slog.LevelWarn, "Warning": slog.LevelWarn, "error": slog.LevelError,
	}
	for in, want := range cases {
		got, err := ParseLevel(in)
		if err != nil || got != want {
			t.Errorf("ParseLevel(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseLevel("loud"); err == nil {
		t.Error("expected error for unknown level")
	}
}

func TestNewJSONRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(&buf, Options{Level: "warn", Format: FormatJSON})
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hidden")
	logger.Warn("shown", "k", "v")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d: %q", len(lines), buf.String())
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if rec["msg"] != "shown" || rec["k"] != "v" {
		t.Fatalf("unexpected record %v", rec)
	}
}

func TestNewText(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(&buf, Options{})
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hello")
	if !strings.Contains(buf.String(), "msg=hello") {
		t.Fatalf("unexpected text output %q", buf.String())
	}
}

func TestNewUnknownFormat(t *testing.T) {
	if _, err := New(&bytes.Buffer{}, Options{Format: "xml"}); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestDiscard(t *testing.T) {
	l := Discard().With("a", 1).WithGroup("g")
	l.Error("nothing happens")
}
