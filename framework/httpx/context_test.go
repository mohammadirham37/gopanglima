package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type userInput struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func bindHandler(t *testing.T, opts HandlerOptions) http.Handler {
	t.Helper()
	r := NewRouter()
	r.Post("/bind", func(c *Context) error {
		var in userInput
		if err := c.Bind(&in); err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, in)
	})
	h, err := r.Build(opts)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func postJSON(h http.Handler, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/bind", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestBind(t *testing.T) {
	h := bindHandler(t, HandlerOptions{MaxBodyBytes: 64})

	rec := postJSON(h, "application/json; charset=utf-8", `{"name":"Ana","age":30,"extra":true}`)
	if rec.Code != 201 || strings.TrimSpace(rec.Body.String()) != `{"name":"Ana","age":30}` {
		t.Fatalf("valid bind: %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if rec := postJSON(h, "application/vnd.api+json", `{"name":"x"}`); rec.Code != 201 {
		t.Errorf("+json media type: %d", rec.Code)
	}

	cases := []struct {
		name, ct, body string
		status         int
		msg            string
	}{
		{"no content type", "", `{}`, 415, "Content-Type must be application/json"},
		{"form content type", "application/x-www-form-urlencoded", "a=b", 415, "Content-Type"},
		{"bad content type", "application/json; =", "{}", 415, "Malformed Content-Type"},
		{"empty body", "application/json", "", 400, "must not be empty"},
		{"syntax error", "application/json", `{"name":}`, 400, "Malformed JSON at position"},
		{"truncated", "application/json", `{"name":"x"`, 400, "unexpected end"},
		{"type error", "application/json", `{"age":"old"}`, 400, `Invalid value for field \"age\"`},
		{"wrong top-level", "application/json", `[1]`, 400, "Invalid JSON value type"},
		{"trailing data", "application/json", `{"name":"a"}{"name":"b"}`, 400, "single JSON value"},
		{"too large", "application/json", `{"name":"` + strings.Repeat("x", 100) + `"}`, 413, "must not exceed 64 bytes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postJSON(h, tc.ct, tc.body)
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.msg) {
				t.Fatalf("got %d %q, want %d containing %q", rec.Code, rec.Body.String(), tc.status, tc.msg)
			}
		})
	}
}

func TestBindNonPointerIsInternalError(t *testing.T) {
	r := NewRouter()
	r.Post("/x", func(c *Context) error { return c.Bind(userInput{}) })
	h := mustBuild(t, r)
	req := httptest.NewRequest("POST", "/x", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 500 {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestResponseHelpers(t *testing.T) {
	r := NewRouter()
	r.Get("/string", func(c *Context) error { return c.String(200, "hi") })
	r.Get("/html", func(c *Context) error { return c.HTML(200, "<b>x</b>") })
	r.Get("/blob", func(c *Context) error { return c.Blob(202, "image/png", []byte{1, 2}) })
	r.Get("/redirect", func(c *Context) error { return c.Redirect("/dest") })
	r.Get("/redirect301", func(c *Context) error { return c.RedirectWithStatus(301, "/dest") })
	r.Get("/badredirect", func(c *Context) error { return c.RedirectWithStatus(200, "/dest") })
	r.Get("/nocontent", func(c *Context) error { return c.NoContent() })
	r.Get("/query", func(c *Context) error {
		return c.String(200, c.Query("q")+"|"+c.QueryDefault("page", "1")+"|"+c.Header("X-Test"))
	})
	h := mustBuild(t, r)

	check := func(target string, status int, ct, body string) {
		t.Helper()
		rec := do(h, "GET", target, nil)
		if rec.Code != status || rec.Header().Get("Content-Type") != ct || (body != "" && rec.Body.String() != body) {
			t.Errorf("GET %s = %d %q %q", target, rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
		}
	}
	check("/string", 200, "text/plain; charset=utf-8", "hi")
	check("/html", 200, "text/html; charset=utf-8", "<b>x</b>")
	check("/blob", 202, "image/png", "\x01\x02")
	check("/nocontent", 204, "", "")
	check("/badredirect", 500, "application/json; charset=utf-8", "")

	if rec := do(h, "GET", "/redirect", nil); rec.Code != 302 || rec.Header().Get("Location") != "/dest" {
		t.Errorf("redirect = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := do(h, "GET", "/redirect301", nil); rec.Code != 301 {
		t.Errorf("redirect301 = %d", rec.Code)
	}
	if rec := do(h, "GET", "/string", nil); rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff header")
	}

	req := httptest.NewRequest("GET", "/query?q=go&page=", nil)
	req.Header.Set("X-Test", "yes")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Body.String() != "go|1|yes" {
		t.Errorf("query = %q", rec.Body.String())
	}
}

func TestDoubleWriteReturnsErrResponseCommitted(t *testing.T) {
	var second error
	r := NewRouter()
	r.Get("/", func(c *Context) error {
		if err := c.String(200, "first"); err != nil {
			return err
		}
		second = c.JSON(500, M{"x": 1})
		return nil
	})
	h := mustBuild(t, r)
	rec := do(h, "GET", "/", nil)
	if !errors.Is(second, ErrResponseCommitted) {
		t.Fatalf("second write err = %v", second)
	}
	if rec.Code != 200 || rec.Body.String() != "first" {
		t.Fatalf("response corrupted: %d %q", rec.Code, rec.Body.String())
	}
}

func TestJSONEncodingErrorWritesNothing(t *testing.T) {
	r := NewRouter()
	var encErr error
	r.Get("/", func(c *Context) error {
		encErr = c.JSON(200, M{"ch": make(chan int)})
		if c.Response().Committed() {
			t.Error("response committed despite encoding failure")
		}
		return encErr
	})
	h := mustBuild(t, r)
	rec := do(h, "GET", "/", nil)
	if encErr == nil || rec.Code != 500 {
		t.Fatalf("encErr=%v code=%d", encErr, rec.Code)
	}
}

func TestErrorHandlerHidesInternalErrors(t *testing.T) {
	secret := errors.New("db password=hunter2 rejected")
	newHandler := func(debug bool, logs *bytes.Buffer) http.Handler {
		r := NewRouter()
		r.Get("/internal", func(c *Context) error { return fmt.Errorf("query users: %w", secret) })
		r.Get("/notfound", func(c *Context) error { return NotFound("User not found").WithCause(secret) })
		r.Get("/conflict", func(c *Context) error { return fmt.Errorf("wrapped: %w", Conflict("")) })
		h, err := r.Build(HandlerOptions{Debug: debug, Logger: slog.New(slog.NewTextHandler(logs, nil))})
		if err != nil {
			t.Fatal(err)
		}
		return h
	}

	var logs bytes.Buffer
	h := newHandler(false, &logs)

	rec := do(h, "GET", "/internal", nil)
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "hunter2") {
		t.Fatalf("internal error leaked: %d %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(logs.String(), "hunter2") {
		t.Errorf("root cause should be logged, logs=%q", logs.String())
	}

	rec = do(h, "GET", "/notfound", nil)
	var body struct {
		Error map[string]any `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != 404 || body.Error["message"] != "User not found" || strings.Contains(rec.Body.String(), "hunter2") {
		t.Fatalf("not found = %d %q", rec.Code, rec.Body.String())
	}

	if rec := do(h, "GET", "/conflict", nil); rec.Code != 409 || !strings.Contains(rec.Body.String(), "Conflict") {
		t.Fatalf("wrapped *Error should keep status, got %d %q", rec.Code, rec.Body.String())
	}

	debugH := newHandler(true, &bytes.Buffer{})
	if rec := do(debugH, "GET", "/internal", nil); !strings.Contains(rec.Body.String(), "detail") {
		t.Fatalf("debug mode should include detail, got %q", rec.Body.String())
	}
}

func TestErrorHandlerSkipsCanceledRequests(t *testing.T) {
	r := NewRouter()
	r.Get("/", func(c *Context) error { return c.Context().Err() })
	h := mustBuild(t, r)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Body.Len() != 0 {
		t.Fatalf("expected no body for canceled request, got %q", rec.Body.String())
	}
}

func TestCustomErrorHandler(t *testing.T) {
	r := NewRouter()
	r.Get("/", func(c *Context) error { return errors.New("boom") })
	h, err := r.Build(HandlerOptions{ErrorHandler: func(c *Context, err error) {
		_ = c.String(418, "custom:"+err.Error())
	}})
	if err != nil {
		t.Fatal(err)
	}
	if rec := do(h, "GET", "/", nil); rec.Code != 418 || rec.Body.String() != "custom:boom" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestContextValuesAndLogger(t *testing.T) {
	var logs bytes.Buffer
	r := NewRouter()
	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Set("user", "ana")
			c.SetLogger(c.Logger().With("tag", "mw"))
			c.SetLogger(nil) // ignored
			type key struct{}
			c.SetRequest(c.Request().WithContext(context.WithValue(c.Context(), key{}, "v")))
			return next(c)
		}
	})
	r.Get("/", func(c *Context) error {
		u, ok := c.Get("user")
		_, missing := c.Get("nope")
		c.Logger().Info("hello")
		return c.String(200, fmt.Sprintf("%v %v %v", u, ok, missing))
	})
	h, err := r.Build(HandlerOptions{Logger: slog.New(slog.NewTextHandler(&logs, nil))})
	if err != nil {
		t.Fatal(err)
	}
	if rec := do(h, "GET", "/", nil); rec.Body.String() != "ana true false" {
		t.Fatalf("got %q", rec.Body.String())
	}
	if !strings.Contains(logs.String(), "tag=mw") {
		t.Fatalf("enriched logger not used: %q", logs.String())
	}
}

func TestHTTPErrorHelpers(t *testing.T) {
	cases := map[int]*Error{
		400: BadRequest(""), 401: Unauthorized(""), 403: Forbidden(""), 404: NotFound(""),
		405: MethodNotAllowed(""), 409: Conflict(""), 413: RequestEntityTooLarge(""),
		415: UnsupportedMediaType(""), 422: UnprocessableEntity(""), 429: TooManyRequests(""),
		500: InternalServerError(""),
	}
	for code, e := range cases {
		if e.Code != code || e.Message != http.StatusText(code) {
			t.Errorf("%d: %+v", code, e)
		}
	}
	cause := errors.New("cause")
	e := NotFound("x").WithCause(cause)
	if !errors.Is(e, cause) || !strings.Contains(e.Error(), "cause") {
		t.Errorf("WithCause: %v", e)
	}
	base := NotFound("x")
	_ = base.WithCause(cause)
	if base.Err != nil {
		t.Error("WithCause must not mutate the receiver")
	}
}

func TestBadBuildOptions(t *testing.T) {
	if _, err := NewRouter().Build(HandlerOptions{MaxBodyBytes: -1}); err == nil {
		t.Fatal("negative MaxBodyBytes should fail")
	}
}
