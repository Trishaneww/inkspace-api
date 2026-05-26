package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
)

// OAuthClaims is the subset of provider id_token claims we care about.
type OAuthClaims struct {
	Subject   string // provider-specific user id
	Email     string
	FirstName string
	LastName  string
}

// tokenResponse is the shape returned by both Google and Microsoft token endpoints.
type tokenResponse struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	// Microsoft uses snake_case + camelCase mixed; the rest of the body
	// is ignored.
}

// exchangeOAuthCode trades an authorization code for tokens and returns
// the parsed claims from the id_token.
//
// NOTE: For now we parse the id_token without verifying the signature.
// The token came from a direct HTTPS POST to the provider's token endpoint
// using our client secret, so the channel is trusted. Production hardening
// should verify against the provider's JWKS (RS256).
func exchangeOAuthCode(
	ctx context.Context,
	cfg *config.Config,
	provider, code, redirectURI string,
) (*OAuthClaims, error) {
	var (
		tokenURL     string
		clientID     string
		clientSecret string
		scope        string
	)

	switch provider {
	case "google":
		tokenURL = "https://oauth2.googleapis.com/token"
		clientID = cfg.GoogleClientID
		clientSecret = cfg.GoogleClientSecret
		// scope param is optional for Google's token endpoint
	case "microsoft":
		tenant := cfg.MicrosoftTenantID
		if tenant == "" {
			tenant = "common"
		}
		tokenURL = fmt.Sprintf(
			"https://login.microsoftonline.com/%s/oauth2/v2.0/token",
			tenant,
		)
		clientID = cfg.MicrosoftClientID
		clientSecret = cfg.MicrosoftClientSecret
		// Microsoft requires the scope echoed back. Match what the
		// frontend asked for in lib/auth/oauth.ts.
		scope = "openid email profile User.Read"
	default:
		return nil, ErrProviderNotConfigured
	}

	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}
	if scope != "" {
		form.Set("scope", scope)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		reqCtx, "POST", tokenURL, strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"oauth token exchange returned %d: %s", resp.StatusCode, string(body),
		)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if tr.IDToken == "" {
		return nil, errors.New("oauth token response missing id_token")
	}

	return parseIDTokenClaims(tr.IDToken)
}

// parseIDTokenClaims decodes the id_token JWT without verifying its signature
// (see security note in exchangeOAuthCode).
func parseIDTokenClaims(idToken string) (*OAuthClaims, error) {
	parser := jwt.NewParser()
	tok, _, err := parser.ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("parse id_token: %w", err)
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("id_token has unexpected claims shape")
	}

	out := &OAuthClaims{}
	if v, ok := claims["sub"].(string); ok {
		out.Subject = v
	}
	if v, ok := claims["email"].(string); ok {
		out.Email = strings.ToLower(strings.TrimSpace(v))
	}
	// Microsoft uses `preferred_username` instead of `email` when only
	// "User.Read" scope is granted; fall back to it.
	if out.Email == "" {
		if v, ok := claims["preferred_username"].(string); ok {
			out.Email = strings.ToLower(strings.TrimSpace(v))
		}
	}
	if v, ok := claims["given_name"].(string); ok {
		out.FirstName = v
	}
	if v, ok := claims["family_name"].(string); ok {
		out.LastName = v
	}
	// If given/family aren't present, split `name` as a best-effort fallback.
	if out.FirstName == "" && out.LastName == "" {
		if name, ok := claims["name"].(string); ok && name != "" {
			parts := strings.SplitN(name, " ", 2)
			out.FirstName = parts[0]
			if len(parts) == 2 {
				out.LastName = parts[1]
			}
		}
	}

	if out.Subject == "" || out.Email == "" {
		return nil, errors.New("id_token missing required claims (sub/email)")
	}
	return out, nil
}

// ── OAuth-session token (short-lived, signed with our JWT secret) ──

const oauthSessionTTL = 15 * time.Minute

// signOAuthSession returns a JWT we hand back to the frontend after a
// successful provider sign-in for a brand-new user. The frontend echoes
// it on POST /v1/auth/oauth/complete so we can trust the carried claims
// without re-running the provider round-trip.
func signOAuthSession(cfg *config.Config, claims *OAuthClaims, provider string) (string, error) {
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"purpose":    "oauth_session",
		"provider":   provider,
		"sub":        claims.Subject,
		"email":      claims.Email,
		"first_name": claims.FirstName,
		"last_name":  claims.LastName,
		"iat":        now.Unix(),
		"exp":        now.Add(oauthSessionTTL).Unix(),
	})
	return tok.SignedString([]byte(cfg.JWTSecret))
}

func verifyOAuthSession(cfg *config.Config, raw string) (*OAuthClaims, error) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid oauth session token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid oauth session claims")
	}
	if purpose, _ := claims["purpose"].(string); purpose != "oauth_session" {
		return nil, errors.New("token purpose mismatch")
	}
	out := &OAuthClaims{}
	out.Subject, _ = claims["sub"].(string)
	out.Email, _ = claims["email"].(string)
	out.FirstName, _ = claims["first_name"].(string)
	out.LastName, _ = claims["last_name"].(string)
	if out.Email == "" || out.Subject == "" {
		return nil, errors.New("oauth session token missing required claims")
	}
	return out, nil
}
