package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	_ "relay-control-plane/migrations"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Custom routes will be registered here
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
