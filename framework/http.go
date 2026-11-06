package framework

import "github.com/mohammadirham37/gopanglima/framework/httpx"

// Aliases for the HTTP types applications use most, so that typical
// application code only needs to import the framework package.
type (
	// Context is the per-request context passed to handlers.
	Context = httpx.Context
	// HandlerFunc is the handler contract: func(*Context) error.
	HandlerFunc = httpx.HandlerFunc
	// Middleware wraps a HandlerFunc.
	Middleware = httpx.Middleware
	// Router registers routes, groups, and middleware.
	Router = httpx.Router
	// Route is a registered route.
	Route = httpx.Route
	// M is a shorthand for map[string]any.
	M = httpx.M
)
