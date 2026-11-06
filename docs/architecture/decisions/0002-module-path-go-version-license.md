# ADR 0002: Module path, minimum Go version, and license

- Status: Accepted
- Date: 2026-11-06

## Context

The blueprint requires the owner to confirm the module path and license, and
requires verifying the toolchain rather than assuming it.

## Decision

- Module path: `github.com/mohammadirham37/gopanglima` (confirmed by the owner,
  matching the repository `git@github.com:mohammadirham37/gopanglima.git`).
- Minimum Go version: **1.23**, declared in `go.mod`. Development used Go
  1.27.3; the full test suite was also run with Go 1.23.12. CI tests 1.23.x
  and the latest stable release.
- License: **MIT**, copyright holder **Mohammad Irham** (confirmed by the owner).

## Consequences

Code must not use standard-library APIs newer than Go 1.23 (for example
`testing.B.Loop`, `strings.SplitSeq`, `sync.WaitGroup.Go`, `os.Root`). The
1.23 CI job enforces this.
