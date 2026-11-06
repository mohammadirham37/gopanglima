// Package routes registers the example's HTTP routes.
package routes

import (
	"crypto/subtle"

	"github.com/mohammadirham37/gopanglima/examples/rest-api/app/controllers"
	"github.com/mohammadirham37/gopanglima/framework"
)

// API registers the /api/v1 routes. Write routes require the
// "Authorization: Bearer <token>" header to match apiToken.
func API(r *framework.Router, users *controllers.UserController, apiToken string) {
	r.Prefix("/api/v1").Group(func(r *framework.Router) {
		r.Get("/users", users.Index).Name("users.index")
		r.Get("/users/{id}", users.Show).Name("users.show")

		r.Middleware(RequireToken(apiToken)).Group(func(r *framework.Router) {
			r.Post("/users", users.Store).Name("users.store")
			r.Delete("/users/{id}", users.Delete).Name("users.destroy")
		})
	})
}

// RequireToken is a minimal bearer-token middleware for the example. It
// compares tokens in constant time and rejects every request when the
// configured token is empty.
func RequireToken(token string) framework.Middleware {
	return func(next framework.HandlerFunc) framework.HandlerFunc {
		return func(c *framework.Context) error {
			const prefix = "Bearer "
			h := c.Header("Authorization")
			if token == "" || len(h) <= len(prefix) || h[:len(prefix)] != prefix ||
				subtle.ConstantTimeCompare([]byte(h[len(prefix):]), []byte(token)) != 1 {
				return framework.Unauthorized("Missing or invalid API token")
			}
			return next(c)
		}
	}
}
