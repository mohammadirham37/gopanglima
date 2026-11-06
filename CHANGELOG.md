# Changelog

All notable changes to this project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project intends to follow [Semantic Versioning](https://semver.org/)
once it reaches a first tagged release.

## [Unreleased]

### Added

- Repository bootstrap: README, license (MIT), contribution guide, code of
  conduct, security policy, CI workflow (Go 1.23 and stable), Makefile,
  issue and pull request templates, architecture decision records.
- `framework`: `Application` with `New`, functional options, service
  providers (`Provider`, `Booter`, `ProviderFunc`), deterministic
  register/boot order, deferred initialization errors, shutdown hooks,
  `Handler`, `Serve`, and `Run` with graceful shutdown on SIGINT/SIGTERM.
- `framework/foundation`: `.env` parser and layered `Env`, concurrency-safe
  `Config` with typed getters and strict `Lookup`, type-keyed `Container`.
- `framework/logging`: `slog` logger construction from level/format.
- `framework/httpx`: router with path parameters, catch-all parameters,
  groups, prefixes, global/group/route middleware, named routes, URL
  generation, route listing, 404/405 with `Allow`, automatic `OPTIONS` and
  `HEAD`; request `Context` with JSON binding and response helpers; `Error`
  type and centralized error handler; graceful `Serve`.
- `framework/httpx/middleware`: `Recover`, `RequestID`, `RequestLogger`.
- Examples: `hello-world`, `rest-api`.
