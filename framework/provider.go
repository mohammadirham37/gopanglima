package framework

import "context"

// Provider is a service provider: the place where an application or
// package wires its services into the Application.
//
// Register is called as soon as the provider is registered. It should only
// bind services and read configuration; it must not depend on services
// from providers registered later.
//
//	type DatabaseProvider struct{}
//
//	func (DatabaseProvider) Register(app *framework.Application) error {
//		db, err := sql.Open(...)
//		if err != nil {
//			return err
//		}
//		foundation.Bind(app.Container(), db)
//		app.OnShutdown(func(context.Context) error { return db.Close() })
//		return nil
//	}
type Provider interface {
	Register(app *Application) error
}

// Booter is an optional interface for providers that need a second phase
// after every provider has been registered, for example to register routes
// that depend on services bound by other providers.
//
// Boot methods run during Application.Bootstrap in provider registration
// order, so the order is deterministic.
type Booter interface {
	Boot(ctx context.Context, app *Application) error
}

// ProviderFunc adapts a function to the Provider interface.
type ProviderFunc func(app *Application) error

// Register calls f(app).
func (f ProviderFunc) Register(app *Application) error { return f(app) }
