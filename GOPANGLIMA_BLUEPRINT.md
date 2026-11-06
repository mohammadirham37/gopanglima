# goPanglima --- Project Blueprint & AI Agent Execution Specification

> **Tagline:** Laravel Developer Experience. Native Go Foundation.\
> **Project name:** goPanglima\
> **Repository target:** `gopanglima`\
> **Project type:** Open-source Go web framework\
> **Document purpose:** Master specification for an AI coding agent to
> design, implement, test, document, and prepare goPanglima for a public
> Git repository.

------------------------------------------------------------------------

## 0. Instructions for the AI coding agent

You are the principal engineer responsible for building **goPanglima**,
a Laravel-inspired web framework written in Go.

Treat this document as the project's source of truth. Do not attempt to
implement every feature in one pass. Work incrementally, keep the
repository compiling, write tests with each feature, and update
documentation as the public API evolves.

### 0.1 Non-negotiable working rules

1.  **Inspect before changing.** If a repository already exists, inspect
    its files, Git status, Go version, module path, tests, and existing
    conventions before editing.
2.  **Do not overwrite user work.** Never delete, reset, force-push, or
    replace existing files or Git history without explicit
    authorization.
3.  **Build in vertical slices.** Each milestone must produce a
    runnable, tested result.
4.  **Keep `go test ./...` green.** Run relevant tests after each
    meaningful change and the full suite before completing a milestone.
5.  **No fake implementations.** Do not expose public methods that
    silently do nothing, return fabricated success, or contain TODO-only
    behavior. If a feature is not implemented, keep it out of the public
    API or return a clear unsupported error.
6.  **No unnecessary dependencies.** Prefer the Go standard library. Add
    a third-party dependency only when it materially reduces complexity
    or risk.
7.  **Use idiomatic Go internally.** Laravel-like ergonomics are a
    design goal, not permission to imitate PHP language features that do
    not fit Go.
8.  **Preserve explicit errors.** Avoid panic-driven control flow and
    hidden error swallowing.
9.  **Respect context cancellation.** Network, database, queue, and
    other blocking operations must accept and honor `context.Context`.
10. **Document public APIs.** Exported types, functions, methods, and
    packages need meaningful GoDoc comments.
11. **Keep security in scope from the start.** Use parameterized SQL,
    safe HTML escaping, secure password hashing, CSRF protection where
    applicable, and conservative defaults.
12. **Do not claim performance superiority without benchmarks.** Add
    reproducible benchmarks before making performance claims.
13. **Do not claim Laravel compatibility.** goPanglima is
    Laravel-inspired, not a PHP Laravel port and not API-compatible with
    Laravel.
14. **Do not publish secrets.** Never commit `.env`, credentials,
    tokens, private keys, database dumps, or personal data.
15. **Ask before irreversible public actions.** Preparing a GitHub-ready
    repository is allowed. Creating a remote repository, publishing a
    release, pushing commits, or changing repository visibility requires
    explicit user authorization and appropriate credentials.

### 0.2 Required agent workflow

For every milestone:

1.  Read this specification and inspect the current repository.
2.  State the milestone objective and acceptance criteria in the work
    log.
3.  Propose a concise implementation plan.
4.  Implement the smallest coherent slice.
5.  Add unit tests and integration tests where appropriate.
6.  Run `gofmt`, `go vet ./...`, and `go test ./...`.
7.  Run race tests when relevant: `go test -race ./...`.
8.  Update README, docs, examples, and changelog if public behavior
    changed.
9.  Review `git diff` for accidental changes, secrets, generated
    clutter, and API inconsistencies.
10. Report files changed, tests run, known limitations, and the exact
    next step.

If a command fails, diagnose and fix the root cause. Do not hide
failures or mark a milestone complete while acceptance criteria remain
unmet.

------------------------------------------------------------------------

# 1. Product vision

goPanglima aims to make building web applications in Go feel productive
and approachable to developers who enjoy Laravel's developer experience.

The framework should provide:

-   Expressive route definitions.
-   Clean request and response APIs.
-   Controller and middleware conventions.
-   A cohesive application lifecycle.
-   Configuration and environment loading.
-   Database connections, query builder, ORM, migrations, and seeders.
-   Request validation.
-   Authentication and session support.
-   Server-rendered views.
-   Cache, events, queue, and scheduler integrations over time.
-   An Artisan-inspired command-line tool.
-   A reliable testing toolkit.
-   Clear documentation and runnable examples.

### 1.1 Product principles

  -----------------------------------------------------------------------
  Principle                           Meaning
  ----------------------------------- -----------------------------------
  Laravel-like DX                     Familiar conventions, readable
                                      APIs, useful generators

  Native Go foundation                Standard library first, idiomatic
                                      implementation

  Explicitness                        Errors, dependencies, and lifecycle
                                      remain understandable

  Type safety                         Use Go's static type system and
                                      generics where beneficial

  Convention over configuration       Sensible defaults, with escape
                                      hatches

  Modular monolith                    One repository and Go module
                                      initially; clear package boundaries

  Security by default                 Safe defaults, no hidden unsafe
                                      shortcuts

  Incremental maturity                Stable features only; experimental
                                      features are clearly marked
  -----------------------------------------------------------------------

### 1.2 Target users and use cases

-   Developers moving from Laravel/PHP to Go.
-   Go developers who want a cohesive full-stack framework.
-   Teams building APIs, dashboards, internal tools, and business
    applications.
-   Teams that prefer a modular monolith over a distributed microservice
    architecture.

### 1.3 Non-goals

The initial project is not intended to:

-   Reimplement the PHP language or Laravel internals.
-   Guarantee source-level compatibility with Laravel.
-   Replace every specialized Go library.
-   Hide Go's error handling or concurrency model.
-   Build a frontend JavaScript framework.
-   Provide automatic dependency injection based on extensive
    reflection.
-   Promise benchmark superiority before evidence exists.

------------------------------------------------------------------------

# 2. Naming, repository, and module conventions

### 2.1 Branding

-   Framework display name: **goPanglima**
-   Repository slug: **gopanglima**
-   CLI executable: **gopanglima**
-   Go package/module path: use a lowercase path.

Recommended initial module path:

``` text
github.com/<GITHUB_USERNAME>/gopanglima
```

The agent must not invent the GitHub username. Keep the module path
configurable until the owner supplies it. If the project is being
developed locally before the remote repository exists, use a clearly
marked temporary module path and update it consistently before the first
public release.

Use lowercase Go package names, for example `httpx`, `database`,
`validation`, and `console`. The framework brand may retain the capital
`P` in prose and logo text, but repository paths and import paths should
remain lowercase.

### 2.2 Public repository readiness

Prepare these files before public release:

-   `README.md`
-   `LICENSE`
-   `CONTRIBUTING.md`
-   `CODE_OF_CONDUCT.md`
-   `SECURITY.md`
-   `CHANGELOG.md`
-   `.gitignore`
-   `.editorconfig`
-   `.github/ISSUE_TEMPLATE/`
-   `.github/PULL_REQUEST_TEMPLATE.md`
-   `.github/workflows/ci.yml`

Do not choose a license silently if the owner has not confirmed it.
Recommend MIT for a permissive framework, but ask the owner to confirm
the license and copyright holder before adding the final license text.

------------------------------------------------------------------------

# 3. Technical baseline and architecture decisions

## 3.1 Language and compatibility

-   Primary language: Go.
-   Initial compatibility target: Go 1.23 or newer, subject to
    verification against the installed toolchain and dependencies.
-   Declare the minimum supported Go version in `go.mod` and document
    it.
-   Do not use language or standard-library APIs newer than the declared
    minimum.
-   Re-evaluate the minimum version before a stable `v1.0.0` release.

The agent must verify the current installed Go toolchain rather than
assuming it.

## 3.2 Architectural style

Use a **modular monolith**:

-   One primary Git repository.
-   One Go module initially.
-   Separate packages with explicit responsibilities.
-   No circular package dependencies.
-   Public API separated from internal implementation where appropriate.
-   External adapters behind small interfaces.

Do not split into multiple Go modules until there is a demonstrated need
for independent release/versioning.

## 3.3 High-level architecture

``` mermaid
flowchart TD
    APP["User Application"] --> BOOT["Bootstrap"]
    BOOT --> KERNEL["Application Kernel"]
    KERNEL --> PROVIDERS["Service Providers"]
    KERNEL --> CONFIG["Configuration"]
    KERNEL --> HTTP["HTTP Layer"]
    KERNEL --> DB["Database Layer"]
    KERNEL --> CONSOLE["Console / CLI"]
    HTTP --> ROUTER["Router"]
    HTTP --> MW["Middleware Pipeline"]
    HTTP --> CTX["Request Context"]
    HTTP --> CTRL["Handlers / Controllers"]
    CTRL --> SERVICES["Application Services"]
    SERVICES --> ORM["ORM / Query Builder"]
    ORM --> DIALECT["SQL Dialects"]
    DIALECT --> SQL["database/sql"]
    KERNEL --> SECURITY["Security Services"]
    KERNEL --> VIEW["View Engine"]
    KERNEL --> CACHE["Cache"]
    KERNEL --> EVENTS["Events / Queue / Scheduler"]
```

### 3.4 Dependency direction

Preferred dependency direction:

``` text
cmd/CLI -> public framework API -> internal framework packages -> standard library / adapters
```

Rules:

-   Low-level packages must not import application packages.
-   `database` must not depend on HTTP.
-   `http` must not depend on a concrete ORM implementation.
-   CLI commands may call public framework interfaces and
    project-generation utilities.
-   Application code depends on goPanglima; the framework never imports
    application code.
-   Avoid global mutable state.

### 3.5 HTTP foundation

Use `net/http` as the underlying server foundation.

Build a goPanglima router abstraction on top of standard-library
primitives. Do not write a custom HTTP server or TCP stack.

The router API should support:

-   GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD.
-   Path parameters.
-   Route groups and prefixes.
-   Middleware at application, group, and route levels.
-   Named routes.
-   Route listing.
-   404 and 405 handling.
-   Graceful shutdown.
-   Request context and cancellation.

Evaluate `http.ServeMux` capabilities for the declared minimum Go
version. If a third-party router is selected, document why and isolate
it behind the framework router API. Do not expose third-party router
types throughout the public API.

### 3.6 Database foundation

Use `database/sql` as the initial execution contract.

-   PostgreSQL: use a maintained PostgreSQL driver, with pgx's
    `database/sql` adapter as a candidate.
-   MySQL: use a maintained MySQL driver.
-   SQLite: select a maintained driver and document CGO requirements, if
    any.

The ORM, query builder, migration engine, and SQL dialect abstraction
are goPanglima-owned components. Drivers are adapters.

Every database operation must support context-aware execution. All user
values must be bound parameters. Dynamic identifiers must be validated
or represented through safe identifier APIs.

### 3.7 Other foundational choices

  -----------------------------------------------------------------------
  Area                                Initial direction
  ----------------------------------- -----------------------------------
  CLI                                 Cobra is a candidate; keep command
                                      registration isolated

  Configuration                       Standard library plus a small
                                      `.env` loader if needed

  Logging                             `log/slog`

  Validation                          Framework-owned API; optionally
                                      delegate rule validation to a
                                      mature library

  HTML templates                      `html/template`

  Password hashing                    Argon2id via a maintained
                                      `golang.org/x/crypto` package

  UUID                                Optional maintained UUID library;
                                      avoid requiring UUID for all models

  Testing                             Standard `testing` package,
                                      `httptest`, test helpers

  Metrics                             Optional adapter; not required for
                                      the first milestone

  Redis                               Optional adapter for cache/queue,
                                      not a hard dependency
  -----------------------------------------------------------------------

Before adding any dependency, verify maintenance status, license, Go
compatibility, security advisories, and whether the standard library
already solves the problem.

------------------------------------------------------------------------

# 4. Proposed repository structure

The agent should implement this structure progressively. Do not create
dozens of empty packages just to match the diagram.

``` text
gopanglima/
├── cmd/
│   └── gopanglima/
│       └── main.go
│
├── framework/
│   ├── application.go
│   ├── kernel.go
│   ├── errors.go
│   ├── foundation/
│   │   ├── app.go
│   │   ├── config.go
│   │   ├── env.go
│   │   ├── provider.go
│   │   └── container.go
│   │
│   ├── http/
│   │   ├── server.go
│   │   ├── router/
│   │   ├── context/
│   │   ├── request/
│   │   ├── response/
│   │   └── middleware/
│   │
│   ├── database/
│   │   ├── manager.go
│   │   ├── connection.go
│   │   ├── transaction.go
│   │   ├── dialect/
│   │   ├── query/
│   │   ├── orm/
│   │   ├── schema/
│   │   ├── migration/
│   │   └── drivers/
│   │
│   ├── validation/
│   ├── auth/
│   ├── session/
│   ├── security/
│   ├── cache/
│   ├── events/
│   ├── queue/
│   ├── scheduler/
│   ├── view/
│   ├── filesystem/
│   ├── mail/
│   ├── logging/
│   └── testing/
│
├── console/
│   ├── commands/
│   ├── generators/
│   ├── stubs/
│   └── output/
│
├── skeleton/
│   └── (starter application template)
│
├── examples/
│   ├── hello-world/
│   ├── rest-api/
│   ├── crud/
│   └── server-rendered/
│
├── docs/
│   ├── getting-started/
│   ├── architecture/
│   ├── routing/
│   ├── database/
│   ├── orm/
│   ├── validation/
│   ├── security/
│   ├── deployment/
│   └── contributing/
│
├── tests/
│   ├── integration/
│   ├── compatibility/
│   └── benchmarks/
│
├── .github/
│   └── workflows/
│       └── ci.yml
│
├── .gitignore
├── .editorconfig
├── .golangci.yml (optional; only if configured and tested)
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── CONTRIBUTING.md
├── SECURITY.md
├── CHANGELOG.md
└── LICENSE (after owner confirms license)
```

### 4.1 Starter application structure

The `gopanglima new <name>` command should generate a separate
application:

``` text
myapp/
├── app/
│   ├── controllers/
│   ├── models/
│   ├── middleware/
│   ├── services/
│   └── providers/
├── bootstrap/
│   └── app.go
├── config/
│   ├── app.go
│   ├── database.go
│   └── auth.go
├── database/
│   ├── migrations/
│   └── seeders/
├── routes/
│   ├── web.go
│   └── api.go
├── resources/
│   └── views/
├── storage/
│   ├── logs/
│   ├── cache/
│   └── sessions/
├── tests/
├── .env.example
├── .gitignore
├── go.mod
└── main.go
```

Use lowercase Go package directories. The framework may offer familiar
conceptual folder names, but should not use uppercase package
directories such as `Controllers` or `Models`.

------------------------------------------------------------------------

# 5. Public API design

The public API is the most important product deliverable. Keep it
coherent, documented, and stable.

The snippets below are **design targets**, not permission to expose
uncompilable APIs. The agent must reconcile signatures, imports, error
behavior, and package boundaries into a consistent implementation.

## 5.1 Application bootstrap

Desired user experience:

``` go
package main

import (
    "log"

    "github.com/<GITHUB_USERNAME>/gopanglima/framework"
)

func main() {
    app := framework.New()

    if err := app.Run(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

A generated application may use a bootstrap package:

``` go
package bootstrap

import "github.com/<GITHUB_USERNAME>/gopanglima/framework"

func New() (*framework.Application, error) {
    app := framework.New()

    // Register configuration, providers, routes, and services.
    // Return errors rather than hiding initialization failures.

    return app, nil
}
```

Do not create multiple competing bootstrap patterns. Pick one canonical
pattern and document it.

## 5.2 Route API

Desired style:

``` go
router.Get("/", HomeController.Index)
router.Get("/users", UserController.Index)
router.Post("/users", UserController.Store)
router.Get("/users/{id}", UserController.Show)
router.Put("/users/{id}", UserController.Update)
router.Delete("/users/{id}", UserController.Delete)
```

Route grouping:

``` go
router.Prefix("/api/v1").Group(func(r *framework.Router) {
    r.Get("/users", UserController.Index)

    r.Middleware(Auth.Middleware()).Group(func(r *framework.Router) {
        r.Get("/profile", UserController.Profile)
    })
})
```

Required semantics:

-   Group middleware runs in a documented order.
-   Middleware execution order must be tested.
-   Route registration errors must be visible.
-   Duplicate named routes must fail clearly.
-   Path parameter behavior must be documented.
-   Route definitions must be safe for concurrent serving after startup.
-   Do not use global route registries.

## 5.3 Handler and controller conventions

Use a small handler contract. Preferred initial shape:

``` go
type HandlerFunc func(*Context) error
```

A controller may be a struct with methods that satisfy the handler
contract:

``` go
type UserController struct {
    users UserService
}

func NewUserController(users UserService) *UserController {
    return &UserController{users: users}
}

func (c *UserController) Index(ctx *framework.Context) error {
    users, err := c.users.List(ctx.Request().Context())
    if err != nil {
        return err
    }

    return ctx.JSON(200, framework.M{
        "data": users,
    })
}
```

Do not require reflection-based controller discovery. Explicit route
registration is easier to debug and more idiomatic.

## 5.4 Context, request, and response

Target API:

``` go
ctx.Param("id")
ctx.Query("page")
ctx.Header("X-Request-ID")
ctx.Bind(&input)
ctx.JSON(200, data)
ctx.String(200, "Hello")
ctx.HTML(200, html)
ctx.View("users/index", data)
ctx.Redirect("/dashboard")
ctx.NoContent()
```

Requirements:

-   Framework context wraps `http.ResponseWriter` and `*http.Request`.
-   Expose the underlying standard request through a clearly named
    method.
-   Do not retain request-scoped context beyond the request lifecycle.
-   Define behavior for writing multiple responses.
-   JSON encoding errors must be handled.
-   Set content types consistently.
-   Do not automatically expose internal errors or stack traces in
    production.
-   Add request size limits and safe defaults for body parsing.

## 5.5 Error handling

Use Go errors internally. The HTTP layer should provide centralized
error mapping.

Design requirements:

-   Typed framework errors for common HTTP conditions.
-   Support wrapping with `%w`.
-   Preserve root causes for logs.
-   Return safe public error messages.
-   Do not serialize arbitrary internal errors to clients.
-   Provide configurable error handlers.
-   Distinguish validation errors, not found, unauthorized, forbidden,
    conflict, and internal server errors.

Example conceptual error:

``` go
return framework.NotFound("User not found")
```

Only expose this helper if its semantics and return type are consistent
with the handler contract.

## 5.6 Query builder and ORM

The ORM is a major custom component. Build it in layers:

1.  SQL execution abstraction.
2.  Dialect/compiler.
3.  Query builder.
4.  Struct mapping.
5.  Generic query API.
6.  Model CRUD helpers.
7.  Relationships and eager loading.
8.  Scopes and lifecycle hooks only after core behavior is stable.

Desired query style:

``` go
users, err := orm.Query[User](db).
    Where("active", true).
    OrderBy("name").
    Get(ctx)
```

Other desired operations:

``` go
user, err := orm.Query[User](db).Find(ctx, id)
user, err := orm.Query[User](db).First(ctx)

err := db.Create(ctx, &user)

err := db.Model(&user).Update(ctx, map[string]any{
    "name": "New Name",
})

err := db.Delete(ctx, &user)
```

These examples are illustrative. Design one consistent API for
create/update/delete and avoid duplicating the same operation across
incompatible entry points.

### ORM requirements

-   Struct field mapping through explicit tags and documented naming
    conventions.
-   Primary key support.
-   Insert, select, update, delete.
-   Nullable values and pointer fields.
-   Time fields.
-   Parameterized SQL.
-   Context-aware execution.
-   Transactions.
-   Pagination.
-   Safe handling of empty `IN` lists.
-   Clear behavior for zero values and omitted fields.
-   Explicit selected columns.
-   No implicit database access during struct reflection.
-   SQL dialect tests for PostgreSQL, MySQL, and SQLite.

### Generic API caution

Go generics are useful for typed query results, but do not force
generics into every API. Keep generic constraints simple and avoid
unsafe reflection where a clear explicit API is possible.

### Relationships --- later milestone

Potential API direction:

``` go
type User struct {
    ID      int64
    Posts   []Post `relation:"hasMany,foreignKey:UserID"`
    Profile *Profile `relation:"hasOne,foreignKey:UserID"`
}
```

Do not promise relationship support until the mapping rules, query count
behavior, eager loading, and error semantics are tested.

## 5.7 Migrations and schema builder

Desired migration style:

``` go
func Up(schema *database.Schema) error {
    return schema.Create("users", func(table *database.Blueprint) {
        table.ID()
        table.String("name", 150)
        table.String("email", 255).Unique()
        table.Boolean("active").Default(true)
        table.Timestamps()
    })
}
```

CLI targets:

``` bash
gopanglima make:migration create_users_table
gopanglima migrate
gopanglima migrate:rollback
gopanglima migrate:status
gopanglima migrate:fresh
gopanglima db:seed
```

Migration engine requirements:

-   Ordered migration identifiers.
-   Migration tracking table.
-   Up/down behavior.
-   Clear failure reporting.
-   Locking to prevent concurrent migration runs.
-   Transactions where supported.
-   Database-specific behavior documented.
-   Destructive commands require clear confirmation or an explicit
    `--force` flag.
-   Never run destructive migrations automatically during application
    startup.

Prefer compiled Go migration files or another transparent, type-safe
approach. Avoid arbitrary `eval`-style migration execution.

## 5.8 Validation

Desired API:

``` go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=3,max=100"`
    Email string `json:"email" validate:"required,email"`
}
```

Potential usage:

``` go
if err := ctx.Validate(&input); err != nil {
    return ctx.ValidationError(err)
}
```

Requirements:

-   Field-level errors.
-   Custom messages.
-   Custom validators.
-   Nested struct validation.
-   Slice/array validation.
-   JSON field names in returned errors.
-   Clear distinction between decoding and validation.
-   Database-dependent rules such as `unique` must be explicit and must
    not run hidden queries.

## 5.9 Authentication and authorization

Build in stages:

1.  Password hashing service.
2.  Session abstraction.
3.  Cookie security.
4.  Login/logout.
5.  Token authentication.
6.  Password reset.
7.  Role/permission authorization.

Requirements:

-   Use Argon2id or another well-reviewed password hashing algorithm
    with documented parameters.
-   Never store plaintext passwords.
-   Constant-time verification where applicable.
-   Secure, HttpOnly, SameSite cookies.
-   Secure cookie mode in production.
-   CSRF protection for cookie/session-authenticated browser forms.
-   Token storage and expiration policy.
-   Session rotation after authentication.
-   Rate limiting for login and password reset.
-   No hardcoded secrets.
-   No default production credentials.

Do not implement a custom cryptographic algorithm.

## 5.10 Views

Use `html/template` as the initial engine.

Desired usage:

``` go
return ctx.View("home/index", framework.M{
    "title": "Dashboard",
    "name":  "Guest",
})
```

Requirements:

-   HTML contextual escaping.
-   Layouts and partials.
-   Template parsing at startup where practical.
-   Clear missing-template errors.
-   Development reload support only if implemented safely.
-   No `template.HTML` conversion for untrusted input.

A Blade-like syntax is not required for version 1.0. Prioritize safe,
conventional Go templates.

## 5.11 Cache, events, queue, scheduler, filesystem, mail

These are later modules. Define small interfaces first and implement
adapters incrementally.

-   Cache: Get, Set, Delete, expiration, context support.
-   Events: explicit event registration and dispatch.
-   Queue: durable job backend, retries, timeout, visibility/locking
    semantics.
-   Scheduler: timezone-aware scheduling and overlap prevention.
-   Filesystem: local storage first; cloud adapters later.
-   Mail: provider interface and SMTP adapter.

Do not present an in-memory queue as production durable storage. Clearly
label development-only adapters.

------------------------------------------------------------------------

# 6. CLI design

Executable: `gopanglima`

Target commands:

``` bash
# Create a project
gopanglima new shop

# Run development server
gopanglima serve

# Generators
gopanglima make:controller UserController
gopanglima make:model User
gopanglima make:middleware AuthMiddleware
gopanglima make:service UserService
gopanglima make:migration create_users_table
gopanglima make:model User --migration

# Database
gopanglima migrate
gopanglima migrate:rollback
gopanglima migrate:status
gopanglima db:seed

# Introspection
gopanglima route:list
gopanglima about

# Quality
gopanglima test
```

Command requirements:

-   Helpful `--help` output.
-   Predictable exit codes.
-   Useful error messages.
-   No secret values in logs.
-   Destructive operations require confirmation or `--force`.
-   Generators must produce compilable Go code.
-   Generator output must respect the application's module path.
-   Generators must not overwrite existing files without explicit force.
-   Commands must work from the application root and detect invalid
    working directories.

CLI commands should be registered in a maintainable command tree. Avoid
one giant `main.go`.

------------------------------------------------------------------------

# 7. Testing and quality strategy

## 7.1 Required test layers

  Test type       Coverage
  --------------- ------------------------------------------------
  Unit            Individual package behavior
  Integration     Database, migration, adapters
  HTTP feature    Routes, middleware, request/response
  Security        Auth, cookies, CSRF, SQL injection protections
  Compatibility   SQL dialects and supported Go versions
  Benchmark       Router, middleware, ORM hot paths

Use `httptest` for HTTP tests. Use disposable databases or containers
for integration tests when available. Tests must not depend on a
developer's personal database.

## 7.2 Quality commands

At minimum, CI should run:

``` bash
gofmt -w .
go vet ./...
go test ./...
```

Also run where practical:

``` bash
go test -race ./...
go test -cover ./...
```

Do not make coverage percentage the only quality metric. Focus on
meaningful behavior and edge cases.

## 7.3 Benchmarks

Create reproducible benchmarks for:

-   Router dispatch.
-   Middleware pipeline.
-   JSON response.
-   Query builder compilation.
-   Struct mapping.
-   ORM reads.
-   ORM inserts.

Report allocations as well as execution time. Compare against a simple
`net/http` baseline only when the benchmark is equivalent.

------------------------------------------------------------------------

# 8. Development roadmap and acceptance criteria

Build in milestones. The estimates are planning ranges, not promises.

## Milestone 0 --- Repository bootstrap

**Goal:** Establish a clean, public-repository-ready Go project.

Tasks:

-   Verify Go toolchain.
-   Confirm module path with owner.
-   Create repository structure incrementally.
-   Add README draft and project conventions.
-   Add `.gitignore` and `.editorconfig`.
-   Add CI workflow.
-   Add contribution and security policy drafts.
-   Decide license with owner.
-   Add architecture decision records (ADRs).

Acceptance criteria:

-   Project builds.
-   CI runs on pull requests.
-   No secrets or generated clutter are tracked.
-   README explains project status honestly.
-   License decision is recorded.

## Milestone 1 --- Foundation and application lifecycle

**Goal:** Create and run a framework application.

Tasks:

-   Application type.
-   Kernel/lifecycle.
-   Configuration.
-   Environment loading.
-   Logger.
-   Service provider registration.
-   Error types.
-   Basic shutdown hooks.

Acceptance criteria:

-   A minimal application initializes.
-   Configuration can be loaded and tested.
-   Provider registration order is deterministic.
-   Initialization errors are returned.
-   Unit tests cover lifecycle and configuration.

## Milestone 2 --- HTTP and routing

**Goal:** Run a real HTTP application.

Tasks:

-   Server wrapper.
-   Router abstraction.
-   Handler contract.
-   Context.
-   Request and response helpers.
-   Route parameters.
-   Route groups.
-   Middleware.
-   404/405 behavior.
-   Graceful shutdown.

Acceptance criteria:

-   GET/POST/PUT/PATCH/DELETE work.
-   Route parameters work.
-   Middleware ordering is tested.
-   Errors map to safe HTTP responses.
-   `httptest` feature tests pass.
-   Example HTTP application runs.

## Milestone 3 --- Database foundation and query builder

**Goal:** Execute safe, context-aware SQL.

Tasks:

-   Database configuration.
-   Connection manager.
-   `database/sql` abstraction.
-   PostgreSQL adapter.
-   SQL dialect interface.
-   Query representation.
-   SQL compiler.
-   Query builder.
-   Transactions.

Acceptance criteria:

-   Connection lifecycle works.
-   Context cancellation works.
-   Parameter binding is tested.
-   SQL compilation tests cover dialect placeholders.
-   Transactions commit and rollback correctly.
-   Integration tests pass against PostgreSQL.

## Milestone 4 --- ORM and migrations

**Goal:** Build a useful CRUD application.

Tasks:

-   Struct mapping.
-   Generic query API.
-   CRUD.
-   Pagination.
-   Schema builder.
-   Migration registry and runner.
-   Migration tracking.
-   Seeders.

Acceptance criteria:

-   CRUD integration tests pass.
-   SQL injection tests cover user-provided values.
-   Null and zero-value mapping is documented and tested.
-   Migration ordering and rollback are tested.
-   Concurrent migration execution is guarded.
-   Example CRUD app works.

## Milestone 5 --- CLI and project generator

**Goal:** Create and manage an application through CLI.

Tasks:

-   CLI executable.
-   `new` command.
-   Starter skeleton.
-   `serve`.
-   `make:*` generators.
-   `migrate` commands.
-   `route:list`.
-   `about`.

Acceptance criteria:

-   Generated application compiles.
-   Generators respect module path.
-   Existing files are not silently overwritten.
-   CLI errors and exit codes are tested.
-   Getting-started documentation works from a clean directory.

## Milestone 6 --- Validation and security foundation

**Goal:** Safely process application input and user identity.

Tasks:

-   Request validation.
-   Validation error format.
-   Password hashing.
-   Session interface and cookie implementation.
-   Authentication middleware.
-   CSRF.
-   Rate limiting.
-   Authorization primitives.

Acceptance criteria:

-   Invalid input returns consistent errors.
-   Passwords are never stored in plaintext.
-   Session cookies use secure settings in production.
-   CSRF protections are tested for browser session flows.
-   Authentication tests cover success, failure, expiration, and logout.

## Milestone 7 --- Full-stack features

**Goal:** Support server-rendered applications.

Tasks:

-   View engine.
-   Layout and partial support.
-   Flash messages if session support is stable.
-   Filesystem abstraction.
-   File upload limits and validation.
-   Mail interface and initial adapter.
-   Cache interface and initial adapter.

Acceptance criteria:

-   Templates escape untrusted data.
-   Missing templates return useful errors.
-   Upload limits are enforced.
-   Cache expiration behavior is tested.
-   Example server-rendered application works.

## Milestone 8 --- Advanced features

**Goal:** Add capabilities after core APIs have stabilized.

Tasks:

-   ORM relationships.
-   Eager loading.
-   Soft deletes.
-   Model scopes.
-   Events.
-   Durable queue adapter.
-   Scheduler.
-   Redis integrations.
-   MySQL and SQLite compatibility.

Acceptance criteria:

-   Each feature has documentation and tests.
-   Database-specific limitations are documented.
-   Queue retry and failure semantics are tested.
-   Scheduler overlap behavior is tested.
-   No experimental feature is mislabeled stable.

## Milestone 9 --- Release candidate and v1.0

**Goal:** Prepare a trustworthy public release.

Tasks:

-   API review.
-   Security review.
-   Compatibility matrix.
-   Benchmarks.
-   Complete examples.
-   Documentation review.
-   Changelog.
-   Versioning policy.
-   Release workflow.
-   Migration and upgrade guide.

Acceptance criteria:

-   All supported CI jobs pass.
-   Public API is documented.
-   Installation and quickstart work from a clean environment.
-   No known critical security issue remains.
-   Supported Go and database versions are explicit.
-   License is confirmed.
-   Release notes are accurate.
-   No performance claims lack benchmark evidence.

------------------------------------------------------------------------

# 9. Definition of done for every feature

A feature is not complete until all applicable items are satisfied:

-   [ ] Public API design reviewed.
-   [ ] Implementation follows package boundaries.
-   [ ] Input validation and error paths implemented.
-   [ ] Context cancellation handled where applicable.
-   [ ] Unit tests added.
-   [ ] Integration tests added where applicable.
-   [ ] Race behavior considered for concurrent code.
-   [ ] Security implications reviewed.
-   [ ] `gofmt` completed.
-   [ ] `go vet ./...` passes.
-   [ ] `go test ./...` passes.
-   [ ] Public documentation updated.
-   [ ] Example updated or added.
-   [ ] No TODO-only public implementation.
-   [ ] No accidental breaking API changes.
-   [ ] Work log and changelog updated when appropriate.

------------------------------------------------------------------------

# 10. Documentation plan

The public documentation should be task-oriented.

Recommended docs navigation:

``` text
docs/
├── 01-getting-started/
│   ├── installation.md
│   ├── create-project.md
│   ├── directory-structure.md
│   └── first-app.md
├── 02-foundation/
│   ├── lifecycle.md
│   ├── configuration.md
│   ├── environment.md
│   └── service-providers.md
├── 03-http/
│   ├── routing.md
│   ├── controllers.md
│   ├── middleware.md
│   ├── requests.md
│   └── responses.md
├── 04-database/
│   ├── connections.md
│   ├── query-builder.md
│   ├── models.md
│   ├── migrations.md
│   └── transactions.md
├── 05-validation/
├── 06-security/
├── 07-views/
├── 08-cli/
├── 09-testing/
├── 10-deployment/
└── architecture/
    ├── overview.md
    ├── package-boundaries.md
    └── decisions/
```

Every guide should include:

-   What the feature does.
-   A minimal working example.
-   Configuration.
-   Error behavior.
-   Security considerations.
-   Relevant CLI commands.
-   Links to related guides.

Never document planned features as if they already exist. Mark roadmap
items clearly.

------------------------------------------------------------------------

# 11. README requirements

The root README should include:

1.  goPanglima name and tagline.
2.  Honest project status (`experimental`, `pre-alpha`, `beta`, or
    `stable`).
3.  Short description.
4.  Key design principles.
5.  Installation instructions once available.
6.  Quickstart that actually compiles.
7.  Feature status table.
8.  Documentation links.
9.  Contribution instructions.
10. License information after confirmation.
11. Security reporting instructions.

Do not use fake badges, fake test status, fake download counts, or
unverified benchmark claims.

Suggested opening:

``` markdown
# goPanglima

**Laravel Developer Experience. Native Go Foundation.**

goPanglima is a Laravel-inspired web framework built in Go.
It aims to combine expressive application development with Go's
type safety, explicitness, and standard-library foundation.

> Project status: experimental. APIs may change before v1.0.
```

------------------------------------------------------------------------

# 12. Git workflow and public publishing

The agent may prepare local commits if the user has requested that. It
must not publish remotely without authorization.

Recommended branch model:

-   `main`: always buildable.
-   `feat/<name>`: feature development.
-   `fix/<name>`: bug fixes.
-   `docs/<name>`: documentation.
-   `release/<version>`: release preparation.

Commit style:

``` text
feat(http): add route groups
feat(database): add postgres connection manager
fix(router): preserve middleware order
docs(quickstart): add first application guide
test(orm): cover nullable field mapping
```

Before a public push:

1.  Confirm the exact remote URL and owner.
2.  Confirm repository visibility is public.
3.  Confirm license and copyright holder.
4.  Run all tests and inspect Git status.
5.  Scan for secrets and personal data.
6.  Review README and project status.
7.  Confirm the user explicitly authorizes the push.

Never force-push to `main` unless explicitly authorized and necessary.

------------------------------------------------------------------------

# 13. Security checklist

-   [ ] Parameterized SQL everywhere.
-   [ ] Safe identifier handling in query builder.
-   [ ] HTML escaping by default.
-   [ ] No untrusted string inserted into `template.HTML`.
-   [ ] Request body size limits.
-   [ ] Upload size and content validation.
-   [ ] Secure cookie defaults in production.
-   [ ] CSRF protection for cookie-authenticated browser forms.
-   [ ] Password hashing with a reviewed algorithm.
-   [ ] Authentication rate limiting.
-   [ ] Session rotation and expiration.
-   [ ] No secrets in logs.
-   [ ] No detailed internal errors exposed to clients.
-   [ ] Safe file path handling.
-   [ ] Dependency vulnerability review.
-   [ ] Security policy and responsible disclosure process.

------------------------------------------------------------------------

# 14. Performance and reliability principles

Do not optimize prematurely, but do not design avoidable overhead into
the framework.

-   Avoid unnecessary reflection in request hot paths.
-   Cache parsed route metadata after registration.
-   Parse templates during startup where practical.
-   Reuse database connection pools.
-   Avoid global locks in request execution.
-   Avoid unbounded goroutine creation.
-   Respect context deadlines.
-   Use bounded queues and worker pools.
-   Measure allocations in benchmarks.
-   Include graceful shutdown and resource cleanup.
-   Make concurrency guarantees explicit.

Every benchmark must describe hardware/software environment and exact
command used.

------------------------------------------------------------------------

# 15. Initial implementation plan: what to build first

The first implementation session should only build **Milestone 0 and
Milestone 1**, not the whole framework.

### First coding task

1.  Inspect repository and environment.
2.  Confirm module path placeholder and installed Go version.
3.  Create minimal repository structure.
4.  Implement `framework.Application`.
5.  Implement application options/configuration.
6.  Implement a small service provider contract.
7.  Implement lifecycle ordering.
8.  Implement structured logger using `slog`.
9.  Add unit tests.
10. Add README with experimental status.
11. Add CI workflow.
12. Run format, vet, and tests.
13. Summarize implementation and propose the next milestone.

### First milestone acceptance test

A minimal example should eventually look like this:

``` go
package main

import (
    "log"

    "github.com/<GITHUB_USERNAME>/gopanglima/framework"
)

func main() {
    app := framework.New()

    if err := app.Run(":8080"); err != nil {
        log.Fatal(err)
    }
}
```

For Milestone 1, if the HTTP server is not yet implemented, do not
pretend `Run` works. Instead, expose only the lifecycle APIs that are
actually implemented and add the HTTP server in Milestone 2.

The agent must keep examples synchronized with real code.

------------------------------------------------------------------------

# 16. Master prompt for the AI coding agent

Copy the following prompt into the coding agent together with this
document.

------------------------------------------------------------------------

## MASTER EXECUTION PROMPT

You are the lead Go framework engineer for **goPanglima**, a
Laravel-inspired web framework written in Go.

Read the complete `GOPANGLIMA_BLUEPRINT.md` before making changes. Treat
it as the authoritative project specification.

Your job is to implement goPanglima incrementally and prepare it for an
eventual public GitHub repository.

### Engineering expectations

-   Build a real, maintainable Go framework, not a demo or mockup.
-   Follow idiomatic Go internally while keeping the public developer
    experience expressive and Laravel-inspired.
-   Prefer standard-library foundations and small, maintained
    dependencies.
-   Keep the framework modular within one Go module initially.
-   Avoid circular dependencies and global mutable state.
-   Preserve explicit errors and context cancellation.
-   Write tests with every feature.
-   Keep the repository compiling after each milestone.
-   Document every public API.
-   Do not claim unfinished features are implemented.
-   Do not invent repository owner, module path, credentials, license
    holder, or remote URL.
-   Never publish, push, or change remote repository visibility without
    explicit authorization.

### Execution process

1.  Inspect the existing repository and environment.
2.  Identify the current milestone and what is already implemented.
3.  Compare implementation against the blueprint.
4.  Present a short plan for the next coherent milestone.
5.  Implement only that milestone.
6.  Add tests and examples.
7.  Run `gofmt`, `go vet ./...`, and `go test ./...`.
8.  Run race tests where appropriate.
9.  Review the diff for correctness, security, and accidental changes.
10. Update README/docs/changelog.
11. Report:

-   What was implemented.
-   Important design decisions.
-   Files created or changed.
-   Commands and tests executed.
-   Test results.
-   Known limitations.
-   Recommended next task.

### Scope control

Start with Milestone 0 and Milestone 1 only. Do not jump ahead to ORM,
authentication, queue, or advanced features.

If a design decision is ambiguous, choose the simplest idiomatic and
maintainable option, document it as an ADR, and continue. Ask the owner
only when a decision requires personal/repository information or an
irreversible action.

Do not stop after merely generating a plan. If the environment permits
file editing and command execution, implement the approved milestone in
the repository.

**Begin by inspecting the repository and reporting the current state.
Then implement the first milestone in small, verifiable steps.**

------------------------------------------------------------------------

# 17. Open decisions for the project owner

The agent can begin the local foundation without all these answers, but
they must be resolved before public release.

  ---------------------------------------------------------------------------------
  Decision                Recommended default               Status
  ----------------------- --------------------------------- -----------------------
  GitHub owner            Owner's actual username           Needs confirmation

  Module path             `github.com/<owner>/gopanglima`   Needs confirmation

  License                 MIT                               Owner must approve

  Initial database        PostgreSQL                        Recommended

  Additional databases    MySQL, SQLite                     Later

  Minimum Go version      Verify current toolchain; target  Verify
                          Go 1.23+ initially                

  Framework status        Experimental / pre-alpha          Initial

  Documentation hosting   GitHub repository docs initially  Recommended

  Release policy          Semantic Versioning               Recommended
  ---------------------------------------------------------------------------------

------------------------------------------------------------------------

# 18. Final product definition

goPanglima is successful when a developer can:

1.  Create a project with one CLI command.
2.  Understand the folder structure immediately.
3.  Register routes without boilerplate.
4.  Write clean controllers.
5.  Validate requests consistently.
6.  Work with database models and queries comfortably.
7.  Create and run migrations.
8.  Implement authentication securely.
9.  Render server-side views.
10. Test application behavior easily.
11. Deploy a conventional Go application.
12. Understand the underlying Go behavior when debugging.

The goal is not to make Go look exactly like PHP. The goal is to make Go
application development feel as cohesive and productive as Laravel,
while preserving Go's strengths.

**goPanglima --- Laravel Developer Experience. Native Go Foundation.**
