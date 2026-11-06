// Command hello-world is the smallest goPanglima application.
//
//	go run ./examples/hello-world
//	curl http://localhost:8080/hello/Panglima
package main

import (
	"log"
	"net/http"

	"github.com/mohammadirham37/gopanglima/framework"
)

func main() {
	app := framework.New(framework.WithEnvFile(".env"))

	app.Router().Get("/", func(c *framework.Context) error {
		return c.String(http.StatusOK, "Hello from goPanglima!")
	})
	app.Router().Get("/hello/{name}", func(c *framework.Context) error {
		return c.JSON(http.StatusOK, framework.M{"message": "Hello, " + c.Param("name") + "!"})
	})

	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
