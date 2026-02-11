package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	_ "relay-control-plane/migrations"
	"relay-control-plane/routes"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		routes.RegisterTokenRoutes(se)
		routes.RegisterFileTokenRoutes(se)
		routes.RegisterInvitationRoutes(se)
		routes.RegisterRotateKeyRoutes(se)
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
