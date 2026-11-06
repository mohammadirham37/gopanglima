package foundation

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env provides read access to environment variables.
//
// Values come from two layers. The process environment always wins; values
// loaded from .env files are used only when the process environment does not
// define the key. This mirrors the common convention that real deployment
// environments override local development files.
//
// An Env is immutable after construction and safe for concurrent use.
type Env struct {
	file   map[string]string
	lookup func(string) (string, bool)
}

// NewEnv creates an Env from explicit file values and a lookup function for
// the process environment. A nil lookup disables the process layer, which is
// useful in tests. The values map is copied.
func NewEnv(values map[string]string, lookup func(string) (string, bool)) *Env {
	copied := make(map[string]string, len(values))
	for k, v := range values {
		copied[k] = v
	}
	return &Env{file: copied, lookup: lookup}
}

// SystemEnv returns an Env backed only by the process environment.
func SystemEnv() *Env {
	return NewEnv(nil, os.LookupEnv)
}

// LoadEnv reads the given .env files and returns an Env layered over the
// process environment.
//
// Files that do not exist are skipped, so a missing .env file is not an error
// (production deployments usually provide real environment variables
// instead). Malformed files return an error that includes the file name and
// line number. When several files define the same key, the later file wins.
//
// LoadEnv never modifies the process environment.
func LoadEnv(paths ...string) (*Env, error) {
	values := make(map[string]string)
	for _, path := range paths {
		f, err := os.Open(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("foundation: open env file: %w", err)
		}
		parsed, err := ParseDotEnv(f)
		closeErr := f.Close()
		if err != nil {
			return nil, fmt.Errorf("foundation: parse %s: %w", path, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("foundation: close env file: %w", closeErr)
		}
		for k, v := range parsed {
			values[k] = v
		}
	}
	return &Env{file: values, lookup: os.LookupEnv}, nil
}

// Lookup returns the value of key and whether it was defined in either layer.
func (e *Env) Lookup(key string) (string, bool) {
	if e == nil {
		return "", false
	}
	if e.lookup != nil {
		if v, ok := e.lookup(key); ok {
			return v, true
		}
	}
	v, ok := e.file[key]
	return v, ok
}

// String returns the value of key, or def when the key is not defined.
// A key defined with an empty value returns the empty string.
func (e *Env) String(key, def string) string {
	if v, ok := e.Lookup(key); ok {
		return v
	}
	return def
}

// Bool parses key as a boolean. Accepted values are those understood by
// strconv.ParseBool plus "yes"/"no" and "on"/"off" (case-insensitive).
// It returns def when the key is undefined or empty, and an error when the
// value cannot be parsed.
func (e *Env) Bool(key string, def bool) (bool, error) {
	v, ok := e.Lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def, nil
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "yes", "on":
		return true, nil
	case "no", "off":
		return false, nil
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return def, fmt.Errorf("foundation: env %s: invalid boolean %q", key, v)
	}
	return b, nil
}

// Int parses key as a base-10 integer. It returns def when the key is
// undefined or empty, and an error when the value cannot be parsed.
func (e *Env) Int(key string, def int) (int, error) {
	v, ok := e.Lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def, fmt.Errorf("foundation: env %s: invalid integer %q", key, v)
	}
	return n, nil
}

// Duration parses key with time.ParseDuration (for example "30s" or "1m").
// It returns def when the key is undefined or empty, and an error when the
// value cannot be parsed.
func (e *Env) Duration(key string, def time.Duration) (time.Duration, error) {
	v, ok := e.Lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return def, fmt.Errorf("foundation: env %s: invalid duration %q", key, v)
	}
	return d, nil
}

// ParseDotEnv parses .env formatted content.
//
// Supported syntax:
//
//	# comment lines and blank lines are ignored
//	KEY=value                 # unquoted; trailing " #comment" is stripped
//	export KEY=value          # optional "export " prefix
//	KEY="double quoted\n"     # supports \n \r \t \" \\ escapes
//	KEY='single quoted'       # taken literally
//
// Keys must match [A-Za-z_][A-Za-z0-9_]*. Values must fit on one line.
// Variable interpolation (${OTHER}) is intentionally not supported.
func ParseDotEnv(r io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if lineNo == 1 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "export "); ok {
			line = strings.TrimSpace(rest)
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", lineNo)
		}
		key = strings.TrimSpace(key)
		if !validEnvKey(key) {
			return nil, fmt.Errorf("line %d: invalid key %q", lineNo, key)
		}
		value, err := parseEnvValue(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		values[key] = value
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func validEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		switch {
		case r == '_', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

func parseEnvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	switch raw[0] {
	case '\'':
		end := strings.IndexByte(raw[1:], '\'')
		if end < 0 {
			return "", errors.New("unterminated single-quoted value")
		}
		if err := checkTrailing(raw[end+2:]); err != nil {
			return "", err
		}
		return raw[1 : end+1], nil
	case '"':
		var b strings.Builder
		for i := 1; i < len(raw); i++ {
			c := raw[i]
			switch c {
			case '\\':
				if i+1 >= len(raw) {
					return "", errors.New("unterminated double-quoted value")
				}
				i++
				switch raw[i] {
				case 'n':
					b.WriteByte('\n')
				case 'r':
					b.WriteByte('\r')
				case 't':
					b.WriteByte('\t')
				case '"', '\\':
					b.WriteByte(raw[i])
				default:
					b.WriteByte('\\')
					b.WriteByte(raw[i])
				}
			case '"':
				if err := checkTrailing(raw[i+1:]); err != nil {
					return "", err
				}
				return b.String(), nil
			default:
				b.WriteByte(c)
			}
		}
		return "", errors.New("unterminated double-quoted value")
	default:
		// Strip an inline comment that is preceded by whitespace.
		for i := 1; i < len(raw); i++ {
			if raw[i] == '#' && (raw[i-1] == ' ' || raw[i-1] == '\t') {
				raw = raw[:i]
				break
			}
		}
		return strings.TrimSpace(raw), nil
	}
}

func checkTrailing(rest string) error {
	rest = strings.TrimSpace(rest)
	if rest == "" || strings.HasPrefix(rest, "#") {
		return nil
	}
	return fmt.Errorf("unexpected characters after quoted value: %q", rest)
}
