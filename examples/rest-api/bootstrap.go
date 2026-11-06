package main

import (
	"github.com/mohammadirham37/gopanglima/examples/rest-api/app/controllers"
	"github.com/mohammadirham37/gopanglima/examples/rest-api/app/services"
	"github.com/mohammadirham37/gopanglima/examples/rest-api/routes"
	"github.com/mohammadirham37/gopanglima/framework"
	"github.com/mohammadirham37/gopanglima/framework/foundation"
)

// newApp builds the application. It is shared by main and the tests.
func newApp(opts ...framework.Option) (*framework.Application, error) {
	app := framework.New(append([]framework.Option{framework.WithEnvFile(".env")}, opts...)...)

	users := services.NewMemoryUserService()
	foundation.Bind[services.UserService](app.Container(), users)

	token := app.Env().String("API_TOKEN", "")
	if token == "" {
		app.Logger().Warn("API_TOKEN is not set; write endpoints will reject every request")
	}
	routes.API(app.Router(), controllers.NewUserController(users), token)

	if err := app.Router().Err(); err != nil {
		return nil, err
	}
	return app, nil
}
