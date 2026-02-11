package routes

import (
	"github.com/pocketbase/pocketbase/core"
)

const defaultQuota = 10737418240    // 10 GB
const defaultMaxFileSize = 524288000 // 500 MB

// AutoCreateRelayDeps creates the standard associated records after a relay is created:
// storage_quotas, relay_roles (Owner), relay_invitations (Member).
// It also sets the relay's storage_quota relation and saves.
func AutoCreateRelayDeps(app core.App, relay *core.Record, creatorID string) error {
	// Create storage quota
	sqCol, err := app.FindCollectionByNameOrId("storage_quotas")
	if err != nil {
		return err
	}
	sq := core.NewRecord(sqCol)
	sq.Set("name", relay.GetString("name"))
	sq.Set("quota", defaultQuota)
	sq.Set("max_file_size", defaultMaxFileSize)
	if err := app.Save(sq); err != nil {
		return err
	}

	relay.Set("storage_quota", sq.Id)
	if err := app.Save(relay); err != nil {
		return err
	}

	// Create Owner relay_role
	rrCol, err := app.FindCollectionByNameOrId("relay_roles")
	if err != nil {
		return err
	}
	rr := core.NewRecord(rrCol)
	rr.Set("user", creatorID)
	rr.Set("role", ownerRoleID)
	rr.Set("relay", relay.Id)
	if err := app.Save(rr); err != nil {
		return err
	}

	// Create Member invitation
	riCol, err := app.FindCollectionByNameOrId("relay_invitations")
	if err != nil {
		return err
	}
	invKey, err := generateRandomHex(16)
	if err != nil {
		return err
	}
	ri := core.NewRecord(riCol)
	ri.Set("role", memberRoleID)
	ri.Set("relay", relay.Id)
	ri.Set("key", invKey)
	ri.Set("enabled", true)
	return app.Save(ri)
}
