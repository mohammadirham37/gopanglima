# Your first application

## Minimal program

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

	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
```

```bash
go run .
curl http://localhost:8080/
```

`Run` blocks until the process receives SIGINT (Ctrl+C) or SIGTERM, then stops
accepting connections, waits for in-flight requests (10 s by default), and
runs shutdown hooks.

## Recommended layout

Until the project generator exists, this layout mirrors the planned
`gopanglima new` skeleton and is used by [`examples/rest-api`](../../examples/rest-api):

```text
myapp/
├── app/
│   ├── controllers/   # HTTP controllers (structs whose methods are handlers)
│   └── services/      # application services
├── routes/
│   └── api.go         # func API(r *framework.Router, ...)
├── bootstrap.go       # builds the *framework.Application
├── main.go            # calls bootstrap and app.Run
├── .env.example       # documented settings (commit this)
└── .env               # local settings (never commit this)
```

## The canonical bootstrap pattern

Build the application in one function that returns errors instead of hiding
them. `main` and your tests both call it:

```go
func newApp(opts ...framework.Option) (*framework.Application, error) {
	app := framework.New(append([]framework.Option{framework.WithEnvFile(".env")}, opts...)...)

	users := services.NewMemoryUserService()
	foundation.Bind[services.UserService](app.Container(), users)
	routes.API(app.Router(), controllers.NewUserController(users))

	if err := app.Router().Err(); err != nil {
		return nil, err
	}
	return app, nil
}
```

## Testing

`app.Handler()` returns an `http.Handler` you can drive with `httptest`:

```go
h, err := app.Handler()
if err != nil {
	t.Fatal(err)
}
rec := httptest.NewRecorder()
h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
```

Related: [Lifecycle](../02-foundation/lifecycle.md), [Routing](../03-http/routing.md), [Controllers](../03-http/controllers.md).
