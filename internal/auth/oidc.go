package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var ErrOIDCNotConfigured = errors.New("oidc is not configured")
var ErrOIDCAccountNotAllowed = errors.New("oidc account is not allowed")

type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	TenantID     string
	DefaultRole  string
	StateSecret  string
}

type OIDCService struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
	config   OIDCConfig
}

func NewOIDCService(ctx context.Context, config OIDCConfig) (*OIDCService, error) {
	if config.IssuerURL == "" || config.ClientID == "" || config.ClientSecret == "" || config.RedirectURL == "" || config.TenantID == "" || len(config.StateSecret) < 32 {
		return nil, ErrOIDCNotConfigured
	}
	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, err
	}
	return &OIDCService{provider: provider, verifier: provider.Verifier(&oidc.Config{ClientID: config.ClientID}), oauth: oauth2.Config{ClientID: config.ClientID, ClientSecret: config.ClientSecret, Endpoint: provider.Endpoint(), RedirectURL: config.RedirectURL, Scopes: []string{oidc.ScopeOpenID, "profile", "email"}}, config: config}, nil
}

func (s *OIDCService) AuthorizationURL() (string, string, error) {
	if s == nil {
		return "", "", ErrOIDCNotConfigured
	}
	verifier := oauth2.GenerateVerifier()
	state := s.signState(verifier)
	return s.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), state, nil
}

type OIDCIdentity struct {
	Subject string
	Email   string
	Name    string
}

func (s *OIDCService) Verify(ctx context.Context, code, state, cookieState string) (OIDCIdentity, error) {
	if s == nil {
		return OIDCIdentity{}, ErrOIDCNotConfigured
	}
	verifier, ok := s.verifyState(state, cookieState)
	if !ok {
		return OIDCIdentity{}, errors.New("invalid oidc state")
	}
	token, err := s.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return OIDCIdentity{}, err
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return OIDCIdentity{}, errors.New("oidc response has no id_token")
	}
	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return OIDCIdentity{}, err
	}
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return OIDCIdentity{}, err
	}
	if strings.TrimSpace(claims.Email) == "" || !strings.Contains(claims.Email, "@") {
		return OIDCIdentity{}, ErrOIDCAccountNotAllowed
	}
	return OIDCIdentity{Subject: idToken.Subject, Email: strings.ToLower(strings.TrimSpace(claims.Email)), Name: claims.Name}, nil
}

func (s *OIDCService) Issuer() string {
	if s == nil {
		return ""
	}
	return s.config.IssuerURL
}
func (s *OIDCService) TenantID() string { return s.config.TenantID }
func (s *OIDCService) DefaultRole() string {
	if s.config.DefaultRole == "" {
		return RoleViewer
	}
	return s.config.DefaultRole
}

func (s *OIDCService) signState(verifier string) string {
	mac := hmac.New(sha256.New, []byte(s.config.StateSecret))
	mac.Write([]byte(verifier))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(verifier + "." + base64.RawURLEncoding.EncodeToString(sig)))
}
func (s *OIDCService) verifyState(state, cookieState string) (string, bool) {
	if state == "" || state != cookieState {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(raw), ".")
	if len(parts) != 2 {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(s.config.StateSecret))
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return "", false
	}
	return parts[0], true
}

func OIDCSession(user *User) *Session {
	return &Session{ID: "", UserID: user.ID, TenantID: user.TenantID, Role: user.Role, ExpiresAt: time.Now().UTC().Add(8 * time.Hour), CreatedAt: time.Now().UTC()}
}
