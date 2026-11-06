# HTTP errors

Handlers and middleware return errors; one error handler turns them into
responses.

## Typed HTTP errors

```go
return framework.NotFound("User not found")
return framework.Forbidden("")                     // message defaults to "Forbidden"
return framework.NewHTTPError(418, "I'm a teapot")
return framework.Conflict("Email already taken").WithCause(err)
```

Helpers: `BadRequest` (400), `Unauthorized` (401), `Forbidden` (403),
`NotFound` (404), `Conflict` (409), `UnprocessableEntity` (422),
`TooManyRequests` (429), `InternalServerError` (500). `httpx` additionally
provides `MethodNotAllowed`, `RequestEntityTooLarge`, and `UnsupportedMediaType`.

`*HTTPError` works with `errors.Is` / `errors.As` and may be wrapped with
`fmt.Errorf("...: %w", err)` — the status is preserved. `WithCause` attaches an
internal cause that is logged but never sent to the client.

## Default error handler

| Returned error | Response |
| --- | --- |
| `*HTTPError` (possibly wrapped) with a 4xx/5xx code | Its status and public message |
| `*HTTPError` with any other code | `500` (the invalid code is logged) |
| anything else | `500 Internal Server Error` |
| client disconnected (`context.Canceled`) | nothing written; debug log |

Body:

```json
{"error":{"status":404,"message":"User not found"}}
```

- 5xx errors are logged at `ERROR` with the full error chain; 4xx at `DEBUG`.
- Internal details are **never** sent to clients unless `app.debug` is true,
  in which case 5xx bodies include a `detail` field. Never enable debug mode
  in production.
- If the response was already committed, nothing more is written.

## Custom error handler

```go
app := framework.New(framework.WithErrorHandler(func(c *framework.Context, err error) {
	var he *framework.HTTPError
	if errors.As(err, &he) {
		_ = c.String(he.Code, he.Message)
		return
	}
	c.Logger().Error("unhandled", "error", err)
	_ = c.String(500, "Something went wrong")
}))
```

Related: [Requests and responses](requests-responses.md), [Middleware](middleware.md).
