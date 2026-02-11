package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(upRelays, downRelays, "relays")
}

func upRelays(app core.App) error {
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	rolesCol, err := app.FindCollectionByNameOrId("roles")
	if err != nil {
		return err
	}
	providersCol, err := app.FindCollectionByNameOrId("providers")
	if err != nil {
		return err
	}
	storageQuotasCol, err := app.FindCollectionByNameOrId("storage_quotas")
	if err != nil {
		return err
	}

	relaysCol, err := createRelaysCollection(app, usersCol.Id, providersCol.Id, storageQuotasCol.Id)
	if err != nil {
		return err
	}

	sharedFoldersCol, err := createSharedFoldersCollection(app, relaysCol.Id, usersCol.Id)
	if err != nil {
		return err
	}

	if err := createRelayRolesCollection(app, usersCol.Id, rolesCol.Id, relaysCol.Id); err != nil {
		return err
	}
	if err := setRelaysCollectionRules(app); err != nil {
		return err
	}
	if err := setSharedFoldersCollectionRules(app); err != nil {
		return err
	}
	if err := createSharedFolderRolesCollection(app, usersCol.Id, rolesCol.Id, sharedFoldersCol.Id); err != nil {
		return err
	}
	if err := createRelayInvitationsCollection(app, rolesCol.Id, relaysCol.Id); err != nil {
		return err
	}
	if err := createSubscriptionsCollection(app, usersCol.Id, relaysCol.Id); err != nil {
		return err
	}
	if err := createOAuth2ResponseCollection(app, usersCol.Id); err != nil {
		return err
	}
	return createCodeExchangeCollection(app)
}

func downRelays(app core.App) error {
	collections := []string{
		"code_exchange",
		"oauth2_response",
		"subscriptions",
		"relay_invitations",
		"shared_folder_roles",
		"relay_roles",
		"shared_folders",
		"relays",
	}
	for _, name := range collections {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}
		if err := app.Delete(col); err != nil {
			return err
		}
	}
	return nil
}

func createRelaysCollection(app core.App, usersId, providersId, storageQuotasId string) (*core.Collection, error) {
	if existing, err := app.FindCollectionByNameOrId("relays"); err == nil {
		return existing, nil
	}

	col := core.NewBaseCollection("relays")

	col.CreateRule = types.Pointer("@request.auth.id != ''")

	col.Fields.Add(
		&core.TextField{Name: "guid", Required: true},
		&core.TextField{Name: "name"},
		&core.NumberField{Name: "version"},
		&core.TextField{Name: "path"},
		&core.NumberField{Name: "user_limit"},
		&core.RelationField{Name: "creator", CollectionId: usersId, MaxSelect: 1},
		&core.TextField{Name: "cta"},
		&core.TextField{Name: "plan"},
		&core.RelationField{Name: "provider", CollectionId: providersId, MaxSelect: 1},
		&core.RelationField{Name: "storage_quota", CollectionId: storageQuotasId, MaxSelect: 1},
	)

	if err := app.Save(col); err != nil {
		return nil, err
	}
	return col, nil
}

func setRelaysCollectionRules(app core.App) error {
	col, err := app.FindCollectionByNameOrId("relays")
	if err != nil {
		return err
	}

	relayUserRule := "@request.auth.id != '' && relay_roles_via_relay.user ?= @request.auth.id"
	col.ListRule = types.Pointer(relayUserRule)
	col.ViewRule = types.Pointer(relayUserRule)
	col.UpdateRule = types.Pointer(relayUserRule)
	col.DeleteRule = types.Pointer(relayUserRule)

	return app.Save(col)
}

func setSharedFoldersCollectionRules(app core.App) error {
	col, err := app.FindCollectionByNameOrId("shared_folders")
	if err != nil {
		return err
	}

	relayUserRule := "@request.auth.id != '' && relay.relay_roles_via_relay.user ?= @request.auth.id"
	col.ListRule = types.Pointer(relayUserRule)
	col.ViewRule = types.Pointer(relayUserRule)
	col.CreateRule = types.Pointer(relayUserRule)
	col.UpdateRule = types.Pointer(relayUserRule)
	col.DeleteRule = types.Pointer(relayUserRule)

	return app.Save(col)
}

func createSharedFoldersCollection(app core.App, relaysId, usersId string) (*core.Collection, error) {
	if existing, err := app.FindCollectionByNameOrId("shared_folders"); err == nil {
		return existing, nil
	}

	col := core.NewBaseCollection("shared_folders")

	col.Fields.Add(
		&core.TextField{Name: "guid", Required: true},
		&core.TextField{Name: "name", Required: true},
		&core.RelationField{Name: "relay", CollectionId: relaysId, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "creator", CollectionId: usersId, MaxSelect: 1, Required: true},
		&core.BoolField{Name: "private"},
	)

	if err := app.Save(col); err != nil {
		return nil, err
	}
	return col, nil
}

func createRelayRolesCollection(app core.App, usersId, rolesId, relaysId string) error {
	if _, err := app.FindCollectionByNameOrId("relay_roles"); err == nil {
		return nil
	}

	col := core.NewBaseCollection("relay_roles")

	relayUserRule := "@request.auth.id != '' && relay.relay_roles_via_relay.user ?= @request.auth.id"
	col.ListRule = types.Pointer(relayUserRule)
	col.ViewRule = types.Pointer(relayUserRule)

	col.Fields.Add(
		&core.RelationField{Name: "user", CollectionId: usersId, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "role", CollectionId: rolesId, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "relay", CollectionId: relaysId, MaxSelect: 1, Required: true},
	)

	return app.Save(col)
}

func createSharedFolderRolesCollection(app core.App, usersId, rolesId, sharedFoldersId string) error {
	if _, err := app.FindCollectionByNameOrId("shared_folder_roles"); err == nil {
		return nil
	}

	col := core.NewBaseCollection("shared_folder_roles")

	rule := "@request.auth.id != '' && shared_folder.relay.relay_roles_via_relay.user ?= @request.auth.id"
	col.ListRule = types.Pointer(rule)
	col.ViewRule = types.Pointer(rule)

	col.Fields.Add(
		&core.RelationField{Name: "user", CollectionId: usersId, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "role", CollectionId: rolesId, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "shared_folder", CollectionId: sharedFoldersId, MaxSelect: 1, Required: true},
	)

	return app.Save(col)
}

func createRelayInvitationsCollection(app core.App, rolesId, relaysId string) error {
	if _, err := app.FindCollectionByNameOrId("relay_invitations"); err == nil {
		return nil
	}

	col := core.NewBaseCollection("relay_invitations")

	authRule := "@request.auth.id != ''"
	ownerRule := "@request.auth.id != '' && relay.relay_roles_via_relay.user ?= @request.auth.id && relay.relay_roles_via_relay.role ?= '2arnubkcv7jpce8'"
	col.ListRule = types.Pointer(authRule)
	col.ViewRule = types.Pointer(authRule)
	col.CreateRule = types.Pointer(ownerRule)
	col.UpdateRule = types.Pointer(ownerRule)
	col.DeleteRule = types.Pointer(ownerRule)

	col.Fields.Add(
		&core.RelationField{Name: "role", CollectionId: rolesId, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "relay", CollectionId: relaysId, MaxSelect: 1, Required: true},
		&core.TextField{Name: "key", Required: true},
		&core.BoolField{Name: "enabled"},
	)

	return app.Save(col)
}

func createSubscriptionsCollection(app core.App, usersId, relaysId string) error {
	if _, err := app.FindCollectionByNameOrId("subscriptions"); err == nil {
		return nil
	}

	col := core.NewBaseCollection("subscriptions")

	authRule := "@request.auth.id != ''"
	col.ListRule = types.Pointer(authRule)
	col.ViewRule = types.Pointer(authRule)

	col.Fields.Add(
		&core.BoolField{Name: "active"},
		&core.RelationField{Name: "user", CollectionId: usersId, MaxSelect: 1},
		&core.RelationField{Name: "relay", CollectionId: relaysId, MaxSelect: 1},
		&core.NumberField{Name: "stripe_cancel_at"},
		&core.NumberField{Name: "stripe_quantity"},
		&core.TextField{Name: "token"},
	)

	return app.Save(col)
}

func createOAuth2ResponseCollection(app core.App, usersId string) error {
	if _, err := app.FindCollectionByNameOrId("oauth2_response"); err == nil {
		return nil
	}

	col := core.NewBaseCollection("oauth2_response")

	col.CreateRule = types.Pointer("@request.auth.id != ''")

	col.Fields.Add(
		&core.RelationField{Name: "user", CollectionId: usersId, MaxSelect: 1},
		&core.JSONField{Name: "oauth_response"},
	)

	return app.Save(col)
}

func createCodeExchangeCollection(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("code_exchange"); err == nil {
		return nil
	}

	col := core.NewBaseCollection("code_exchange")

	col.ViewRule = types.Pointer("")

	col.Fields.Add(
		&core.TextField{Name: "code"},
	)

	return app.Save(col)
}
