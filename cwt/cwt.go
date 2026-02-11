package cwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

const DefaultIssuer = "relay-control-plane"

// GenerateDocToken creates a CWT token for document access.
// Authorization should be "full" (→ rw) or "read-only" (→ r).
func GenerateDocToken(key []byte, keyId string, docId string, userId string, audience string, authorization string, expirySeconds int) (string, error) {
	suffix := authSuffix(authorization)
	scope := fmt.Sprintf("doc:%s:%s", docId, suffix)
	return generateToken(key, keyId, userId, audience, scope, expirySeconds)
}

// GenerateFileToken creates a CWT token for file access.
// Authorization should be "full" (→ rw) or "read-only" (→ r).
func GenerateFileToken(key []byte, keyId string, docId string, userId string, audience string, authorization string, expirySeconds int, fileHash string) (string, error) {
	suffix := authSuffix(authorization)
	scope := fmt.Sprintf("file:%s:%s:%s", fileHash, docId, suffix)
	return generateToken(key, keyId, userId, audience, scope, expirySeconds)
}

func authSuffix(authorization string) string {
	if authorization == "full" {
		return "rw"
	}
	return "r"
}

func generateToken(key []byte, keyId string, userId string, audience string, scope string, expirySeconds int) (string, error) {
	now := uint64(time.Now().Unix())
	exp := now + uint64(expirySeconds)

	claims := buildClaimsMap(DefaultIssuer, userId, audience, exp, now, scope)
	payload, err := cbor.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encoding claims: %w", err)
	}

	protectedHeaders := buildProtectedHeaders(keyId)
	protectedBytes, err := cbor.Marshal(protectedHeaders)
	if err != nil {
		return "", fmt.Errorf("encoding protected headers: %w", err)
	}

	tag, err := computeMAC(protectedBytes, payload, key)
	if err != nil {
		return "", fmt.Errorf("computing MAC: %w", err)
	}

	// COSE_Mac0 = [protected, unprotected, payload, tag]
	coseMac0 := []cbor.RawMessage{
		mustMarshal(protectedBytes), // bstr-wrapped protected headers
		mustMarshal(map[any]any{}),  // empty unprotected headers
		mustMarshal(payload),        // bstr-wrapped payload
		mustMarshal(tag),            // bstr-wrapped tag
	}

	coseMac0Bytes, err := cbor.Marshal(coseMac0)
	if err != nil {
		return "", fmt.Errorf("encoding COSE_Mac0: %w", err)
	}

	// Wrap with CBOR tag 17 (COSE_Mac0), then tag 61 (CWT)
	tagged17 := cbor.Tag{Number: 17, Content: cbor.RawMessage(coseMac0Bytes)}
	tagged17Bytes, err := cbor.Marshal(tagged17)
	if err != nil {
		return "", fmt.Errorf("encoding COSE_Mac0 tag: %w", err)
	}

	tagged61 := cbor.Tag{Number: 61, Content: cbor.RawMessage(tagged17Bytes)}
	cwtBytes, err := cbor.Marshal(tagged61)
	if err != nil {
		return "", fmt.Errorf("encoding CWT tag: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(cwtBytes), nil
}

// buildClaimsMap builds a CBOR map with integer keys per CWT spec.
func buildClaimsMap(issuer, subject, audience string, exp, iat uint64, scope string) map[int64]any {
	return map[int64]any{
		1:      issuer,   // iss
		2:      subject,  // sub
		3:      audience, // aud
		4:      exp,      // exp
		6:      iat,      // iat
		-80201: scope,    // scope (private claim)
	}
}

// buildProtectedHeaders builds the COSE protected headers map.
// Key 1 = algorithm (4 = HMAC_256_64), Key 4 = key_id.
func buildProtectedHeaders(keyId string) map[int]any {
	return map[int]any{
		1: 4,              // alg: HMAC 256/64
		4: []byte(keyId),  // kid
	}
}

// computeMAC computes HMAC-SHA-256 over the MAC_structure, truncated to 8 bytes.
// MAC_structure = ['MAC0', protected_bytes, external_aad, payload]
func computeMAC(protectedBytes, payload, key []byte) ([]byte, error) {
	macStructure := []any{
		"MAC0",
		protectedBytes,
		[]byte{}, // external_aad (empty)
		payload,
	}

	macInput, err := cbor.Marshal(macStructure)
	if err != nil {
		return nil, fmt.Errorf("encoding MAC_structure: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(macInput)
	fullMAC := mac.Sum(nil)

	return fullMAC[:8], nil
}

func mustMarshal(v any) cbor.RawMessage {
	b, err := cbor.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("cbor.Marshal: %v", err))
	}
	return b
}
