package auth

import (
	"time"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Role string

const (
	RoleArtist      Role = "artist"
	RoleStudioAdmin Role = "studio_admin"
	RoleUser        Role = "user"
)

type User struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	Role            Role    `json:"role"`
	FirstName       string  `json:"firstName"`
	LastName        string  `json:"lastName"`
	Phone           string  `json:"phone,omitempty"`
	AvatarURL       string  `json:"avatarUrl,omitempty"`
	PhoneVerifiedAt *string `json:"phoneVerifiedAt,omitempty"`
	CreatedAt       string  `json:"createdAt"`
}

func userFromRecord(u sqlc.User) User {
	out := User{
		ID:    u.ID.String(),
		Email: u.Email,
		Role:  Role(u.Role),
	}
	if u.FirstName != nil {
		out.FirstName = *u.FirstName
	}
	if u.LastName != nil {
		out.LastName = *u.LastName
	}
	if u.Phone != nil {
		out.Phone = *u.Phone
	}
	if u.CreatedAt.Valid {
		out.CreatedAt = u.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	if u.PhoneVerifiedAt.Valid {
		s := u.PhoneVerifiedAt.Time.UTC().Format(time.RFC3339)
		out.PhoneVerifiedAt = &s
	}
	return out
}

// ── Request payloads ─────────────────────────────────────────────

type RegisterInput struct {
	FirstName string `json:"firstName" binding:"required,min=1,max=50"`
	LastName  string `json:"lastName"  binding:"required,min=1,max=50"`
	Email     string `json:"email"     binding:"required,email"`
	Phone     string `json:"phone"     binding:"required,min=7,max=20"`
	Password  string `json:"password"  binding:"required,min=8"`
	Role      Role   `json:"role"      binding:"required,oneof=artist studio_admin user"`
	// Username is required for artists (it backs their public booking link) and
	// optional otherwise; enforced in the service since it's role-dependent.
	Username     string `json:"username"     binding:"omitempty,min=3,max=30"`
	InstagramURL string `json:"instagramUrl" binding:"omitempty,url,max=255"`
}

type LoginInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type VerifyPhoneInput struct {
	VerificationID string `json:"verificationId" binding:"required,uuid"`
	Code           string `json:"code"           binding:"required,len=6,numeric"`
}

type ResendPhoneInput struct {
	VerificationID string `json:"verificationId" binding:"required,uuid"`
}

type OAuthCallbackInput struct {
	Provider    string `json:"provider"    binding:"required,oneof=google microsoft"`
	Code        string `json:"code"        binding:"required"`
	RedirectURI string `json:"redirectUri" binding:"required"`
}

// Used by POST /v1/auth/oauth/complete when a first-time OAuth user
// submits the prefilled signup form with the remaining fields.
type RefreshInput struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type OAuthCompleteInput struct {
	OAuthSession string `json:"oauthSession" binding:"required"`
	FirstName    string `json:"firstName"    binding:"required,min=1,max=50"`
	LastName     string `json:"lastName"     binding:"required,min=1,max=50"`
	Password     string `json:"password"     binding:"required,min=8"`
	Phone        string `json:"phone"        binding:"required,min=7,max=20"`
	Role         Role   `json:"role"         binding:"required,oneof=artist studio_admin user"`
	Username     string `json:"username"     binding:"omitempty,min=3,max=30"`
	InstagramURL string `json:"instagramUrl" binding:"omitempty,url,max=255"`
}

// ── Response shapes ──────────────────────────────────────────────

type AuthenticatedResponse struct {
	Status       string `json:"status"` // always "authenticated"
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	User         User   `json:"user"`
}

type PhoneVerificationRequiredResponse struct {
	Status         string `json:"status"` // always "phone_verification_required"
	VerificationID string `json:"verificationId"`
	MaskedPhone    string `json:"maskedPhone"`
}

// Returned by OAuthCallback when the OAuth identity doesn't match an
// existing user. The frontend prefills the signup form with email/name,
// asks the user for password/phone/role, and POSTs the captured session
// token back to /v1/auth/oauth/complete to finish the account.
type OAuthCompleteProfileRequiredResponse struct {
	Status       string `json:"status"` // always "oauth_complete_profile_required"
	OAuthSession string `json:"oauthSession"`
	Email        string `json:"email"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
