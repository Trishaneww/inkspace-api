package openbook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

var (
	ErrNotFound     = errors.New("open book not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotAccepting = errors.New("not accepting bookings")
)

const (
	presignViewTTL   = 1 * time.Hour
	presignUploadTTL = 15 * time.Minute

	maxReferenceImages = 3
	referenceKeyPrefix = "booking-refs/"
)

var referenceContentTypeToExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

type Service interface {
	GetProfile(ctx context.Context, slug string) (Profile, error)
	PresignReferenceUpload(ctx context.Context, slug, contentType string) (PresignReferenceResult, error)
	CreateRequest(ctx context.Context, slug string, input CreateRequestInput) (CreateRequestResult, error)
	SetConversationCreator(c ConversationCreator)
}

type ConversationCreator interface {
	CreateConversationForBooking(ctx context.Context, artistID, bookingID uuid.UUID, clientName, clientEmail string) (sqlc.Conversation, error)
}

type service struct {
	repo                Repository
	s3                  *s3client.Client
	log                 *slog.Logger
	conversationCreator ConversationCreator
}

func NewService(repo Repository, s3 *s3client.Client) Service {
	return &service{repo: repo, s3: s3, log: slog.Default()}
}

func (s *service) SetConversationCreator(c ConversationCreator) {
	s.conversationCreator = c
}

type bookable struct {
	user     sqlc.User
	artist   sqlc.Artist
	settings sqlc.ArtistSetting
	openBook sqlc.OpenBook
}

func (s *service) resolveBookable(ctx context.Context, slug string) (bookable, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return bookable{}, ErrNotFound
	}

	user, err := s.repo.GetUserByUsername(ctx, slug)
	if err != nil {
		return bookable{}, notFoundOrErr(err)
	}

	artist, err := s.repo.GetArtistByUserID(ctx, user.ID)
	if err != nil {
		return bookable{}, notFoundOrErr(err)
	}
	if !artist.OnboardedAt.Valid {
		return bookable{}, ErrNotFound
	}

	openBook, err := s.repo.GetOpenBookByArtist(ctx, artist.ID)
	if err != nil {
		return bookable{}, notFoundOrErr(err)
	}

	settings, err := s.repo.GetArtistSettings(ctx, artist.ID)
	if err != nil {
		return bookable{}, notFoundOrErr(err)
	}

	return bookable{user: user, artist: artist, settings: settings, openBook: openBook}, nil
}

func notFoundOrErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *service) GetProfile(ctx context.Context, slug string) (Profile, error) {
	b, err := s.resolveBookable(ctx, slug)
	if err != nil {
		return Profile{}, err
	}

	windows, err := s.repo.ListAvailabilityWindows(ctx, b.artist.ID)
	if err != nil {
		return Profile{}, err
	}
	locations, err := s.repo.ListArtistLocations(ctx, b.artist.ID)
	if err != nil {
		return Profile{}, err
	}
	availableFlashes, err := s.repo.CountAvailableFlashes(ctx, b.artist.ID)
	if err != nil {
		return Profile{}, err
	}
	publishedPortfolio, err := s.repo.CountPublishedPortfolio(ctx, b.artist.ID)
	if err != nil {
		return Profile{}, err
	}

	profile := s.buildProfile(ctx, b.user, b.settings, b.openBook, windows, locations)
	profile.ArtistID = b.artist.ID.String()
	profile.HasFlashes = availableFlashes > 0
	profile.HasPortfolio = publishedPortfolio > 0
	return profile, nil
}

func (s *service) PresignReferenceUpload(ctx context.Context, slug, contentType string) (PresignReferenceResult, error) {
	b, err := s.resolveBookable(ctx, slug)
	if err != nil {
		return PresignReferenceResult{}, err
	}
	if !b.settings.AcceptingBookings {
		return PresignReferenceResult{}, ErrNotAccepting
	}

	ext, ok := referenceContentTypeToExt[contentType]
	if !ok {
		return PresignReferenceResult{}, fmt.Errorf("%w: unsupported image type (allowed: jpeg, png, webp)", ErrInvalidInput)
	}

	key := fmt.Sprintf("%s%s/%s.%s", referenceKeyPrefix, b.artist.ID, uuid.New(), ext)
	url, err := s.s3.PresignPut(ctx, key, contentType, presignUploadTTL)
	if err != nil {
		return PresignReferenceResult{}, err
	}

	return PresignReferenceResult{
		URL:       url,
		Key:       key,
		ExpiresAt: time.Now().UTC().Add(presignUploadTTL).Format(time.RFC3339),
	}, nil
}

func (s *service) CreateRequest(ctx context.Context, slug string, input CreateRequestInput) (CreateRequestResult, error) {
	b, err := s.resolveBookable(ctx, slug)
	if err != nil {
		return CreateRequestResult{}, err
	}
	if !b.settings.AcceptingBookings {
		return CreateRequestResult{}, ErrNotAccepting
	}

	var params sqlc.CreateBookingRequestParams
	if input.FlashID != nil && strings.TrimSpace(*input.FlashID) != "" {
		params, err = s.buildFlashRequestParams(ctx, b, input)
	} else {
		params, err = buildCustomRequestParams(b, input)
	}
	if err != nil {
		return CreateRequestResult{}, err
	}

	locations, err := s.repo.ListArtistLocations(ctx, b.artist.ID)
	if err != nil {
		return CreateRequestResult{}, err
	}
	location, err := resolveRequestLocation(input.LocationID, b.settings, locations)
	if err != nil {
		return CreateRequestResult{}, err
	}
	if location != nil {
		params.LocationID = pgUUID(location.ID)
	}

	row, err := s.repo.CreateBookingRequest(ctx, params)
	if err != nil {
		return CreateRequestResult{}, err
	}

	if s.conversationCreator != nil {
		if _, err := s.conversationCreator.CreateConversationForBooking(ctx, b.artist.ID, row.ID, row.ClientName, row.ClientEmail); err != nil {
			s.log.Warn("conversation_create_failed", "booking_request_id", row.ID, "error", err)
		}
	}

	return CreateRequestResult{
		ID:        row.ID.String(),
		Status:    row.Status,
		CreatedAt: row.CreatedAt.Time.UTC().Format(time.RFC3339),
	}, nil
}

// validateContact pulls the shared guest contact fields, required for every
// request regardless of type.
func validateContact(input CreateRequestInput) (name, email, phone string, err error) {
	name = strings.TrimSpace(input.ClientName)
	if name == "" {
		return "", "", "", fmt.Errorf("%w: your name is required", ErrInvalidInput)
	}
	email = strings.TrimSpace(input.ClientEmail)
	if _, err := mail.ParseAddress(email); err != nil {
		return "", "", "", fmt.Errorf("%w: a valid email is required", ErrInvalidInput)
	}
	phone = strings.TrimSpace(input.ClientPhone)
	if phone == "" {
		return "", "", "", fmt.Errorf("%w: a phone number is required", ErrInvalidInput)
	}
	return name, email, phone, nil
}

func buildCustomRequestParams(b bookable, input CreateRequestInput) (sqlc.CreateBookingRequestParams, error) {
	var empty sqlc.CreateBookingRequestParams

	name, email, phone, err := validateContact(input)
	if err != nil {
		return empty, err
	}
	description := strings.TrimSpace(input.Description)
	if description == "" {
		return empty, fmt.Errorf("%w: please describe the tattoo you want", ErrInvalidInput)
	}

	styles := normalizeStyles(input.Styles)
	keys, err := validateReferenceKeys(input.ReferenceImageKeys)
	if err != nil {
		return empty, err
	}
	if input.ApproxSizeInches != nil && *input.ApproxSizeInches <= 0 {
		return empty, fmt.Errorf("%w: size must be greater than zero", ErrInvalidInput)
	}

	colorType := strings.TrimSpace(input.ColorType)
	if !validColorType(colorType) {
		return empty, fmt.Errorf("%w: choose black & grey, color, or either", ErrInvalidInput)
	}

	answersJSON, err := json.Marshal(cleanAnswers(input.CustomAnswers))
	if err != nil {
		return empty, err
	}
	availabilityJSON, err := json.Marshal(cleanAvailability(input.ClientAvailability))
	if err != nil {
		return empty, err
	}

	waiverStatus := "not_required"
	if b.settings.WaiverRequired {
		waiverStatus = "pending"
	}

	params := sqlc.CreateBookingRequestParams{
		ArtistID:           b.artist.ID,
		OpenBookID:         b.openBook.ID,
		Type:               "custom",
		Description:        description,
		ReferenceImageKeys: keys,
		Placement:          strings.TrimSpace(input.Placement),
		ApproxSizeInches:   input.ApproxSizeInches,
		ColorType:          colorType,
		Styles:             styles,
		ClientAvailability: availabilityJSON,
		CustomAnswers:      answersJSON,
		ClientName:         name,
		ClientEmail:        email,
		ClientPhone:        phone,
		Status:             "pending",
		DepositStatus:      "not_required",
		WaiverStatus:       waiverStatus,
	}
	return params, nil
}

// buildFlashRequestParams builds a booking request that claims a specific flash.
// The design (color, styles) comes from the flash; the client only chooses size,
// placement, and location, so most custom fields are left empty.
func (s *service) buildFlashRequestParams(ctx context.Context, b bookable, input CreateRequestInput) (sqlc.CreateBookingRequestParams, error) {
	var empty sqlc.CreateBookingRequestParams

	name, email, phone, err := validateContact(input)
	if err != nil {
		return empty, err
	}

	flashID, err := uuid.Parse(strings.TrimSpace(*input.FlashID))
	if err != nil {
		return empty, fmt.Errorf("%w: invalid flash", ErrInvalidInput)
	}
	flash, err := s.repo.GetFlash(ctx, flashID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return empty, fmt.Errorf("%w: that flash isn't available", ErrInvalidInput)
		}
		return empty, err
	}
	if flash.ArtistID != b.artist.ID || flash.Status != "available" {
		return empty, fmt.Errorf("%w: that flash isn't available", ErrInvalidInput)
	}

	placement := strings.TrimSpace(input.Placement)
	if placement == "" {
		return empty, fmt.Errorf("%w: choose a placement", ErrInvalidInput)
	}
	if len(flash.Placements) > 0 && !containsString(flash.Placements, placement) {
		return empty, fmt.Errorf("%w: that placement isn't offered for this flash", ErrInvalidInput)
	}

	duration, err := s.resolveFlashDuration(ctx, flash, input.SizeCode)
	if err != nil {
		return empty, err
	}

	answersJSON, err := json.Marshal(cleanAnswers(input.CustomAnswers))
	if err != nil {
		return empty, err
	}
	availabilityJSON, err := json.Marshal(cleanAvailability(input.ClientAvailability))
	if err != nil {
		return empty, err
	}

	waiverStatus := "not_required"
	if b.settings.WaiverRequired {
		waiverStatus = "pending"
	}

	styles := flash.Styles
	if styles == nil {
		styles = []string{}
	}

	return sqlc.CreateBookingRequestParams{
		ArtistID:               b.artist.ID,
		OpenBookID:             b.openBook.ID,
		Type:                   "flash",
		FlashID:                pgUUID(flashID),
		Description:            "",
		ReferenceImageKeys:     []string{},
		Placement:              placement,
		ColorType:              normalizeFlashColorType(flash.ColorType),
		Styles:                 styles,
		ClientAvailability:     availabilityJSON,
		CustomAnswers:          answersJSON,
		ClientName:             name,
		ClientEmail:            email,
		ClientPhone:            phone,
		Status:                 "pending",
		DepositStatus:          "not_required",
		WaiverStatus:           waiverStatus,
		SessionDurationMinutes: duration,
		FlashSizeCode:          strings.TrimSpace(input.SizeCode),
	}, nil
}

func (s *service) resolveFlashDuration(ctx context.Context, flash sqlc.Flash, sizeCode string) (*int32, error) {
	if flash.PricingMode == "flat" {
		return flash.FlatDurationMinutes, nil
	}
	code := strings.TrimSpace(sizeCode)
	if code == "" {
		return nil, fmt.Errorf("%w: choose a size", ErrInvalidInput)
	}
	tiers, err := s.repo.ListFlashPricingTiers(ctx, flash.ID)
	if err != nil {
		return nil, err
	}
	for _, t := range tiers {
		if t.SizeCode == code {
			duration := t.DurationMinutes
			return &duration, nil
		}
	}
	return nil, fmt.Errorf("%w: that size isn't offered for this flash", ErrInvalidInput)
}

func normalizeFlashColorType(colorType string) string {
	if colorType == "both" {
		return "either"
	}
	return colorType
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func normalizeStyles(selected []string) []string {
	seen := make(map[string]bool, len(selected))
	out := make([]string, 0, len(selected))
	for _, style := range selected {
		style = strings.TrimSpace(style)
		if style == "" || seen[style] {
			continue
		}
		seen[style] = true
		out = append(out, style)
	}
	return out
}

func validateReferenceKeys(keys []string) ([]string, error) {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if !strings.HasPrefix(key, referenceKeyPrefix) {
			return nil, fmt.Errorf("%w: invalid reference image", ErrInvalidInput)
		}
		out = append(out, key)
	}
	if len(out) > maxReferenceImages {
		return nil, fmt.Errorf("%w: up to %d reference images", ErrInvalidInput, maxReferenceImages)
	}
	return out, nil
}

func cleanAnswers(answers []CustomAnswer) []CustomAnswer {
	out := make([]CustomAnswer, 0, len(answers))
	for _, qa := range answers {
		prompt := strings.TrimSpace(qa.Prompt)
		if prompt == "" {
			continue
		}
		out = append(out, CustomAnswer{Prompt: prompt, Answer: strings.TrimSpace(qa.Answer)})
	}
	return out
}

func validColorType(value string) bool {
	switch value {
	case "black_and_grey", "color", "either":
		return true
	default:
		return false
	}
}

func cleanAvailability(choices []AvailabilityChoice) []AvailabilityChoice {
	out := make([]AvailabilityChoice, 0, len(choices))
	for _, choice := range choices {
		if choice.Weekday < 0 || choice.Weekday > 6 {
			continue
		}
		if choice.StartMinute < 0 ||
			choice.EndMinute > 1440 ||
			choice.StartMinute >= choice.EndMinute {
			continue
		}
		out = append(out, choice)
	}
	return out
}

func (s *service) buildProfile(
	ctx context.Context,
	user sqlc.User,
	settings sqlc.ArtistSetting,
	openBook sqlc.OpenBook,
	windows []sqlc.ArtistAvailabilityWindow,
	locations []sqlc.ArtistLocation,
) Profile {
	active := resolveActiveLocation(settings, locations)

	profile := Profile{
		Username:          derefString(user.Username),
		DisplayName:       formatDisplayName(user),
		AvatarURL:         s.presignKey(ctx, derefString(user.AvatarURL)),
		InstagramURL:      derefString(user.InstagramURL),
		AcceptingBookings: settings.AcceptingBookings,
		SchedulingMode:    openBook.SchedulingMode,
		Theme:             openBook.Theme,
		CustomTheme:       parseCustomTheme(openBook.CustomTheme),
		Styles:            settings.Styles,
		CustomQuestions:   parseStringList(openBook.CustomQuestions),
		Aftercare:         settings.Aftercare,
		FAQs:              parseFAQs(settings.Faqs),
		Availability:      formatWindows(windows),
		Locations:         formatPublicLocations(locations, active),
	}
	if openBook.BackgroundImageKey != nil && *openBook.BackgroundImageKey != "" {
		profile.BackgroundImageURL = s.presignKey(ctx, *openBook.BackgroundImageKey)
	}
	if active != nil {
		profile.Location = formatLocation(active.City, active.Country)
	}
	if profile.Styles == nil {
		profile.Styles = []string{}
	}
	return profile
}

func resolveRequestLocation(
	locationID string,
	settings sqlc.ArtistSetting,
	locations []sqlc.ArtistLocation,
) (*sqlc.ArtistLocation, error) {
	if id := strings.TrimSpace(locationID); id != "" {
		for i := range locations {
			if locations[i].ID.String() == id {
				return &locations[i], nil
			}
		}
		return nil, fmt.Errorf("%w: unknown location", ErrInvalidInput)
	}
	return resolveActiveLocation(settings, locations), nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func resolveActiveLocation(
	settings sqlc.ArtistSetting,
	locations []sqlc.ArtistLocation,
) *sqlc.ArtistLocation {
	currentID := ""
	if settings.CurrentLocationID.Valid {
		currentID = uuid.UUID(settings.CurrentLocationID.Bytes).String()
	}
	for i := range locations {
		if locations[i].ID.String() == currentID {
			return &locations[i]
		}
	}
	for i := range locations {
		if locations[i].IsPrimary {
			return &locations[i]
		}
	}
	if len(locations) > 0 {
		return &locations[0]
	}
	return nil
}

func formatPublicLocations(
	locations []sqlc.ArtistLocation,
	active *sqlc.ArtistLocation,
) []PublicLocation {
	out := make([]PublicLocation, 0, len(locations))
	for i := range locations {
		l := locations[i]
		item := PublicLocation{
			ID:        l.ID.String(),
			Label:     l.Label,
			City:      l.City,
			Country:   l.Country,
			IsPrimary: l.IsPrimary,
			IsCurrent: active != nil && active.ID == l.ID,
		}
		if l.StartDate.Valid {
			s := l.StartDate.Time.Format("2006-01-02")
			item.StartDate = &s
		}
		if l.EndDate.Valid {
			s := l.EndDate.Time.Format("2006-01-02")
			item.EndDate = &s
		}
		out = append(out, item)
	}
	return out
}

func formatDisplayName(user sqlc.User) string {
	parts := make([]string, 0, 2)
	if name := strings.TrimSpace(derefString(user.FirstName)); name != "" {
		parts = append(parts, name)
	}
	if name := strings.TrimSpace(derefString(user.LastName)); name != "" {
		parts = append(parts, name)
	}
	return strings.Join(parts, " ")
}

func formatLocation(city, country string) string {
	parts := make([]string, 0, 2)
	if c := strings.TrimSpace(city); c != "" {
		parts = append(parts, c)
	}
	if c := strings.TrimSpace(country); c != "" {
		parts = append(parts, c)
	}
	return strings.Join(parts, ", ")
}

func formatWindows(windows []sqlc.ArtistAvailabilityWindow) []AvailabilityWindow {
	out := make([]AvailabilityWindow, 0, len(windows))
	for _, w := range windows {
		out = append(out, AvailabilityWindow{
			Weekday:     w.Weekday,
			StartMinute: w.StartMinute,
			EndMinute:   w.EndMinute,
		})
	}
	return out
}

func parseStringList(raw []byte) []string {
	values := []string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &values)
	}
	if values == nil {
		values = []string{}
	}
	return values
}

func parseFAQs(raw []byte) []FAQItem {
	faqs := []FAQItem{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &faqs)
	}
	if faqs == nil {
		faqs = []FAQItem{}
	}
	return faqs
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *service) presignKey(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	url, err := s.s3.PresignGet(ctx, key, presignViewTTL)
	if err != nil {
		s.log.Warn("openbook_presign_failed", "key", key, "error", err)
		return ""
	}
	return url
}
