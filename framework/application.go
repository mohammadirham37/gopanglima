// Package framework is the main entry point of goPanglima.
//
// An Application ties together configuration, logging, the service
// container, service providers, and the HTTP router:
//
//	app := framework.New(framework.WithEnvFile(".env"))
//	app.Router().Get("/", func(c *framework.Context) error {
//		return c.String(200, "Hello from goPanglima")
//	})
//	if err := app.Run(":8080"); err != nil {
//		log.Fatal(err)
//	}
//
// Lifecycle: New → Register providers → Bootstrap (boot providers) →
// Serve/Run → Shutdown (shutdown hooks in reverse order).
package framework

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mohammadirham37/gopanglima/framework/foundation"
	"github.com/mohammadirham37/gopanglima/framework/httpx"
	"github.com/mohammadirham37/gopanglima/framework/httpx/middleware"
	"github.com/mohammadirham37/gopanglima/framework/logging"
)

// Version is the goPanglima framework version.
const Version = "0.1.0-dev"

type state int

const (
	stateCreated state = iota
	stateBooted
	stateShutdown
)

// Application is a goPanglima application instance.
//
// Create one with New. All methods are safe for concurrent use. Concurrent
// Bootstrap and Shutdown calls wait for the first call to finish and then
// return its result.
type Application struct {
	env       *foundation.Env
	config    *foundation.Config
	logger    *slog.Logger
	container *foundation.Container
	router    *httpx.Router

	serverCfg    httpx.ServerConfig
	errorHandler httpx.ErrorHandler
	maxBodyBytes int64

	mu            sync.Mutex
	state         state
	initErrs      []error
	providers     []Provider
	shutdownHooks []func(context.Context) error

	bootDone chan struct{} // closed when the first Bootstrap finishes
	bootErr  error

	hooksDone   chan struct{} // closed when shutdown hooks have run
	shutdownErr error

	serving   bool
	stopServe context.CancelFunc // cancels the running Serve
	serveDone chan struct{}      // closed when Serve has fully returned
}

// New creates an Application.
//
// New never fails. Problems detected while applying options (for example a
// malformed .env file or an invalid LOG_LEVEL) are recorded and returned by
// Bootstrap, Handler, Serve, and Run, so initialization errors are never
// silently ignored.
//
// Unless WithoutDefaultMiddleware is given, New installs the RequestID,
// RequestLogger, and Recover middleware globally.
func New(opts ...Option) *Application {
	s := &settings{}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	app := &Application{
		container:    foundation.NewContainer(),
		router:       httpx.NewRouter(),
		serverCfg:    s.serverCfg,
		errorHandler: s.errorHandler,
		maxBodyBytes: s.maxBodyBytes,
	}
	app.initErrs = append(app.initErrs, s.errs...)

	app.env = s.env
	if app.env == nil {
		app.env = foundation.SystemEnv()
	}

	app.config = foundation.NewConfig()
	if err := app.loadDefaultConfig(); err != nil {
		app.initErrs = append(app.initErrs, err)
	}
	app.config.SetMany(s.config)

	app.logger = s.logger
	if app.logger == nil {
		l, err := logging.New(os.Stderr, logging.Options{
			Level:  app.config.String("log.level", "info"),
			Format: logging.Format(app.config.String("log.format", "text")),
		})
		if err != nil {
			app.initErrs = append(app.initErrs, err)
			l = slog.New(slog.NewTextHandler(os.Stderr, nil))
		}
		app.logger = l
	}

	if !s.noDefaultMW {
		app.router.Use(middleware.RequestID(), middleware.RequestLogger(), middleware.Recover())
	}

	for _, p := range s.providers {
		if err := app.Register(p); err != nil {
			app.initErrs = append(app.initErrs, err)
		}
	}
	return app
}

// loadDefaultConfig populates framework configuration keys from the
// environment:
//
//	app.name   APP_NAME   (default "goPanglima")
//	app.env    APP_ENV    (default "production")
//	app.debug  APP_DEBUG  (default false)
//	log.level  LOG_LEVEL  (default "info")
//	log.format LOG_FORMAT (default "text")
func (a *Application) loadDefaultConfig() error {
	debug, err := a.env.Bool("APP_DEBUG", false)
	a.config.SetMany(map[string]any{
		"app.name":   a.env.String("APP_NAME", "goPanglima"),
		"app.env":    a.env.String("APP_ENV", "production"),
		"app.debug":  debug,
		"log.level":  a.env.String("LOG_LEVEL", "info"),
		"log.format": a.env.String("LOG_FORMAT", "text"),
	})
	return err
}

// Config returns the configuration repository.
func (a *Application) Config() *foundation.Config { return a.config }

// Env returns the environment the application was created with.
func (a *Application) Env() *foundation.Env { return a.env }

// Logger returns the application logger.
func (a *Application) Logger() *slog.Logger { return a.logger }

// Container returns the service container.
func (a *Application) Container() *foundation.Container { return a.container }

// Router returns the HTTP router. Register routes before calling Handler,
// Serve, or Run; the router is frozen once the HTTP handler is built.
func (a *Application) Router() *Router { return a.router }

// Name returns the "app.name" configuration value.
func (a *Application) Name() string { return a.config.String("app.name", "goPanglima") }

// Environment returns the "app.env" configuration value, for example
// "production", "local", or "testing".
func (a *Application) Environment() string { return a.config.String("app.env", "production") }

// IsProduction reports whether the application runs in the "production"
// environment.
func (a *Application) IsProduction() bool { return a.Environment() == "production" }

// Debug reports whether debug mode ("app.debug") is enabled.
func (a *Application) Debug() bool { return a.config.Bool("app.debug", false) }

// Register registers service providers in order, calling each provider's
// Register method immediately. It stops at the first error.
//
// Providers must be registered before Bootstrap; afterwards Register
// returns ErrAlreadyBooted.
func (a *Application) Register(providers ...Provider) error {
	for _, p := range providers {
		if p == nil {
			return errors.New("framework: nil provider")
		}
		a.mu.Lock()
		if a.state != stateCreated {
			a.mu.Unlock()
			return ErrAlreadyBooted
		}
		a.providers = append(a.providers, p)
		a.mu.Unlock()

		if err := p.Register(a); err != nil {
			return fmt.Errorf("framework: register provider %T: %w", p, err)
		}
	}
	return nil
}

// Bootstrap finishes initialization: it reports deferred initialization
// errors and then boots every provider implementing Booter, in
// registration order. It stops at the first error.
//
// Bootstrap is idempotent: later and concurrent calls wait for the first
// call to finish and return its result (or ctx.Err() if ctx ends first).
// Serve and Run call it automatically. After Shutdown, Bootstrap returns
// ErrShutdown.
func (a *Application) Bootstrap(ctx context.Context) error {
	a.mu.Lock()
	if a.state == stateShutdown {
		err := a.bootErr
		a.mu.Unlock()
		if err == nil {
			err = ErrShutdown
		}
		return err
	}
	if a.bootDone != nil {
		done := a.bootDone
		a.mu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.bootErr
	}
	a.bootDone = make(chan struct{})
	a.state = stateBooted
	providers := append([]Provider(nil), a.providers...)
	initErr := errors.Join(a.initErrs...)
	a.mu.Unlock()

	err := initErr
	if err != nil {
		err = fmt.Errorf("framework: initialization failed: %w", err)
	}
	for _, p := range providers {
		if err != nil {
			break
		}
		if b, ok := p.(Booter); ok {
			if cerr := ctx.Err(); cerr != nil {
				err = cerr
				break
			}
			if berr := b.Boot(ctx, a); berr != nil {
				err = fmt.Errorf("framework: boot provider %T: %w", p, berr)
			}
		}
	}

	a.mu.Lock()
	a.bootErr = err
	a.mu.Unlock()
	close(a.bootDone)
	return err
}

// OnShutdown registers a hook that runs during Shutdown. Hooks run in
// reverse registration order (last registered, first run), mirroring the
// order in which resources are usually acquired.
func (a *Application) OnShutdown(hook func(ctx context.Context) error) {
	if hook == nil {
		return
	}
	a.mu.Lock()
	a.shutdownHooks = append(a.shutdownHooks, hook)
	a.mu.Unlock()
}

// Shutdown stops the application and returns the joined errors of the
// shutdown hooks.
//
// If Serve is running, Shutdown first stops it (exactly as if its context
// had been canceled), waits for in-flight requests to drain, and only then
// are the hooks run, so resources are never released while requests still
// use them. Otherwise Shutdown runs the hooks directly.
//
// Hooks run once, in reverse registration order, even if some fail. Later
// and concurrent calls wait for the first run to finish and return its
// result, or return ctx.Err() if ctx ends first.
//
// Applications that serve Handler with their own server should call
// Shutdown after that server has stopped.
func (a *Application) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	if a.serving {
		stop, done := a.stopServe, a.serveDone
		a.mu.Unlock()
		stop()
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.shutdownErr
	}
	a.mu.Unlock()
	return a.runShutdownHooks(ctx)
}

func (a *Application) runShutdownHooks(ctx context.Context) error {
	a.mu.Lock()
	if a.hooksDone != nil {
		done := a.hooksDone
		a.mu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.shutdownErr
	}
	a.hooksDone = make(chan struct{})
	a.state = stateShutdown
	hooks := append([]func(context.Context) error(nil), a.shutdownHooks...)
	a.mu.Unlock()

	var errs []error
	for i := len(hooks) - 1; i >= 0; i-- {
		if err := hooks[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	err := errors.Join(errs...)

	a.mu.Lock()
	a.shutdownErr = err
	a.mu.Unlock()
	close(a.hooksDone)
	return err
}

// Handler bootstraps the application and builds its HTTP handler. It is
// useful for tests with net/http/httptest and for serving with a custom
// http.Server. After Handler succeeds the router is frozen.
func (a *Application) Handler() (http.Handler, error) {
	if err := a.Bootstrap(context.Background()); err != nil {
		return nil, err
	}
	return a.buildHandler()
}

func (a *Application) buildHandler() (http.Handler, error) {
	h, err := a.router.Build(httpx.HandlerOptions{
		Logger:       a.logger,
		Debug:        a.Debug(),
		MaxBodyBytes: a.maxBodyBytes,
		ErrorHandler: a.errorHandler,
	})
	if err != nil {
		return nil, fmt.Errorf("framework: build routes: %w", err)
	}
	return h, nil
}

// Serve bootstraps the application and serves HTTP on ln until ctx is
// canceled or Shutdown is called, then shuts down gracefully and runs the
// shutdown hooks. It returns nil after a clean shutdown. If bootstrapping
// or building the routes fails, the shutdown hooks still run so that
// resources acquired by providers are released. Serve always closes ln.
func (a *Application) Serve(ctx context.Context, ln net.Listener) error {
	if err := a.Bootstrap(ctx); err != nil {
		ln.Close()
		return errors.Join(err, a.shutdownWithTimeout())
	}

	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	a.mu.Lock()
	if a.serving || a.state == stateShutdown {
		a.mu.Unlock()
		ln.Close()
		if a.serving {
			return ErrAlreadyServing
		}
		return ErrShutdown
	}
	a.serving = true
	a.stopServe = cancel
	a.serveDone = make(chan struct{})
	done := a.serveDone
	a.mu.Unlock()
	defer close(done)

	h, err := a.buildHandler()
	if err != nil {
		ln.Close()
		return errors.Join(err, a.shutdownWithTimeout())
	}

	a.logger.Info("server started",
		slog.String("app", a.Name()),
		slog.String("env", a.Environment()),
		slog.String("addr", ln.Addr().String()))
	serveErr := httpx.Serve(serveCtx, ln, h, a.serverCfg, a.logger)
	a.logger.Info("server stopped")
	return errors.Join(serveErr, a.shutdownWithTimeout())
}

func (a *Application) shutdownWithTimeout() error {
	timeout := a.serverCfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = httpx.DefaultServerConfig().ShutdownTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return a.runShutdownHooks(ctx)
}

// Run listens on addr (for example ":8080") and serves until the process
// receives SIGINT or SIGTERM, then shuts down gracefully. After the first
// signal, default signal handling is restored, so a second Ctrl+C
// terminates the process immediately.
func (a *Application) Run(addr string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := a.Bootstrap(ctx); err != nil {
		return errors.Join(err, a.shutdownWithTimeout())
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Join(fmt.Errorf("framework: listen on %s: %w", addr, err), a.shutdownWithTimeout())
	}
	return a.Serve(ctx, ln)
}
