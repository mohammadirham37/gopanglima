# Routing

Routes are registered on `app.Router()` (a `*framework.Router`):

```go
r := app.Router()
r.Get("/", home.Index)
r.Get("/users", users.Index)
r.Post("/users", users.Store)
r.Get("/users/{id}", users.Show).Name("users.show")
r.Put("/users/{id}", users.Update)
r.Patch("/users/{id}", users.Update)
r.Delete("/users/{id}", users.Delete)
r.Options("/users", users.Options)
r.Head("/users", users.Head)
r.Handle("PURGE", "/cache", cache.Purge) // any valid method token
```

## Path syntax

| Pattern | Matches | Notes |
| --- | --- | --- |
| `/users` | `/users` only | Trailing slash is significant: `/users/` is a different route. |
| `/users/{id}` | `/users/42` | Exactly one **non-empty** segment. `c.Param("id")` → `"42"`. |
| `/files/{path...}` | `/files/`, `/files/a/b.txt` | Catch-all; must be the last segment; may be empty. |

- Parameter names are identifiers (`[A-Za-z_][A-Za-z0-9_]*`) and must be unique
  per route. Parameters occupy a whole segment (`/user-{id}` is invalid).
- Parameter values are percent-decoded: `/users/a%20b` → `"a b"`.
- A `{name}` value never contains `/`: a segment with an encoded slash
  (`%2F`) does not match a single-segment parameter. Catch-all values may
  contain `/` but never a `.` or `..` element.
- Write static segments unescaped (`/café`, not `/caf%C3%A9`); patterns
  containing `%` are rejected. `URL` escapes them when generating paths.
- Priority: static segments beat parameters, which beat catch-alls. The router
  backtracks, so `/a/static/y` still matches `/a/{p}/y` when `/a/static/x` exists.
- Parameters never contain `.` or `..` path elements, including encoded forms
  such as `%2E%2E` or `..%2F..%2F`. Still treat parameters as untrusted input:
  validate them before using them in file paths.
- Requests with non-canonical paths (`//`, `/./`, `/../`) are redirected with
  `308 Permanent Redirect` to the cleaned path (global middleware runs for
  the redirect).

## Groups and prefixes

`Prefix` and `Middleware` return router views; `Group` runs a function with
the view, which keeps nested definitions readable:

```go
r.Prefix("/api/v1").Group(func(r *framework.Router) {
	r.Get("/users", users.Index)              // GET /api/v1/users

	r.Middleware(auth).Group(func(r *framework.Router) {
		r.Get("/profile", users.Profile)      // GET /api/v1/profile, with auth
	})
})
```

Prefixes may contain parameters: `r.Prefix("/teams/{team}")`.

## 404, 405, HEAD, OPTIONS

- Unknown paths return **404**.
- Known paths with an unregistered method return **405** with an `Allow` header.
- `HEAD` requests fall back to the `GET` route (body suppressed) unless a `HEAD` route exists.
- `OPTIONS` requests to a known path get an automatic `204` with `Allow`, unless an `OPTIONS` route exists.

Error bodies are produced by the error handler; see [Errors](errors.md).

## Named routes and URL generation

```go
r.Get("/users/{id}", users.Show).Name("users.show")

path, err := r.URL("users.show", map[string]string{"id": "42"}) // "/users/42"
```

Every parameter must be supplied, unknown parameters are rejected, and values
are percent-encoded.

## Registration errors

Mistakes (invalid patterns, nil handlers, duplicate routes, duplicate names,
nil middleware) do not panic. They are collected and returned by
`r.Err()` and when the handler is built (`app.Handler`, `Serve`, `Run`).

After the handler is built the router is **frozen**: the route table is
immutable and safe for concurrent requests. Later registrations are recorded
as errors and have no effect.

## Listing routes

```go
for _, rt := range app.Router().Routes() {
	fmt.Println(rt.Method, rt.Pattern, rt.Name, rt.Middleware)
}
```

Routes are listed in registration order. (A `gopanglima route:list` command is
planned with the CLI.)

Related: [Middleware](middleware.md), [Controllers](controllers.md).
