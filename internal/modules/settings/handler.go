package settings

import (
	"errors"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetSettings(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	resp, err := h.svc.GetSettings(c.Request.Context(), userID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

// ── Account ──────────────────────────────────────────────────────────────
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	account, err := h.svc.UpdateProfile(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, account)
}

func (h *Handler) ChangeEmail(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input ChangeEmailInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	account, err := h.svc.ChangeEmail(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, account)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), userID, input); err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) PresignAvatar(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input PresignUploadInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.PresignAvatarUpload(c.Request.Context(), userID, input.ContentType)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	httpx.NotImplemented(c)
}

// ── Onboarding ──────────────────────────────────────────────────────────────
func (h *Handler) CompleteOnboarding(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input OnboardingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", onboardingBindMessage(err))
		return
	}
	resp, err := h.svc.CompleteOnboarding(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) CheckUsername(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	resp, err := h.svc.CheckUsernameAvailable(c.Request.Context(), userID, c.Query("username"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

// ── Artist business config ─────────────────────────────────────────────────
func (h *Handler) UpdateSettings(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input UpdateSettingsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	settings, err := h.svc.UpdateSettings(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, settings)
}

func (h *Handler) SetAvailability(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input SetAvailabilityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	windows, err := h.svc.SetAvailability(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, gin.H{"windows": windows})
}

func (h *Handler) PresignWaiver(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input PresignUploadInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.PresignWaiverUpload(c.Request.Context(), userID, input.ContentType)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) CreatePreset(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input CreatePresetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	preset, err := h.svc.CreatePreset(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.Created(c, preset)
}

func (h *Handler) UpdatePreset(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	presetID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var input UpdatePresetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	preset, err := h.svc.UpdatePreset(c.Request.Context(), userID, presetID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, preset)
}

func (h *Handler) DeletePreset(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	presetID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.DeletePreset(c.Request.Context(), userID, presetID); err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) AddDayOff(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input DayOffInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	if err := h.svc.AddDayOff(c.Request.Context(), userID, input); err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) RemoveDayOff(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := h.svc.RemoveDayOff(c.Request.Context(), userID, c.Param("day")); err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) AddBlocklistEntry(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input CreateBlocklistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	entry, err := h.svc.AddBlocklistEntry(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.Created(c, entry)
}

func (h *Handler) RemoveBlocklistEntry(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	entryID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.RemoveBlocklistEntry(c.Request.Context(), userID, entryID); err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ConnectStripe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	resp, err := h.svc.ConnectStripe(c.Request.Context(), userID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) RefreshStripeStatus(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	settings, err := h.svc.RefreshStripeStatus(c.Request.Context(), userID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, settings)
}

func (h *Handler) DisconnectStripe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	settings, err := h.svc.DisconnectStripe(c.Request.Context(), userID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, settings)
}

// StripeWebhook is a public endpoint (no JWT) — authenticity is established by
// the Stripe-Signature header, verified in the service against the webhook
// secret. We read the raw body since signature verification needs the exact
// bytes Stripe signed.
func (h *Handler) StripeWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		httpx.Error(c, 400, "invalid_request", "could not read request body")
		return
	}
	if err := h.svc.HandleStripeWebhook(c.Request.Context(), payload, c.GetHeader("Stripe-Signature")); err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, gin.H{"received": true})
}

func (h *Handler) ConnectGoogleCalendar(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input ConnectGoogleCalendarInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	settings, err := h.svc.ConnectGoogleCalendar(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, settings)
}

func (h *Handler) DisconnectGoogleCalendar(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	settings, err := h.svc.DisconnectGoogleCalendar(c.Request.Context(), userID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, settings)
}

func requireUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, 401, "unauthorized", "missing user context")
		return uuid.Nil, false
	}
	return userID, true
}

func parseIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httpx.Error(c, 400, "invalid_id", "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

var onboardingFieldLabels = map[string]string{
	"Username":         "username",
	"InstagramURL":     "Instagram handle",
	"StudioName":       "studio name",
	"StudioAddress":    "studio address",
	"StudioCity":       "city",
	"StudioProvince":   "province",
	"StudioPostalCode": "postal code",
	"StudioCountry":    "country",
	"Timezone":         "timezone",
	"SchedulingMode":   "scheduling option",
}

func onboardingBindMessage(err error) string {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) || len(verrs) == 0 {
		return "Some details look incorrect. Please review and try again."
	}

	fe := verrs[0]
	label := onboardingFieldLabels[fe.Field()]
	if label == "" {
		label = "details"
	}

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("Please enter your %s.", label)
	case "url":
		return "That doesn't look like a valid Instagram handle."
	case "oneof":
		return "Please choose how you want to schedule."
	case "min", "max":
		if fe.Field() == "Username" {
			return "Your username must be 3–30 characters."
		}
		return fmt.Sprintf("Please check your %s.", label)
	default:
		return fmt.Sprintf("Please check your %s.", label)
	}
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, 404, "not_found", err.Error())
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, 403, "forbidden", err.Error())
	case errors.Is(err, ErrArtistMissing):
		httpx.Error(c, 403, "no_artist_profile", err.Error())
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(c, 409, "email_taken", err.Error())
	case errors.Is(err, ErrUsernameTaken):
		httpx.Error(c, 409, "username_taken", err.Error())
	case errors.Is(err, ErrPhoneTaken):
		httpx.Error(c, 409, "phone_taken", err.Error())
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, 400, "invalid_input", err.Error())
	case errors.Is(err, ErrOAuthExchange):
		httpx.Error(c, 400, "oauth_exchange_failed", err.Error())
	case errors.Is(err, ErrStripeAPI):
		httpx.Error(c, 502, "stripe_error", err.Error())
	case errors.Is(err, ErrIntegrationConfig):
		httpx.Error(c, 501, "integration_not_configured", err.Error())
	default:
		httpx.Error(c, 500, "internal_error", err.Error())
	}
}
