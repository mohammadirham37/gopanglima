# Security Policy

## Supported versions

goPanglima is experimental and has no stable release yet. Security fixes are
applied to the `main` branch only.

## Reporting a vulnerability

**Please do not open public issues for security vulnerabilities.**

Report vulnerabilities privately through GitHub's private vulnerability
reporting:
<https://github.com/mohammadirham37/gopanglima/security/advisories/new>

Please include:

- A description of the issue and its impact.
- Steps or code to reproduce it.
- The affected version or commit.

You can expect an acknowledgement within 7 days. Once a fix is available, the
issue will be disclosed in a GitHub security advisory and credited to you
unless you prefer otherwise.

## Security defaults

goPanglima aims to be secure by default. Current safeguards include:

- Request bodies decoded by `Context.Bind` are limited (1 MiB by default).
- The HTTP server sets read-header, read, write, and idle timeouts.
- Internal error details are logged but never sent to clients unless debug
  mode (`APP_DEBUG=true`) is explicitly enabled. Never enable debug mode in
  production.
- Panics are recovered and returned as generic 500 responses.
- Route parameters never contain `.` or `..` path elements (including
  encoded forms such as `..%2F`), single-segment parameters never contain
  `/`, and non-canonical request paths are redirected to their canonical
  form.
- The query string is excluded from access logs.
