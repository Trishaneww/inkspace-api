package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

const (
	googleTokenURL  = "https://oauth2.googleapis.com/token"
	googleRevokeURL = "https://oauth2.googleapis.com/revoke"
)

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *service) ConnectGoogleCalendar(ctx context.Context, userID uuid.UUID, input ConnectGoogleCalendarInput) (ArtistSettings, error) {
	if s.cipher == nil {
		return ArtistSettings{}, fmt.Errorf("%w: google calendar", ErrIntegrationConfig)
	}
	if s.cfg.GoogleClientID == "" || s.cfg.GoogleClientSecret == "" {
		return ArtistSettings{}, fmt.Errorf("%w: google oauth credentials", ErrIntegrationConfig)
	}

	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return ArtistSettings{}, err
	}
	if err := s.repo.EnsureArtistSettings(ctx, artist.ID); err != nil {
		return ArtistSettings{}, err
	}

	tokens, err := s.exchangeGoogleCode(ctx, input.Code, input.RedirectURI)
	if err != nil {
		s.log.Warn("google_calendar_exchange_failed", "error", err)
		return ArtistSettings{}, ErrOAuthExchange
	}

	email := emailFromIDToken(tokens.IDToken)
	if email == "" {
		return ArtistSettings{}, fmt.Errorf("%w: provider did not return an account email", ErrOAuthExchange)
	}

	accessEnc, err := s.cipher.Encrypt(tokens.AccessToken)
	if err != nil {
		return ArtistSettings{}, fmt.Errorf("encrypt access token: %w", err)
	}

	params := sqlc.SetGoogleCalendarConnectionParams{
		ArtistID:            artist.ID,
		GoogleCalendarEmail: email,
		AccessToken:         accessEnc,
		TokenExpiry:         expiryFromNow(tokens.ExpiresIn),
	}
	if tokens.RefreshToken != "" {
		refreshEnc, err := s.cipher.Encrypt(tokens.RefreshToken)
		if err != nil {
			return ArtistSettings{}, fmt.Errorf("encrypt refresh token: %w", err)
		}
		params.RefreshToken = &refreshEnc
	}

	row, err := s.repo.SetGoogleCalendarConnection(ctx, params)
	if err != nil {
		return ArtistSettings{}, err
	}
	return s.settingsResponse(ctx, row), nil
}

func (s *service) DisconnectGoogleCalendar(ctx context.Context, userID uuid.UUID) (ArtistSettings, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return ArtistSettings{}, err
	}
	if err := s.repo.EnsureArtistSettings(ctx, artist.ID); err != nil {
		return ArtistSettings{}, err
	}

	current, err := s.repo.GetArtistSettings(ctx, artist.ID)
	if err != nil {
		return ArtistSettings{}, err
	}
	if s.cipher != nil && current.GoogleCalendarRefreshToken != nil {
		if token, err := s.cipher.Decrypt(*current.GoogleCalendarRefreshToken); err == nil {
			s.revokeGoogleToken(ctx, token)
		}
	}

	row, err := s.repo.ClearGoogleCalendarConnection(ctx, artist.ID)
	if err != nil {
		return ArtistSettings{}, err
	}
	return s.settingsResponse(ctx, row), nil
}

func (s *service) exchangeGoogleCode(ctx context.Context, code, redirectURI string) (googleTokenResponse, error) {
	form := url.Values{
		"client_id":     {s.cfg.GoogleClientID},
		"client_secret": {s.cfg.GoogleClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, googleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return googleTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return googleTokenResponse{}, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return googleTokenResponse{}, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tr googleTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return googleTokenResponse{}, fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return googleTokenResponse{}, fmt.Errorf("token response missing access_token")
	}
	return tr, nil
}

func (s *service) revokeGoogleToken(ctx context.Context, token string) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	form := url.Values{"token": {token}}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, googleRevokeURL, strings.NewReader(form.Encode()))
	if err != nil {
		s.log.Warn("google_calendar_revoke_failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Warn("google_calendar_revoke_failed", "error", err)
		return
	}
	_ = resp.Body.Close()
}

func emailFromIDToken(idToken string) string {
	if idToken == "" {
		return ""
	}
	tok, _, err := jwt.NewParser().ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return ""
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}
	email, _ := claims["email"].(string)
	return strings.ToLower(strings.TrimSpace(email))
}

func expiryFromNow(expiresIn int) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
		Valid: true,
	}
}
