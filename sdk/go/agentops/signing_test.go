package agentops

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestSignatureUsesFrozenV1ProtocolVector(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	body := []byte("hello")
	bodyHash := hashBody(body)
	if bodyHash != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("body hash = %q", bodyHash)
	}
	canonical := canonicalRequest("POST", "/api/v1/ingest/events", 1700000000, "nonce-fixed", bodyHash)
	expectedCanonical := "v1\nPOST\n/api/v1/ingest/events\n1700000000\nnonce-fixed\n2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if canonical != expectedCanonical {
		t.Fatalf("canonical request = %q", canonical)
	}
	signature := signRequest(secret, "POST", "/api/v1/ingest/events", body, 1700000000, "nonce-fixed")
	if signature != "v1=f7151e596375c3562ff93158cd2dd289b4a17bdf23f5b8149902b8bb8f4b3cb3" {
		t.Fatalf("signature = %q", signature)
	}
	if strings.ToLower(signature) != signature {
		t.Fatal("signature must use lowercase hex")
	}
}

func TestNonceIsRandomPrintableHex(t *testing.T) {
	first, err := newNonce()
	if err != nil {
		t.Fatal(err)
	}
	second, err := newNonce()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || len(first) != 64 || len(second) != 64 {
		t.Fatalf("nonces = %q and %q", first, second)
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Fatalf("nonce is not lowercase hex: %v", err)
	}
}
