package routes

import (
	"bytes"
	"os"
	"text/template"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterTemplateRoutes(se *core.ServeEvent) {
	se.Router.GET("/templates/relay.toml", handleRelayToml).Bind(apis.RequireAuth())
	se.Router.GET("/api/collections/relays/records/{id}/relay.toml", handleRelayToml).Bind(apis.RequireAuth())
}

func handleRelayToml(e *core.RequestEvent) error {
	tmplBytes, err := os.ReadFile("templates/relay.toml.tmpl")
	if err != nil {
		return e.InternalServerError("Template not found", nil)
	}

	tmpl, err := template.New("relay.toml").Parse(string(tmplBytes))
	if err != nil {
		return e.InternalServerError("Invalid template", nil)
	}

	publicKey := os.Getenv("RELAY_HMAC_KEY")
	if publicKey == "" {
		publicKey = "<HMAC_KEY_BASE64>"
	}

	data := map[string]string{
		"URL":       "{url}",
		"KeyID":     getHMACKeyID(),
		"PublicKey": publicKey,
		"Issuer":    getIssuer(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return e.InternalServerError("Failed to render template", nil)
	}

	return e.Blob(200, "text/plain", buf.Bytes())
}
