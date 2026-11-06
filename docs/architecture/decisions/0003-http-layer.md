# ADR 0003: HTTP layer — own router on net/http, package name `httpx`

- Status: Accepted
- Date: 2026-11-06

## Context

The blueprint asks for a router abstraction on `net/http` supporting methods,
parameters, groups, global/group/route middleware, named routes, route
listing, 404/405, and visible registration errors, and asks us to evaluate
`http.ServeMux` (Go 1.22+ patterns).

`http.ServeMux` supports method and wildcard patterns, but:

- it **panics** on conflicting or invalid registrations, while the blueprint
  requires errors to be returned;
- it has no route names, URL generation, route listing, or middleware model;
- its 404/405 responses bypass our error handler and middleware.

Third-party routers would add a dependency and leak their types unless
wrapped.

## Decision

- Implement a small segment-tree router in `framework/httpx` on top of
  `net/http` types. No third-party dependency.
- Name the package `httpx` to avoid shadowing `net/http` in user code.
- Keep router, context, response, and errors in **one package**: they are
  mutually dependent (handlers take `*Context`, the context references the
  matched route), and splitting them into the blueprint's suggested
  `router/`, `context/`, `request/`, `response/` sub-packages would create
  cycles or force awkward interfaces. Built-in middleware lives in
  `httpx/middleware`.
- Handler contract: `func(*Context) error`. Middleware: `func(HandlerFunc) HandlerFunc`.
- Registration errors are collected and returned by `Router.Err` / `Build`;
  the router is frozen after a successful build so serving is lock-free.
- Errors returned by handlers go to a configurable `ErrorHandler`; the
  default never leaks internal errors outside debug mode.
- Trailing slashes are significant; non-canonical paths are redirected (308).

## Consequences

We own routing correctness and must keep it well-tested (priority,
backtracking, encoding, 405 computation). Benchmarks exist in
`framework/httpx/bench_test.go`; no performance claims are made.
