package framework

import (
	"fmt"
	"log/slog"

	"github.com/mohammadirham37/gopanglima/framework/foundation"
	"github.com/mohammadirham37/gopanglima/framework/httpx"
)

// Option configures an Application in New.
type Option func(*settings)

type settings struct {
	env          *foundation.Env
	config       map[string]any
	logger       *slog.Logger
	providers    []Provider
	serverCfg    httpx.ServerConfig
	errorHandler httpx.ErrorHandler
	maxBodyBytes int64
	noDefaultMW  bool
	errs         []error
}

// WithEnv uses env as the application environment instead of the process
// environment.
func WithEnv(env *foundation.Env) Option {
	return func(s *settings) { s.env = env }
}

// WithEnvFile loads the given .env files (see foundation.LoadEnv). Missing
// files are skipped; malformed files cause Bootstrap to fail.
func WithEnvFile(paths ...string) Option {
	return func(s *settings) {
		env, err := foundation.LoadEnv(paths...)
		if err != nil {
			s.errs = append(s.errs, err)
			return
		}
		s.env = env
	}
}

// WithConfig sets configuration values. They are applied after the
// framework defaults derived from the environment, so they take precedence.
// Multiple WithConfig options are merged in order.
func WithConfig(values map[string]any) Option {
	return func(s *settings) {
		if s.config == nil {
			s.config = make(map[string]any, len(values))
		}
		for k, v := range values {
			s.config[k] = v
		}
	}
}

// WithLogger uses logger instead of building one from LOG_LEVEL and
// LOG_FORMAT.
func WithLogger(logger *slog.Logger) Option {
	return func(s *settings) { s.logger = logger }
}

// WithProviders registers service providers, in order, during New.
func WithProviders(providers ...Provider) Option {
	return func(s *settings) { s.providers = append(s.providers, providers...) }
}

// WithServerConfig sets HTTP server timeouts and limits. Zero fields keep
// their defaults (see httpx.DefaultServerConfig).
func WithServerConfig(cfg httpx.ServerConfig) Option {
	return func(s *settings) { s.serverCfg = cfg }
}

// WithErrorHandler replaces httpx.DefaultErrorHandler.
func WithErrorHandler(h httpx.ErrorHandler) Option {
	return func(s *settings) { s.errorHandler = h }
}

// WithMaxBodyBytes sets the request body limit used by Context.Bind.
// The default is httpx.DefaultMaxBodyBytes (1 MiB).
func WithMaxBodyBytes(n int64) Option {
	return func(s *settings) {
		if n <= 0 {
			s.errs = append(s.errs, fmt.Errorf("framework: max body bytes must be positive, got %d", n))
			return
		}
		s.maxBodyBytes = n
	}
}

// WithoutDefaultMiddleware disables the global RequestID, RequestLogger,
// and Recover middleware installed by New.
func WithoutDefaultMiddleware() Option {
	return func(s *settings) { s.noDefaultMW = true }
}
