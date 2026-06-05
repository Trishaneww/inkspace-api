package settings

import (
	"context"
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
	ErrPhoneTaken        = errors.New("phone number already in use")
	ErrNotImplemented    = errors.New("not implemented")
	ErrIntegrationConfig = errors.New("integration not configured on this server")
	ErrOAuthExchange     = errors.New("could not complete the connection with the provider")
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

	// Artist business config
	UpdateSettings(ctx context.Context, userID uuid.UUID, input UpdateSettingsInput) (ArtistSettings, error)
	SetAvailability(ctx context.Context, userID uuid.UUID, input SetAvailabilityInput) ([]AvailabilityWindow, error)
	PresignWaiverUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error)

	// Integrations
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

	resp := SettingsResponse{
		Account:        s.accountResponse(ctx, user),
		Settings:       s.settingsResponse(ctx, srow),
		Availability:   make([]AvailabilityWindow, 0, len(windows)),
		SessionPresets: make([]SessionPreset, 0, len(presets)),
		DaysOff:        make([]string, 0, len(daysOff)),
		Blocklist:      make([]BlocklistEntry, 0, len(blocklist)),
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
	params.AvatarUrl = trimmedPtr(input.AvatarURL)
	params.InstagramUrl = trimmedPtr(input.InstagramURL)

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
	return s.accountResponse(ctx, updated), nil
}

func (s *service) ChangeEmail(ctx context.Context, userID uuid.UUID, input ChangeEmailInput) (Account, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return Account{}, err
	}

	newEmail := strings.ToLower(strings.TrimSpace(input.NewEmail))
	if newEmail == strings.ToLower(user.Email) {
		// No-op change; return current account.
		return s.accountResponse(ctx, user), nil
	}

	existing, err := s.repo.GetUserByEmail(ctx, newEmail)
	switch {
	case err == nil:
		if existing.ID != userID {
			return Account{}, ErrEmailTaken
		}
	case errors.Is(err, pgx.ErrNoRows):
		// free
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

// ── Artist business config ─────────────────────────────────────────────────
func (s *service) UpdateSettings(ctx context.Context, userID uuid.UUID, input UpdateSettingsInput) (ArtistSettings, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return ArtistSettings{}, err
	}
	if err := s.repo.EnsureArtistSettings(ctx, artist.ID); err != nil {
		return ArtistSettings{}, err
	}

	params := sqlc.UpdateArtistSettingsParams{
		ArtistID:            artist.ID,
		StudioName:          trimmedPtr(input.StudioName),
		StudioAddress:       trimmedPtr(input.StudioAddress),
		StudioCity:          trimmedPtr(input.StudioCity),
		StudioProvince:      trimmedPtr(input.StudioProvince),
		StudioPostalCode:    trimmedPtr(input.StudioPostalCode),
		StudioCountry:       trimmedPtr(input.StudioCountry),
		Timezone:            trimmedPtr(input.Timezone),
		TermsText:           input.TermsText,
		AcceptingBookings:   input.AcceptingBookings,
		TermsShowOnBooking:  input.TermsShowOnBooking,
		TermsShowAtDeposit:  input.TermsShowAtDeposit,
		WaiverRequired:      input.WaiverRequired,
		NotifyByEmail:       input.NotifyByEmail,
		NotifyBySms:         input.NotifyBySms,
		BufferMinutes:       input.BufferMinutes,
		MinNoticeMinutes:    input.MinNoticeMinutes,
		DepositFlatFeeCents: input.DepositFlatFeeCents,
		ClearDepositFlatFee: input.ClearDepositFlatFee,
		MaxAdvanceDays:      input.MaxAdvanceDays,
		ClearMaxAdvance:     input.ClearMaxAdvance,
		WaiverFileUrl:       input.WaiverFileURL,
		ClearWaiverFile:     input.ClearWaiverFile,
	}

	if input.PayoutFrequency != nil {
		if !validPayoutFrequency(*input.PayoutFrequency) {
			return ArtistSettings{}, fmt.Errorf("%w: payoutFrequency must be weekly, biweekly, or monthly", ErrInvalidInput)
		}
		params.PayoutFrequency = input.PayoutFrequency
	}
	if input.PlatformFeePayer != nil {
		if !validFeePayer(*input.PlatformFeePayer) {
			return ArtistSettings{}, fmt.Errorf("%w: platformFeePayer must be artist, client, or split", ErrInvalidInput)
		}
		params.PlatformFeePayer = input.PlatformFeePayer
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

	srow, err := s.repo.UpdateArtistSettings(ctx, params)
	if err != nil {
		return ArtistSettings{}, err
	}
	return s.settingsResponse(ctx, srow), nil
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
		if err := tx.DeleteAvailabilityWindows(ctx, artist.ID); err != nil {
			return err
		}
		for _, w := range input.Windows {
			row, err := tx.InsertAvailabilityWindow(ctx, sqlc.InsertAvailabilityWindowParams{
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

// presignKey turns a stored S3 key into a short-lived GET URL. Empty keys and
// presign failures degrade to an empty string rather than failing the request.
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
	case PayoutWeekly, PayoutBiweekly, PayoutMonthly:
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
