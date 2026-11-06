package foundation

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// ErrServiceNotFound is returned by Make when no service is bound for a type.
var ErrServiceNotFound = errors.New("foundation: service not bound")

// Container is a small, explicit, type-keyed service registry.
//
// Services are bound and resolved by their static Go type using the generic
// Bind and Make functions. There is no constructor injection, autowiring,
// or struct-tag reflection: what you bind is exactly what you get back.
// Binding an interface type is the recommended way to decouple consumers
// from concrete implementations.
//
// A Container is safe for concurrent use.
type Container struct {
	mu       sync.RWMutex
	services map[reflect.Type]any
}

// NewContainer returns an empty Container.
func NewContainer() *Container {
	return &Container{services: make(map[reflect.Type]any)}
}

// Bind registers value as the service for type T, replacing any previous
// binding for T.
//
//	foundation.Bind[UserRepository](c, postgresUsers)
func Bind[T any](c *Container, value T) {
	key := reflect.TypeOf((*T)(nil)).Elem()
	c.mu.Lock()
	c.services[key] = value
	c.mu.Unlock()
}

// Make returns the service bound for type T, or an error wrapping
// ErrServiceNotFound.
//
//	users, err := foundation.Make[UserRepository](c)
func Make[T any](c *Container) (T, error) {
	var zero T
	key := reflect.TypeOf((*T)(nil)).Elem()
	c.mu.RLock()
	v, ok := c.services[key]
	c.mu.RUnlock()
	if !ok {
		return zero, fmt.Errorf("%w: %s", ErrServiceNotFound, key)
	}
	t, _ := v.(T) // a nil interface binding yields the zero value
	return t, nil
}

// Bound reports whether a service is bound for type T.
func Bound[T any](c *Container) bool {
	key := reflect.TypeOf((*T)(nil)).Elem()
	c.mu.RLock()
	_, ok := c.services[key]
	c.mu.RUnlock()
	return ok
}
