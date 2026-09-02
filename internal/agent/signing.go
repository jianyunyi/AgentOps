package agent

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const signingVersion = "v1"

var (
	ErrInvalidAgentSignature    = errors.New("invalid agent signature")
	ErrSigningSecretUnavailable = errors.New("agent signing secret unavailable")
	ErrSignatureRequired        = errors.New("agent request signature required")
)

type SigningSecretProtector interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type AESGCMProtector struct {
	key []byte
}

func NewAESGCMProtector(key []byte) (*AESGCMProtector, error) {
	if len(key) != 32 {
		return nil, errors.New("agent signing encryption key must be 32 bytes")
	}
	return &AESGCMProtector{key: append([]byte(nil), key...)}, nil
}

func NewAESGCMProtectorFromString(raw string) (*AESGCMProtector, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrSigningSecretUnavailable
	}
	var key []byte
	var err error
	if len(raw) == 64 {
		key, err = hex.DecodeString(raw)
	}
	if err != nil || len(key) != 32 {
		key, err = base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != 32 {
			key, err = base64.RawStdEncoding.DecodeString(raw)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("decode agent signing encryption key: %w", err)
	}
	return NewAESGCMProtector(key)
}

func (p *AESGCMProtector) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	result := make([]byte, 0, len(signingVersion)+1+len(nonce)+len(sealed))
	result = append(result, signingVersion+":"...)
	result = append(result, nonce...)
	return append(result, sealed...), nil
}

func (p *AESGCMProtector) Decrypt(ciphertext []byte) ([]byte, error) {
	if !strings.HasPrefix(string(ciphertext), signingVersion+":") {
		return nil, errors.New("unsupported signing secret ciphertext version")
	}
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ciphertext = ciphertext[len(signingVersion)+1:]
	if len(ciphertext) < gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("signing secret ciphertext is truncated")
	}
	nonce, sealed := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, sealed, nil)
}

func HashRequestBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func CanonicalRequest(method, path string, metadata AuthenticationMetadata) string {
	return strings.Join([]string{signingVersion, method, path, fmt.Sprintf("%d", metadata.Timestamp), metadata.Nonce, metadata.BodyHash}, "\n")
}

func BuildAgentSignature(secret []byte, method, path string, body []byte, timestamp int64, nonce string) string {
	metadata := AuthenticationMetadata{Timestamp: timestamp, Nonce: nonce, Method: method, Path: path, BodyHash: HashRequestBody(body)}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(CanonicalRequest(method, path, metadata)))
	return signingVersion + "=" + hex.EncodeToString(mac.Sum(nil))
}

func VerifyAgentSignature(secret []byte, metadata AuthenticationMetadata) error {
	if len(secret) == 0 || metadata.Method == "" || metadata.Path == "" || len(metadata.BodyHash) != sha256.Size*2 {
		return ErrInvalidAgentSignature
	}
	if bodyHash, err := hex.DecodeString(metadata.BodyHash); err != nil || len(bodyHash) != sha256.Size {
		return ErrInvalidAgentSignature
	}
	header := strings.TrimSpace(metadata.Signature)
	encoded := strings.TrimPrefix(header, signingVersion+"=")
	if encoded == header || encoded == "" || strings.ToLower(encoded) != encoded {
		return ErrInvalidAgentSignature
	}
	provided, err := hex.DecodeString(encoded)
	if err != nil || len(provided) != sha256.Size {
		return ErrInvalidAgentSignature
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(CanonicalRequest(metadata.Method, metadata.Path, metadata)))
	if subtle.ConstantTimeCompare(provided, mac.Sum(nil)) != 1 {
		return ErrInvalidAgentSignature
	}
	return nil
}

func encodeSigningSecret(secret []byte) string {
	return base64.RawStdEncoding.EncodeToString(secret)
}

func decodeSigningSecret(raw string) ([]byte, error) {
	secret, err := base64.RawStdEncoding.DecodeString(raw)
	if err != nil || len(secret) != 32 {
		return nil, ErrSigningSecretUnavailable
	}
	return secret, nil
}
