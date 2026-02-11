package routes

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterInvitationRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/accept-invitation", handleAcceptInvitation).Bind(apis.RequireAuth())
}

func handleAcceptInvitation(e *core.RequestEvent) error {
	var body struct {
		Key string `json:"key"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.BadRequestError("Invalid request body", nil)
	}

	invitation, err := e.App.FindFirstRecordByFilter(
		"relay_invitations",
		"key = {:key} && enabled = true",
		dbx.Params{"key": body.Key},
	)
	if err != nil {
		return e.NotFoundError("Invitation not found", nil)
	}

	relayID := invitation.GetString("relay")
	userID := e.Auth.Id

	// Check for duplicate membership
	_, err = e.App.FindFirstRecordByFilter(
		"relay_roles",
		"user = {:user} && relay = {:relay}",
		dbx.Params{"user": userID, "relay": relayID},
	)
	if err == nil {
		return e.Error(409, "Already a member of this relay", nil)
	}

	// Create relay_role
	relayRolesCol, err := e.App.FindCollectionByNameOrId("relay_roles")
	if err != nil {
		return e.InternalServerError("Collection not found", nil)
	}

	role := core.NewRecord(relayRolesCol)
	role.Set("user", userID)
	role.Set("role", invitation.GetString("role"))
	role.Set("relay", relayID)
	if err := e.App.Save(role); err != nil {
		return e.InternalServerError("Failed to create role", nil)
	}

	// Return relay with expands
	relay, err := e.App.FindRecordById("relays", relayID)
	if err != nil {
		return e.InternalServerError("Failed to load relay", nil)
	}

	apis.EnrichRecord(e, relay, "relay_roles_via_relay", "relay_invitations_via_relay", "storage_quota")

	return e.JSON(200, relay)
}
