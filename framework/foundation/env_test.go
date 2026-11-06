package foundation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseDotEnv(t *testing.T) {
	input := "\ufeff# comment\n" +
		"\n" +
		"APP_NAME=goPanglima\n" +
		"export APP_ENV=local\n" +
		"SPACED =  value with spaces   \n" +
		"INLINE=abc # trailing comment\n" +
		"HASH=abc#def\n" +
		"EMPTY=\n" +
		`DQ="line1\nline2 \"quoted\" \\ end" # comment` + "\n" +
		`SQ='raw \n $HOME'` + "\n" +
		"EQUALS=a=b=c\n"

	got, err := ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDotEnv: %v", err)
	}
	want := map[string]string{
		"APP_NAME": "goPanglima",
		"APP_ENV":  "local",
		"SPACED":   "value with spaces",
		"INLINE":   "abc",
		"HASH":     "abc#def",
		"EMPTY":    "",
		"DQ":       "line1\nline2 \"quoted\" \\ end",
		"SQ":       `raw \n $HOME`,
		"EQUALS":   "a=b=c",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d keys %v, want %d", len(got), got, len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseDotEnvErrors(t *testing.T) {
	cases := map[string]string{
		"missing equals":     "JUST_A_KEY\n",
		"invalid key":        "1KEY=value\n",
		"key with dash":      "MY-KEY=value\n",
		"unterminated dq":    `KEY="abc` + "\n",
		"unterminated sq":    "KEY='abc\n",
		"trailing after dq":  `KEY="abc" junk` + "\n",
		"trailing after sq":  "KEY='abc' junk\n",
		"dangling backslash": `KEY="abc\`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseDotEnv(strings.NewReader(input))
			if err == nil {
				t.Fatalf("expected error for %q", input)
			}
			if !strings.Contains(err.Error(), "line 1") {
				t.Errorf("error %q should mention the line number", err)
			}
		})
	}
}

func TestLoadEnvLayering(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, ".env")
	second := filepath.Join(dir, ".env.local")
	writeFile(t, first, "A=from-first\nB=from-first\nC=from-first\n")
	writeFile(t, second, "B=from-second\n")

	env, err := LoadEnv(first, filepath.Join(dir, "missing.env"), second)
	if err != nil {
		t.Fatalf("LoadEnv: %v", err)
	}
	if got := env.String("A", ""); got != "from-first" {
		t.Errorf("A = %q", got)
	}
	if got := env.String("B", ""); got != "from-second" {
		t.Errorf("later file should win, B = %q", got)
	}

	t.Setenv("C", "from-process")
	if got := env.String("C", ""); got != "from-process" {
		t.Errorf("process env should win, C = %q", got)
	}
	if got := env.String("NOT_DEFINED_ANYWHERE_XYZ", "fallback"); got != "fallback" {
		t.Errorf("default not used: %q", got)
	}
}

func TestLoadEnvReportsParseErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	writeFile(t, path, "OK=1\nBROKEN\n")
	_, err := LoadEnv(path)
	if err == nil || !strings.Contains(err.Error(), "line 2") || !strings.Contains(err.Error(), path) {
		t.Fatalf("expected error mentioning file and line 2, got %v", err)
	}
}

func TestEnvTypedGetters(t *testing.T) {
	env := NewEnv(map[string]string{
		"T1": "true", "T2": "yes", "F1": "off", "BAD_BOOL": "maybe",
		"N": " 42 ", "BAD_N": "4x2",
		"D": "1m30s", "BAD_D": "soon",
		"BLANK": "",
	}, nil)

	for key, want := range map[string]bool{"T1": true, "T2": true, "F1": false} {
		got, err := env.Bool(key, !want)
		if err != nil || got != want {
			t.Errorf("Bool(%s) = %v, %v; want %v", key, got, err, want)
		}
	}
	if got, err := env.Bool("BLANK", true); err != nil || !got {
		t.Errorf("blank bool should use default, got %v %v", got, err)
	}
	if _, err := env.Bool("BAD_BOOL", false); err == nil {
		t.Error("expected error for invalid bool")
	}
	if got, err := env.Int("N", 0); err != nil || got != 42 {
		t.Errorf("Int = %v, %v", got, err)
	}
	if _, err := env.Int("BAD_N", 0); err == nil {
		t.Error("expected error for invalid int")
	}
	if got, err := env.Duration("D", 0); err != nil || got != 90*time.Second {
		t.Errorf("Duration = %v, %v", got, err)
	}
	if _, err := env.Duration("BAD_D", 0); err == nil {
		t.Error("expected error for invalid duration")
	}
	if got, err := env.Int("MISSING", 7); err != nil || got != 7 {
		t.Errorf("missing int should use default, got %v %v", got, err)
	}
}

func TestNilEnvIsSafe(t *testing.T) {
	var env *Env
	if _, ok := env.Lookup("X"); ok {
		t.Fatal("nil env should not find keys")
	}
}

func TestNewEnvCopiesValues(t *testing.T) {
	values := map[string]string{"K": "v1"}
	env := NewEnv(values, nil)
	values["K"] = "v2"
	if got := env.String("K", ""); got != "v1" {
		t.Fatalf("env should not observe later mutations, got %q", got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
