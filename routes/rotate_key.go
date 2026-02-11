package routes

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterRotateKeyRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/rotate-key", handleRotateKey).Bind(apis.RequireAuth())
}

func handleRotateKey(e *core.RequestEvent) error {
	var body struct {
		ID string `json:"id"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.BadRequestError("Invalid request body", nil)
	}

	invitation, err := e.App.FindRecordById("relay_invitations", body.ID)
	if err != nil {
		return e.NotFoundError("Invitation not found", nil)
	}

	relayID := invitation.GetString("relay")
	_, err = e.App.FindFirstRecordByFilter(
		"relay_roles",
		"user = {:user} && relay = {:relay} && role = {:role}",
		dbx.Params{"user": e.Auth.Id, "relay": relayID, "role": ownerRoleID},
	)
	if err != nil {
		return e.ForbiddenError("Only relay owners can rotate keys", nil)
	}

	newKey, err := generateRandomHex(16)
	if err != nil {
		return e.InternalServerError("Failed to generate key", nil)
	}

	invitation.Set("key", newKey)
	if err := e.App.Save(invitation); err != nil {
		return e.InternalServerError("Failed to update invitation", nil)
	}

	return e.JSON(200, invitation)
}

func generateRandomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
