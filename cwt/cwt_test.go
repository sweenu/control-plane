package cwt

import (
	"encoding/base64"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

var testKey = []byte("test_key_1234567890123456789012")
var testKeyId = "test-key-id"

func TestGenerateDocToken_Base64Decodable(t *testing.T) {
	token, err := GenerateDocToken(testKey, testKeyId, "doc123", "user1", "https://relay.example.com", "full", 3600)
	if err != nil {
		t.Fatalf("GenerateDocToken failed: %v", err)
	}

	_, err = base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token is not valid base64url: %v", err)
	}
}

func TestGenerateDocToken_CWTTag61(t *testing.T) {
	token, err := GenerateDocToken(testKey, testKeyId, "doc123", "user1", "https://relay.example.com", "full", 3600)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := base64.RawURLEncoding.DecodeString(token)

	var tag cbor.Tag
	if err := cbor.Unmarshal(raw, &tag); err != nil {
		t.Fatalf("failed to unmarshal outer CBOR: %v", err)
	}
	if tag.Number != 61 {
		t.Fatalf("expected CWT tag 61, got %d", tag.Number)
	}
}

func TestGenerateDocToken_COSEMac0Tag17(t *testing.T) {
	token, err := GenerateDocToken(testKey, testKeyId, "doc123", "user1", "https://relay.example.com", "full", 3600)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := base64.RawURLEncoding.DecodeString(token)

	// Unwrap CWT tag 61
	var outerTag cbor.Tag
	if err := cbor.Unmarshal(raw, &outerTag); err != nil {
		t.Fatal(err)
	}

	innerBytes, err := cbor.Marshal(outerTag.Content)
	if err != nil {
		t.Fatal(err)
	}

	var innerTag cbor.Tag
	if err := cbor.Unmarshal(innerBytes, &innerTag); err != nil {
		t.Fatalf("inner content is not a CBOR tag: %v", err)
	}
	if innerTag.Number != 17 {
		t.Fatalf("expected COSE_Mac0 tag 17, got %d", innerTag.Number)
	}
}

func TestGenerateDocToken_ProtectedHeaders(t *testing.T) {
	token, err := GenerateDocToken(testKey, testKeyId, "doc123", "user1", "https://relay.example.com", "full", 3600)
	if err != nil {
		t.Fatal(err)
	}

	mac0Array := decodeCOSEMac0(t, token)

	// First element is protected headers (bstr)
	var protectedBytes []byte
	if err := cbor.Unmarshal(mac0Array[0], &protectedBytes); err != nil {
		t.Fatalf("failed to decode protected headers bstr: %v", err)
	}

	var headers map[int]cbor.RawMessage
	if err := cbor.Unmarshal(protectedBytes, &headers); err != nil {
		t.Fatalf("failed to decode protected headers map: %v", err)
	}

	// Check algorithm (key 1 = value 4 for HMAC_256_64)
	var alg int
	if err := cbor.Unmarshal(headers[1], &alg); err != nil {
		t.Fatal(err)
	}
	if alg != 4 {
		t.Fatalf("expected algorithm 4 (HMAC_256_64), got %d", alg)
	}

	// Check key_id (key 4 = UTF-8 bytes of keyId)
	var kid []byte
	if err := cbor.Unmarshal(headers[4], &kid); err != nil {
		t.Fatal(err)
	}
	if string(kid) != testKeyId {
		t.Fatalf("expected key_id %q, got %q", testKeyId, string(kid))
	}
}

func TestGenerateDocToken_ClaimsKeys(t *testing.T) {
	token, err := GenerateDocToken(testKey, testKeyId, "doc123", "user1", "https://relay.example.com", "full", 3600)
	if err != nil {
		t.Fatal(err)
	}

	mac0Array := decodeCOSEMac0(t, token)

	// Third element is payload (bstr containing CBOR claims)
	var payload []byte
	if err := cbor.Unmarshal(mac0Array[2], &payload); err != nil {
		t.Fatalf("failed to decode payload bstr: %v", err)
	}

	var claims map[int64]cbor.RawMessage
	if err := cbor.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("failed to decode claims map: %v", err)
	}

	expectedKeys := []int64{1, 2, 3, 4, 6, -80201}
	for _, key := range expectedKeys {
		if _, ok := claims[key]; !ok {
			t.Errorf("missing claim key %d", key)
		}
	}
}

func TestGenerateDocToken_HMACTag8Bytes(t *testing.T) {
	token, err := GenerateDocToken(testKey, testKeyId, "doc123", "user1", "https://relay.example.com", "full", 3600)
	if err != nil {
		t.Fatal(err)
	}

	mac0Array := decodeCOSEMac0(t, token)

	// Fourth element is the MAC tag (bstr)
	var tag []byte
	if err := cbor.Unmarshal(mac0Array[3], &tag); err != nil {
		t.Fatalf("failed to decode MAC tag: %v", err)
	}
	if len(tag) != 8 {
		t.Fatalf("expected 8-byte HMAC tag, got %d bytes", len(tag))
	}
}

func TestGenerateDocToken_ScopeFormats(t *testing.T) {
	tests := []struct {
		auth  string
		scope string
	}{
		{"full", "doc:doc123:rw"},
		{"read-only", "doc:doc123:r"},
	}
	for _, tt := range tests {
		token, err := GenerateDocToken(testKey, testKeyId, "doc123", "user1", "https://relay.example.com", tt.auth, 3600)
		if err != nil {
			t.Fatal(err)
		}
		scope := extractScope(t, token)
		if scope != tt.scope {
			t.Errorf("auth=%q: expected scope %q, got %q", tt.auth, tt.scope, scope)
		}
	}
}

func TestGenerateFileToken_ScopeFormats(t *testing.T) {
	tests := []struct {
		auth  string
		scope string
	}{
		{"full", "file:abc123:doc456:rw"},
		{"read-only", "file:abc123:doc456:r"},
	}
	for _, tt := range tests {
		token, err := GenerateFileToken(testKey, testKeyId, "doc456", "user1", "https://relay.example.com", tt.auth, 3600, "abc123")
		if err != nil {
			t.Fatal(err)
		}
		scope := extractScope(t, token)
		if scope != tt.scope {
			t.Errorf("auth=%q: expected scope %q, got %q", tt.auth, tt.scope, scope)
		}
	}
}

// decodeCOSEMac0 unwraps CWT tag 61 → COSE_Mac0 tag 17 → returns the 4-element array.
func decodeCOSEMac0(t *testing.T, token string) []cbor.RawMessage {
	t.Helper()
	raw, _ := base64.RawURLEncoding.DecodeString(token)

	var cwtTag cbor.Tag
	if err := cbor.Unmarshal(raw, &cwtTag); err != nil {
		t.Fatalf("unmarshal CWT tag: %v", err)
	}

	innerBytes, _ := cbor.Marshal(cwtTag.Content)
	var mac0Tag cbor.Tag
	if err := cbor.Unmarshal(innerBytes, &mac0Tag); err != nil {
		t.Fatalf("unmarshal COSE_Mac0 tag: %v", err)
	}

	arrayBytes, _ := cbor.Marshal(mac0Tag.Content)
	var arr []cbor.RawMessage
	if err := cbor.Unmarshal(arrayBytes, &arr); err != nil {
		t.Fatalf("unmarshal COSE_Mac0 array: %v", err)
	}
	if len(arr) != 4 {
		t.Fatalf("expected 4-element COSE_Mac0, got %d", len(arr))
	}
	return arr
}

func extractScope(t *testing.T, token string) string {
	t.Helper()
	mac0Array := decodeCOSEMac0(t, token)

	var payload []byte
	if err := cbor.Unmarshal(mac0Array[2], &payload); err != nil {
		t.Fatal(err)
	}

	var claims map[int64]cbor.RawMessage
	if err := cbor.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}

	var scope string
	if err := cbor.Unmarshal(claims[-80201], &scope); err != nil {
		t.Fatal(err)
	}
	return scope
}
