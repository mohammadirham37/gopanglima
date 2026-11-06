// Package controllers contains the example's HTTP controllers.
package controllers

import (
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/mohammadirham37/gopanglima/examples/rest-api/app/services"
	"github.com/mohammadirham37/gopanglima/framework"
)

// UserController handles /users endpoints.
type UserController struct {
	users services.UserService
}

// NewUserController returns a UserController backed by users.
func NewUserController(users services.UserService) *UserController {
	return &UserController{users: users}
}

// Index lists users.
func (uc *UserController) Index(c *framework.Context) error {
	users, err := uc.users.List(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, framework.M{"data": users})
}

// Show returns one user.
func (uc *UserController) Show(c *framework.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	user, err := uc.users.Find(c.Context(), id)
	if errors.Is(err, services.ErrUserNotFound) {
		return framework.NotFound("User not found")
	}
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, framework.M{"data": user})
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Store creates a user. Validation is done by hand here; a declarative
// validation package is planned for a later milestone.
func (uc *UserController) Store(c *framework.Context) error {
	var in createUserRequest
	if err := c.Bind(&in); err != nil {
		return err
	}
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Name) < 3 || len(in.Name) > 100 {
		return framework.UnprocessableEntity("name must be between 3 and 100 characters")
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return framework.UnprocessableEntity("email must be a valid email address")
	}
	user, err := uc.users.Create(c.Context(), in.Name, in.Email)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, framework.M{"data": user})
}

// Delete removes a user.
func (uc *UserController) Delete(c *framework.Context) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := uc.users.Delete(c.Context(), id); errors.Is(err, services.ErrUserNotFound) {
		return framework.NotFound("User not found")
	} else if err != nil {
		return err
	}
	return c.NoContent()
}

func parseID(c *framework.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, framework.NotFound("User not found")
	}
	return id, nil
}
