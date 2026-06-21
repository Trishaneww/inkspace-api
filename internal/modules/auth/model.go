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

const (
	ProviderPassword  = "password"
	ProviderGoogle    = "google"
	ProviderMicrosoft = "microsoft"
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
	OnboardedAt     *string `json:"onboardedAt,omitempty"`
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
	out.Phone = u.Phone
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
}

// ── Response shapes ──────────────────────────────────────────────
type AuthenticatedResponse struct {
	Status       string `json:"status"`
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	User         User   `json:"user"`
}

type PhoneVerificationRequiredResponse struct {
	Status         string `json:"status"`
	VerificationID string `json:"verificationId"`
	MaskedPhone    string `json:"maskedPhone"`
}

type OAuthCompleteProfileRequiredResponse struct {
	Status       string `json:"status"`
	OAuthSession string `json:"oauthSession"`
	Email        string `json:"email"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
