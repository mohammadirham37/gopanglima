// Package middleware provides goPanglima's built-in HTTP middleware:
// panic recovery, request IDs, and access logging.
//
// framework.New installs RequestID, RequestLogger, and Recover (in that
// order) as global middleware unless framework.WithoutDefaultMiddleware is
// used.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/mohammadirham37/gopanglima/framework/httpx"
)

// Recover converts panics in downstream handlers into errors, which the
// error handler renders as 500 responses. The panic value and stack trace
// are logged; they are never sent to clients.
//
// http.ErrAbortHandler is re-panicked so that net/http can abort the
// response as intended.
func Recover() httpx.Middleware {
	return func(next httpx.HandlerFunc) httpx.HandlerFunc {
		return func(c *httpx.Context) (err error) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				c.Logger().Error("panic recovered",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())))
				err = httpx.InternalServerError("").WithCause(fmt.Errorf("panic: %v", rec))
			}()
			return next(c)
		}
	}
}

// RequestIDHeader is the header used to read and return request IDs.
const RequestIDHeader = "X-Request-ID"

type requestIDKey struct{}

// RequestID assigns every request an ID.
//
// A client-supplied X-Request-ID header is reused when it is 1-128
// characters drawn from [A-Za-z0-9-_.:]; otherwise a random 128-bit hex ID
// is generated. The ID is echoed in the response header, stored in the
// request context (see RequestIDFrom), and added to the request logger as
// the "request_id" attribute.
func RequestID() httpx.Middleware {
	return func(next httpx.HandlerFunc) httpx.HandlerFunc {
		return func(c *httpx.Context) error {
			id := c.Header(RequestIDHeader)
			if !validRequestID(id) {
				var err error
				if id, err = newRequestID(); err != nil {
					return fmt.Errorf("middleware: generate request id: %w", err)
				}
			}
			c.SetHeader(RequestIDHeader, id)
			c.SetRequest(c.Request().WithContext(context.WithValue(c.Context(), requestIDKey{}, id)))
			c.SetLogger(c.Logger().With(slog.String("request_id", id)))
			return next(c)
		}
	}
}

// RequestIDFrom returns the request ID stored by RequestID, or "".
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-', c == '_', c == '.', c == ':':
		default:
			return false
		}
	}
	return true
}

func newRequestID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// RequestLogger logs one record per request with the method, path, route
// pattern, status, response size, and duration.
//
// To log the final status, RequestLogger hands errors from downstream
// handlers to the error handler itself (via Context.HandleError) and then
// returns nil. Middleware placed before RequestLogger therefore never sees
// handler errors. The query string is not logged because it may contain
// secrets.
func RequestLogger() httpx.Middleware {
	return func(next httpx.HandlerFunc) httpx.HandlerFunc {
		return func(c *httpx.Context) error {
			start := time.Now()
			if err := next(c); err != nil {
				c.HandleError(err)
			}
			status := c.Response().Status()
			if status == 0 {
				status = http.StatusOK
			}
			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			} else if status >= 400 {
				level = slog.LevelWarn
			}
			c.Logger().LogAttrs(c.Context(), level, "http request",
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.String("route", c.RoutePattern()),
				slog.Int("status", status),
				slog.Int64("bytes", c.Response().Size()),
				slog.Duration("duration", time.Since(start)),
			)
			return nil
		}
	}
}
