package routes

import (
	"fmt"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"relay-control-plane/cwt"
)

func RegisterFileTokenRoutes(se *core.ServeEvent) {
	se.Router.POST("/file-token", handleFileToken).Bind(apis.RequireAuth())
}

func handleFileToken(e *core.RequestEvent) error {
	var body struct {
		DocID         string `json:"docId"`
		Relay         string `json:"relay"`
		Folder        string `json:"folder"`
		Hash          string `json:"hash"`
		ContentType   string `json:"contentType"`
		ContentLength int64  `json:"contentLength"`
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
	token, err := cwt.GenerateFileToken(key, keyID, issuer, body.DocID, e.Auth.Id, ra.ProviderURL, ra.Authorization, expirySeconds, body.Hash)
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
		"fileHash":      body.Hash,
		"contentType":   body.ContentType,
		"contentLength": body.ContentLength,
	})
}
