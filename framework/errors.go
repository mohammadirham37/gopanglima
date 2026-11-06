package framework

import (
	"errors"

	"github.com/mohammadirham37/gopanglima/framework/httpx"
)

var (
	// ErrAlreadyBooted is returned when providers are registered after
	// Bootstrap has run.
	ErrAlreadyBooted = errors.New("framework: application already booted")
	// ErrShutdown is returned when bootstrapping an application that has
	// already been shut down.
	ErrShutdown = errors.New("framework: application has been shut down")
	// ErrAlreadyServing is returned when Serve or Run is called while the
	// application is already serving.
	ErrAlreadyServing = errors.New("framework: application is already serving")
)

// HTTPError is an HTTP error with a status code and a client-safe message.
// It is an alias of httpx.Error.
type HTTPError = httpx.Error

// NewHTTPError returns an *HTTPError with the given status and public
// message. An empty message defaults to the standard status text.
func NewHTTPError(code int, message string) *HTTPError { return httpx.NewError(code, message) }

// BadRequest returns a 400 error.
func BadRequest(message string) *HTTPError { return httpx.BadRequest(message) }

// Unauthorized returns a 401 error.
func Unauthorized(message string) *HTTPError { return httpx.Unauthorized(message) }

// Forbidden returns a 403 error.
func Forbidden(message string) *HTTPError { return httpx.Forbidden(message) }

// NotFound returns a 404 error.
//
//	return framework.NotFound("User not found")
func NotFound(message string) *HTTPError { return httpx.NotFound(message) }

// Conflict returns a 409 error.
func Conflict(message string) *HTTPError { return httpx.Conflict(message) }

// UnprocessableEntity returns a 422 error.
func UnprocessableEntity(message string) *HTTPError { return httpx.UnprocessableEntity(message) }

// TooManyRequests returns a 429 error.
func TooManyRequests(message string) *HTTPError { return httpx.TooManyRequests(message) }

// InternalServerError returns a 500 error.
func InternalServerError(message string) *HTTPError { return httpx.InternalServerError(message) }
