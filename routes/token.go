package routes

import (
	"fmt"
	"net/url"
	"os"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"relay-control-plane/cwt"
)

func RegisterTokenRoutes(se *core.ServeEvent) {
	se.Router.POST("/token", handleToken).Bind(apis.RequireAuth())
}

func getWSScheme() string {
	if os.Getenv("RELAY_WS_SCHEME") != "" {
		return os.Getenv("RELAY_WS_SCHEME")
	}
	return "wss"
}

// buildProviderURLs takes a provider URL (with or without scheme) and returns
// properly constructed WebSocket and HTTP URLs.
func buildProviderURLs(providerURL string) (wsURL string, httpURL string, err error) {
	parsed, err := url.Parse(providerURL)
	if err != nil {
		return "", "", err
	}

	// If no scheme was provided, treat the whole string as host
	if parsed.Scheme == "" || parsed.Host == "" {
		parsed, err = url.Parse("http://" + providerURL)
		if err != nil {
			return "", "", err
		}
	}

	wsScheme := getWSScheme()
	httpScheme := "https"
	if wsScheme == "ws" {
		httpScheme = "http"
	}

	wsURL = fmt.Sprintf("%s://%s%s", wsScheme, parsed.Host, parsed.Path)
	httpURL = fmt.Sprintf("%s://%s%s", httpScheme, parsed.Host, parsed.Path)
	return wsURL, httpURL, nil
}

func handleToken(e *core.RequestEvent) error {
	var body struct {
		DocID  string `json:"docId"`
		Relay  string `json:"relay"`
		Folder string `json:"folder"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.BadRequestError("Invalid request body", nil)
	}

	ra, err := resolveRelayAuth(e, body.Relay)
	if err != nil {
		return err
	}

	key, err := getHMACKey()
	if err != nil {
		return e.InternalServerError("HMAC key not configured", nil)
	}
	keyID := getHMACKeyID()
	issuer := getIssuer()

	const expirySeconds = 3600
	token, err := cwt.GenerateDocToken(key, keyID, issuer, body.DocID, e.Auth.Id, ra.ProviderURL, ra.Authorization, expirySeconds)
	if err != nil {
		return e.InternalServerError("Failed to generate token", nil)
	}

	wsURL, httpURL, err := buildProviderURLs(ra.ProviderURL)
	if err != nil {
		return e.InternalServerError("Invalid provider URL", nil)
	}

	return e.JSON(200, map[string]any{
		"url":           fmt.Sprintf("%s/d/%s/ws", wsURL, body.DocID),
		"baseUrl":       fmt.Sprintf("%s/d/%s", httpURL, body.DocID),
		"docId":         body.DocID,
		"token":         token,
		"authorization": ra.Authorization,
		"expiryTime":    expiryTime(expirySeconds),
	})
}
