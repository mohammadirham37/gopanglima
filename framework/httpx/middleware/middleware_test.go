package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mohammadirham37/gopanglima/framework/httpx"
)

func build(t *testing.T, logs *bytes.Buffer, setup func(r *httpx.Router)) http.Handler {
	t.Helper()
	r := httpx.NewRouter()
	setup(r)
	h, err := r.Build(httpx.HandlerOptions{Logger: slog.New(slog.NewTextHandler(logs, nil))})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestRecover(t *testing.T) {
	var logs bytes.Buffer
	h := build(t, &logs, func(r *httpx.Router) {
		r.Use(Recover())
		r.Get("/panic", func(c *httpx.Context) error { panic("secret-panic-value") })
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/panic", nil))
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "secret-panic-value") {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(logs.String(), "panic recovered") || !strings.Contains(logs.String(), "secret-panic-value") {
		t.Fatalf("panic not logged: %q", logs.String())
	}
}

func TestRecoverRepanicsAbortHandler(t *testing.T) {
	h := build(t, &bytes.Buffer{}, func(r *httpx.Router) {
		r.Use(Recover())
		r.Get("/", func(c *httpx.Context) error { panic(http.ErrAbortHandler) })
	})
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Fatalf("expected ErrAbortHandler re-panic, got %v", rec)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRequestID(t *testing.T) {
	var logs bytes.Buffer
	h := build(t, &logs, func(r *httpx.Router) {
		r.Use(RequestID())
		r.Get("/", func(c *httpx.Context) error {
			c.Logger().Info("inside")
			return c.String(200, RequestIDFrom(c.Context()))
		})
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	id := rec.Header().Get(RequestIDHeader)
	if len(id) != 32 || rec.Body.String() != id {
		t.Fatalf("generated id %q body %q", id, rec.Body.String())
	}
	if !strings.Contains(logs.String(), "request_id="+id) {
		t.Fatalf("logger not enriched: %q", logs.String())
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(RequestIDHeader, "client-id_1.2:3")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get(RequestIDHeader) != "client-id_1.2:3" {
		t.Fatalf("valid client id not reused: %q", rec.Header().Get(RequestIDHeader))
	}

	for _, bad := range []string{"has space", "<script>", strings.Repeat("a", 129)} {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(RequestIDHeader, bad)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get(RequestIDHeader); got == bad || len(got) != 32 {
			t.Errorf("invalid id %q should be replaced, got %q", bad, got)
		}
	}
}

func TestRequestLogger(t *testing.T) {
	var logs bytes.Buffer
	h := build(t, &logs, func(r *httpx.Router) {
		r.Use(RequestLogger())
		r.Get("/users/{id}", func(c *httpx.Context) error { return httpx.NotFound("User not found") })
		r.Get("/ok", func(c *httpx.Context) error { return c.String(200, "hello") })
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/users/9?token=secret", nil))
	if rec.Code != 404 {
		t.Fatalf("status %d", rec.Code)
	}
	out := logs.String()
	for _, want := range []string{"level=WARN", "status=404", "route=/users/{id}", "path=/users/9"} {
		if !strings.Contains(out, want) {
			t.Errorf("log missing %q: %s", want, out)
		}
	}
	if strings.Contains(out, "secret") {
		t.Errorf("query string must not be logged: %s", out)
	}

	logs.Reset()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/ok", nil))
	if !strings.Contains(logs.String(), "level=INFO") || !strings.Contains(logs.String(), "bytes=5") {
		t.Errorf("unexpected log: %s", logs.String())
	}
}
