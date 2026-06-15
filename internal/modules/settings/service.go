package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/crypto"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrForbidden         = errors.New("not allowed")
	ErrInvalidInput      = errors.New("invalid input")
	ErrArtistMissing     = errors.New("user has no artist profile")
	ErrEmailTaken        = errors.New("email already in use")
	ErrUsernameTaken     = errors.New("username already in use")
	ErrSlugTaken         = errors.New("that booking link is already taken")
	ErrPhoneTaken        = errors.New("phone number already in use")
	ErrNotImplemented    = errors.New("not implemented")
	ErrIntegrationConfig = errors.New("integration not configured on this server")
	ErrOAuthExchange     = errors.New("could not complete the connection with the provider")
	ErrStripeAPI         = errors.New("could not complete the request with Stripe")
)

const (
	presignUploadTTL = 15 * time.Minute
	presignViewTTL   = 1 * time.Hour

	dayLayout = "2006-01-02"
)

var avatarContentTypeToExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

var waiverContentTypeToExt = map[string]string{
	"application/pdf": "pdf",
	"image/jpeg":      "jpg",
	"image/png":       "png",
	"image/webp":      "webp",
}

type Service interface {
	GetSettings(ctx context.Context, userID uuid.UUID) (SettingsResponse, error)

	// Account
	UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (Account, error)
	ChangeEmail(ctx context.Context, userID uuid.UUID, input ChangeEmailInput) (Account, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, input ChangePasswordInput) error
	PresignAvatarUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error)

	// Onboarding
	CompleteOnboarding(ctx context.Context, userID uuid.UUID, input OnboardingInput) (OnboardingResponse, error)
	CheckUsernameAvailable(ctx context.Context, userID uuid.UUID, username string) (UsernameAvailabilityResponse, error)

	// Open Book
	GetOpenBook(ctx context.Context, userID uuid.UUID) (OpenBookResponse, error)
	UpdateOpenBook(ctx context.Context, userID uuid.UUID, input UpdateOpenBookInput) (OpenBookResponse, error)

	// Artist business config
	UpdateSettings(ctx context.Context, userID uuid.UUID, input UpdateSettingsInput) (ArtistSettings, error)
	SetAvailability(ctx context.Context, userID uuid.UUID, input SetAvailabilityInput) ([]AvailabilityWindow, error)
	PresignWaiverUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error)

	// Locations (home studio + guest spots)
	CreateLocation(ctx context.Context, userID uuid.UUID, input CreateLocationInput) (Location, error)
	UpdateLocation(ctx context.Context, userID, locationID uuid.UUID, input UpdateLocationInput) (Location, error)
	DeleteLocation(ctx context.Context, userID, locationID uuid.UUID) error
	SetCurrentLocation(ctx context.Context, userID uuid.UUID, input SetCurrentLocationInput) error

	// Integrations
	ConnectStripe(ctx context.Context, userID uuid.UUID) (StripeConnectResponse, error)
	RefreshStripeStatus(ctx context.Context, userID uuid.UUID) (ArtistSettings, error)
	DisconnectStripe(ctx context.Context, userID uuid.UUID) (ArtistSettings, error)
	HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error

	ConnectGoogleCalendar(ctx context.Context, userID uuid.UUID, input ConnectGoogleCalendarInput) (ArtistSettings, error)
	DisconnectGoogleCalendar(ctx context.Context, userID uuid.UUID) (ArtistSettings, error)

	CreatePreset(ctx context.Context, userID uuid.UUID, input CreatePresetInput) (SessionPreset, error)
	UpdatePreset(ctx context.Context, userID, presetID uuid.UUID, input UpdatePresetInput) (SessionPreset, error)
	DeletePreset(ctx context.Context, userID, presetID uuid.UUID) error

	AddDayOff(ctx context.Context, userID uuid.UUID, input DayOffInput) error
	RemoveDayOff(ctx context.Context, userID uuid.UUID, day string) error

	AddBlocklistEntry(ctx context.Context, userID uuid.UUID, input CreateBlocklistInput) (BlocklistEntry, error)
	RemoveBlocklistEntry(ctx context.Context, userID, entryID uuid.UUID) error
}

type service struct {
	cfg    *config.Config
	repo   Repository
	s3     *s3client.Client
	cipher *crypto.Cipher
	log    *slog.Logger
}

func NewService(cfg *config.Config, repo Repository, s3 *s3client.Client, cipher *crypto.Cipher) Service {
	return &service{cfg: cfg, repo: repo, s3: s3, cipher: cipher, log: slog.Default()}
}

func (s *service) GetSettings(ctx context.Context, userID uuid.UUID) (SettingsResponse, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return SettingsResponse{}, err
	}
	if err := s.repo.EnsureArtistSettings(ctx, artist.ID); err != nil {
		return SettingsResponse{}, fmt.Errorf("ensure artist settings: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return SettingsResponse{}, err
	}
	srow, err := s.repo.GetArtistSettings(ctx, artist.ID)
	if err != nil {
		return SettingsResponse{}, err
	}
	windows, err := s.repo.ListAvailabilityWindows(ctx, artist.ID)
	if err != nil {
		return SettingsResponse{}, err
	}
	presets, err := s.repo.ListSessionPresets(ctx, artist.ID)
	if err != nil {
		return SettingsResponse{}, err
	}
	daysOff, err := s.repo.ListDaysOff(ctx, artist.ID)
	if err != nil {
		return SettingsResponse{}, err
	}
	blocklist, err := s.repo.ListBlocklist(ctx, artist.ID)
	if err != nil {
		return SettingsResponse{}, err
	}
	locations, err := s.repo.ListArtistLocations(ctx, artist.ID)
	if err != nil {
		return SettingsResponse{}, err
	}

	resp := SettingsResponse{
		Account:        s.accountResponse(ctx, user),
		Locations:      make([]Location, 0, len(locations)),
		Settings:       s.settingsResponse(ctx, srow),
		Availability:   make([]AvailabilityWindow, 0, len(windows)),
		SessionPresets: make([]SessionPreset, 0, len(presets)),
		DaysOff:        make([]string, 0, len(daysOff)),
		Blocklist:      make([]BlocklistEntry, 0, len(blocklist)),
	}
	for _, l := range locations {
		resp.Locations = append(resp.Locations, getLocationFromRow(l))
	}
	for _, w := range windows {
		resp.Availability = append(resp.Availability, availabilityFromRow(w))
	}
	for _, p := range presets {
		resp.SessionPresets = append(resp.SessionPresets, presetFromRow(p))
	}
	for _, d := range daysOff {
		resp.DaysOff = append(resp.DaysOff, d.Day.Time.Format(dayLayout))
	}
	for _, b := range blocklist {
		resp.Blocklist = append(resp.Blocklist, blocklistFromRow(b))
	}
	return resp, nil
}

// ── Account ──────────────────────────────────────────────────────────────
func (s *service) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (Account, error) {
	params := sqlc.UpdateUserProfileParams{ID: userID}

	params.FirstName = trimmedPtr(input.FirstName)
	params.LastName = trimmedPtr(input.LastName)
	params.AvatarURL = trimmedPtr(input.AvatarURL)
	params.InstagramURL = trimmedPtr(input.InstagramURL)

	if input.Username != nil {
		username := strings.TrimSpace(*input.Username)
		if username == "" {
			return Account{}, fmt.Errorf("%w: username cannot be empty", ErrInvalidInput)
		}
		if err := s.assertUsernameAvailable(ctx, username, userID); err != nil {
			return Account{}, err
		}
		params.Username = &username
	}
	if input.Phone != nil {
		phone := strings.TrimSpace(*input.Phone)
		if phone == "" {
			return Account{}, fmt.Errorf("%w: phone cannot be empty", ErrInvalidInput)
		}
		params.Phone = &phone
	}

	updated, err := s.repo.UpdateUserProfile(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return Account{}, ErrPhoneTaken
		}
		return Account{}, err
	}

	if input.Username != nil && params.Username != nil {
		if artist, aerr := s.repo.GetArtistByUserID(ctx, userID); aerr == nil {
			if _, oerr := s.repo.UpdateOpenBook(ctx, sqlc.UpdateOpenBookParams{
				ArtistID: artist.ID,
				Slug:     params.Username,
			}); oerr != nil && !errors.Is(oerr, pgx.ErrNoRows) {
				return Account{}, oerr
			}
		}
	}

	return s.accountResponse(ctx, updated), nil
}

func (s *service) ChangeEmail(ctx context.Context, userID uuid.UUID, input ChangeEmailInput) (Account, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return Account{}, err
	}

	newEmail := strings.ToLower(strings.TrimSpace(input.NewEmail))
	if newEmail == strings.ToLower(user.Email) {
		return s.accountResponse(ctx, user), nil
	}

	existing, err := s.repo.GetUserByEmail(ctx, newEmail)
	switch {
	case err == nil:
		if existing.ID != userID {
			return Account{}, ErrEmailTaken
		}
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return Account{}, err
	}

	updated, err := s.repo.UpdateUserEmail(ctx, sqlc.UpdateUserEmailParams{ID: userID, Email: newEmail})
	if err != nil {
		if isUniqueViolation(err) {
			return Account{}, ErrEmailTaken
		}
		return Account{}, err
	}
	return s.accountResponse(ctx, updated), nil
}

func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, input ChangePasswordInput) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{ID: userID, PasswordHash: string(hash)})
}

func (s *service) PresignAvatarUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error) {
	ext, ok := avatarContentTypeToExt[contentType]
	if !ok {
		return PresignUploadResponse{}, fmt.Errorf("%w: unsupported content type %q (allowed: image/jpeg, image/png, image/webp)", ErrInvalidInput, contentType)
	}
	key := fmt.Sprintf("avatars/%s/%s.%s", userID, uuid.New(), ext)
	return s.presignPut(ctx, key, contentType)
}

// ── Onboarding ──────────────────────────────────────────────────────────────
func (s *service) CheckUsernameAvailable(ctx context.Context, userID uuid.UUID, username string) (UsernameAvailabilityResponse, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 30 {
		return UsernameAvailabilityResponse{}, fmt.Errorf("%w: username must be 3–30 characters", ErrInvalidInput)
	}
	err := s.assertUsernameAvailable(ctx, username, userID)
	switch {
	case err == nil:
		return UsernameAvailabilityResponse{Username: username, Available: true}, nil
	case errors.Is(err, ErrUsernameTaken):
		return UsernameAvailabilityResponse{Username: username, Available: false}, nil
	default:
		return UsernameAvailabilityResponse{}, err
	}
}

func (s *service) CompleteOnboarding(ctx context.Context, userID uuid.UUID, input OnboardingInput) (OnboardingResponse, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return OnboardingResponse{}, err
	}
	if artist.OnboardedAt.Valid {
		return OnboardingResponse{}, fmt.Errorf("%w: onboarding already completed", ErrInvalidInput)
	}

	username := strings.TrimSpace(input.Username)
	if username == "" {
		return OnboardingResponse{}, fmt.Errorf("%w: username is required", ErrInvalidInput)
	}
	if err := s.assertUsernameAvailable(ctx, username, userID); err != nil {
		return OnboardingResponse{}, err
	}

	mode := SchedulingMode(input.SchedulingMode)
	if mode != SchedulingArtist && mode != SchedulingClient {
		return OnboardingResponse{}, fmt.Errorf("%w: schedulingMode must be artist_scheduled or client_scheduled", ErrInvalidInput)
	}

	if !allValidStyles(input.Styles) {
		return OnboardingResponse{}, fmt.Errorf("%w: unknown tattoo style", ErrInvalidInput)
	}

	for _, w := range input.Availability {
		if w.Weekday < 0 || w.Weekday > 6 {
			return OnboardingResponse{}, fmt.Errorf("%w: weekday must be 0..6", ErrInvalidInput)
		}
		if w.EndMinute <= w.StartMinute {
			return OnboardingResponse{}, fmt.Errorf("%w: a window's end must be after its start", ErrInvalidInput)
		}
	}

	instagram := nilIfEmpty(trimmedPtr(&input.InstagramURL))

	err = s.repo.RunInTx(ctx, func(tx Repository) error {
		if err := tx.EnsureArtistSettings(ctx, artist.ID); err != nil {
			return err
		}

		if _, err := tx.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
			ID:           userID,
			Username:     &username,
			InstagramURL: instagram,
		}); err != nil {
			return err
		}

		if _, err := tx.UpdateArtistSettings(ctx, sqlc.UpdateArtistSettingsParams{
			ArtistID:            artist.ID,
			Timezone:            trimmedPtr(&input.Timezone),
			DepositFlatFeeCents: input.DepositFlatFeeCents,
			Styles:              input.Styles,
		}); err != nil {
			return err
		}

		// The home studio is the artist's primary location, and the one they
		// start out working from.
		home, err := tx.CreateArtistLocation(ctx, sqlc.CreateArtistLocationParams{
			ArtistID:   artist.ID,
			Label:      strings.TrimSpace(input.StudioName),
			Address:    strings.TrimSpace(input.StudioAddress),
			City:       strings.TrimSpace(input.StudioCity),
			Province:   strings.TrimSpace(input.StudioProvince),
			PostalCode: strings.TrimSpace(input.StudioPostalCode),
			Country:    strings.TrimSpace(input.StudioCountry),
			Timezone:   strings.TrimSpace(input.Timezone),
			IsPrimary:  true,
		})
		if err != nil {
			return err
		}
		if err := tx.SetCurrentLocation(ctx, sqlc.SetCurrentLocationParams{
			ArtistID:          artist.ID,
			CurrentLocationID: pgUUID(home.ID),
		}); err != nil {
			return err
		}

		if len(input.Availability) > 0 {
			if err := tx.DeleteAllAvailabilityWindows(ctx, artist.ID); err != nil {
				return err
			}
			for _, w := range input.Availability {
				if _, err := tx.CreateAvailabilityWindow(ctx, sqlc.CreateAvailabilityWindowParams{
					ArtistID:    artist.ID,
					Weekday:     w.Weekday,
					StartMinute: w.StartMinute,
					EndMinute:   w.EndMinute,
				}); err != nil {
					return err
				}
			}
		}

		if _, err := tx.CreateOpenBook(ctx, sqlc.CreateOpenBookParams{
			ArtistID:       artist.ID,
			Slug:           username,
			SchedulingMode: string(mode),
		}); err != nil {
			return err
		}

		return tx.SetArtistOnboardedAt(ctx, artist.ID)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return OnboardingResponse{}, ErrUsernameTaken
		}
		return OnboardingResponse{}, err
	}

	return OnboardingResponse{
		OnboardedAt:    time.Now().UTC().Format(time.RFC3339),
		Slug:           username,
		SchedulingMode: string(mode),
	}, nil
}

// ── Open Book ────────────────────────────────────────────────────────────────
func (s *service) GetOpenBook(ctx context.Context, userID uuid.UUID) (OpenBookResponse, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return OpenBookResponse{}, err
	}
	book, err := s.repo.GetOpenBookByArtist(ctx, artist.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OpenBookResponse{}, ErrNotFound
		}
		return OpenBookResponse{}, err
	}
	return openBookResponse(book), nil
}

func (s *service) UpdateOpenBook(ctx context.Context, userID uuid.UUID, input UpdateOpenBookInput) (OpenBookResponse, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return OpenBookResponse{}, err
	}

	params := sqlc.UpdateOpenBookParams{ArtistID: artist.ID}

	if input.SchedulingMode != nil {
		mode := SchedulingMode(*input.SchedulingMode)
		if mode != SchedulingArtist && mode != SchedulingClient {
			return OpenBookResponse{}, fmt.Errorf("%w: schedulingMode must be artist_scheduled or client_scheduled", ErrInvalidInput)
		}
		params.SchedulingMode = input.SchedulingMode
	}

	if input.Slug != nil {
		slug := strings.TrimSpace(*input.Slug)
		if !validSlug(slug) {
			return OpenBookResponse{}, fmt.Errorf("%w: booking link must be 3–30 characters (letters, numbers, underscores)", ErrInvalidInput)
		}
		owner, err := s.repo.GetOpenBookBySlug(ctx, slug)
		switch {
		case err == nil:
			if owner.ArtistID != artist.ID {
				return OpenBookResponse{}, ErrSlugTaken
			}
		case errors.Is(err, pgx.ErrNoRows):
		default:
			return OpenBookResponse{}, err
		}
		params.Slug = &slug
	}

	if input.CustomQuestions != nil {
		questions := make([]string, 0, len(*input.CustomQuestions))
		for _, q := range *input.CustomQuestions {
			if q = strings.TrimSpace(q); q != "" {
				questions = append(questions, q)
			}
		}
		if len(questions) > 3 {
			return OpenBookResponse{}, fmt.Errorf("%w: you can add up to 3 custom questions", ErrInvalidInput)
		}
		raw, err := json.Marshal(questions)
		if err != nil {
			return OpenBookResponse{}, err
		}
		params.CustomQuestions = raw
	}

	book, err := s.repo.UpdateOpenBook(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return OpenBookResponse{}, ErrSlugTaken
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return OpenBookResponse{}, ErrNotFound
		}
		return OpenBookResponse{}, err
	}
	return openBookResponse(book), nil
}

func openBookResponse(book sqlc.OpenBook) OpenBookResponse {
	return OpenBookResponse{
		Slug:            book.Slug,
		SchedulingMode:  book.SchedulingMode,
		CustomQuestions: parseQuestions(book.CustomQuestions),
	}
}

func parseQuestions(raw []byte) []string {
	questions := []string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &questions)
	}
	if questions == nil {
		questions = []string{}
	}
	return questions
}

func validSlug(s string) bool {
	if len(s) < 3 || len(s) > 30 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

// ── Artist business config ─────────────────────────────────────────────────
func (s *service) UpdateSettings(ctx context.Context, userID uuid.UUID, input UpdateSettingsInput) (ArtistSettings, error) {
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

	params := sqlc.UpdateArtistSettingsParams{
		ArtistID:                     artist.ID,
		Timezone:                     trimmedPtr(input.Timezone),
		TermsText:                    input.TermsText,
		Aftercare:                    input.Aftercare,
		AcceptingBookings:            input.AcceptingBookings,
		TermsShowOnBooking:           input.TermsShowOnBooking,
		TermsShowAtDeposit:           input.TermsShowAtDeposit,
		WaiverRequired:               input.WaiverRequired,
		NotifyByEmail:                input.NotifyByEmail,
		NotifyBySMS:                  input.NotifyBySMS,
		BufferMinutes:                input.BufferMinutes,
		MinNoticeMinutes:             input.MinNoticeMinutes,
		DepositFlatFeeCents:          input.DepositFlatFeeCents,
		ClearDepositFlatFee:          input.ClearDepositFlatFee,
		MaxAdvanceDays:               input.MaxAdvanceDays,
		ClearMaxAdvanceDays:          input.ClearMaxAdvanceDays,
		WaiverFileURL:                input.WaiverFileURL,
		ClearWaiverFile:              input.ClearWaiverFile,
		CancellationNoticeHours:      input.CancellationNoticeHours,
		ClearCancellationNoticeHours: input.ClearCancellationNoticeHours,
	}

	if input.PayoutFrequency != nil {
		if !validPayoutFrequency(*input.PayoutFrequency) {
			return ArtistSettings{}, fmt.Errorf("%w: payoutFrequency must be weekly or monthly", ErrInvalidInput)
		}
		params.PayoutFrequency = input.PayoutFrequency
	}
	if input.PlatformFeePayer != nil {
		if !validFeePayer(*input.PlatformFeePayer) {
			return ArtistSettings{}, fmt.Errorf("%w: platformFeePayer must be artist, client, or split", ErrInvalidInput)
		}
		params.PlatformFeePayer = input.PlatformFeePayer
	}
	if input.DepositRefundPolicy != nil {
		if !validRefundPolicy(*input.DepositRefundPolicy) {
			return ArtistSettings{}, fmt.Errorf("%w: depositRefundPolicy must be non_refundable, refundable_within_window, or always_refundable", ErrInvalidInput)
		}
		params.DepositRefundPolicy = input.DepositRefundPolicy
	}
	if input.Currency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*input.Currency))
		if len(currency) != 3 {
			return ArtistSettings{}, fmt.Errorf("%w: currency must be a 3-letter code", ErrInvalidInput)
		}
		params.Currency = &currency
	}
	if input.SlotIntervalMinutes != nil {
		if !validSlotInterval(*input.SlotIntervalMinutes) {
			return ArtistSettings{}, fmt.Errorf("%w: slotIntervalMinutes must be 15, 30, or 60", ErrInvalidInput)
		}
		params.SlotIntervalMinutes = input.SlotIntervalMinutes
	}
	if input.Styles != nil {
		if !allValidStyles(*input.Styles) {
			return ArtistSettings{}, fmt.Errorf("%w: unknown tattoo style", ErrInvalidInput)
		}
		params.Styles = *input.Styles
	}
	if input.FAQs != nil {
		faqs := make([]FAQItem, 0, len(*input.FAQs))
		for _, f := range *input.FAQs {
			question := strings.TrimSpace(f.Question)
			answer := strings.TrimSpace(f.Answer)
			if question == "" && answer == "" {
				continue
			}
			if question == "" || answer == "" {
				return ArtistSettings{}, fmt.Errorf("%w: each FAQ needs both a question and an answer", ErrInvalidInput)
			}
			faqs = append(faqs, FAQItem{Question: question, Answer: answer})
		}
		if len(faqs) > 5 {
			return ArtistSettings{}, fmt.Errorf("%w: you can add up to 5 FAQs", ErrInvalidInput)
		}
		raw, err := json.Marshal(faqs)
		if err != nil {
			return ArtistSettings{}, err
		}
		params.Faqs = raw
	}

	// If the payout frequency changed and a Stripe account is connected, push the
	// new schedule to Stripe first — only persist locally if Stripe accepts it,
	// so the live schedule and the saved setting can't drift apart.
	if input.PayoutFrequency != nil &&
		*input.PayoutFrequency != current.PayoutFrequency &&
		current.StripeAccountID != nil && *current.StripeAccountID != "" {
		if err := s.syncStripePayoutSchedule(ctx, *current.StripeAccountID, *input.PayoutFrequency); err != nil {
			return ArtistSettings{}, err
		}
	}

	srow, err := s.repo.UpdateArtistSettings(ctx, params)
	if err != nil {
		return ArtistSettings{}, err
	}
	return s.settingsResponse(ctx, srow), nil
}

// ── Locations (home studio + guest spots) ──────────────────────────────────
func (s *service) CreateLocation(ctx context.Context, userID uuid.UUID, input CreateLocationInput) (Location, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Location{}, err
	}

	city := strings.TrimSpace(input.City)
	country := strings.TrimSpace(input.Country)
	if city == "" || country == "" {
		return Location{}, fmt.Errorf("%w: a guest spot needs at least a city and country", ErrInvalidInput)
	}

	start, err := parseDate(input.StartDate)
	if err != nil {
		return Location{}, err
	}
	end, err := parseDate(input.EndDate)
	if err != nil {
		return Location{}, err
	}
	if !start.Valid || !end.Valid {
		return Location{}, fmt.Errorf("%w: a guest spot needs a start and end date", ErrInvalidInput)
	}
	if end.Time.Before(start.Time) {
		return Location{}, fmt.Errorf("%w: the end date must be on or after the start date", ErrInvalidInput)
	}

	existing, err := s.repo.ListArtistLocations(ctx, artist.ID)
	if err != nil {
		return Location{}, err
	}
	if overlapsAnyGuestSpot(start.Time, end.Time, existing, uuid.Nil) {
		return Location{}, fmt.Errorf("%w: those dates overlap another guest spot", ErrInvalidInput)
	}

	row, err := s.repo.CreateArtistLocation(ctx, sqlc.CreateArtistLocationParams{
		ArtistID:   artist.ID,
		Label:      strings.TrimSpace(input.Label),
		Address:    strings.TrimSpace(input.Address),
		City:       city,
		Province:   strings.TrimSpace(input.Province),
		PostalCode: strings.TrimSpace(input.PostalCode),
		Country:    country,
		Timezone:   strings.TrimSpace(input.Timezone),
		IsPrimary:  false,
		StartDate:  start,
		EndDate:    end,
	})
	if err != nil {
		return Location{}, err
	}
	return getLocationFromRow(row), nil
}

func (s *service) UpdateLocation(ctx context.Context, userID, locationID uuid.UUID, input UpdateLocationInput) (Location, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Location{}, err
	}

	current, err := s.repo.GetArtistLocation(ctx, sqlc.GetArtistLocationParams{ID: locationID, ArtistID: artist.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Location{}, ErrNotFound
		}
		return Location{}, err
	}

	params := sqlc.UpdateArtistLocationParams{
		ID:         locationID,
		ArtistID:   artist.ID,
		Label:      trimmedPtr(input.Label),
		Address:    trimmedPtr(input.Address),
		City:       trimmedPtr(input.City),
		Province:   trimmedPtr(input.Province),
		PostalCode: trimmedPtr(input.PostalCode),
		Country:    trimmedPtr(input.Country),
		Timezone:   trimmedPtr(input.Timezone),
	}

	if !current.IsPrimary && !input.ClearDates {
		start, err := parseDate(input.StartDate)
		if err != nil {
			return Location{}, err
		}
		end, err := parseDate(input.EndDate)
		if err != nil {
			return Location{}, err
		}

		effectiveStart := current.StartDate
		if start.Valid {
			effectiveStart = start
		}
		effectiveEnd := current.EndDate
		if end.Valid {
			effectiveEnd = end
		}
		if effectiveStart.Valid && effectiveEnd.Valid {
			if effectiveEnd.Time.Before(effectiveStart.Time) {
				return Location{}, fmt.Errorf("%w: the end date must be on or after the start date", ErrInvalidInput)
			}
			existing, err := s.repo.ListArtistLocations(ctx, artist.ID)
			if err != nil {
				return Location{}, err
			}
			if overlapsAnyGuestSpot(effectiveStart.Time, effectiveEnd.Time, existing, locationID) {
				return Location{}, fmt.Errorf("%w: those dates overlap another guest spot", ErrInvalidInput)
			}
		}
		params.StartDate = start
		params.EndDate = end
	} else if input.ClearDates {
		params.ClearDates = true
	}

	row, err := s.repo.UpdateArtistLocation(ctx, params)
	if err != nil {
		return Location{}, err
	}
	return getLocationFromRow(row), nil
}

// DeleteLocation closes a guest spot. The row is kept (soft-deleted) so booking
// requests retain their location and we keep a history of where the artist has
// worked; the home studio can't be closed.
func (s *service) DeleteLocation(ctx context.Context, userID, locationID uuid.UUID) error {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.repo.CloseArtistLocation(ctx, sqlc.CloseArtistLocationParams{ID: locationID, ArtistID: artist.ID}); err != nil {
		return err
	}
	return s.resetCurrentLocationIfClosed(ctx, artist.ID, locationID)
}

// resetCurrentLocationIfClosed falls the artist back to their home studio when
// they close the very spot they were working from, so the active location never
// points at a closed one.
func (s *service) resetCurrentLocationIfClosed(ctx context.Context, artistID, closedID uuid.UUID) error {
	settings, err := s.repo.GetArtistSettings(ctx, artistID)
	if err != nil {
		return err
	}
	if !settings.CurrentLocationID.Valid || uuid.UUID(settings.CurrentLocationID.Bytes) != closedID {
		return nil
	}
	primary, err := s.repo.GetPrimaryLocation(ctx, artistID)
	if err != nil {
		return err
	}
	return s.repo.SetCurrentLocation(ctx, sqlc.SetCurrentLocationParams{
		ArtistID:          artistID,
		CurrentLocationID: pgUUID(primary.ID),
	})
}

func (s *service) SetCurrentLocation(ctx context.Context, userID uuid.UUID, input SetCurrentLocationInput) error {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return err
	}

	locationID, err := uuid.Parse(strings.TrimSpace(input.LocationID))
	if err != nil {
		return fmt.Errorf("%w: invalid location", ErrInvalidInput)
	}
	if _, err := s.repo.GetArtistLocation(ctx, sqlc.GetArtistLocationParams{ID: locationID, ArtistID: artist.ID}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	return s.repo.SetCurrentLocation(ctx, sqlc.SetCurrentLocationParams{
		ArtistID:          artist.ID,
		CurrentLocationID: pgUUID(locationID),
	})
}

func overlapsAnyGuestSpot(start, end time.Time, locations []sqlc.ArtistLocation, excludeID uuid.UUID) bool {
	for _, l := range locations {
		if l.IsPrimary || l.ID == excludeID || !l.StartDate.Valid || !l.EndDate.Valid {
			continue
		}
		if !start.After(l.EndDate.Time) && !l.StartDate.Time.After(end) {
			return true
		}
	}
	return false
}

func (s *service) SetAvailability(ctx context.Context, userID uuid.UUID, input SetAvailabilityInput) ([]AvailabilityWindow, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, w := range input.Windows {
		if w.Weekday < 0 || w.Weekday > 6 {
			return nil, fmt.Errorf("%w: weekday must be 0..6", ErrInvalidInput)
		}
		if w.EndMinute <= w.StartMinute {
			return nil, fmt.Errorf("%w: a window's end must be after its start", ErrInvalidInput)
		}
	}

	var saved []sqlc.ArtistAvailabilityWindow
	err = s.repo.RunInTx(ctx, func(tx Repository) error {
		if err := tx.DeleteAllAvailabilityWindows(ctx, artist.ID); err != nil {
			return err
		}
		for _, w := range input.Windows {
			row, err := tx.CreateAvailabilityWindow(ctx, sqlc.CreateAvailabilityWindowParams{
				ArtistID:    artist.ID,
				Weekday:     w.Weekday,
				StartMinute: w.StartMinute,
				EndMinute:   w.EndMinute,
			})
			if err != nil {
				return err
			}
			saved = append(saved, row)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]AvailabilityWindow, 0, len(saved))
	for _, w := range saved {
		out = append(out, availabilityFromRow(w))
	}
	return out, nil
}

func (s *service) PresignWaiverUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return PresignUploadResponse{}, err
	}
	ext, ok := waiverContentTypeToExt[contentType]
	if !ok {
		return PresignUploadResponse{}, fmt.Errorf("%w: unsupported content type %q (allowed: application/pdf, image/jpeg, image/png, image/webp)", ErrInvalidInput, contentType)
	}
	key := fmt.Sprintf("waivers/%s/%s.%s", artist.ID, uuid.New(), ext)
	return s.presignPut(ctx, key, contentType)
}

// ── Session presets ────────────────────────────────────────────────────────
func (s *service) CreatePreset(ctx context.Context, userID uuid.UUID, input CreatePresetInput) (SessionPreset, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return SessionPreset{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return SessionPreset{}, fmt.Errorf("%w: preset name is required", ErrInvalidInput)
	}
	if input.ApproxDurationMinutes <= 0 {
		return SessionPreset{}, fmt.Errorf("%w: approxDurationMinutes must be positive", ErrInvalidInput)
	}
	row, err := s.repo.CreateSessionPreset(ctx, sqlc.CreateSessionPresetParams{
		ArtistID:              artist.ID,
		Name:                  name,
		Description:           strings.TrimSpace(input.Description),
		ApproxDurationMinutes: input.ApproxDurationMinutes,
		Position:              input.Position,
	})
	if err != nil {
		return SessionPreset{}, err
	}
	return presetFromRow(row), nil
}

func (s *service) UpdatePreset(ctx context.Context, userID, presetID uuid.UUID, input UpdatePresetInput) (SessionPreset, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return SessionPreset{}, err
	}
	row, err := s.repo.UpdateSessionPreset(ctx, sqlc.UpdateSessionPresetParams{
		ID:                    presetID,
		ArtistID:              artist.ID,
		Name:                  trimmedPtr(input.Name),
		Description:           trimmedPtr(input.Description),
		ApproxDurationMinutes: input.ApproxDurationMinutes,
		Position:              input.Position,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionPreset{}, ErrNotFound
		}
		return SessionPreset{}, err
	}
	return presetFromRow(row), nil
}

func (s *service) DeletePreset(ctx context.Context, userID, presetID uuid.UUID) error {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.DeleteSessionPreset(ctx, sqlc.DeleteSessionPresetParams{ID: presetID, ArtistID: artist.ID})
}

// ── Days off ────────────────────────────────────────────────────────────────
func (s *service) AddDayOff(ctx context.Context, userID uuid.UUID, input DayOffInput) error {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return err
	}
	day, err := parseDay(input.Day)
	if err != nil {
		return err
	}
	return s.repo.AddDayOff(ctx, sqlc.AddDayOffParams{ArtistID: artist.ID, Day: day})
}

func (s *service) RemoveDayOff(ctx context.Context, userID uuid.UUID, day string) error {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return err
	}
	parsed, err := parseDay(day)
	if err != nil {
		return err
	}
	return s.repo.RemoveDayOff(ctx, sqlc.RemoveDayOffParams{ArtistID: artist.ID, Day: parsed})
}

// ── Blocklist ───────────────────────────────────────────────────────────────
func (s *service) AddBlocklistEntry(ctx context.Context, userID uuid.UUID, input CreateBlocklistInput) (BlocklistEntry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return BlocklistEntry{}, err
	}
	email := trimmedPtr(input.Email)
	phone := trimmedPtr(input.Phone)
	if (email == nil || *email == "") && (phone == nil || *phone == "") {
		return BlocklistEntry{}, fmt.Errorf("%w: provide an email and/or a phone number", ErrInvalidInput)
	}
	row, err := s.repo.AddBlocklistEntry(ctx, sqlc.AddBlocklistEntryParams{
		ArtistID: artist.ID,
		Email:    nilIfEmpty(email),
		Phone:    nilIfEmpty(phone),
		Note:     strings.TrimSpace(input.Note),
	})
	if err != nil {
		return BlocklistEntry{}, err
	}
	return blocklistFromRow(row), nil
}

func (s *service) RemoveBlocklistEntry(ctx context.Context, userID, entryID uuid.UUID) error {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.RemoveBlocklistEntry(ctx, sqlc.RemoveBlocklistEntryParams{ID: entryID, ArtistID: artist.ID})
}

func (s *service) requireArtist(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err == nil {
		return artist, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Artist{}, err
	}
	if err := s.repo.EnsureArtist(ctx, userID); err != nil {
		return sqlc.Artist{}, err
	}
	return s.repo.GetArtistByUserID(ctx, userID)
}

func (s *service) assertUsernameAvailable(ctx context.Context, username string, userID uuid.UUID) error {
	owner, err := s.repo.GetUserByUsername(ctx, username)
	switch {
	case err == nil:
		if owner.ID != userID {
			return ErrUsernameTaken
		}
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil
	default:
		return err
	}
}

func (s *service) accountResponse(ctx context.Context, u sqlc.User) Account {
	account := accountFromUser(u)
	account.AvatarURL = s.presignKey(ctx, account.AvatarURL)
	return account
}

func (s *service) settingsResponse(ctx context.Context, r sqlc.ArtistSetting) ArtistSettings {
	out := settingsFromRow(r)
	out.WaiverFileURL = s.presignKey(ctx, out.WaiverFileURL)
	return out
}

func (s *service) presignKey(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	url, err := s.s3.PresignGet(ctx, key, presignViewTTL)
	if err != nil {
		s.log.Warn("settings_presign_failed", "key", key, "error", err)
		return ""
	}
	return url
}

func (s *service) presignPut(ctx context.Context, key, contentType string) (PresignUploadResponse, error) {
	url, err := s.s3.PresignPut(ctx, key, contentType, presignUploadTTL)
	if err != nil {
		return PresignUploadResponse{}, err
	}
	return PresignUploadResponse{
		URL:       url,
		S3Key:     key,
		ExpiresAt: time.Now().UTC().Add(presignUploadTTL),
	}, nil
}

func parseDay(s string) (pgtype.Date, error) {
	t, err := time.Parse(dayLayout, strings.TrimSpace(s))
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("%w: invalid date %q (want YYYY-MM-DD)", ErrInvalidInput, s)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func validPayoutFrequency(v string) bool {
	switch PayoutFrequency(v) {
	case PayoutWeekly, PayoutMonthly:
		return true
	}
	return false
}

func validRefundPolicy(v string) bool {
	switch DepositRefundPolicy(v) {
	case RefundNonRefundable, RefundWithinWindow, RefundAlways:
		return true
	}
	return false
}

func validFeePayer(v string) bool {
	switch PlatformFeePayer(v) {
	case FeePayerArtist, FeePayerClient, FeePayerSplit:
		return true
	}
	return false
}

func validSlotInterval(n int32) bool {
	return n == 15 || n == 30 || n == 60
}

func trimmedPtr(in *string) *string {
	if in == nil {
		return nil
	}
	t := strings.TrimSpace(*in)
	return &t
}

func nilIfEmpty(in *string) *string {
	if in == nil || *in == "" {
		return nil
	}
	return in
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func parseDate(value *string) (pgtype.Date, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse(dateLayout, strings.TrimSpace(*value))
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("%w: invalid date %q (use YYYY-MM-DD)", ErrInvalidInput, *value)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
