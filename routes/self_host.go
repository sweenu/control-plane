package routes

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterSelfHostRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/collections/relays/self-host", handleSelfHost).Bind(apis.RequireAuth())
}

func handleSelfHost(e *core.RequestEvent) error {
	var body struct {
		URL      string `json:"url"`
		Provider string `json:"provider"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.BadRequestError("Invalid request body", nil)
	}

	var provider *core.Record

	if body.URL != "" {
		// Create a new self-hosted provider
		hmacKey := make([]byte, 32)
		if _, err := rand.Read(hmacKey); err != nil {
			return e.InternalServerError("Failed to generate HMAC key", nil)
		}

		provCol, err := e.App.FindCollectionByNameOrId("providers")
		if err != nil {
			return e.InternalServerError("Collection not found", nil)
		}

		provider = core.NewRecord(provCol)
		provider.Set("url", body.URL)
		provider.Set("self_hosted", true)
		provider.Set("public_key", base64.StdEncoding.EncodeToString(hmacKey))
		provider.Set("key_id", fmt.Sprintf("self_host_%d", time.Now().Unix()))
		provider.Set("key_type", "hmac")

		parsed, err := url.Parse(body.URL)
		if err == nil && parsed.Host != "" {
			provider.Set("name", parsed.Host)
		} else {
			provider.Set("name", body.URL)
		}

		if err := e.App.Save(provider); err != nil {
			return e.InternalServerError("Failed to create provider", nil)
		}
	} else if body.Provider != "" {
		var err error
		provider, err = e.App.FindRecordById("providers", body.Provider)
		if err != nil {
			return e.NotFoundError("Provider not found", nil)
		}
	} else {
		return e.BadRequestError("Either url or provider is required", nil)
	}

	// Derive relay name from provider URL hostname
	relayName := provider.GetString("name")
	if relayName == "" {
		relayName = "self-hosted-relay"
	}

	relaysCol, err := e.App.FindCollectionByNameOrId("relays")
	if err != nil {
		return e.InternalServerError("Collection not found", nil)
	}

	relay := core.NewRecord(relaysCol)
	relay.Set("guid", uuid.NewString())
	relay.Set("name", relayName)
	relay.Set("creator", e.Auth.Id)
	relay.Set("provider", provider.Id)
	if err := e.App.Save(relay); err != nil {
		return e.InternalServerError("Failed to create relay", nil)
	}

	if err := AutoCreateRelayDeps(e.App, relay, e.Auth.Id); err != nil {
		return e.InternalServerError("Failed to create relay dependencies", nil)
	}

	return e.JSON(200, relay)
}
