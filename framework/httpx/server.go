package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/mohammadirham37/gopanglima/framework/logging"
)

// ServerConfig holds net/http server limits and the graceful shutdown
// timeout. Zero values are replaced by the defaults from
// DefaultServerConfig, so a partially filled config is safe.
//
// A negative ReadHeaderTimeout, ReadTimeout, WriteTimeout, or IdleTimeout
// disables that timeout (useful for streaming responses such as
// server-sent events). ShutdownTimeout must not be negative.
type ServerConfig struct {
	// ReadHeaderTimeout bounds the time to read request headers
	// (mitigates slowloris attacks). Default 10s.
	ReadHeaderTimeout time.Duration
	// ReadTimeout bounds the time to read the entire request. Default 30s.
	ReadTimeout time.Duration
	// WriteTimeout bounds the time to write the response. Default 30s.
	WriteTimeout time.Duration
	// IdleTimeout bounds keep-alive idle time. Default 120s.
	IdleTimeout time.Duration
	// MaxHeaderBytes limits request header size. Default 1 MiB.
	MaxHeaderBytes int
	// ShutdownTimeout bounds graceful shutdown before connections are
	// forcibly closed. Default 10s.
	ShutdownTimeout time.Duration
}

// DefaultServerConfig returns conservative production defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ShutdownTimeout:   10 * time.Second,
	}
}

func (c ServerConfig) withDefaults() ServerConfig {
	d := DefaultServerConfig()
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = d.ReadHeaderTimeout
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = d.ReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = d.WriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = d.IdleTimeout
	}
	if c.MaxHeaderBytes == 0 {
		c.MaxHeaderBytes = d.MaxHeaderBytes
	}
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = d.ShutdownTimeout
	}
	return c
}

// Serve serves h on ln until ctx is canceled or the server fails.
//
// When ctx is canceled, Serve stops accepting connections and waits up to
// cfg.ShutdownTimeout for in-flight requests to finish. Request contexts
// are not canceled by ctx itself, so in-flight requests can complete during
// the grace period. Serve returns nil after a clean shutdown, and an error
// if serving failed or the grace period expired (remaining connections are
// then closed). Serve always closes ln.
func Serve(ctx context.Context, ln net.Listener, h http.Handler, cfg ServerConfig, logger *slog.Logger) error {
	if logger == nil {
		logger = logging.Discard()
	}
	cfg = cfg.withDefaults()
	if cfg.ShutdownTimeout < 0 {
		ln.Close()
		return fmt.Errorf("httpx: ShutdownTimeout must not be negative, got %v", cfg.ShutdownTimeout)
	}
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: disabledIfNegative(cfg.ReadHeaderTimeout),
		ReadTimeout:       disabledIfNegative(cfg.ReadTimeout),
		WriteTimeout:      disabledIfNegative(cfg.WriteTimeout),
		IdleTimeout:       disabledIfNegative(cfg.IdleTimeout),
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		BaseContext:       func(net.Listener) context.Context { return context.WithoutCancel(ctx) },
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpx: serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	shutdownErr := srv.Shutdown(shutdownCtx)
	serveErr := <-errCh
	if shutdownErr != nil {
		closeErr := srv.Close()
		return errors.Join(fmt.Errorf("httpx: graceful shutdown: %w", shutdownErr), closeErr)
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return fmt.Errorf("httpx: serve: %w", serveErr)
	}
	return nil
}

// disabledIfNegative maps a negative timeout to 0, which net/http treats as
// "no timeout".
func disabledIfNegative(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}
