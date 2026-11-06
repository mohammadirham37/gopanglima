package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Run with: go test -run '^$' -bench . -benchmem ./framework/httpx

func benchRouter(b *testing.B) http.Handler {
	r := NewRouter()
	noop := func(c *Context) error { return c.NoContent() }
	for _, p := range []string{"/", "/users", "/users/{id}", "/users/{id}/posts", "/users/{id}/posts/{post}", "/static/{path...}"} {
		r.Get(p, noop)
	}
	h, err := r.Build(HandlerOptions{})
	if err != nil {
		b.Fatal(err)
	}
	return h
}

func benchServe(b *testing.B, h http.Handler, target string) {
	req := httptest.NewRequest("GET", target, nil)
	w := &discardWriter{h: http.Header{}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(w, req)
	}
}

func BenchmarkRouterStatic(b *testing.B)   { benchServe(b, benchRouter(b), "/users") }
func BenchmarkRouterParam(b *testing.B)    { benchServe(b, benchRouter(b), "/users/42/posts/7") }
func BenchmarkRouterCatchAll(b *testing.B) { benchServe(b, benchRouter(b), "/static/css/app.css") }

func BenchmarkMiddlewareChain(b *testing.B) {
	r := NewRouter()
	pass := func(next HandlerFunc) HandlerFunc { return func(c *Context) error { return next(c) } }
	r.Use(pass, pass, pass, pass, pass)
	r.Get("/", func(c *Context) error { return c.NoContent() })
	h, err := r.Build(HandlerOptions{})
	if err != nil {
		b.Fatal(err)
	}
	benchServe(b, h, "/")
}

func BenchmarkJSONResponse(b *testing.B) {
	r := NewRouter()
	payload := M{"id": 1, "name": "Ana", "tags": []string{"a", "b"}}
	r.Get("/", func(c *Context) error { return c.JSON(200, payload) })
	h, err := r.Build(HandlerOptions{})
	if err != nil {
		b.Fatal(err)
	}
	benchServe(b, h, "/")
}

type discardWriter struct{ h http.Header }

func (d *discardWriter) Header() http.Header         { return d.h }
func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }
func (d *discardWriter) WriteHeader(int)             {}
