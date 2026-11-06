package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// ErrResponseCommitted is returned by response helpers when the status line
// and headers have already been sent. A handler may write exactly one
// response; later attempts fail with this error instead of corrupting the
// output.
var ErrResponseCommitted = errors.New("httpx: response already committed")

// Error is an HTTP error with a status code and a message that is safe to
// show to clients.
//
// The optional Err field holds the internal cause. It is available to
// errors.Is/errors.As and to logs, but the default error handler never
// sends it to clients.
type Error struct {
	// Code is the HTTP status code, for example 404.
	Code int
	// Message is the public, client-safe message.
	Message string
	// Err is the internal cause. It is never serialized to clients.
	Err error
}

// NewError returns an *Error with the given status code and public message.
// An empty message defaults to the standard status text.
func NewError(code int, message string) *Error {
	if message == "" {
		message = http.StatusText(code)
	}
	return &Error{Code: code, Message: message}
}

// Error implements the error interface. The string includes the internal
// cause and is intended for logs, not for clients.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("http %d: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("http %d: %s", e.Code, e.Message)
}

// Unwrap returns the internal cause.
func (e *Error) Unwrap() error { return e.Err }

// WithCause returns a copy of e whose internal cause is err.
//
//	return httpx.NotFound("User not found").WithCause(err)
func (e *Error) WithCause(err error) *Error {
	cp := *e
	cp.Err = err
	return &cp
}

// BadRequest returns a 400 error.
func BadRequest(message string) *Error { return NewError(http.StatusBadRequest, message) }

// Unauthorized returns a 401 error.
func Unauthorized(message string) *Error { return NewError(http.StatusUnauthorized, message) }

// Forbidden returns a 403 error.
func Forbidden(message string) *Error { return NewError(http.StatusForbidden, message) }

// NotFound returns a 404 error.
func NotFound(message string) *Error { return NewError(http.StatusNotFound, message) }

// MethodNotAllowed returns a 405 error.
func MethodNotAllowed(message string) *Error {
	return NewError(http.StatusMethodNotAllowed, message)
}

// Conflict returns a 409 error.
func Conflict(message string) *Error { return NewError(http.StatusConflict, message) }

// RequestEntityTooLarge returns a 413 error.
func RequestEntityTooLarge(message string) *Error {
	return NewError(http.StatusRequestEntityTooLarge, message)
}

// UnsupportedMediaType returns a 415 error.
func UnsupportedMediaType(message string) *Error {
	return NewError(http.StatusUnsupportedMediaType, message)
}

// UnprocessableEntity returns a 422 error.
func UnprocessableEntity(message string) *Error {
	return NewError(http.StatusUnprocessableEntity, message)
}

// TooManyRequests returns a 429 error.
func TooManyRequests(message string) *Error { return NewError(http.StatusTooManyRequests, message) }

// InternalServerError returns a 500 error.
func InternalServerError(message string) *Error {
	return NewError(http.StatusInternalServerError, message)
}

// ErrorHandler converts an error returned by a handler or middleware into
// an HTTP response. It must not panic.
type ErrorHandler func(c *Context, err error)

// DefaultErrorHandler is the error handler used when none is configured.
//
// Behavior:
//   - *Error values keep their status code and public message.
//   - Any other error becomes 500 "Internal Server Error".
//   - Errors caused by the client disconnecting are logged at debug level
//     and no response is written.
//   - 5xx errors are logged at error level with their full cause; 4xx
//     errors are logged at debug level.
//   - The response is JSON: {"error":{"status":404,"message":"..."}}.
//   - In debug mode only, 5xx responses include a "detail" field holding
//     err.Error(). Never enable debug mode in production.
//   - If the response has already been committed, nothing is written.
func DefaultErrorHandler(c *Context, err error) {
	logger := c.Logger()
	if errors.Is(err, context.Canceled) && c.Request().Context().Err() != nil {
		logger.Debug("request canceled by client", slog.String("error", err.Error()))
		return
	}

	status := http.StatusInternalServerError
	message := http.StatusText(status)
	var he *Error
	if errors.As(err, &he) {
		if he.Code >= 400 && he.Code <= 599 {
			status, message = he.Code, he.Message
		} else {
			logger.Error("invalid HTTP error status; using 500", slog.Int("status", he.Code))
		}
	}

	if status >= 500 {
		logger.Error("request failed", slog.Int("status", status), slog.String("error", err.Error()))
	} else {
		logger.Debug("request rejected", slog.Int("status", status), slog.String("error", err.Error()))
	}

	if c.Response().Committed() {
		return
	}
	body := M{"status": status, "message": message}
	if status >= 500 && c.Debug() {
		body["detail"] = err.Error()
	}
	if werr := c.JSON(status, M{"error": body}); werr != nil {
		logger.Error("write error response", slog.String("error", werr.Error()))
	}
}
