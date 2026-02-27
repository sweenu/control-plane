package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(upCodeExchange, downCodeExchange, "code_exchange")
}

func upCodeExchange(app core.App) error {
	_, err := app.FindCollectionByNameOrId("code_exchange")
	if err == nil {
		return nil // already exists
	}

	col := core.NewBaseCollection("code_exchange")
	col.ViewRule = types.Pointer("")
	col.Fields.Add(
		&core.TextField{Name: "code"},
	)
	return app.Save(col)
}

func downCodeExchange(app core.App) error {
	col, err := app.FindCollectionByNameOrId("code_exchange")
	if err != nil {
		return nil
	}
	return app.Delete(col)
}
