# Service providers and the container

## Container

`app.Container()` is a small, explicit registry keyed by Go type. There is
no autowiring or reflection-based injection: you bind values and resolve them
by type.

```go
foundation.Bind[services.UserService](app.Container(), services.NewMemoryUserService())

users, err := foundation.Make[services.UserService](app.Container())
// errors.Is(err, foundation.ErrServiceNotFound) when nothing is bound

ok := foundation.Bound[services.UserService](app.Container())
```

Binding interface types keeps consumers decoupled from implementations.
Interface and concrete types are distinct keys.

## Providers

A provider groups the wiring for one area of the application.

```go
type UserProvider struct{}

// Register binds services. It runs as soon as the provider is registered.
func (UserProvider) Register(app *framework.Application) error {
	foundation.Bind[services.UserService](app.Container(), services.NewMemoryUserService())
	return nil
}

// Boot is optional (framework.Booter). It runs during Bootstrap, after all
// providers are registered, so it may use services from other providers.
func (UserProvider) Boot(ctx context.Context, app *framework.Application) error {
	users, err := foundation.Make[services.UserService](app.Container())
	if err != nil {
		return err
	}
	ctrl := controllers.NewUserController(users)
	app.Router().Get("/users", ctrl.Index)
	return nil
}

app := framework.New(framework.WithProviders(UserProvider{}))
```

For one-off wiring, `framework.ProviderFunc(func(app *framework.Application) error { ... })`
adapts a function.

Ordering guarantees:

- `Register` methods run in registration order, immediately.
- `Boot` methods run in registration order during `Bootstrap`, stopping at the
  first error.
- Registering providers after `Bootstrap` returns `framework.ErrAlreadyBooted`.

## Releasing resources

Register cleanup with `app.OnShutdown`. Hooks run in reverse order of
registration, all hooks run even if some fail, and their errors are joined:

```go
db, err := sql.Open(driver, dsn)
if err != nil {
	return err
}
app.OnShutdown(func(ctx context.Context) error { return db.Close() })
```

Related: [Lifecycle](lifecycle.md).
