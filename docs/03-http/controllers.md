# Controllers

A handler has the signature `func(*framework.Context) error`. A controller is
a plain struct whose methods have that signature; dependencies are passed to
its constructor explicitly. There is no reflection-based discovery: routes are
registered explicitly, which keeps them easy to find and debug.

```go
type UserController struct {
	users services.UserService
}

func NewUserController(users services.UserService) *UserController {
	return &UserController{users: users}
}

func (uc *UserController) Show(c *framework.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return framework.NotFound("User not found")
	}
	user, err := uc.users.Find(c.Context(), id)
	if errors.Is(err, services.ErrUserNotFound) {
		return framework.NotFound("User not found")
	}
	if err != nil {
		return err // logged; client sees a generic 500
	}
	return c.JSON(http.StatusOK, framework.M{"data": user})
}
```

```go
users := NewUserController(userService)
r.Get("/users/{id}", users.Show).Name("users.show")
```

Always pass `c.Context()` to blocking calls (database, HTTP clients) so work
stops when the client disconnects.

A complete example with tests lives in [`examples/rest-api`](../../examples/rest-api).

Related: [Routing](routing.md), [Requests and responses](requests-responses.md).
