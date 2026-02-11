package routes

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"relay-control-plane/cwt"
)

const ownerRoleID = "2arnubkcv7jpce8"
const memberRoleID = "x6lllh2qsf9lxk6"

type relayAuth struct {
	Relay         *core.Record
	Provider      *core.Record
	Authorization string
	ProviderURL   string
}

// resolveRelayAuth loads the relay, verifies user access, and determines authorization level.
// relayID can be either a PocketBase record ID or a relay guid.
func resolveRelayAuth(e *core.RequestEvent, relayID string) (*relayAuth, error) {
	relay, err := e.App.FindRecordById("relays", relayID)
	if err != nil {
		// Fall back to lookup by guid
		relay, err = e.App.FindFirstRecordByFilter(
			"relays",
			"guid = {:guid}",
			dbx.Params{"guid": relayID},
		)
		if err != nil {
			return nil, e.NotFoundError("Relay not found", nil)
		}
	}

	userID := e.Auth.Id
	relayRole, err := e.App.FindFirstRecordByFilter(
		"relay_roles",
		"user = {:user} && relay = {:relay}",
		dbx.Params{"user": userID, "relay": relay.Id},
	)
	if err != nil {
		return nil, e.ForbiddenError("No access to this relay", nil)
	}

	roleID := relayRole.GetString("role")
	authorization := "read-only"
	if roleID == ownerRoleID || roleID == memberRoleID {
		authorization = "full"
	}

	providerID := relay.GetString("provider")
	provider, err := e.App.FindRecordById("providers", providerID)
	if err != nil {
		return nil, e.NotFoundError("Provider not found", nil)
	}

	return &relayAuth{
		Relay:         relay,
		Provider:      provider,
		Authorization: authorization,
		ProviderURL:   provider.GetString("url"),
	}, nil
}

func getHMACKey() ([]byte, error) {
	keyB64 := os.Getenv("RELAY_HMAC_KEY")
	if keyB64 == "" {
		return nil, fmt.Errorf("RELAY_HMAC_KEY not set")
	}
	return base64.StdEncoding.DecodeString(keyB64)
}

func getHMACKeyID() string {
	kid := os.Getenv("RELAY_HMAC_KEY_ID")
	if kid == "" {
		return "default"
	}
	return kid
}

func getIssuer() string {
	issuer := os.Getenv("RELAY_ISSUER")
	if issuer == "" {
		return cwt.DefaultIssuer
	}
	return issuer
}

func expiryTime(seconds int) int64 {
	return time.Now().Unix() + int64(seconds)
}
