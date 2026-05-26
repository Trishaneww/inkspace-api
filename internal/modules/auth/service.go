package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
)

// ── Sentinel errors — handlers map these to HTTP codes ───────────

var (
	ErrInvalidCredentials          = errors.New("invalid credentials")
	ErrEmailTaken                  = errors.New("email already in use")
	ErrPhoneTaken                  = errors.New("phone number already in use")
	ErrPhoneVerificationNotFound   = errors.New("phone verification not found or expired")
	ErrInvalidVerificationCode     = errors.New("invalid verification code")
	ErrTooManyVerificationAttempts = errors.New("too many verification attempts")
	ErrProviderNotConfigured       = errors.New("oauth provider not configured")
	ErrProviderNotImplemented      = errors.New("oauth provider integration not implemented")
)

// ── Service ──────────────────────────────────────────────────────

type Service interface {
	Register(ctx context.Context, in RegisterInput) (*PhoneVerificationRequiredResponse, error)
	Login(ctx context.Context, in LoginInput) (interface{}, error)
	VerifyPhone(ctx context.Context, in VerifyPhoneInput) (*AuthenticatedResponse, error)
	ResendPhoneCode(ctx context.Context, in ResendPhoneInput) error
	OAuthCallback(ctx context.Context, in OAuthCallbackInput) (interface{}, error)
	CompleteOAuthSignup(ctx context.Context, in OAuthCompleteInput) (*PhoneVerificationRequiredResponse, error)
	Me(ctx context.Context, userID uuid.UUID) (*User, error)
	Logout(ctx context.Context, userID uuid.UUID) error
}

type service struct {
	cfg    *config.Config
	repo   Repository
	events *events.Publisher
}

func NewService(cfg *config.Config, repo Repository, pub *events.Publisher) Service {
	return &service{cfg: cfg, repo: repo, events: pub}
}

// ── Register ─────────────────────────────────────────────────────

func (s *service) Register(
	ctx context.Context, in RegisterInput,
) (*PhoneVerificationRequiredResponse, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hash := string(hashBytes)

	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	phone := strings.TrimSpace(in.Phone)

	// Look up any existing row by email. An in-progress (unverified)
	// signup is allowed to re-submit to correct typos; a verified user
	// must use the login flow.
	existing, err := s.repo.GetUserByEmail(ctx, email)
	emailExists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if emailExists && existing.PhoneVerifiedAt.Valid {
		return nil, ErrEmailTaken
	}

	// Reject phone reuse by ANY user other than the in-progress signup
	// we're about to update. (Verified users own their phone; unverified
	// users have soft-reserved it.)
	phoneOwner, err := s.repo.GetUserByPhone(ctx, &phone)
	switch {
	case err == nil:
		if !emailExists || phoneOwner.ID != existing.ID {
			return nil, ErrPhoneTaken
		}
	case errors.Is(err, pgx.ErrNoRows):
		// phone is free
	default:
		return nil, err
	}

	if emailExists {
		updated, err := s.repo.UpdateUnverifiedUser(
			ctx, sqlc.UpdateUnverifiedUserParams{
				ID:           existing.ID,
				PasswordHash: hash,
				Role:         string(in.Role),
				FirstName:    &firstName,
				LastName:     &lastName,
				Phone:        &phone,
			},
		)
		if err != nil {
			return nil, err
		}
		return s.issuePhoneVerification(ctx, updated)
	}

	user, err := s.repo.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		Role:         string(in.Role),
		FirstName:    &firstName,
		LastName:     &lastName,
		Phone:        &phone,
	})
	if err != nil {
		return nil, err
	}

	return s.issuePhoneVerification(ctx, user)
}

// ── Login ────────────────────────────────────────────────────────
//
// Returns either *AuthenticatedResponse OR *PhoneVerificationRequiredResponse
// depending on whether the user has already verified their phone.
//
// (The mixed return is wrapped as interface{} so the handler can JSON-encode
// whichever payload comes back; the discriminator is the `status` field.)

func (s *service) Login(ctx context.Context, in LoginInput) (interface{}, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash), []byte(in.Password),
	); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Always require phone verification on email login (per product spec).
	return s.issuePhoneVerification(ctx, user)
}

// ── Phone verification issuance ──────────────────────────────────

func (s *service) issuePhoneVerification(
	ctx context.Context, user sqlc.User,
) (*PhoneVerificationRequiredResponse, error) {
	if user.Phone == nil || *user.Phone == "" {
		return nil, errors.New("user has no phone on file")
	}

	if err := s.repo.RevokeActivePhoneVerificationsForUser(ctx, user.ID); err != nil {
		return nil, err
	}

	code, err := generateNumericCode(s.cfg.PhoneCodeLength)
	if err != nil {
		return nil, err
	}
	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	expires := pgtype.Timestamptz{
		Time:  time.Now().Add(s.cfg.PhoneCodeTTL),
		Valid: true,
	}

	verification, err := s.repo.CreatePhoneVerification(
		ctx, sqlc.CreatePhoneVerificationParams{
			UserID:    user.ID,
			Phone:     *user.Phone,
			CodeHash:  string(codeHash),
			ExpiresAt: expires,
		},
	)
	if err != nil {
		return nil, err
	}

	// Per product spec: log to stdout instead of sending via Vonage.
	// Swap with a real SMS provider call once VONAGE_API_KEY is set.
	log.Printf(
		"\n"+
			"✅ ═════════════════════════════════════════════════════ ✅\n"+
			"   ✅✅✅  PHONE VERIFICATION CODE: %s  ✅✅✅\n"+
			"✅ ═════════════════════════════════════════════════════ ✅\n"+
			"      user:  %s\n"+
			"      phone: %s\n"+
			"      ttl:   %s (expires %s)\n",
		code,
		user.ID,
		*user.Phone,
		s.cfg.PhoneCodeTTL,
		expires.Time.Format(time.RFC3339),
	)

	return &PhoneVerificationRequiredResponse{
		Status:         "phone_verification_required",
		VerificationID: verification.ID.String(),
		MaskedPhone:    maskPhone(*user.Phone),
	}, nil
}

// ── Verify phone code ────────────────────────────────────────────

func (s *service) VerifyPhone(
	ctx context.Context, in VerifyPhoneInput,
) (*AuthenticatedResponse, error) {
	verificationID, err := uuid.Parse(in.VerificationID)
	if err != nil {
		return nil, ErrPhoneVerificationNotFound
	}

	verification, err := s.repo.GetActivePhoneVerification(ctx, verificationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPhoneVerificationNotFound
		}
		return nil, err
	}

	if int(verification.Attempts) >= s.cfg.PhoneMaxAttempts {
		_ = s.repo.ConsumePhoneVerification(ctx, verification.ID)
		return nil, ErrTooManyVerificationAttempts
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(verification.CodeHash), []byte(in.Code),
	); err != nil {
		_ = s.repo.IncrementPhoneVerificationAttempts(ctx, verification.ID)
		return nil, ErrInvalidVerificationCode
	}

	if err := s.repo.ConsumePhoneVerification(ctx, verification.ID); err != nil {
		return nil, err
	}
	if err := s.repo.MarkPhoneVerified(ctx, verification.UserID); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, verification.UserID)
	if err != nil {
		return nil, err
	}

	token, err := s.issueAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthenticatedResponse{
		Status: "authenticated",
		Token:  token,
		User:   toAPIUser(user),
	}, nil
}

// ── Resend phone code ────────────────────────────────────────────
//
// Rotates the code on the EXISTING verification rather than creating a
// new row. This keeps the verificationId stable on the frontend so the
// user's next submit still finds the active verification.

func (s *service) ResendPhoneCode(
	ctx context.Context, in ResendPhoneInput,
) error {
	verificationID, err := uuid.Parse(in.VerificationID)
	if err != nil {
		return ErrPhoneVerificationNotFound
	}
	verification, err := s.repo.GetActivePhoneVerification(ctx, verificationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPhoneVerificationNotFound
		}
		return err
	}

	code, err := generateNumericCode(s.cfg.PhoneCodeLength)
	if err != nil {
		return err
	}
	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	expires := pgtype.Timestamptz{
		Time:  time.Now().Add(s.cfg.PhoneCodeTTL),
		Valid: true,
	}

	if err := s.repo.RefreshPhoneVerificationCode(
		ctx, sqlc.RefreshPhoneVerificationCodeParams{
			ID:        verification.ID,
			CodeHash:  string(codeHash),
			ExpiresAt: expires,
		},
	); err != nil {
		return err
	}

	log.Printf(
		"\n"+
			"♻️ ═════════════════════════════════════════════════════ ♻️\n"+
			"   ♻️♻️♻️  RESENT PHONE VERIFICATION CODE: %s  ♻️♻️♻️\n"+
			"♻️ ═════════════════════════════════════════════════════ ♻️\n"+
			"      verification: %s\n"+
			"      phone:        %s\n"+
			"      ttl:          %s (expires %s)\n",
		code,
		verification.ID,
		verification.Phone,
		s.cfg.PhoneCodeTTL,
		expires.Time.Format(time.RFC3339),
	)
	return nil
}

// ── OAuth ────────────────────────────────────────────────────────
//
// Scaffolded shape. Real provider integration (token exchange + id_token
// verification + user upsert) lands when credentials are available.

// OAuthCallback returns one of two response shapes:
//   - AuthenticatedResponse                  → email already maps to a full user, just log them in
//   - OAuthCompleteProfileRequiredResponse   → new user; frontend prefills the signup form and
//                                              calls CompleteOAuthSignup once they pick a password/phone/role
func (s *service) OAuthCallback(
	ctx context.Context, in OAuthCallbackInput,
) (interface{}, error) {
	switch in.Provider {
	case "google":
		if s.cfg.GoogleClientID == "" || s.cfg.GoogleClientSecret == "" {
			return nil, ErrProviderNotConfigured
		}
	case "microsoft":
		if s.cfg.MicrosoftClientID == "" || s.cfg.MicrosoftClientSecret == "" {
			return nil, ErrProviderNotConfigured
		}
	default:
		return nil, ErrProviderNotConfigured
	}

	claims, err := exchangeOAuthCode(ctx, s.cfg, in.Provider, in.Code, in.RedirectURI)
	if err != nil {
		log.Printf("[oauth] %s token exchange failed: %v", in.Provider, err)
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByEmail(ctx, claims.Email)
	switch {
	case err == nil:
		// Existing fully-provisioned user — log them in directly.
		token, err := s.issueAccessToken(user)
		if err != nil {
			return nil, err
		}
		return &AuthenticatedResponse{
			Status: "authenticated",
			Token:  token,
			User:   toAPIUser(user),
		}, nil
	case errors.Is(err, pgx.ErrNoRows):
		// First-time OAuth signup. Sign a short-lived session that
		// carries the verified claims; the frontend echoes it back to
		// CompleteOAuthSignup with the rest of the profile.
		sessionTok, err := signOAuthSession(s.cfg, claims, in.Provider)
		if err != nil {
			return nil, err
		}
		return &OAuthCompleteProfileRequiredResponse{
			Status:       "oauth_complete_profile_required",
			OAuthSession: sessionTok,
			Email:        claims.Email,
			FirstName:    claims.FirstName,
			LastName:     claims.LastName,
		}, nil
	default:
		return nil, err
	}
}

// CompleteOAuthSignup finalizes a first-time OAuth user: validates the
// session token issued by OAuthCallback, creates the row with the
// password/phone/role they just entered, and starts the phone OTP flow.
func (s *service) CompleteOAuthSignup(
	ctx context.Context, in OAuthCompleteInput,
) (*PhoneVerificationRequiredResponse, error) {
	claims, err := verifyOAuthSession(s.cfg, in.OAuthSession)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Email is taken from the signed OAuth session token (unspoofable);
	// first/last name are user-editable on the prefilled signup form so
	// we trust the form-submitted values.
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	phone := strings.TrimSpace(in.Phone)
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	_ = claims // .Subject still useful once oauth_identities table lands

	hashBytes, err := bcrypt.GenerateFromPassword(
		[]byte(in.Password), bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}
	hash := string(hashBytes)

	// Same email/phone uniqueness rules as plain signup.
	existing, err := s.repo.GetUserByEmail(ctx, email)
	emailExists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if emailExists && existing.PhoneVerifiedAt.Valid {
		return nil, ErrEmailTaken
	}

	phoneOwner, err := s.repo.GetUserByPhone(ctx, &phone)
	switch {
	case err == nil:
		if !emailExists || phoneOwner.ID != existing.ID {
			return nil, ErrPhoneTaken
		}
	case errors.Is(err, pgx.ErrNoRows):
		// free
	default:
		return nil, err
	}

	if emailExists {
		updated, err := s.repo.UpdateUnverifiedUser(
			ctx, sqlc.UpdateUnverifiedUserParams{
				ID:           existing.ID,
				PasswordHash: hash,
				Role:         string(in.Role),
				FirstName:    &firstName,
				LastName:     &lastName,
				Phone:        &phone,
			},
		)
		if err != nil {
			return nil, err
		}
		return s.issuePhoneVerification(ctx, updated)
	}

	user, err := s.repo.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		Role:         string(in.Role),
		FirstName:    &firstName,
		LastName:     &lastName,
		Phone:        &phone,
	})
	if err != nil {
		return nil, err
	}
	return s.issuePhoneVerification(ctx, user)
}

// ── Me / Logout ──────────────────────────────────────────────────

func (s *service) Me(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u := toAPIUser(user)
	return &u, nil
}

func (s *service) Logout(ctx context.Context, userID uuid.UUID) error {
	// Refresh-token rotation isn't wired up on the frontend yet; once it is,
	// revoke all refresh tokens for the user here.
	_ = userID
	return nil
}

// ── JWT issuance ─────────────────────────────────────────────────

func (s *service) issueAccessToken(u sqlc.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  u.ID.String(),
		"role": u.Role,
		"iat":  now.Unix(),
		"exp":  now.Add(s.cfg.JWTAccessTTL).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.cfg.JWTSecret))
}

// ── Helpers ──────────────────────────────────────────────────────

func generateNumericCode(length int) (string, error) {
	const digits = "0123456789"
	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		out[i] = digits[n.Int64()]
	}
	return string(out), nil
}

// maskPhone returns a "•••• ••• 1234" style for a phone number, preserving
// only the last 4 digits. Non-digits are stripped before masking.
func maskPhone(phone string) string {
	digits := make([]byte, 0, len(phone))
	for i := 0; i < len(phone); i++ {
		c := phone[i]
		if c >= '0' && c <= '9' {
			digits = append(digits, c)
		}
	}
	if len(digits) <= 4 {
		return "•••• ••• ••••"
	}
	return "•••• ••• " + string(digits[len(digits)-4:])
}
