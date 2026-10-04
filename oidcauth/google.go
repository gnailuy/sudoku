package oidcauth

import (
	"context"
	"errors"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const GoogleIssuer = "https://accounts.google.com"

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// GoogleProvider performs discovery, authorization-code exchange, signature
// verification, and standard OIDC issuer/audience/time validation.
type GoogleProvider struct {
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func NewGoogleProvider(ctx context.Context, config GoogleConfig) (*GoogleProvider, error) {
	if config.ClientID == "" || config.RedirectURL == "" {
		return nil, ErrInvalidConfiguration
	}
	provider, err := oidc.NewProvider(ctx, GoogleIssuer)
	if err != nil {
		return nil, err
	}
	return &GoogleProvider{
		oauth: oauth2.Config{
			ClientID: config.ClientID, ClientSecret: config.ClientSecret,
			Endpoint: provider.Endpoint(), RedirectURL: config.RedirectURL,
			Scopes: []string{oidc.ScopeOpenID, "email", "profile"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
	}, nil
}

func (p *GoogleProvider) AuthorizationURL(state, nonce, codeChallenge string) string {
	return p.oauth.AuthCodeURL(state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (p *GoogleProvider) Exchange(ctx context.Context, code, codeVerifier string) (Claims, error) {
	token, err := p.oauth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return Claims{}, err
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return Claims{}, errors.New("provider response omitted ID token")
	}
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Claims{}, err
	}
	var profile struct {
		Nonce         string `json:"nonce"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&profile); err != nil {
		return Claims{}, err
	}
	return Claims{
		Issuer: idToken.Issuer, Subject: idToken.Subject,
		Audience: append([]string(nil), idToken.Audience...), Nonce: profile.Nonce,
		Email: profile.Email, EmailVerified: profile.EmailVerified, Name: profile.Name,
		IssuedAt: idToken.IssuedAt, ExpiresAt: idToken.Expiry,
	}, nil
}
