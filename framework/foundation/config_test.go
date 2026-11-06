package foundation

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestConfigGetters(t *testing.T) {
	c := NewConfig()
	c.SetMany(map[string]any{
		"app.name":    "goPanglima",
		"app.debug":   true,
		"db.port":     5432,
		"db.max":      int64(10),
		"srv.timeout": 5 * time.Second,
	})

	if got := c.String("app.name", ""); got != "goPanglima" {
		t.Errorf("String = %q", got)
	}
	if got := c.Bool("app.debug", false); !got {
		t.Error("Bool = false")
	}
	if got := c.Int("db.port", 0); got != 5432 {
		t.Errorf("Int = %d", got)
	}
	if got := c.Int64("db.max", 0); got != 10 {
		t.Errorf("Int64 = %d", got)
	}
	if got := c.Duration("srv.timeout", 0); got != 5*time.Second {
		t.Errorf("Duration = %v", got)
	}
	// Missing key and wrong type both fall back to the default.
	if got := c.String("missing", "def"); got != "def" {
		t.Errorf("missing String = %q", got)
	}
	if got := c.String("db.port", "def"); got != "def" {
		t.Errorf("mistyped String = %q", got)
	}
}

func TestConfigLookup(t *testing.T) {
	c := NewConfig()
	c.Set("app.name", "x")

	if v, err := Lookup[string](c, "app.name"); err != nil || v != "x" {
		t.Fatalf("Lookup = %q, %v", v, err)
	}
	if _, err := Lookup[string](c, "nope"); !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("want ErrConfigNotFound, got %v", err)
	}
	if _, err := Lookup[int](c, "app.name"); err == nil || errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("want type mismatch error, got %v", err)
	}
}

func TestConfigKeysSortedAndHas(t *testing.T) {
	c := NewConfig()
	c.Set("b", 1)
	c.Set("a", 2)
	keys := c.Keys()
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("Keys = %v", keys)
	}
	if !c.Has("a") || c.Has("z") {
		t.Fatal("Has mismatch")
	}
}

func TestConfigConcurrentAccess(t *testing.T) {
	c := NewConfig()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("k%d", i%4)
			c.Set(key, i)
			_ = c.Int(key, 0)
			_ = c.Keys()
		}(i)
	}
	wg.Wait()
}
