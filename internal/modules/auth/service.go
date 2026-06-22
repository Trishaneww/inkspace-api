package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"log/slog"
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
	"github.com/trishaneupnexx/inkspace-api/internal/messaging"
	"github.com/trishaneupnexx/inkspace-api/internal/ratelimit"
)

var (
	ErrInvalidCredentials          = errors.New("invalid credentials")
	ErrEmailTaken                  = errors.New("email already in use")
	ErrEmailUsedWithOtherProvider  = errors.New("This email is already registered. Please sign in using the method you originally signed up with.")
	ErrPhoneTaken                  = errors.New("phone number already in use")
	ErrPhoneVerificationNotFound   = errors.New("phone verification not found or expired")
	ErrInvalidVerificationCode     = errors.New("invalid verification code")
	ErrTooManyVerificationAttempts = errors.New("too many verification attempts")
	ErrTooManyLoginAttempts        = errors.New("too many failed login attempts; try again later")
	ErrTooManyOTPRequests          = errors.New("too many verification codes requested; try again later")
	ErrProviderNotConfigured       = errors.New("oauth provider not configured")
	ErrProviderNotImplemented      = errors.New("oauth provider integration not implemented")
	ErrInvalidRefreshToken         = errors.New("invalid or expired refresh token")
	ErrUsernameRequired            = errors.New("a username is required for artist accounts")
	ErrUsernameTaken               = errors.New("username already in use")
)

type Service interface {
	Register(ctx context.Context, in RegisterInput) (*PhoneVerificationRequiredResponse, error)
	Login(ctx context.Context, in LoginInput) (interface{}, error)
	VerifyPhone(ctx context.Context, in VerifyPhoneInput) (*AuthenticatedResponse, error)
	ResendPhoneCode(ctx context.Context, in ResendPhoneInput) error
	OAuthCallback(ctx context.Context, in OAuthCallbackInput) (interface{}, error)
	CompleteOAuthSignup(ctx context.Context, in OAuthCompleteInput) (*PhoneVerificationRequiredResponse, error)
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (*User, error)
	RefreshSession(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, userID uuid.UUID) error
}

type service struct {
	cfg    *config.Config
	repo   Repository
	events *events.Publisher

	loginLockout *ratelimit.Lockout
	otpLimiter   ratelimit.Limiter
	sms          messaging.Sender
	log          *slog.Logger
}

func NewService(
	cfg *config.Config,
	repo Repository,
	pub *events.Publisher,
	loginLockout *ratelimit.Lockout,
	otpLimiter ratelimit.Limiter,
) Service {
	log := slog.Default()
	return &service{
		cfg:          cfg,
		repo:         repo,
		events:       pub,
		loginLockout: loginLockout,
		otpLimiter:   otpLimiter,
		sms:          messaging.NewSender(cfg, log),
		log:          log,
	}
}

func (s *service) sendOTP(ctx context.Context, phone, code string) {
	body := fmt.Sprintf("Your Inkspace verification code is %s", code)
	if err := s.sms.Send(ctx, phone, body); err != nil {
		s.log.Warn("otp_send_failed", "error", err)
	}
}

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
	var username, instagram *string

	existing, err := s.repo.GetUserByEmail(ctx, email)
	emailExists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if emailExists && existing.PhoneVerifiedAt.Valid {
		return nil, ErrEmailTaken
	}

	phoneOwner, err := s.repo.GetUserByPhone(ctx, phone)
	switch {
	case err == nil:
		if !emailExists || phoneOwner.ID != existing.ID {
			return nil, ErrPhoneTaken
		}
	case errors.Is(err, pgx.ErrNoRows):
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
				Phone:        phone,
				Username:     username,
				InstagramURL: instagram,
				AuthProvider: ProviderPassword,
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
		Phone:        phone,
		Username:     username,
		InstagramURL: instagram,
		AuthProvider: ProviderPassword,
	})
	if err != nil {
		return nil, err
	}

	return s.issuePhoneVerification(ctx, user)
}

// ── Login ────────────────────────────────────────────────────────
func (s *service) Login(ctx context.Context, in LoginInput) (interface{}, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	if locked, err := s.isLoginLocked(ctx, email); err != nil {
		s.log.Warn("login_lockout_check_failed", "error", err) // fail open
	} else if locked {
		return nil, ErrTooManyLoginAttempts
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.recordLoginFailure(ctx, email)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash), []byte(in.Password),
	); err != nil {
		s.recordLoginFailure(ctx, email)
		return nil, ErrInvalidCredentials
	}

	s.resetLoginFailures(ctx, email)

	return s.issuePhoneVerification(ctx, user)
}

func loginFailKey(email string) string { return "auth:login_fail:" + email }
func otpSendKey(phone string) string   { return "auth:otp_send:" + phone }

func (s *service) isLoginLocked(ctx context.Context, email string) (bool, error) {
	if s.loginLockout == nil {
		return false, nil
	}
	return s.loginLockout.Locked(ctx, loginFailKey(email))
}

func (s *service) recordLoginFailure(ctx context.Context, email string) {
	if s.loginLockout == nil {
		return
	}
	if _, err := s.loginLockout.RecordFailure(ctx, loginFailKey(email)); err != nil {
		s.log.Warn("login_lockout_record_failed", "error", err)
	}
}

func (s *service) resetLoginFailures(ctx context.Context, email string) {
	if s.loginLockout == nil {
		return
	}
	if err := s.loginLockout.Reset(ctx, loginFailKey(email)); err != nil {
		s.log.Warn("login_lockout_reset_failed", "error", err)
	}
}

func (s *service) otpSendAllowed(ctx context.Context, phone string) bool {
	if s.otpLimiter == nil {
		return true
	}
	res, err := s.otpLimiter.CheckLimit(ctx, otpSendKey(phone))
	if err != nil {
		s.log.Warn("otp_send_limit_check_failed", "error", err)
		return true // fail open
	}
	return res.Allowed
}

// ── Phone verification ──────────────────────────────────
func (s *service) issuePhoneVerification(
	ctx context.Context, user sqlc.User,
) (*PhoneVerificationRequiredResponse, error) {
	if user.Phone == "" {
		return nil, errors.New("user has no phone on file")
	}

	if !s.otpSendAllowed(ctx, user.Phone) {
		return nil, ErrTooManyOTPRequests
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
			Phone:     user.Phone,
			CodeHash:  string(codeHash),
			ExpiresAt: expires,
		},
	)
	if err != nil {
		return nil, err
	}

	s.sendOTP(ctx, user.Phone, code)

	return &PhoneVerificationRequiredResponse{
		Status:         "phone_verification_required",
		VerificationID: verification.ID.String(),
		MaskedPhone:    maskPhone(user.Phone),
	}, nil
}

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

	if Role(user.Role) == RoleArtist {
		if err := s.repo.EnsureArtist(ctx, user.ID); err != nil {
			return nil, err
		}
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	respUser, err := s.buildUserResponse(ctx, user)
	if err != nil {
		return nil, err
	}

	return &AuthenticatedResponse{
		Status:       "authenticated",
		Token:        pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		User:         respUser,
	}, nil
}

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

	if !s.otpSendAllowed(ctx, verification.Phone) {
		return ErrTooManyOTPRequests
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

	s.sendOTP(ctx, verification.Phone, code)
	return nil
}

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
		// An email belongs to exactly one provider. If it was registered with a
		// password or a different OAuth provider, refuse rather than silently
		// signing into (or duplicating) the existing account.
		if user.AuthProvider != in.Provider {
			return nil, ErrEmailUsedWithOtherProvider
		}
		pair, err := s.issueTokenPair(ctx, user)
		if err != nil {
			return nil, err
		}
		respUser, err := s.buildUserResponse(ctx, user)
		if err != nil {
			return nil, err
		}
		return &AuthenticatedResponse{
			Status:       "authenticated",
			Token:        pair.AccessToken,
			RefreshToken: pair.RefreshToken,
			User:         respUser,
		}, nil
	case errors.Is(err, pgx.ErrNoRows):
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

func (s *service) CompleteOAuthSignup(
	ctx context.Context, in OAuthCompleteInput,
) (*PhoneVerificationRequiredResponse, error) {
	claims, err := verifyOAuthSession(s.cfg, in.OAuthSession)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	phone := strings.TrimSpace(in.Phone)
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	var username, instagram *string

	provider := claims.Provider
	if provider != ProviderGoogle && provider != ProviderMicrosoft {
		return nil, ErrInvalidCredentials
	}

	hashBytes, err := bcrypt.GenerateFromPassword(
		[]byte(in.Password), bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}
	hash := string(hashBytes)

	existing, err := s.repo.GetUserByEmail(ctx, email)
	emailExists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if emailExists && existing.PhoneVerifiedAt.Valid {
		return nil, ErrEmailTaken
	}
	// An unverified row from a password signup or a different provider must not
	// be silently taken over by this provider.
	if emailExists && existing.AuthProvider != provider {
		return nil, ErrEmailUsedWithOtherProvider
	}

	phoneOwner, err := s.repo.GetUserByPhone(ctx, phone)
	switch {
	case err == nil:
		if !emailExists || phoneOwner.ID != existing.ID {
			return nil, ErrPhoneTaken
		}
	case errors.Is(err, pgx.ErrNoRows):
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
				Phone:        phone,
				Username:     username,
				InstagramURL: instagram,
				AuthProvider: provider,
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
		Phone:        phone,
		Username:     username,
		InstagramURL: instagram,
		AuthProvider: provider,
	})
	if err != nil {
		return nil, err
	}
	return s.issuePhoneVerification(ctx, user)
}

func (s *service) buildUserResponse(ctx context.Context, user sqlc.User) (User, error) {
	u := userFromRecord(user)
	if Role(user.Role) != RoleArtist {
		return u, nil
	}

	onboardedAt, err := s.repo.GetArtistOnboardedAt(ctx, user.ID)
	switch {
	case err == nil:
		if onboardedAt.Valid {
			ts := onboardedAt.Time.UTC().Format(time.RFC3339)
			u.OnboardedAt = &ts
		}
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return User{}, err
	}
	return u, nil
}

func (s *service) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u, err := s.buildUserResponse(ctx, user)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *service) RefreshSession(ctx context.Context, refreshToken string) (*TokenPair, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrInvalidRefreshToken
	}

	hash := hashRefreshToken(refreshToken)
	existing, err := s.repo.GetActiveRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, existing.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	if err := s.repo.RevokeRefreshToken(ctx, hash); err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, user)
}

func (s *service) Logout(ctx context.Context, userID uuid.UUID) error {
	return s.repo.RevokeAllRefreshTokensForUser(ctx, userID)
}

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

func (s *service) issueTokenPair(ctx context.Context, u sqlc.User) (*TokenPair, error) {
	access, err := s.issueAccessToken(u)
	if err != nil {
		return nil, err
	}

	raw, err := generateRefreshTokenString()
	if err != nil {
		return nil, err
	}

	_, err = s.repo.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    u.ID,
		TokenHash: hashRefreshToken(raw),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(s.cfg.JWTRefreshTTL), Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: access, RefreshToken: raw}, nil
}

func generateRefreshTokenString() (string, error) {
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
