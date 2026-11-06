package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/mohammadirham37/gopanglima/framework/logging"
)

// DefaultMaxBodyBytes is the request body limit applied by Context.Bind
// when HandlerOptions.MaxBodyBytes is zero (1 MiB).
const DefaultMaxBodyBytes int64 = 1 << 20

// HandlerOptions configures the http.Handler produced by Router.Build.
type HandlerOptions struct {
	// Logger receives framework log records. Nil discards them.
	Logger *slog.Logger
	// Debug exposes internal error details in 5xx responses. Never enable
	// it in production.
	Debug bool
	// MaxBodyBytes limits request bodies read by Context.Bind. Zero means
	// DefaultMaxBodyBytes; negative values are rejected.
	MaxBodyBytes int64
	// ErrorHandler renders handler errors. Nil means DefaultErrorHandler.
	ErrorHandler ErrorHandler
}

type handler struct {
	opts             HandlerOptions
	root             *node
	chains           map[*Route]HandlerFunc
	notFound         HandlerFunc
	methodNotAllowed HandlerFunc
	autoOptions      HandlerFunc
	badPath          HandlerFunc
	cleanRedirect    HandlerFunc
}

// Build validates the route table and returns an immutable http.Handler.
//
// Build returns every registration error recorded so far, plus route
// conflicts and duplicate route names, joined into one error. On success
// the router is frozen. Build may be called more than once (for example
// with different options); each call returns a new handler over the same
// routes.
func (r *Router) Build(opts HandlerOptions) (http.Handler, error) {
	if opts.Logger == nil {
		opts.Logger = logging.Discard()
	}
	if opts.ErrorHandler == nil {
		opts.ErrorHandler = DefaultErrorHandler
	}
	if opts.MaxBodyBytes < 0 {
		return nil, fmt.Errorf("httpx: MaxBodyBytes must not be negative, got %d", opts.MaxBodyBytes)
	}
	if opts.MaxBodyBytes == 0 {
		opts.MaxBodyBytes = DefaultMaxBodyBytes
	}

	r.reg.mu.Lock()
	defer r.reg.mu.Unlock()

	errs := append([]error(nil), r.reg.errs...)
	h := &handler{opts: opts, root: &node{}, chains: make(map[*Route]HandlerFunc, len(r.reg.routes))}
	names := make(map[string]*Route)
	for _, rt := range r.reg.routes {
		if err := h.root.insert(rt); err != nil {
			errs = append(errs, err)
			continue
		}
		if rt.name != "" {
			if prev, ok := names[rt.name]; ok {
				errs = append(errs, fmt.Errorf("duplicate route name %q: %s %s and %s %s",
					rt.name, prev.method, prev.pattern, rt.method, rt.pattern))
				continue
			}
			names[rt.name] = rt
		}
		h.chains[rt] = compose(rt.middlewareChain(), rt.handler)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	global := append([]Middleware(nil), r.reg.root.mw...)
	h.notFound = compose(global, func(*Context) error { return NotFound("") })
	h.methodNotAllowed = compose(global, func(*Context) error { return MethodNotAllowed("") })
	h.autoOptions = compose(global, func(c *Context) error { return c.NoContent() })
	h.badPath = compose(global, func(*Context) error { return BadRequest("Malformed request path") })
	h.cleanRedirect = compose(global, func(c *Context) error {
		return c.RedirectWithStatus(http.StatusPermanentRedirect, c.redirectTo)
	})
	r.reg.frozen = true
	return h, nil
}

// ServeHTTP dispatches the request.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := newContext(w, r, h)
	raw := r.URL.EscapedPath()
	if !strings.HasPrefix(raw, "/") {
		h.run(c, h.notFound)
		return
	}
	if clean := cleanPath(raw); clean != raw {
		// Redirect non-canonical paths ("//", "/./", "/../") like
		// http.ServeMux does. 308 preserves the method and body.
		loc := clean
		if r.URL.RawQuery != "" {
			loc += "?" + r.URL.RawQuery
		}
		c.redirectTo = loc
		h.run(c, h.cleanRedirect)
		return
	}
	segs, ok := splitPath(raw)
	if !ok {
		h.run(c, h.badPath)
		return
	}

	var vals []string
	rt := h.root.lookup(r.Method, segs, &vals)
	if rt == nil && r.Method == http.MethodHead {
		vals = vals[:0]
		rt = h.root.lookup(http.MethodGet, segs, &vals)
	}
	if rt != nil {
		c.route = rt
		if len(vals) > 0 {
			c.params = make([]Param, len(vals))
			for i, v := range vals {
				c.params[i] = Param{Name: rt.paramNames[i], Value: v}
			}
		}
		h.run(c, h.chains[rt])
		return
	}

	methods := make(map[string]bool)
	h.root.allowed(segs, methods)
	if len(methods) == 0 {
		h.run(c, h.notFound)
		return
	}
	c.res.Header().Set("Allow", allowHeader(methods))
	if r.Method == http.MethodOptions {
		h.run(c, h.autoOptions)
		return
	}
	h.run(c, h.methodNotAllowed)
}

func (h *handler) run(c *Context, fn HandlerFunc) {
	if err := fn(c); err != nil {
		h.opts.ErrorHandler(c, err)
	}
}

// cleanPath returns the canonical form of p, preserving a trailing slash.
func cleanPath(p string) string {
	np := path.Clean(p)
	if strings.HasSuffix(p, "/") && np != "/" {
		np += "/"
	}
	return np
}

// splitPath splits an escaped, canonical path into percent-decoded
// segments. "/" yields [""], and a trailing slash yields a final "".
func splitPath(p string) ([]string, bool) {
	segs := strings.Split(p[1:], "/")
	for i, s := range segs {
		if strings.IndexByte(s, '%') < 0 {
			continue
		}
		dec, err := url.PathUnescape(s)
		if err != nil {
			return nil, false
		}
		segs[i] = dec
	}
	return segs, true
}
