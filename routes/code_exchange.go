package routes

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterCodeExchangeMiddleware(se *core.ServeEvent) {
	se.Router.BindFunc(func(e *core.RequestEvent) error {
		if e.Request.URL.Path != "/api/oauth2-redirect" || e.Request.Method != "GET" {
			return e.Next()
		}

		code := e.Request.URL.Query().Get("code")
		state := e.Request.URL.Query().Get("state")

		if code != "" && state != "" && len(state) >= 15 {
			col, err := e.App.FindCollectionByNameOrId("code_exchange")
			if err == nil {
				id := state[:15]

				// Try to find existing record to update, otherwise create new
				record, err := e.App.FindRecordById("code_exchange", id)
				if err != nil {
					record = core.NewRecord(col)
					record.Id = id
				}
				record.Set("code", code)

				if err := e.App.SaveNoValidate(record); err != nil {
					e.App.Logger().Error("Failed to save code_exchange record", "error", err)
				}
			}
		}

		return e.Next()
	})
}
