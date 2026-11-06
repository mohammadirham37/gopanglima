// Command rest-api is a small JSON API built with goPanglima: controllers,
// a service bound in the container, route groups, prefixes, named routes,
// and route-group middleware.
//
//	API_TOKEN=dev-token go run ./examples/rest-api
//	curl -X POST localhost:8080/api/v1/users \
//	  -H 'Authorization: Bearer dev-token' -H 'Content-Type: application/json' \
//	  -d '{"name":"Ana","email":"ana@example.com"}'
//	curl localhost:8080/api/v1/users
package main

import "log"

func main() {
	app, err := newApp()
	if err != nil {
		log.Fatal(err)
	}
	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
