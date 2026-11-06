package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Encoded slashes must not let parameters carry path traversal sequences.
func TestEncodedSlashTraversalIsRejected(t *testing.T) {
	r := NewRouter()
	r.Get("/u/{id}", func(c *Context) error { return c.String(200, "id="+c.Param("id")) })
	r.Get("/files/{p...}", func(c *Context) error { return c.String(200, "p="+c.Param("p")) })
	h := mustBuild(t, r)

	for _, target := range []string{
		"/u/..%2F..%2Fadmin",
		"/u/a%2Fb",
		"/files/a%2F..%2F..%2Fetc%2Fpasswd",
		"/files/%2E%2E%2Fsecret",
		"/files/ok/%2E%2E",
	} {
		if rec := do(h, "GET", target, nil); rec.Code != 404 {
			t.Errorf("GET %s = %d %q, want 404", target, rec.Code, rec.Body.String())
		}
	}
	// Encoded slashes without dot elements remain usable in catch-alls.
	if rec := do(h, "GET", "/files/a%2Fb/c", nil); rec.Body.String() != "p=a/b/c" {
		t.Errorf("catch-all with %%2F = %d %q", rec.Code, rec.Body.String())
	}
}

func TestInvalidErrorStatusBecomes500(t *testing.T) {
	r := NewRouter()
	r.Get("/zero", func(c *Context) error { return &Error{Message: "x"} })
	r.Get("/huge", func(c *Context) error { return NewError(1000, "x") })
	r.Get("/ok-range", func(c *Context) error { return NewError(302, "x") })
	h := mustBuild(t, r)
	for _, target := range []string{"/zero", "/huge", "/ok-range"} {
		if rec := do(h, "GET", target, nil); rec.Code != 500 {
			t.Errorf("GET %s = %d, want 500", target, rec.Code)
		}
	}
}

func TestResponseHelpersRejectInvalidStatus(t *testing.T) {
	r := NewRouter()
	var errs []error
	r.Get("/", func(c *Context) error {
		errs = append(errs, c.String(0, "x"), c.Status(1000), c.JSON(150, M{}))
		return c.String(200, "fine")
	})
	h := mustBuild(t, r)
	rec := do(h, "GET", "/", nil)
	for i, err := range errs {
		if err == nil {
			t.Errorf("call %d: expected error", i)
		}
	}
	if rec.Code != 200 || rec.Body.String() != "fine" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestCleanPathRedirectRunsGlobalMiddleware(t *testing.T) {
	r := NewRouter()
	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error { c.SetHeader("X-Global", "1"); return next(c) }
	})
	r.Get("/u/x", text("x"))
	h := mustBuild(t, r)
	rec := do(h, "GET", "/u//x", nil)
	if rec.Code != http.StatusPermanentRedirect || rec.Header().Get("X-Global") != "1" || rec.Header().Get("Location") != "/u/x" {
		t.Fatalf("got %d headers=%v", rec.Code, rec.Header())
	}
}

func TestInformationalStatusDoesNotCommit(t *testing.T) {
	r := NewRouter()
	r.Get("/", func(c *Context) error {
		c.Response().WriteHeader(http.StatusEarlyHints)
		if c.Response().Committed() {
			t.Error("1xx must not commit the response")
		}
		return c.JSON(201, M{"ok": true})
	})
	h := mustBuild(t, r)
	if rec := do(h, "GET", "/", nil); !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("final response not written: %d %q", rec.Code, rec.Body.String())
	}
}

func TestFlushCommitsStatus(t *testing.T) {
	r := NewRouter()
	var second error
	r.Get("/", func(c *Context) error {
		if err := http.NewResponseController(c.Response()).Flush(); err != nil {
			return err
		}
		if !c.Response().Committed() || c.Response().Status() != 200 {
			t.Errorf("flush should commit 200, status=%d", c.Response().Status())
		}
		second = c.JSON(201, M{})
		return nil
	})
	h := mustBuild(t, r)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !rec.Flushed || rec.Code != 200 || second != ErrResponseCommitted {
		t.Fatalf("flushed=%v code=%d second=%v", rec.Flushed, rec.Code, second)
	}
}

func TestStaticSegmentsEscaping(t *testing.T) {
	r := NewRouter()
	r.Get("/caf%C3%A9", text("x"))
	if _, err := r.Build(HandlerOptions{}); err == nil {
		t.Fatal("percent in pattern should be rejected")
	}

	r = NewRouter()
	r.Get("/café/a b", text("ok")).Name("cafe")
	h := mustBuild(t, r)
	u, err := r.URL("cafe", nil)
	if err != nil || u != "/caf%C3%A9/a%20b" {
		t.Fatalf("URL = %q, %v", u, err)
	}
	if rec := do(h, "GET", u, nil); rec.Body.String() != "ok" {
		t.Fatalf("generated URL does not route: %d", rec.Code)
	}
}
