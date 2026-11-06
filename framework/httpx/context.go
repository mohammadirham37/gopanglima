package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// M is a shorthand for map[string]any, convenient for JSON payloads and
// template data.
type M = map[string]any

// Param is a single matched route parameter.
type Param struct {
	Name  string
	Value string
}

// Context carries the request, the response, matched route parameters, and
// request-scoped values through middleware and handlers.
//
// A Context is created for a single request and must not be retained or
// used after the handler returns. It is not safe for concurrent use by
// multiple goroutines.
type Context struct {
	req    *http.Request
	res    *Response
	route  *Route
	params []Param
	query  url.Values
	values map[string]any
	logger *slog.Logger
	h      *handler

	redirectTo string // set for canonical-path redirects
}

func newContext(w http.ResponseWriter, r *http.Request, h *handler) *Context {
	return &Context{req: r, res: newResponse(w), logger: h.opts.Logger, h: h}
}

// Request returns the underlying *http.Request.
func (c *Context) Request() *http.Request { return c.req }

// SetRequest replaces the request, typically with r.WithContext(...) in a
// middleware that adds request-scoped context values.
func (c *Context) SetRequest(r *http.Request) {
	c.req = r
	c.query = nil
}

// Context returns the request's context.Context. It is canceled when the
// client disconnects or the server shuts down; pass it to every blocking
// operation performed on behalf of the request.
func (c *Context) Context() context.Context { return c.req.Context() }

// Response returns the response writer wrapper.
func (c *Context) Response() *Response { return c.res }

// Logger returns the request logger. Middleware may enrich it with
// SetLogger, for example to add a request ID.
func (c *Context) Logger() *slog.Logger { return c.logger }

// SetLogger replaces the request logger.
func (c *Context) SetLogger(l *slog.Logger) {
	if l != nil {
		c.logger = l
	}
}

// Debug reports whether the handler runs in debug mode. Debug mode may
// expose internal error details and must not be enabled in production.
func (c *Context) Debug() bool { return c.h.opts.Debug }

// Param returns the value of the named route parameter, or "" if the route
// has no such parameter. Values are percent-decoded.
func (c *Context) Param(name string) string {
	for _, p := range c.params {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}

// Params returns all matched route parameters in path order. The returned
// slice must not be modified.
func (c *Context) Params() []Param { return c.params }

// RouteName returns the name of the matched route, or "" when the route is
// unnamed or no route matched.
func (c *Context) RouteName() string {
	if c.route == nil {
		return ""
	}
	return c.route.name
}

// RoutePattern returns the pattern of the matched route (for example
// "/users/{id}"), or "" when no route matched.
func (c *Context) RoutePattern() string {
	if c.route == nil {
		return ""
	}
	return c.route.pattern
}

// Query returns the first value of the named query-string parameter, or "".
func (c *Context) Query(name string) string {
	return c.queryValues().Get(name)
}

// QueryDefault returns the named query-string parameter, or def when it is
// absent or empty.
func (c *Context) QueryDefault(name, def string) string {
	if v := c.Query(name); v != "" {
		return v
	}
	return def
}

// QueryValues returns all parsed query-string values. Malformed pairs are
// skipped, matching url.ParseQuery's lenient behavior.
func (c *Context) QueryValues() url.Values { return c.queryValues() }

func (c *Context) queryValues() url.Values {
	if c.query == nil {
		c.query = c.req.URL.Query()
	}
	return c.query
}

// Header returns the first value of the named request header.
func (c *Context) Header(name string) string { return c.req.Header.Get(name) }

// SetHeader sets a response header. It has no effect once the response has
// been committed.
func (c *Context) SetHeader(name, value string) { c.res.Header().Set(name, value) }

// Set stores a request-scoped value, for example the authenticated user.
func (c *Context) Set(key string, value any) {
	if c.values == nil {
		c.values = make(map[string]any)
	}
	c.values[key] = value
}

// Get returns a request-scoped value stored with Set.
func (c *Context) Get(key string) (any, bool) {
	v, ok := c.values[key]
	return v, ok
}

// Bind decodes a JSON request body into dst, which must be a non-nil
// pointer.
//
// Bind enforces the configured maximum body size, requires a JSON content
// type (application/json or any application/*+json type), and requires the
// body to contain exactly one JSON value. Unknown fields are ignored.
//
// All failures are returned as *Error values with client-safe messages:
// 400 for malformed or empty bodies, 413 when the body is too large, and
// 415 for unsupported content types. Bind only decodes; validation is a
// separate step.
func (c *Context) Bind(dst any) error {
	if dst == nil {
		return errors.New("httpx: Bind destination must be a non-nil pointer")
	}
	ct := c.req.Header.Get("Content-Type")
	if ct == "" {
		return UnsupportedMediaType("Content-Type must be application/json")
	}
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return UnsupportedMediaType("Malformed Content-Type header").WithCause(err)
	}
	if mt != "application/json" && !(strings.HasPrefix(mt, "application/") && strings.HasSuffix(mt, "+json")) {
		return UnsupportedMediaType("Content-Type must be application/json")
	}
	if c.req.Body == nil || c.req.Body == http.NoBody {
		return BadRequest("Request body must not be empty")
	}

	body := http.MaxBytesReader(c.res.w, c.req.Body, c.h.opts.MaxBodyBytes)
	dec := json.NewDecoder(body)
	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return decodeError(err)
		}
		return BadRequest("Request body must contain a single JSON value").WithCause(err)
	}
	return nil
}

func decodeError(err error) error {
	var (
		syntaxErr    *json.SyntaxError
		typeErr      *json.UnmarshalTypeError
		maxErr       *http.MaxBytesError
		invalidUnmar *json.InvalidUnmarshalError
	)
	switch {
	case errors.As(err, &maxErr):
		return RequestEntityTooLarge(fmt.Sprintf("Request body must not exceed %d bytes", maxErr.Limit)).WithCause(err)
	case errors.As(err, &invalidUnmar):
		// Programming error: the destination is not a pointer.
		return fmt.Errorf("httpx: Bind: %w", err)
	case errors.As(err, &syntaxErr):
		return BadRequest(fmt.Sprintf("Malformed JSON at position %d", syntaxErr.Offset)).WithCause(err)
	case errors.Is(err, io.ErrUnexpectedEOF):
		return BadRequest("Malformed JSON: unexpected end of input").WithCause(err)
	case errors.As(err, &typeErr):
		if typeErr.Field != "" {
			return BadRequest(fmt.Sprintf("Invalid value for field %q", typeErr.Field)).WithCause(err)
		}
		return BadRequest("Invalid JSON value type").WithCause(err)
	case errors.Is(err, io.EOF):
		return BadRequest("Request body must not be empty").WithCause(err)
	default:
		return BadRequest("Invalid request body").WithCause(err)
	}
}

// JSON encodes v and writes it with the given status code and the
// "application/json; charset=utf-8" content type.
//
// Encoding happens before anything is written, so an encoding failure
// returns an error and leaves the response untouched for the error handler.
func (c *Context) JSON(code int, v any) error {
	if c.res.committed {
		return ErrResponseCommitted
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return fmt.Errorf("httpx: encode JSON response: %w", err)
	}
	return c.Blob(code, "application/json; charset=utf-8", buf.Bytes())
}

// String writes s as "text/plain; charset=utf-8".
func (c *Context) String(code int, s string) error {
	return c.Blob(code, "text/plain; charset=utf-8", []byte(s))
}

// HTML writes html as "text/html; charset=utf-8".
//
// The string is written verbatim and is NOT escaped. Only pass trusted
// markup; render untrusted data through html/template instead.
func (c *Context) HTML(code int, html string) error {
	return c.Blob(code, "text/html; charset=utf-8", []byte(html))
}

// Blob writes data with the given status code and content type.
func (c *Context) Blob(code int, contentType string, data []byte) error {
	if c.res.committed {
		return ErrResponseCommitted
	}
	if err := checkStatus(code); err != nil {
		return err
	}
	h := c.res.Header()
	h.Set("Content-Type", contentType)
	h.Set("X-Content-Type-Options", "nosniff")
	c.res.WriteHeader(code)
	if c.req.Method == http.MethodHead {
		return nil
	}
	_, err := c.res.Write(data)
	return err
}

// checkStatus rejects status codes that are not final responses
// (200-599 plus 3-digit codes up to 999 accepted by net/http).
func checkStatus(code int) error {
	if code < 200 || code > 999 {
		return fmt.Errorf("httpx: invalid response status %d", code)
	}
	return nil
}

// NoContent writes a 204 No Content response.
func (c *Context) NoContent() error {
	return c.Status(http.StatusNoContent)
}

// Status writes the status code with an empty body.
func (c *Context) Status(code int) error {
	if c.res.committed {
		return ErrResponseCommitted
	}
	if err := checkStatus(code); err != nil {
		return err
	}
	c.res.WriteHeader(code)
	return nil
}

// Redirect sends a 302 Found redirect to url.
//
// Never pass an unvalidated, user-supplied URL: doing so creates an open
// redirect vulnerability.
func (c *Context) Redirect(url string) error {
	return c.RedirectWithStatus(http.StatusFound, url)
}

// RedirectWithStatus sends a redirect with an explicit 3xx status code.
func (c *Context) RedirectWithStatus(code int, url string) error {
	if code < 300 || code > 308 {
		return fmt.Errorf("httpx: invalid redirect status %d", code)
	}
	if c.res.committed {
		return ErrResponseCommitted
	}
	http.Redirect(c.res, c.req, url, code)
	return nil
}

// HandleError passes err to the configured error handler immediately.
//
// Middleware that needs to observe the final response status, such as an
// access logger, can call HandleError and then return nil. A nil err is a
// no-op.
func (c *Context) HandleError(err error) {
	if err == nil {
		return
	}
	c.h.opts.ErrorHandler(c, err)
}
