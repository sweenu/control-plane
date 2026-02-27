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

	routes.RegisterHooks(app)
	routes.RegisterAuthHooks(app)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		routes.RegisterCodeExchangeMiddleware(se)
		routes.RegisterTokenRoutes(se)
		routes.RegisterFileTokenRoutes(se)
		routes.RegisterInvitationRoutes(se)
		routes.RegisterRotateKeyRoutes(se)
		routes.RegisterSelfHostRoutes(se)
		routes.RegisterTemplateRoutes(se)
		routes.RegisterUtilityRoutes(se)
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
