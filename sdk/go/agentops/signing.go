package agentops

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

const signingVersion = "v1"

func hashBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func canonicalRequest(method, path string, timestamp int64, nonce, bodyHash string) string {
	return strings.Join([]string{signingVersion, method, path, strconv.FormatInt(timestamp, 10), nonce, bodyHash}, "\n")
}

func signRequest(secret []byte, method, path string, body []byte, timestamp int64, nonce string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(canonicalRequest(method, path, timestamp, nonce, hashBody(body))))
	return signingVersion + "=" + hex.EncodeToString(mac.Sum(nil))
}

func newNonce() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
