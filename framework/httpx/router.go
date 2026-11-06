package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// HandlerFunc is the goPanglima handler contract. Returning a non-nil error
// delegates response rendering to the configured ErrorHandler.
type HandlerFunc func(c *Context) error

// Middleware wraps a handler. Code before calling next runs on the way in;
// code after it runs on the way out.
//
//	func Timing(next httpx.HandlerFunc) httpx.HandlerFunc {
//		return func(c *httpx.Context) error {
//			start := time.Now()
//			err := next(c)
//			c.Logger().Info("took", "d", time.Since(start))
//			return err
//		}
//	}
type Middleware func(next HandlerFunc) HandlerFunc

// Router collects route definitions and builds an http.Handler from them.
//
// A Router value returned by Prefix or Middleware is a view onto the same
// route table with an additional path prefix and/or middleware; routes
// registered through it are stored in the shared table.
//
// Registration is safe for concurrent use but is intended to happen during
// application startup. Registration problems (invalid patterns, duplicate
// routes, duplicate names, nil handlers) are collected and reported by Err
// and Build. After a successful Build the router is frozen: further
// registration attempts are recorded as errors and have no effect on the
// built handler, which is immutable and safe for concurrent serving.
//
// Middleware order: middleware added with Use on the root router runs
// first (in the order added), then group middleware from the outermost
// group to the innermost, then route middleware, then the handler. Root
// middleware also runs for 404, 405, and automatic OPTIONS responses.
// Group middleware applies to every route of the group, including routes
// registered before the middleware was added.
type Router struct {
	reg *registry
	grp *group
}

type registry struct {
	mu     sync.Mutex
	root   *group
	routes []*Route
	errs   []error
	frozen bool
}

type group struct {
	parent *group
	prefix string // full, already-joined prefix
	mw     []Middleware
}

// NewRouter returns an empty router.
func NewRouter() *Router {
	root := &group{}
	return &Router{reg: &registry{root: root}, grp: root}
}

// Route is a registered route. Its methods configure the route and return
// it for chaining:
//
//	r.Get("/users/{id}", users.Show).Name("users.show")
type Route struct {
	reg        *registry
	grp        *group
	method     string
	pattern    string
	segments   []segment
	paramNames []string
	handler    HandlerFunc
	mw         []Middleware
	name       string
}

// RouteInfo describes a registered route for introspection.
type RouteInfo struct {
	Method     string
	Pattern    string
	Name       string
	Middleware int // number of middleware wrapping the handler, including global middleware
}

// Get registers a GET route.
func (r *Router) Get(path string, h HandlerFunc) *Route { return r.Handle(http.MethodGet, path, h) }

// Post registers a POST route.
func (r *Router) Post(path string, h HandlerFunc) *Route { return r.Handle(http.MethodPost, path, h) }

// Put registers a PUT route.
func (r *Router) Put(path string, h HandlerFunc) *Route { return r.Handle(http.MethodPut, path, h) }

// Patch registers a PATCH route.
func (r *Router) Patch(path string, h HandlerFunc) *Route {
	return r.Handle(http.MethodPatch, path, h)
}

// Delete registers a DELETE route.
func (r *Router) Delete(path string, h HandlerFunc) *Route {
	return r.Handle(http.MethodDelete, path, h)
}

// Options registers an OPTIONS route. Without one, OPTIONS requests to a
// known path receive an automatic 204 response with an Allow header.
func (r *Router) Options(path string, h HandlerFunc) *Route {
	return r.Handle(http.MethodOptions, path, h)
}

// Head registers a HEAD route. Without one, HEAD requests are served by the
// matching GET route with the body suppressed.
func (r *Router) Head(path string, h HandlerFunc) *Route { return r.Handle(http.MethodHead, path, h) }

// Handle registers a route for an arbitrary method. Method names are
// case-sensitive and must be valid HTTP tokens; standard methods are
// upper-case.
//
// Path syntax:
//   - Paths start with "/". A trailing slash is significant: "/users" and
//     "/users/" are different routes.
//   - "{name}" matches exactly one non-empty path segment.
//   - "{name...}" matches the rest of the path (possibly empty) and must be
//     the final segment.
//   - Parameters must occupy a whole segment; "user-{id}" is invalid.
//   - Static segments take priority over parameters, which take priority
//     over catch-all parameters.
func (r *Router) Handle(method, path string, h HandlerFunc) *Route {
	full, joinErr := joinPath(r.grp.prefix, path)
	rt := &Route{reg: r.reg, grp: r.grp, method: method, pattern: full, handler: h}

	r.reg.mu.Lock()
	defer r.reg.mu.Unlock()
	fail := func(err error) *Route {
		r.reg.errs = append(r.reg.errs, fmt.Errorf("route %s %s: %w", method, full, err))
		return rt
	}
	switch {
	case r.reg.frozen:
		return fail(errFrozen)
	case !validMethod(method):
		return fail(fmt.Errorf("invalid method %q", method))
	case h == nil:
		return fail(errors.New("nil handler"))
	case joinErr != nil:
		return fail(joinErr)
	}
	segs, names, err := parsePattern(full)
	if err != nil {
		return fail(err)
	}
	rt.segments, rt.paramNames = segs, names
	r.reg.routes = append(r.reg.routes, rt)
	return rt
}

// Prefix returns a router view whose routes are registered under prefix.
//
//	r.Prefix("/api/v1").Group(func(r *httpx.Router) {
//		r.Get("/users", users.Index) // GET /api/v1/users
//	})
func (r *Router) Prefix(prefix string) *Router {
	p := strings.TrimSuffix(prefix, "/")
	if !strings.HasPrefix(prefix, "/") {
		r.recordErr(fmt.Errorf("prefix %q must start with \"/\"", prefix))
	}
	return &Router{reg: r.reg, grp: &group{parent: r.grp, prefix: r.grp.prefix + p}}
}

// Middleware returns a router view whose routes are wrapped by mw, in
// addition to any middleware inherited from r.
func (r *Router) Middleware(mw ...Middleware) *Router {
	r.checkMiddleware(mw)
	return &Router{reg: r.reg, grp: &group{parent: r.grp, prefix: r.grp.prefix, mw: append([]Middleware(nil), mw...)}}
}

// Use appends middleware to this router's group. On the root router this
// registers global middleware.
func (r *Router) Use(mw ...Middleware) {
	if !r.checkMiddleware(mw) {
		return
	}
	r.reg.mu.Lock()
	defer r.reg.mu.Unlock()
	if r.reg.frozen {
		r.reg.errs = append(r.reg.errs, fmt.Errorf("use middleware: %w", errFrozen))
		return
	}
	r.grp.mw = append(r.grp.mw, mw...)
}

// Group calls fn with r. It exists to make nested route definitions read
// naturally:
//
//	r.Prefix("/admin").Middleware(auth).Group(func(r *httpx.Router) { ... })
func (r *Router) Group(fn func(r *Router)) {
	fn(r)
}

// Name sets the route name used for URL generation. Names must be unique
// across the router; duplicates are reported by Build.
func (rt *Route) Name(name string) *Route {
	rt.reg.mu.Lock()
	defer rt.reg.mu.Unlock()
	switch {
	case rt.reg.frozen:
		rt.reg.errs = append(rt.reg.errs, fmt.Errorf("route %s %s: name %q: %w", rt.method, rt.pattern, name, errFrozen))
	case name == "":
		rt.reg.errs = append(rt.reg.errs, fmt.Errorf("route %s %s: empty route name", rt.method, rt.pattern))
	default:
		rt.name = name
	}
	return rt
}

// Middleware appends route-specific middleware. Route middleware runs after
// global and group middleware.
func (rt *Route) Middleware(mw ...Middleware) *Route {
	for _, m := range mw {
		if m == nil {
			rt.reg.mu.Lock()
			rt.reg.errs = append(rt.reg.errs, fmt.Errorf("route %s %s: nil middleware", rt.method, rt.pattern))
			rt.reg.mu.Unlock()
			return rt
		}
	}
	rt.reg.mu.Lock()
	defer rt.reg.mu.Unlock()
	if rt.reg.frozen {
		rt.reg.errs = append(rt.reg.errs, fmt.Errorf("route %s %s: middleware: %w", rt.method, rt.pattern, errFrozen))
		return rt
	}
	rt.mw = append(rt.mw, mw...)
	return rt
}

// Err returns every registration error recorded so far, joined, or nil.
func (r *Router) Err() error {
	r.reg.mu.Lock()
	defer r.reg.mu.Unlock()
	return errors.Join(r.reg.errs...)
}

// Routes returns the successfully registered routes in registration order.
func (r *Router) Routes() []RouteInfo {
	r.reg.mu.Lock()
	defer r.reg.mu.Unlock()
	out := make([]RouteInfo, 0, len(r.reg.routes))
	for _, rt := range r.reg.routes {
		out = append(out, RouteInfo{
			Method:     rt.method,
			Pattern:    rt.pattern,
			Name:       rt.name,
			Middleware: len(rt.middlewareChain()),
		})
	}
	return out
}

// URL builds the path for the named route. Every route parameter must be
// supplied in params, and params must not contain unknown keys. Values are
// percent-encoded; catch-all values keep their "/" separators.
//
//	path, err := r.URL("users.show", map[string]string{"id": "42"}) // "/users/42"
func (r *Router) URL(name string, params map[string]string) (string, error) {
	r.reg.mu.Lock()
	var rt *Route
	for _, candidate := range r.reg.routes {
		if candidate.name == name {
			rt = candidate
			break
		}
	}
	r.reg.mu.Unlock()
	if rt == nil {
		return "", fmt.Errorf("httpx: no route named %q", name)
	}

	used := 0
	var b strings.Builder
	for _, s := range rt.segments {
		b.WriteByte('/')
		switch s.kind {
		case segStatic:
			b.WriteString(url.PathEscape(s.value))
		case segParam, segCatchAll:
			v, ok := params[s.value]
			if !ok {
				return "", fmt.Errorf("httpx: route %q: missing parameter %q", name, s.value)
			}
			used++
			if s.kind == segParam {
				if v == "" {
					return "", fmt.Errorf("httpx: route %q: parameter %q must not be empty", name, s.value)
				}
				b.WriteString(url.PathEscape(v))
				continue
			}
			parts := strings.Split(v, "/")
			for i, p := range parts {
				parts[i] = url.PathEscape(p)
			}
			b.WriteString(strings.Join(parts, "/"))
		}
	}
	if used != len(params) {
		return "", fmt.Errorf("httpx: route %q: unknown parameters supplied", name)
	}
	return b.String(), nil
}

// middlewareChain returns global, group (outer to inner), and route
// middleware in execution order. The caller must hold reg.mu.
func (rt *Route) middlewareChain() []Middleware {
	return append(groupChain(rt.grp), rt.mw...)
}

func groupChain(g *group) []Middleware {
	var groups []*group
	for ; g != nil; g = g.parent {
		groups = append(groups, g)
	}
	var out []Middleware
	for i := len(groups) - 1; i >= 0; i-- {
		out = append(out, groups[i].mw...)
	}
	return out
}

func compose(mw []Middleware, h HandlerFunc) HandlerFunc {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

var errFrozen = errors.New("router is frozen after Build")

func (r *Router) recordErr(err error) {
	r.reg.mu.Lock()
	r.reg.errs = append(r.reg.errs, err)
	r.reg.mu.Unlock()
}

func (r *Router) checkMiddleware(mw []Middleware) bool {
	for _, m := range mw {
		if m == nil {
			r.recordErr(errors.New("nil middleware"))
			return false
		}
	}
	return true
}

func joinPath(prefix, path string) (string, error) {
	if !strings.HasPrefix(path, "/") {
		return prefix + "/" + path, fmt.Errorf("path %q must start with \"/\"", path)
	}
	if prefix != "" && path == "/" {
		return prefix, nil
	}
	return prefix + path, nil
}

func validMethod(m string) bool {
	if m == "" {
		return false
	}
	for i := 0; i < len(m); i++ {
		c := m[i]
		if c <= ' ' || c >= 0x7f || strings.IndexByte("()<>@,;:\\\"/[]?={}", c) >= 0 {
			return false
		}
	}
	return true
}
