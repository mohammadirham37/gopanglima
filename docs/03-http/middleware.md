# Middleware

A middleware wraps a handler:

```go
type Middleware func(next framework.HandlerFunc) framework.HandlerFunc

func Timing(next framework.HandlerFunc) framework.HandlerFunc {
	return func(c *framework.Context) error {
		start := time.Now()
		err := next(c)
		c.Logger().Info("timing", "duration", time.Since(start))
		return err
	}
}
```

A middleware can short-circuit by returning an error (or writing a response)
without calling `next`:

```go
func RequireAdmin(next framework.HandlerFunc) framework.HandlerFunc {
	return func(c *framework.Context) error {
		if c.Header("X-Role") != "admin" {
			return framework.Forbidden("Admins only")
		}
		return next(c)
	}
}
```

## Where middleware applies

```go
r := app.Router()
r.Use(Timing)                                   // global
r.Middleware(RequireAdmin).Group(func(r *framework.Router) {
	r.Get("/admin", admin.Index)                // group
})
r.Get("/report", reports.Show).Middleware(Audit) // route
```

## Execution order (tested guarantee)

1. Global middleware (`Use` on the root router), in the order added.
2. Group middleware, outermost group first.
3. Route middleware, in the order added.
4. The handler.

Code after `next(c)` runs in reverse order. Middleware added to a group with
`Use` applies to all routes in the group, including routes registered before
it. Global middleware also runs for 404, 405, automatic OPTIONS, and
canonical-path redirect responses; group and route middleware do not.

## Built-in middleware

`framework.New` installs these globally, in this order (disable with
`framework.WithoutDefaultMiddleware()`):

| Middleware | Behavior |
| --- | --- |
| `middleware.RequestID()` | Reuses a safe client `X-Request-ID` (1–128 chars of `[A-Za-z0-9-_.:]`) or generates a random 128-bit ID; sets the response header; adds `request_id` to the request logger; `middleware.RequestIDFrom(ctx)` reads it. |
| `middleware.RequestLogger()` | One log record per request: method, path, route pattern, status, bytes, duration. Never logs the query string. |
| `middleware.Recover()` | Turns panics into 500 errors; logs the panic and stack; re-panics `http.ErrAbortHandler`. |

Note: `RequestLogger` renders handler errors itself (via `c.HandleError`) so
it can log the final status, and then returns `nil`. Middleware *before* it
therefore does not see handler errors. Use the same pattern in your own
middleware when you need the final status.

Related: [Routing](routing.md), [Errors](errors.md).
