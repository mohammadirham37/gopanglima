# Contributing to goPanglima

Thanks for your interest in goPanglima! The project is experimental, so APIs
can still change — discussion before large changes is very welcome.

## Before you start

- For bugs, open an issue with a minimal reproduction.
- For new features or API changes, open an issue first describing the
  application-facing API you want. The roadmap lives in
  [GOPANGLIMA_BLUEPRINT.md](GOPANGLIMA_BLUEPRINT.md).
- Security issues must be reported privately; see [SECURITY.md](SECURITY.md).

## Development setup

Requirements: Go 1.23 or newer.

```bash
git clone git@github.com:mohammadirham37/gopanglima.git
cd gopanglima
make check
```

## Guidelines

- **Standard library first.** Adding a dependency needs a clear justification
  (maintenance, license, security, and why the standard library is not enough).
- **Idiomatic Go.** Laravel-like ergonomics are a goal; imitating PHP is not.
- **Explicit errors.** No panics for control flow, no swallowed errors.
- **Context.** Blocking operations accept and honor `context.Context`.
- **No fake implementations.** Do not add public APIs that do nothing or
  pretend to succeed.
- **Tests.** Every behavior change needs tests, including error paths. Use
  `net/http/httptest` for HTTP behavior.
- **Docs.** Exported identifiers need GoDoc comments. Update `docs/`,
  examples, and `CHANGELOG.md` when public behavior changes.
- **Compatibility.** Do not use APIs newer than the Go version declared in
  `go.mod`. CI tests the minimum version and the latest stable release.

## Commits and pull requests

- Branch from `main`: `feat/<name>`, `fix/<name>`, `docs/<name>`.
- Use conventional-style commit messages, for example
  `feat(http): add route groups` or `fix(router): preserve middleware order`.
- Make sure `make check` passes before opening a pull request.
- Keep pull requests focused; unrelated refactors belong in separate PRs.

By contributing, you agree that your contributions are licensed under the
project's [MIT License](LICENSE).
