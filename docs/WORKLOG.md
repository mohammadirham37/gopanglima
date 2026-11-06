# Work log

## 2026-11-06 — Milestones 0, 1, 2

**Objective.** Bootstrap a public-ready repository (M0), implement the
application foundation and lifecycle (M1), and a real HTTP layer with routing
and middleware (M2), as approved by the owner.

**Acceptance criteria and status**

| Criterion | Status |
| --- | --- |
| Project builds; CI workflow runs on PRs (Go 1.23 + stable) | ✅ |
| No secrets/generated clutter tracked; README states status honestly | ✅ |
| License decision recorded (MIT, Mohammad Irham) | ✅ ADR 0002 |
| Minimal application initializes; config loadable and tested | ✅ |
| Provider order deterministic; init errors returned | ✅ tests |
| GET/POST/PUT/PATCH/DELETE, params, groups work | ✅ tests |
| Middleware order tested | ✅ `TestMiddlewareOrder` |
| Errors map to safe responses | ✅ `TestErrorHandlerHidesInternalErrors` |
| Graceful shutdown | ✅ `TestServeGracefulShutdown`, manual SIGINT run |
| Example HTTP application runs | ✅ `examples/hello-world`, `examples/rest-api` (+ tests) |

**Verification.** `gofmt -l .` (clean), `go vet ./...`, `go test -race ./...`
on Go 1.27.3, and `go test ./...` on Go 1.23.12.

**Known limitations.** No CLI, database, validation, sessions, auth, or views
yet. `Bind` supports JSON only. No CORS or rate-limit middleware yet.

**Next step.** Milestone 3: database configuration, connection manager on
`database/sql`, SQL dialect interface, query builder with parameter binding,
transactions, PostgreSQL integration tests.

**Independent review.** A code review before the first commit found and we
fixed: path traversal through encoded `%2F` in route parameters; a panic on
`*Error` values with invalid status codes; races in concurrent
`Bootstrap`/`Shutdown`; `Shutdown` not stopping a running `Serve`; a second
Ctrl+C not force-quitting; canonical-path redirects bypassing global
middleware; 1xx statuses and `Flush` handling; nil interface bindings in the
container; percent signs in static patterns; negative server timeouts. Each
fix has a regression test.
