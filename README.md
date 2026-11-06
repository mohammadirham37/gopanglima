# goPanglima

**Laravel Developer Experience. Native Go Foundation.**

goPanglima is a Laravel-inspired web framework built in Go.
It aims to combine expressive application development with Go's
type safety, explicitness, and standard-library foundation.

> **Project status: experimental (pre-alpha).** APIs may change before v1.0.
> goPanglima is inspired by Laravel's developer experience. It is not a port
> of Laravel and is not API-compatible with it.

## Design principles

- **Laravel-like DX** — expressive routing, controllers, middleware, service providers.
- **Native Go foundation** — built on `net/http`, `log/slog`, and the standard library. Zero third-party dependencies today.
- **Explicitness** — handlers return errors; initialization errors are reported, never swallowed; no reflection-based autowiring.
- **Security by default** — request body limits, server timeouts, safe error responses, no internal error leakage in production.
- **Incremental maturity** — only implemented features are documented as available.

## Requirements

- Go 1.23 or newer

## Installation

```bash
go get github.com/mohammadirham37/gopanglima@latest
```

## Quickstart

```go
package main

import (
	"log"
	"net/http"

	"github.com/mohammadirham37/gopanglima/framework"
)

func main() {
	app := framework.New(framework.WithEnvFile(".env"))

	app.Router().Get("/", func(c *framework.Context) error {
		return c.String(http.StatusOK, "Hello from goPanglima!")
	})

	app.Router().Prefix("/api/v1").Group(func(r *framework.Router) {
		r.Get("/users/{id}", func(c *framework.Context) error {
			if c.Param("id") != "1" {
				return framework.NotFound("User not found")
			}
			return c.JSON(http.StatusOK, framework.M{"id": 1, "name": "Ana"})
		}).Name("users.show")
	})

	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
```

`Run` serves until SIGINT/SIGTERM and then shuts down gracefully. More complete
programs live in [`examples/`](examples):

- [`examples/hello-world`](examples/hello-world) — minimal application.
- [`examples/rest-api`](examples/rest-api) — controllers, a service bound in the
  container, route groups, prefixes, named routes, and token middleware, with
  `httptest` feature tests.

## Feature status

| Area | Status |
| --- | --- |
| Application lifecycle, service providers, shutdown hooks | ✅ Available (experimental) |
| Configuration repository and `.env` loading | ✅ Available (experimental) |
| Type-keyed service container | ✅ Available (experimental) |
| Structured logging (`log/slog`) | ✅ Available (experimental) |
| Router: methods, params, catch-all, groups, prefixes, named routes, URL generation | ✅ Available (experimental) |
| Middleware (global/group/route), Recover, RequestID, RequestLogger | ✅ Available (experimental) |
| Context: JSON binding with size limits, JSON/text/HTML/redirect responses | ✅ Available (experimental) |
| Centralized HTTP error handling, 404/405, graceful shutdown | ✅ Available (experimental) |
| Database, query builder, ORM, migrations | 🗺️ Planned (Milestones 3–4) |
| `gopanglima` CLI and project generator | 🗺️ Planned (Milestone 5) |
| Validation, sessions, authentication, CSRF | 🗺️ Planned (Milestone 6) |
| Views, filesystem, mail, cache | 🗺️ Planned (Milestone 7) |
| Events, queue, scheduler, relationships | 🗺️ Planned (Milestone 8) |

Planned items do not exist yet. See [GOPANGLIMA_BLUEPRINT.md](GOPANGLIMA_BLUEPRINT.md)
for the full roadmap.

## Documentation

- Getting started: [installation](docs/01-getting-started/installation.md), [first app](docs/01-getting-started/first-app.md)
- Foundation: [lifecycle](docs/02-foundation/lifecycle.md), [configuration & environment](docs/02-foundation/configuration.md), [service providers & container](docs/02-foundation/service-providers.md)
- HTTP: [routing](docs/03-http/routing.md), [middleware](docs/03-http/middleware.md), [controllers](docs/03-http/controllers.md), [requests & responses](docs/03-http/requests-responses.md), [errors](docs/03-http/errors.md)
- Architecture: [overview](docs/architecture/overview.md), [decisions (ADRs)](docs/architecture/decisions)

API reference: `go doc github.com/mohammadirham37/gopanglima/framework` or pkg.go.dev.

## Development

```bash
make check   # gofmt check, go vet, go test -race
make bench   # router/middleware/JSON benchmarks (no performance claims are made)
```

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) and the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Security

Please report vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## License

goPanglima is released under the [MIT License](LICENSE).
Copyright (c) 2026 Mohammad Irham.
