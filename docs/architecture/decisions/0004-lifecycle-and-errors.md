# ADR 0004: Application lifecycle, canonical bootstrap, deferred init errors

- Status: Accepted
- Date: 2026-11-06

## Context

The blueprint's desired API is `app := framework.New()` (no error) followed by
`app.Run(":8080")`, while also requiring that initialization errors are
returned and never hidden.

## Decision

- `framework.New(opts ...Option) *Application` never fails. Errors found while
  applying options (malformed `.env`, invalid `APP_DEBUG`/`LOG_LEVEL`/
  `LOG_FORMAT`, provider `Register` failures) are recorded and returned by
  the first `Bootstrap`, `Handler`, `Serve`, or `Run` call.
- Providers: `Register(*Application) error` runs immediately on registration;
  optional `Boot(ctx, *Application) error` runs during `Bootstrap` in
  registration order. No reflection-based injection.
- The container is type-keyed (`foundation.Bind[T]`, `foundation.Make[T]`).
- Shutdown hooks run in reverse registration order, all of them, with errors
  joined; they also run if bootstrap fails during `Serve`/`Run`.
- Canonical bootstrap pattern: one function returning
  `(*framework.Application, error)` that builds the app, binds services, and
  registers routes; `main` and tests both call it (see `examples/rest-api`).
- Configuration defaults are fail-safe: `APP_ENV` defaults to `production`,
  `APP_DEBUG` to `false`.

## Consequences

Callers who only use `New` and never `Bootstrap`/`Handler`/`Serve`/`Run`
will not observe init errors; every documented path does.
