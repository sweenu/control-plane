package routes

import (
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterAuthHooks(app *pocketbase.PocketBase) {
	app.OnRecordAuthRequest("users").BindFunc(func(e *core.RecordAuthRequestEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		meta, ok := e.Meta.(map[string]any)
		if !ok || meta == nil {
			return nil
		}
		rawUser, ok := meta["rawUser"].(map[string]any)
		if !ok || rawUser == nil {
			return nil
		}

		var pictureURL string
		if v, ok := rawUser["avatar_url"].(string); ok && v != "" {
			pictureURL = v // GitHub
		} else if v, ok := rawUser["picture"].(string); ok && v != "" {
			pictureURL = v // Google, OIDC
		}

		if pictureURL != "" && pictureURL != e.Record.GetString("picture") {
			e.Record.Set("picture", pictureURL)
			if err := e.App.Save(e.Record); err != nil {
				return nil // non-fatal
			}
		}

		return nil
	})
}

func RegisterHooks(app *pocketbase.PocketBase) {
	app.OnRecordCreateRequest("relays").BindFunc(onRelayCreateRequest)
	app.OnRecordCreateRequest("shared_folders").BindFunc(onSharedFolderCreateRequest)
	app.OnRecordDelete("relays").BindFunc(onRelayDelete)
	app.OnRecordDelete("shared_folders").BindFunc(onSharedFolderDelete)
}

func onRelayCreateRequest(e *core.RecordRequestEvent) error {
	if e.Record.GetString("creator") == "" && e.Auth != nil {
		e.Record.Set("creator", e.Auth.Id)
	}

	if e.Record.GetString("provider") == "" {
		if err := assignDefaultProvider(e.App, e.Record); err != nil {
			return err
		}
	}

	if err := e.Next(); err != nil {
		return err
	}

	creatorID := e.Record.GetString("creator")
	return AutoCreateRelayDeps(e.App, e.Record, creatorID)
}

func assignDefaultProvider(app core.App, record *core.Record) error {
	providerURL := os.Getenv("RELAY_DEFAULT_PROVIDER_URL")
	if providerURL == "" {
		return nil
	}

	provider, err := app.FindFirstRecordByFilter(
		"providers",
		"url = {:url}",
		dbx.Params{"url": providerURL},
	)
	if err != nil {
		provCol, err := app.FindCollectionByNameOrId("providers")
		if err != nil {
			return err
		}

		provider = core.NewRecord(provCol)
		provider.Set("url", providerURL)
		provider.Set("name", providerURL)
		provider.Set("self_hosted", false)
		provider.Set("public_key", os.Getenv("RELAY_HMAC_KEY"))
		provider.Set("key_id", getHMACKeyID())
		provider.Set("key_type", "hmac")

		if err := app.Save(provider); err != nil {
			return err
		}
	}

	record.Set("provider", provider.Id)
	return nil
}

func onSharedFolderCreateRequest(e *core.RecordRequestEvent) error {
	if e.Record.GetString("creator") == "" && e.Auth != nil {
		e.Record.Set("creator", e.Auth.Id)
	}

	if err := e.Next(); err != nil {
		return err
	}

	creatorID := e.Record.GetString("creator")
	if creatorID == "" {
		return nil
	}

	sfrCol, err := e.App.FindCollectionByNameOrId("shared_folder_roles")
	if err != nil {
		return err
	}

	sfr := core.NewRecord(sfrCol)
	sfr.Set("user", creatorID)
	sfr.Set("role", ownerRoleID)
	sfr.Set("shared_folder", e.Record.Id)
	return e.App.Save(sfr)
}

func onRelayDelete(e *core.RecordEvent) error {
	relayID := e.Record.Id

	// Cascade delete related records before the relay is deleted
	collections := []string{"relay_roles", "relay_invitations", "shared_folders", "subscriptions"}
	for _, col := range collections {
		records, err := e.App.FindRecordsByFilter(col, "relay = {:relay}", "", 0, 0, dbx.Params{"relay": relayID})
		if err != nil {
			continue
		}
		for _, r := range records {
			if err := e.App.Delete(r); err != nil {
				return err
			}
		}
	}

	// Delete associated storage_quota
	sqID := e.Record.GetString("storage_quota")
	if sqID != "" {
		sq, err := e.App.FindRecordById("storage_quotas", sqID)
		if err == nil {
			if err := e.App.Delete(sq); err != nil {
				return err
			}
		}
	}

	return e.Next()
}

func onSharedFolderDelete(e *core.RecordEvent) error {
	sfID := e.Record.Id

	roles, err := e.App.FindRecordsByFilter("shared_folder_roles", "shared_folder = {:sf}", "", 0, 0, dbx.Params{"sf": sfID})
	if err == nil {
		for _, r := range roles {
			if err := e.App.Delete(r); err != nil {
				return err
			}
		}
	}

	return e.Next()
}
