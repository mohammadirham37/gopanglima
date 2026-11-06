# Requests and responses

## Reading the request

```go
c.Param("id")                 // route parameter ("" if absent)
c.Params()                    // all parameters, in path order
c.Query("page")               // first query value ("" if absent)
c.QueryDefault("page", "1")   // with default for absent/empty
c.QueryValues()               // url.Values
c.Header("X-Request-ID")      // request header
c.Request()                   // the underlying *http.Request
c.Context()                   // request context.Context
c.RouteName(), c.RoutePattern()
```

Request-scoped values (for example, the authenticated user set by
middleware):

```go
c.Set("user", user)
v, ok := c.Get("user")
```

To add `context.Context` values, use
`c.SetRequest(c.Request().WithContext(ctx))`.

## Binding JSON

```go
var in CreateUserRequest
if err := c.Bind(&in); err != nil {
	return err // already a client-safe *HTTPError
}
```

`Bind`:

- Requires `Content-Type: application/json` or `application/*+json` → otherwise **415**.
- Limits the body to `WithMaxBodyBytes` (default 1 MiB) → **413** when exceeded.
- Requires exactly one JSON value → **400** for empty, malformed, mistyped, or trailing data.
- Ignores unknown fields.
- Only decodes. Declarative validation is planned for a later milestone.

## Writing responses

```go
c.JSON(200, framework.M{"data": users})   // application/json; charset=utf-8
c.String(200, "Hello")                    // text/plain; charset=utf-8
c.HTML(200, "<h1>Trusted markup</h1>")    // text/html — NOT escaped
c.Blob(200, "image/png", data)
c.Redirect("/dashboard")                  // 302
c.RedirectWithStatus(301, "/new")         // any 3xx
c.NoContent()                             // 204
c.Status(202)                             // status only
c.SetHeader("Cache-Control", "no-store")
```

Rules:

- **One response per request.** Once the status line is sent, further
  response helpers return `httpx.ErrResponseCommitted` and write nothing.
- `JSON` encodes before writing: an encoding failure returns an error and
  leaves the response untouched, so the error handler can send a clean 500.
- Body-writing helpers (`JSON`, `String`, `HTML`, `Blob`) set
  `X-Content-Type-Options: nosniff`.
- Status codes must be final responses (200–999); other codes return an
  error. Informational `1xx` codes written with `c.Response().WriteHeader`
  (for example 103 Early Hints) do not commit the response.
- `http.NewResponseController(c.Response()).Flush()` commits a `200` if
  nothing was written yet.
- `HTML` writes its argument verbatim. Never pass untrusted input; a safe
  `html/template` view engine is planned.
- Never pass unvalidated user input to `Redirect` (open redirect).

Related: [Errors](errors.md), [Controllers](controllers.md).
