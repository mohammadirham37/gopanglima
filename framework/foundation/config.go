package foundation

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ErrConfigNotFound is returned by Lookup when a configuration key is not set.
var ErrConfigNotFound = errors.New("foundation: config key not found")

// Config is a concurrency-safe key/value configuration repository.
//
// Keys are flat strings that conventionally use dot notation, such as
// "app.name" or "database.host". Values keep their Go type; no implicit
// string conversion is performed.
//
// The convenience getters (String, Int, Bool, Duration) return the provided
// default when the key is missing or holds a value of a different type. Use
// the generic Lookup function when a missing or mistyped value must be
// treated as an error.
type Config struct {
	mu     sync.RWMutex
	values map[string]any
}

// NewConfig returns an empty Config.
func NewConfig() *Config {
	return &Config{values: make(map[string]any)}
}

// Set stores value under key, replacing any previous value.
func (c *Config) Set(key string, value any) {
	c.mu.Lock()
	c.values[key] = value
	c.mu.Unlock()
}

// SetMany stores every entry of values. Existing keys are replaced.
func (c *Config) SetMany(values map[string]any) {
	c.mu.Lock()
	for k, v := range values {
		c.values[k] = v
	}
	c.mu.Unlock()
}

// Get returns the raw value stored under key.
func (c *Config) Get(key string) (any, bool) {
	c.mu.RLock()
	v, ok := c.values[key]
	c.mu.RUnlock()
	return v, ok
}

// Has reports whether key is set.
func (c *Config) Has(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// Keys returns all configured keys in lexical order.
func (c *Config) Keys() []string {
	c.mu.RLock()
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	c.mu.RUnlock()
	sort.Strings(keys)
	return keys
}

// String returns the string stored under key, or def.
func (c *Config) String(key, def string) string {
	return getOr(c, key, def)
}

// Int returns the int stored under key, or def.
func (c *Config) Int(key string, def int) int {
	return getOr(c, key, def)
}

// Int64 returns the int64 stored under key, or def.
func (c *Config) Int64(key string, def int64) int64 {
	return getOr(c, key, def)
}

// Bool returns the bool stored under key, or def.
func (c *Config) Bool(key string, def bool) bool {
	return getOr(c, key, def)
}

// Duration returns the time.Duration stored under key, or def.
func (c *Config) Duration(key string, def time.Duration) time.Duration {
	return getOr(c, key, def)
}

// Lookup returns the value stored under key as type T. It returns an error
// wrapping ErrConfigNotFound when the key is missing, or a type mismatch
// error when the stored value is not a T.
func Lookup[T any](c *Config, key string) (T, error) {
	var zero T
	raw, ok := c.Get(key)
	if !ok {
		return zero, fmt.Errorf("%w: %s", ErrConfigNotFound, key)
	}
	v, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("foundation: config key %s holds %T, want %T", key, raw, zero)
	}
	return v, nil
}

func getOr[T any](c *Config, key string, def T) T {
	v, err := Lookup[T](c, key)
	if err != nil {
		return def
	}
	return v
}
