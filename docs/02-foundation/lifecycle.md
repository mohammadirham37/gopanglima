# Application lifecycle

```text
framework.New(opts...)        options applied, env + config + logger built,
                              default middleware installed,
                              WithProviders → Register() called in order
app.Register(p...)            more providers; Register() called immediately
app.Bootstrap(ctx)            deferred init errors reported, then Boot() of
                              every Booter provider, in registration order
app.Handler() / Serve / Run   routes validated and frozen, HTTP served
app.Shutdown(ctx)             shutdown hooks, last registered runs first
```

## Errors are never hidden

`framework.New` does not return an error so that the common case stays short.
Problems found while applying options — a malformed `.env` file, an invalid
`APP_DEBUG`, `LOG_LEVEL`, or `LOG_FORMAT`, a failing provider `Register` — are
recorded and returned by the first call to `Bootstrap`, `Handler`, `Serve`, or
`Run`. `Bootstrap` is idempotent and returns the same result every time.

## Serving

| Method | Use it when |
| --- | --- |
| `app.Run(addr)` | Production entry point. Handles SIGINT/SIGTERM. |
| `app.Serve(ctx, ln)` | You manage the listener and shutdown context (tests, systemd sockets). |
| `app.Handler()` | You need an `http.Handler` (httptest, custom `http.Server`). Call `app.Shutdown` yourself. |

Graceful shutdown: when the context is canceled, the server stops accepting
connections and waits up to `ShutdownTimeout` (default 10 s) for in-flight
requests. Request contexts are *not* canceled by the shutdown signal, so
running requests can finish. If the grace period expires, remaining
connections are closed and `Serve` returns an error.

Shutdown hooks always run after serving stops, and also when bootstrapping
fails, so resources opened by providers are released.

Calling `app.Shutdown(ctx)` while `Serve` is running stops the server first
(like canceling its context), waits for in-flight requests to drain, and only
then runs the hooks. `Bootstrap` and `Shutdown` are safe to call
concurrently: later callers wait for the first call and get its result.

With `Run`, the first SIGINT/SIGTERM starts a graceful shutdown; a second one
terminates the process immediately.

Server limits are configured with `framework.WithServerConfig(httpx.ServerConfig{...})`;
defaults: ReadHeaderTimeout 10 s, ReadTimeout 30 s, WriteTimeout 30 s,
IdleTimeout 120 s, MaxHeaderBytes 1 MiB, ShutdownTimeout 10 s. Zero means
"use the default"; a negative read/write/idle timeout disables it (for
example `WriteTimeout: -1` for server-sent events). `ShutdownTimeout` must not
be negative.

## Options

| Option | Purpose |
| --- | --- |
| `WithEnvFile(paths...)` | Load `.env` files (missing files are skipped). |
| `WithEnv(env)` | Use an explicit `*foundation.Env` (handy in tests). |
| `WithConfig(map)` | Set configuration values (override env-derived defaults). |
| `WithLogger(l)` | Use your own `*slog.Logger`. |
| `WithProviders(p...)` | Register service providers. |
| `WithServerConfig(cfg)` | HTTP server timeouts and limits. |
| `WithErrorHandler(h)` | Replace the default HTTP error handler. |
| `WithMaxBodyBytes(n)` | Body limit for `Context.Bind` (default 1 MiB). |
| `WithoutDefaultMiddleware()` | Skip RequestID, RequestLogger, Recover. |

Related: [Service providers](service-providers.md), [Configuration](configuration.md).
