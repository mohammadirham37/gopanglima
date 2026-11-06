// Package httpx is goPanglima's HTTP layer: router, request context,
// response helpers, HTTP errors, and a graceful server, all built on
// net/http.
//
// Handlers have the signature func(*Context) error. Returned errors are
// rendered by an ErrorHandler; *Error values carry a status code and a
// client-safe message, and any other error becomes a 500 response whose
// details are only logged.
//
//	r := httpx.NewRouter()
//	r.Get("/users/{id}", func(c *httpx.Context) error {
//		return c.JSON(200, httpx.M{"id": c.Param("id")})
//	}).Name("users.show")
//
//	h, err := r.Build(httpx.HandlerOptions{})
//
// Most applications use this package through the framework package, which
// re-exports the common types (framework.Context, framework.Router, ...).
package httpx
