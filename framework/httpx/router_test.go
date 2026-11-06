package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func mustBuild(t testing.TB, r *Router) http.Handler {
	t.Helper()
	h, err := r.Build(HandlerOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return h
}

func do(h http.Handler, method, target string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func text(s string) HandlerFunc {
	return func(c *Context) error { return c.String(http.StatusOK, s) }
}

func TestRouterMethods(t *testing.T) {
	r := NewRouter()
	r.Get("/res", text("get"))
	r.Post("/res", text("post"))
	r.Put("/res", text("put"))
	r.Patch("/res", text("patch"))
	r.Delete("/res", text("delete"))
	r.Options("/res", text("options"))
	r.Head("/res", func(c *Context) error { c.SetHeader("X-Head", "1"); return c.NoContent() })
	r.Handle("PURGE", "/res", text("purge"))
	h := mustBuild(t, r)

	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "PURGE"} {
		rec := do(h, m, "/res", nil)
		if rec.Code != 200 || rec.Body.String() != strings.ToLower(m) {
			t.Errorf("%s: got %d %q", m, rec.Code, rec.Body.String())
		}
	}
	rec := do(h, "HEAD", "/res", nil)
	if rec.Code != 204 || rec.Header().Get("X-Head") != "1" {
		t.Errorf("HEAD: got %d %v", rec.Code, rec.Header())
	}
}

func TestRouterParams(t *testing.T) {
	r := NewRouter()
	r.Get("/users/{id}", func(c *Context) error { return c.String(200, "user:"+c.Param("id")) })
	r.Get("/users/{id}/posts/{post}", func(c *Context) error {
		return c.String(200, c.Param("id")+"/"+c.Param("post")+"/"+c.Param("missing"))
	})
	r.Get("/users/new", text("static-wins"))
	r.Get("/files/{path...}", func(c *Context) error { return c.String(200, "file:"+c.Param("path")) })
	r.Get("/", text("root"))
	h := mustBuild(t, r)

	cases := []struct{ target, want string }{
		{"/", "root"},
		{"/users/42", "user:42"},
		{"/users/new", "static-wins"},
		{"/users/7/posts/9", "7/9/"},
		{"/users/a%20b", "user:a b"},
		{"/files/a/b/c.txt", "file:a/b/c.txt"},
		{"/files/", "file:"},
	}
	for _, tc := range cases {
		rec := do(h, "GET", tc.target, nil)
		if rec.Code != 200 || rec.Body.String() != tc.want {
			t.Errorf("GET %s = %d %q, want %q", tc.target, rec.Code, rec.Body.String(), tc.want)
		}
	}
	for _, target := range []string{"/users", "/users/", "/files", "/nope", "/users/1/posts", "/users/a%2Fb"} {
		if rec := do(h, "GET", target, nil); rec.Code != 404 {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
	}
}

func TestRouterParamsBacktrack(t *testing.T) {
	r := NewRouter()
	r.Get("/a/static/x", text("static"))
	r.Get("/a/{p}/y", func(c *Context) error { return c.String(200, "param:"+c.Param("p")) })
	h := mustBuild(t, r)
	if rec := do(h, "GET", "/a/static/y", nil); rec.Body.String() != "param:static" {
		t.Fatalf("expected backtracking to param route, got %d %q", rec.Code, rec.Body.String())
	}
}

func TestRouterRejectsEncodedDotParams(t *testing.T) {
	r := NewRouter()
	r.Get("/files/{name}", text("one"))
	r.Get("/all/{rest...}", text("rest"))
	h := mustBuild(t, r)
	for _, target := range []string{"/files/%2E%2E", "/files/%2e", "/all/a/%2E%2E/b"} {
		if rec := do(h, "GET", target, nil); rec.Code != 404 {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
	}
}

func TestRouterCleansPaths(t *testing.T) {
	r := NewRouter()
	r.Get("/a/b", text("ab"))
	h := mustBuild(t, r)
	rec := do(h, "GET", "/a/../a//b?x=1", nil)
	if rec.Code != http.StatusPermanentRedirect || rec.Header().Get("Location") != "/a/b?x=1" {
		t.Fatalf("got %d Location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestRouterTrailingSlashIsSignificant(t *testing.T) {
	r := NewRouter()
	r.Get("/docs/", text("dir"))
	h := mustBuild(t, r)
	if rec := do(h, "GET", "/docs/", nil); rec.Body.String() != "dir" {
		t.Fatalf("GET /docs/ = %q", rec.Body.String())
	}
	if rec := do(h, "GET", "/docs", nil); rec.Code != 404 {
		t.Fatalf("GET /docs = %d", rec.Code)
	}
}

func TestRouter405AndAutoOptions(t *testing.T) {
	r := NewRouter()
	r.Get("/items/{id}", text("get"))
	r.Put("/items/{id}", text("put"))
	r.Post("/items/new", text("new"))
	h := mustBuild(t, r)

	rec := do(h, "DELETE", "/items/1", nil)
	if rec.Code != 405 {
		t.Fatalf("DELETE = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET, HEAD, OPTIONS, PUT" {
		t.Errorf("Allow = %q", got)
	}
	var body struct {
		Error struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Status != 405 {
		t.Errorf("405 body = %q (%v)", rec.Body.String(), err)
	}

	// "/items/new" has POST statically but GET via the param route.
	if got := do(h, "GET", "/items/new", nil); got.Body.String() != "get" {
		t.Errorf("GET /items/new = %d %q", got.Code, got.Body.String())
	}
	rec = do(h, "PATCH", "/items/new", nil)
	if got := rec.Header().Get("Allow"); got != "GET, HEAD, OPTIONS, POST, PUT" {
		t.Errorf("Allow for /items/new = %q", got)
	}

	rec = do(h, "OPTIONS", "/items/1", nil)
	if rec.Code != 204 || rec.Header().Get("Allow") != "GET, HEAD, OPTIONS, PUT" {
		t.Errorf("OPTIONS = %d Allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
	if rec := do(h, "OPTIONS", "/unknown", nil); rec.Code != 404 {
		t.Errorf("OPTIONS unknown = %d", rec.Code)
	}
}

func TestRouterHeadFallsBackToGet(t *testing.T) {
	r := NewRouter()
	r.Get("/page", text("body"))
	h := mustBuild(t, r)
	rec := do(h, "HEAD", "/page", nil)
	if rec.Code != 200 || rec.Body.Len() != 0 || rec.Header().Get("Content-Type") == "" {
		t.Fatalf("HEAD = %d body=%q headers=%v", rec.Code, rec.Body.String(), rec.Header())
	}
}

func recorder(log *[]string, name string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			*log = append(*log, name+">")
			err := next(c)
			*log = append(*log, "<"+name)
			return err
		}
	}
}

func TestMiddlewareOrder(t *testing.T) {
	var log []string
	r := NewRouter()
	r.Use(recorder(&log, "global1"))

	api := r.Prefix("/api").Middleware(recorder(&log, "api"))
	api.Group(func(api *Router) {
		api.Get("/ping", func(c *Context) error {
			log = append(log, "handler")
			return c.NoContent()
		}).Middleware(recorder(&log, "route1"), recorder(&log, "route2"))

		api.Middleware(recorder(&log, "auth")).Group(func(r *Router) {
			r.Get("/me", func(c *Context) error {
				log = append(log, "me")
				return c.NoContent()
			})
		})
	})
	// Global middleware added after routes still applies to them.
	r.Use(recorder(&log, "global2"))
	h := mustBuild(t, r)

	do(h, "GET", "/api/ping", nil)
	want := "global1> global2> api> route1> route2> handler <route2 <route1 <api <global2 <global1"
	if got := strings.Join(log, " "); got != want {
		t.Errorf("order:\n got  %s\n want %s", got, want)
	}

	log = nil
	do(h, "GET", "/api/me", nil)
	want = "global1> global2> api> auth> me <auth <api <global2 <global1"
	if got := strings.Join(log, " "); got != want {
		t.Errorf("nested order:\n got  %s\n want %s", got, want)
	}

	log = nil
	if rec := do(h, "GET", "/missing", nil); rec.Code != 404 {
		t.Fatalf("missing = %d", rec.Code)
	}
	if got := strings.Join(log, " "); got != "global1> global2> <global2 <global1" {
		t.Errorf("404 should run only global middleware, got %s", got)
	}
}

func TestMiddlewareCanShortCircuit(t *testing.T) {
	deny := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error { return Unauthorized("Login required") }
	}
	r := NewRouter()
	called := false
	r.Middleware(deny).Get("/secret", func(c *Context) error { called = true; return nil })
	h := mustBuild(t, r)
	rec := do(h, "GET", "/secret", nil)
	if rec.Code != 401 || called || !strings.Contains(rec.Body.String(), "Login required") {
		t.Fatalf("got %d %q called=%v", rec.Code, rec.Body.String(), called)
	}
}

func TestPrefixJoining(t *testing.T) {
	r := NewRouter()
	v1 := r.Prefix("/api/").Prefix("/v1")
	v1.Get("/", text("index"))
	v1.Get("/users", text("users"))
	r.Prefix("/teams/{team}").Get("/members/{id}", func(c *Context) error {
		return c.String(200, c.Param("team")+":"+c.Param("id"))
	})
	h := mustBuild(t, r)
	for target, want := range map[string]string{"/api/v1": "index", "/api/v1/users": "users", "/teams/red/members/3": "red:3"} {
		if rec := do(h, "GET", target, nil); rec.Body.String() != want {
			t.Errorf("GET %s = %d %q, want %q", target, rec.Code, rec.Body.String(), want)
		}
	}
}

func TestRegistrationErrors(t *testing.T) {
	cases := map[string]func(r *Router){
		"no leading slash":   func(r *Router) { r.Get("users", text("")) },
		"nil handler":        func(r *Router) { r.Get("/x", nil) },
		"bad method":         func(r *Router) { r.Handle("GE T", "/x", text("")) },
		"partial param":      func(r *Router) { r.Get("/user-{id}", text("")) },
		"bad param name":     func(r *Router) { r.Get("/{1d}", text("")) },
		"dup param":          func(r *Router) { r.Get("/{id}/{id}", text("")) },
		"catch-all not last": func(r *Router) { r.Get("/{p...}/x", text("")) },
		"double slash":       func(r *Router) { r.Get("/a//b", text("")) },
		"dot segment":        func(r *Router) { r.Get("/a/../b", text("")) },
		"duplicate route":    func(r *Router) { r.Get("/a/{x}", text("")); r.Get("/a/{y}", text("")) },
		"duplicate catchall": func(r *Router) { r.Get("/a/{x...}", text("")); r.Get("/a/{y...}", text("")) },
		"duplicate name": func(r *Router) {
			r.Get("/a", text("")).Name("n")
			r.Get("/b", text("")).Name("n")
		},
		"empty name":     func(r *Router) { r.Get("/a", text("")).Name("") },
		"nil middleware": func(r *Router) { r.Use(nil) },
		"nil route mw":   func(r *Router) { r.Get("/a", text("")).Middleware(nil) },
		"bad prefix":     func(r *Router) { r.Prefix("api").Get("/x", text("")) },
	}
	for name, register := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewRouter()
			register(r)
			if _, err := r.Build(HandlerOptions{}); err == nil {
				t.Fatal("expected Build error")
			}
		})
	}
}

func TestSameRouteDifferentMethodsIsNotAConflict(t *testing.T) {
	r := NewRouter()
	r.Get("/a/{x}", text("get"))
	r.Post("/a/{y}", text("post"))
	h := mustBuild(t, r)
	if rec := do(h, "POST", "/a/1", nil); rec.Body.String() != "post" {
		t.Fatalf("POST = %q", rec.Body.String())
	}
}

func TestRouterFrozenAfterBuild(t *testing.T) {
	r := NewRouter()
	rt := r.Get("/a", text("a"))
	h := mustBuild(t, r)
	r.Get("/b", text("b"))
	rt.Name("late")
	r.Use(func(next HandlerFunc) HandlerFunc { return next })
	if err := r.Err(); err == nil || !strings.Contains(err.Error(), "frozen") {
		t.Fatalf("expected frozen errors, got %v", err)
	}
	if rec := do(h, "GET", "/b", nil); rec.Code != 404 {
		t.Fatalf("late route must not be served, got %d", rec.Code)
	}
}

func TestRoutesAndURL(t *testing.T) {
	r := NewRouter()
	r.Use(func(next HandlerFunc) HandlerFunc { return next })
	r.Get("/users/{id}", text("")).Name("users.show")
	r.Get("/files/{path...}", text("")).Name("files")
	r.Post("/users", text(""))

	routes := r.Routes()
	if len(routes) != 3 || routes[0].Pattern != "/users/{id}" || routes[0].Name != "users.show" || routes[0].Middleware != 1 {
		t.Fatalf("Routes = %+v", routes)
	}

	if u, err := r.URL("users.show", map[string]string{"id": "a b/c"}); err != nil || u != "/users/a%20b%2Fc" {
		t.Errorf("URL = %q, %v", u, err)
	}
	if u, err := r.URL("files", map[string]string{"path": "docs/read me.md"}); err != nil || u != "/files/docs/read%20me.md" {
		t.Errorf("URL catch-all = %q, %v", u, err)
	}
	if _, err := r.URL("users.show", nil); err == nil {
		t.Error("missing param should error")
	}
	if _, err := r.URL("users.show", map[string]string{"id": "1", "extra": "x"}); err == nil {
		t.Error("unknown param should error")
	}
	if _, err := r.URL("users.show", map[string]string{"id": ""}); err == nil {
		t.Error("empty param should error")
	}
	if _, err := r.URL("nope", nil); err == nil {
		t.Error("unknown route should error")
	}
}

func TestRouteMetadataInContext(t *testing.T) {
	r := NewRouter()
	r.Get("/users/{id}", func(c *Context) error {
		return c.String(200, c.RouteName()+"|"+c.RoutePattern()+"|"+fmt.Sprint(len(c.Params())))
	}).Name("users.show")
	h := mustBuild(t, r)
	if rec := do(h, "GET", "/users/1", nil); rec.Body.String() != "users.show|/users/{id}|1" {
		t.Fatalf("got %q", rec.Body.String())
	}
}

func TestConcurrentServing(t *testing.T) {
	r := NewRouter()
	r.Get("/n/{id}", func(c *Context) error { return c.String(200, c.Param("id")) })
	h := mustBuild(t, r)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprint(i)
			if rec := do(h, "GET", "/n/"+id, nil); rec.Body.String() != id {
				t.Errorf("got %q want %q", rec.Body.String(), id)
			}
		}(i)
	}
	wg.Wait()
}
