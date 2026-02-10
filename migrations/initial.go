package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(upInitial, downInitial, "initial")
}

func upInitial(app core.App) error {
	roles, err := createRolesCollection(app)
	if err != nil {
		return err
	}
	if err := seedRoles(app, roles); err != nil {
		return err
	}
	if err := createProvidersCollection(app); err != nil {
		return err
	}
	if err := addPictureFieldToUsers(app); err != nil {
		return err
	}
	return createStorageQuotasCollection(app)
}

func addPictureFieldToUsers(app core.App) error {
	col, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	col.Fields.Add(&core.URLField{Name: "picture"})
	return app.Save(col)
}

func downInitial(app core.App) error {
	for _, name := range []string{"storage_quotas", "providers", "roles"} {
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

func createRolesCollection(app core.App) (*core.Collection, error) {
	collection := core.NewBaseCollection("roles")
	authRule := "@request.auth.id != ''"
	collection.ListRule = types.Pointer(authRule)
	collection.ViewRule = types.Pointer(authRule)
	collection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
	})
	if err := app.Save(collection); err != nil {
		return nil, err
	}
	return collection, nil
}

func seedRoles(app core.App, collection *core.Collection) error {
	owner := core.NewRecord(collection)
	owner.Set("id", "2arnubkcv7jpce8")
	owner.Set("name", "Owner")
	if err := app.Save(owner); err != nil {
		return err
	}

	member := core.NewRecord(collection)
	member.Set("id", "x6lllh2qsf9lxk6")
	member.Set("name", "Member")
	return app.Save(member)
}

func createProvidersCollection(app core.App) error {
	collection := core.NewBaseCollection("providers")

	authRule := "@request.auth.id != ''"
	collection.ListRule = types.Pointer(authRule)
	collection.ViewRule = types.Pointer(authRule)

	collection.Fields.Add(
		&core.TextField{Name: "url"},
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "self_hosted"},
		&core.TextField{Name: "public_key"},
		&core.TextField{Name: "key_type"},
		&core.TextField{Name: "key_id"},
	)

	return app.Save(collection)
}

func createStorageQuotasCollection(app core.App) error {
	collection := core.NewBaseCollection("storage_quotas")

	authRule := "@request.auth.id != ''"
	collection.ListRule = types.Pointer(authRule)
	collection.ViewRule = types.Pointer(authRule)

	collection.Fields.Add(
		&core.TextField{Name: "name"},
		&core.NumberField{Name: "quota"},
		&core.NumberField{Name: "usage"},
		&core.BoolField{Name: "metered"},
		&core.NumberField{Name: "max_file_size"},
	)

	return app.Save(collection)
}
