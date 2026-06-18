package tokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type RefreshTokenStore interface {
	CreateRefreshToken(ctx context.Context, params sqlc.CreateRefreshTokenParams) (sqlc.RefreshToken, error)
}

type Pair struct {
	AccessToken  string
	RefreshToken string
}

func IssuePair(ctx context.Context, store RefreshTokenStore, cfg *config.Config, user sqlc.User) (Pair, error) {
	access, err := signAccessToken(cfg, user)
	if err != nil {
		return Pair{}, err
	}

	raw, err := generateRefreshToken()
	if err != nil {
		return Pair{}, err
	}

	_, err = store.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashRefreshToken(raw),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(cfg.JWTRefreshTTL), Valid: true},
	})
	if err != nil {
		return Pair{}, err
	}

	return Pair{AccessToken: access, RefreshToken: raw}, nil
}

func signAccessToken(cfg *config.Config, user sqlc.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": user.Role,
		"iat":  now.Unix(),
		"exp":  now.Add(cfg.JWTAccessTTL).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(cfg.JWTSecret))
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
