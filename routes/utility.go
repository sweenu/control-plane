package routes

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterUtilityRoutes(se *core.ServeEvent) {
	se.Router.GET("/api/flags", handleFlags).Bind(apis.RequireAuth())
	se.Router.GET("/api/whoami", handleWhoami).Bind(apis.RequireAuth())
	se.Router.GET("/health", handleHealth)
	se.Router.GET("/api/relay/{guid}/check-host", handleCheckHost).Bind(apis.RequireAuth())
}

func handleFlags(e *core.RequestEvent) error {
	return e.JSON(200, map[string]any{})
}

func handleWhoami(e *core.RequestEvent) error {
	return e.JSON(200, e.Auth)
}

func handleHealth(e *core.RequestEvent) error {
	return e.JSON(200, map[string]string{"status": "ok"})
}

func handleCheckHost(e *core.RequestEvent) error {
	guid := e.Request.PathValue("guid")

	relay, err := e.App.FindFirstRecordByFilter(
		"relays",
		"guid = {:guid}",
		dbx.Params{"guid": guid},
	)
	if err != nil {
		return e.NotFoundError("Relay not found", nil)
	}

	providerID := relay.GetString("provider")
	provider, err := e.App.FindRecordById("providers", providerID)
	if err != nil {
		return e.NotFoundError("Provider not found", nil)
	}

	providerURL := provider.GetString("url")
	httpScheme := "https"
	if getWSScheme() == "ws" {
		httpScheme = "http"
	}
	healthURL := fmt.Sprintf("%s://%s/health", httpScheme, providerURL)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		return e.JSON(502, map[string]string{"status": "unreachable", "error": err.Error()})
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return e.InternalServerError("Failed to read response", nil)
	}

	return e.Blob(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
