# Architecture overview

goPanglima is a **modular monolith**: one repository and one Go module, with
packages that have explicit responsibilities and no import cycles.

## Packages (current)

```text
framework/                 Application, lifecycle, options, providers,
│                          HTTP type aliases and error helpers
├── foundation/            Env (.env loading), Config, Container — no deps
├── logging/               slog logger construction — no deps
└── httpx/                 Router, Context, Response, Error, ErrorHandler, Serve
    └── middleware/        Recover, RequestID, RequestLogger
examples/                  Runnable example applications (compiled and tested in CI)
docs/                      Guides and architecture decision records
```

## Dependency direction

```text
application code ──► framework ──► httpx/middleware ──► httpx ──► logging
                          │                                   (stdlib only)
                          └──────► foundation, logging
```

Rules:

- `foundation` and `logging` import only the standard library.
- `httpx` does not import `framework`; `framework` re-exports `httpx` types as
  aliases (`framework.Context = httpx.Context`) so applications usually import
  only `framework`.
- The framework never imports application code and keeps no global mutable
  state: each `Application` owns its config, container, router, and logger.
- Future packages follow the blueprint's boundaries (for example, `database`
  will not depend on HTTP).

## Request flow

```text
net/http.Server
  └─ httpx handler: canonical-path check → segment tree lookup
       ├─ match       → global → group → route middleware → handler
       ├─ no method   → global middleware → 405 (Allow) / automatic OPTIONS 204
       └─ no path     → global middleware → 404
     handler error   → ErrorHandler (default: safe JSON error body)
```

See [decisions](decisions) for the reasoning behind these choices.
