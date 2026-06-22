package clientaccount

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/tokens"
)

const minPasswordLength = 8

var (
	ErrWeakPassword  = errors.New("password must be at least 8 characters")
	ErrAccountExists = errors.New("an account with this email already exists")
	ErrPhoneRequired = errors.New("a phone number is required")
	ErrPhoneTaken    = errors.New("that phone number is already in use")
)

type Store interface {
	GetUserByEmail(ctx context.Context, email string) (sqlc.User, error)
	CreateUser(ctx context.Context, params sqlc.CreateUserParams) (sqlc.User, error)
	SetUserMarketingOptIn(ctx context.Context, params sqlc.SetUserMarketingOptInParams) error
	LinkBookingRequestsToClient(ctx context.Context, params sqlc.LinkBookingRequestsToClientParams) error
	tokens.RefreshTokenStore
}

type Input struct {
	FirstName      string
	LastName       string
	Phone          string
	Password       string
	MarketingOptIn bool
}

type Session struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
	User         SessionUser `json:"user"`
}

type SessionUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	CreatedAt string `json:"createdAt"`
}

func Create(ctx context.Context, store Store, cfg *config.Config, email string, input Input, marketingSource string) (Session, error) {
	if len(input.Password) < minPasswordLength {
		return Session{}, ErrWeakPassword
	}
	if strings.TrimSpace(input.Phone) == "" {
		return Session{}, ErrPhoneRequired
	}
	email = NormalizeEmail(email)

	if _, err := store.GetUserByEmail(ctx, email); err == nil {
		return Session{}, ErrAccountExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Session{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, err
	}

	first, last := names(input.FirstName, input.LastName)
	user, err := store.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: string(hash),
		Role:         "user",
		FirstName:    first,
		LastName:     last,
		Phone:        strings.TrimSpace(input.Phone),
		AuthProvider: "password",
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Session{}, ErrPhoneTaken
		}
		return Session{}, err
	}

	if input.MarketingOptIn {
		source := marketingSource
		_ = store.SetUserMarketingOptIn(ctx, sqlc.SetUserMarketingOptInParams{
			ID:     user.ID,
			Source: &source,
		})
	}
	_ = store.LinkBookingRequestsToClient(ctx, sqlc.LinkBookingRequestsToClientParams{
		ClientEmail:  email,
		ClientUserID: pgtype.UUID{Bytes: user.ID, Valid: true},
	})

	pair, err := tokens.IssuePair(ctx, store, cfg, user)
	if err != nil {
		return Session{}, err
	}
	return Session{
		Token:        pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User:         sessionUserFromRecord(user),
	}, nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func names(first, last string) (*string, *string) {
	f := strings.TrimSpace(first)
	l := strings.TrimSpace(last)
	var fp, lp *string
	if f != "" {
		fp = &f
	}
	if l != "" {
		lp = &l
	}
	return fp, lp
}

func sessionUserFromRecord(u sqlc.User) SessionUser {
	out := SessionUser{
		ID:    u.ID.String(),
		Email: u.Email,
		Role:  u.Role,
	}
	if u.CreatedAt.Valid {
		out.CreatedAt = u.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	if u.FirstName != nil {
		out.FirstName = *u.FirstName
	}
	if u.LastName != nil {
		out.LastName = *u.LastName
	}
	return out
}
