package flashes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

var (
	ErrNotFound      = errors.New("flash not found")
	ErrForbidden     = errors.New("not allowed")
	ErrInvalidInput  = errors.New("invalid input")
	ErrArtistMissing = errors.New("user has no artist profile")
)

const (
	presignUploadTTL = 15 * time.Minute
	presignViewTTL   = 1 * time.Hour

	defaultListLimit = 50
	maxListLimit     = 100
)

var contentTypeToExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

type Service interface {
	PresignFlashImageUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error)

	Create(ctx context.Context, userID uuid.UUID, input CreateFlashInput) (Flash, error)
	Get(ctx context.Context, flashID uuid.UUID) (Flash, error)
	ListByArtist(ctx context.Context, artistID uuid.UUID, filter ListFilter) (FlashListResult, error)
	ListForCurrentUser(ctx context.Context, userID uuid.UUID, filter ListFilter) (FlashListResult, error)
	Update(ctx context.Context, userID uuid.UUID, flashID uuid.UUID, input UpdateFlashInput) (Flash, error)
	Publish(ctx context.Context, userID uuid.UUID, flashID uuid.UUID) (Flash, error)
	Archive(ctx context.Context, userID uuid.UUID, flashID uuid.UUID) (Flash, error)
	Unarchive(ctx context.Context, userID uuid.UUID, flashID uuid.UUID) (Flash, error)
	Delete(ctx context.Context, userID uuid.UUID, flashID uuid.UUID) error
}

type service struct {
	repo   Repository
	events *events.Publisher
	s3     *s3client.Client
	log    *slog.Logger
}

func NewService(repo Repository, pub *events.Publisher, s3 *s3client.Client) Service {
	return &service{repo: repo, events: pub, s3: s3, log: slog.Default()}
}

func (s *service) PresignFlashImageUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error) {
	ext, ok := contentTypeToExt[contentType]
	if !ok {
		return PresignUploadResponse{}, fmt.Errorf("%w: unsupported content_type %q (allowed: image/jpeg, image/png, image/webp)", ErrInvalidInput, contentType)
	}

	key := fmt.Sprintf("flashes/%s/%s.%s", userID, uuid.New(), ext)

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

func (s *service) Create(ctx context.Context, userID uuid.UUID, input CreateFlashInput) (Flash, error) {
	if err := validateCreateInput(input); err != nil {
		return Flash{}, err
	}

	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Flash{}, err
	}

	status := string(FlashStatusDraft)
	publishedAt := pgtype.Timestamptz{}
	if input.Publish {
		status = string(FlashStatusAvailable)
		publishedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}

	params := sqlc.CreateFlashParams{
		ArtistID:            artist.ID,
		Status:              status,
		Title:               input.Title,
		Description:         input.Description,
		S3Key:               input.S3Key,
		ReferenceS3Key:      input.ReferenceS3Key,
		ColorType:           string(input.ColorType),
		Styles:              defaultStringSlice(input.Styles),
		Placements:          defaultStringSlice(input.Placements),
		PricingMode:         string(input.PricingMode),
		FlatPriceCents:      input.FlatPriceCents,
		FlatDurationMinutes: input.FlatDurationMinutes,
		DepositCents:        input.DepositCents,
		Repeatable:          input.Repeatable,
		PublishedAt:         publishedAt,
	}

	var created sqlc.Flash
	var tiers []sqlc.FlashPricingTier

	err = s.repo.RunInTx(ctx, func(txRepo Repository) error {
		row, err := txRepo.Create(ctx, params)
		if err != nil {
			return fmt.Errorf("create flash: %w", err)
		}
		created = row

		if input.PricingMode == PricingPerSize {
			for _, tier := range input.PricingTiers {
				saved, err := txRepo.UpsertPricingTier(ctx, sqlc.UpsertFlashPricingTierParams{
					FlashID:         row.ID,
					SizeCode:        string(tier.SizeCode),
					DurationMinutes: tier.DurationMinutes,
					PriceCents:      tier.PriceCents,
				})
				if err != nil {
					return fmt.Errorf("upsert pricing tier %s: %w", tier.SizeCode, err)
				}
				tiers = append(tiers, saved)
			}
		}
		return nil
	})
	if err != nil {
		return Flash{}, err
	}

	return s.hydrateFlash(ctx, created, tiers, s.artistCurrency(ctx, artist.ID))
}

func (s *service) Get(ctx context.Context, flashID uuid.UUID) (Flash, error) {
	row, err := s.repo.GetByID(ctx, flashID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Flash{}, ErrNotFound
		}
		return Flash{}, err
	}

	tiers, err := s.repo.ListPricingTiers(ctx, flashID)
	if err != nil {
		return Flash{}, fmt.Errorf("list pricing tiers: %w", err)
	}

	// Bump view count in the background — never block the read on it,
	// and never fail the request if the counter update errors.
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.IncrementViewCount(bgCtx, flashID); err != nil {
			s.log.Warn("flash_view_count_bump_failed", "flash_id", flashID, "error", err)
		}
	}()

	return s.hydrateFlash(ctx, row, tiers, s.artistCurrency(ctx, row.ArtistID))
}

func (s *service) ListByArtist(ctx context.Context, artistID uuid.UUID, filter ListFilter) (FlashListResult, error) {
	// Public callers never see drafts.
	if filter.Status == nil {
		availableStatus := FlashStatusAvailable
		filter.Status = &availableStatus
	} else if *filter.Status == FlashStatusDraft {
		return FlashListResult{}, ErrForbidden
	}
	return s.listFlashes(ctx, artistID, filter)
}

func (s *service) ListForCurrentUser(ctx context.Context, userID uuid.UUID, filter ListFilter) (FlashListResult, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return FlashListResult{}, err
	}
	return s.listFlashes(ctx, artist.ID, filter)
}

func (s *service) Update(ctx context.Context, userID, flashID uuid.UUID, input UpdateFlashInput) (Flash, error) {
	existing, err := s.repo.GetByID(ctx, flashID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Flash{}, ErrNotFound
		}
		return Flash{}, err
	}
	if err := s.requireOwnership(ctx, userID, existing); err != nil {
		return Flash{}, err
	}

	params := sqlc.UpdateFlashParams{
		ID:                  flashID,
		Title:               input.Title,
		Description:         input.Description,
		S3Key:               input.S3Key,
		ReferenceS3Key:      input.ReferenceS3Key,
		ClearReferenceS3Key: input.ClearReferenceImage,
		FlatPriceCents:      input.FlatPriceCents,
		FlatDurationMinutes: input.FlatDurationMinutes,
		DepositCents:        input.DepositCents,
		Repeatable:          input.Repeatable,
	}
	if input.ColorType != nil {
		ct := string(*input.ColorType)
		params.ColorType = &ct
	}
	if input.PricingMode != nil {
		pm := string(*input.PricingMode)
		params.PricingMode = &pm
	}
	if input.Styles != nil {
		params.Styles = *input.Styles
	}
	if input.Placements != nil {
		params.Placements = *input.Placements
	}

	var updated sqlc.Flash
	var tiers []sqlc.FlashPricingTier

	err = s.repo.RunInTx(ctx, func(txRepo Repository) error {
		row, err := txRepo.Update(ctx, params)
		if err != nil {
			return fmt.Errorf("update flash: %w", err)
		}
		updated = row

		if input.PricingTiers != nil {
			if err := txRepo.DeleteAllPricingTiers(ctx, flashID); err != nil {
				return fmt.Errorf("clear pricing tiers: %w", err)
			}
			for _, tier := range *input.PricingTiers {
				saved, err := txRepo.UpsertPricingTier(ctx, sqlc.UpsertFlashPricingTierParams{
					FlashID:         flashID,
					SizeCode:        string(tier.SizeCode),
					DurationMinutes: tier.DurationMinutes,
					PriceCents:      tier.PriceCents,
				})
				if err != nil {
					return fmt.Errorf("upsert pricing tier %s: %w", tier.SizeCode, err)
				}
				tiers = append(tiers, saved)
			}
		} else {
			loaded, err := txRepo.ListPricingTiers(ctx, flashID)
			if err != nil {
				return fmt.Errorf("list pricing tiers: %w", err)
			}
			tiers = loaded
		}
		return nil
	})
	if err != nil {
		return Flash{}, err
	}

	// The row no longer points at the old reference image, so drop the
	// bytes. Best-effort — the column is already cleared, an orphaned
	// object is harmless.
	if input.ClearReferenceImage && existing.ReferenceS3Key != nil && *existing.ReferenceS3Key != "" {
		if err := s.s3.Delete(ctx, *existing.ReferenceS3Key); err != nil {
			s.log.Warn("flash_reference_image_delete_failed", "flash_id", flashID, "key", *existing.ReferenceS3Key, "error", err)
		}
	}

	return s.hydrateFlash(ctx, updated, tiers, s.artistCurrency(ctx, updated.ArtistID))
}

func (s *service) Publish(ctx context.Context, userID, flashID uuid.UUID) (Flash, error) {
	existing, err := s.repo.GetByID(ctx, flashID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Flash{}, ErrNotFound
		}
		return Flash{}, err
	}
	if err := s.requireOwnership(ctx, userID, existing); err != nil {
		return Flash{}, err
	}
	if existing.Status != string(FlashStatusDraft) {
		return Flash{}, fmt.Errorf("%w: only draft flashes can be published", ErrInvalidInput)
	}
	if existing.S3Key == nil || *existing.S3Key == "" {
		return Flash{}, fmt.Errorf("%w: flash must have an image before publishing", ErrInvalidInput)
	}

	published, err := s.repo.Publish(ctx, flashID)
	if err != nil {
		return Flash{}, err
	}

	tiers, err := s.repo.ListPricingTiers(ctx, flashID)
	if err != nil {
		return Flash{}, err
	}
	return s.hydrateFlash(ctx, published, tiers, s.artistCurrency(ctx, published.ArtistID))
}

func (s *service) Archive(ctx context.Context, userID, flashID uuid.UUID) (Flash, error) {
	existing, err := s.repo.GetByID(ctx, flashID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Flash{}, ErrNotFound
		}
		return Flash{}, err
	}
	if err := s.requireOwnership(ctx, userID, existing); err != nil {
		return Flash{}, err
	}
	if existing.Status == string(FlashStatusArchived) {
		return Flash{}, fmt.Errorf("%w: flash is already archived", ErrInvalidInput)
	}

	archived, err := s.repo.Archive(ctx, flashID)
	if err != nil {
		return Flash{}, err
	}
	tiers, err := s.repo.ListPricingTiers(ctx, flashID)
	if err != nil {
		return Flash{}, err
	}
	return s.hydrateFlash(ctx, archived, tiers, s.artistCurrency(ctx, archived.ArtistID))
}

func (s *service) Unarchive(ctx context.Context, userID, flashID uuid.UUID) (Flash, error) {
	existing, err := s.repo.GetByID(ctx, flashID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Flash{}, ErrNotFound
		}
		return Flash{}, err
	}
	if err := s.requireOwnership(ctx, userID, existing); err != nil {
		return Flash{}, err
	}
	if existing.Status != string(FlashStatusArchived) {
		return Flash{}, fmt.Errorf("%w: only archived flashes can be unarchived", ErrInvalidInput)
	}

	unarchived, err := s.repo.Unarchive(ctx, flashID)
	if err != nil {
		return Flash{}, err
	}
	tiers, err := s.repo.ListPricingTiers(ctx, flashID)
	if err != nil {
		return Flash{}, err
	}
	return s.hydrateFlash(ctx, unarchived, tiers, s.artistCurrency(ctx, unarchived.ArtistID))
}

func (s *service) Delete(ctx context.Context, userID, flashID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, flashID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if err := s.requireOwnership(ctx, userID, existing); err != nil {
		return err
	}

	for _, key := range []*string{existing.S3Key, existing.ReferenceS3Key} {
		if key == nil || *key == "" {
			continue
		}
		if err := s.s3.Delete(ctx, *key); err != nil {
			s.log.Warn("flash_image_delete_failed", "flash_id", flashID, "key", *key, "error", err)
		}
	}

	return s.repo.Delete(ctx, flashID)
}

func (s *service) listFlashes(ctx context.Context, artistID uuid.UUID, filter ListFilter) (FlashListResult, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var statusFilter *string
	if filter.Status != nil {
		s := string(*filter.Status)
		statusFilter = &s
	}

	rows, err := s.repo.ListByArtist(ctx, sqlc.ListFlashesByArtistParams{
		ArtistID: artistID,
		Status:   statusFilter,
		Lim:      limit,
		Off:      offset,
	})
	if err != nil {
		return FlashListResult{}, err
	}
	counts, err := s.repo.CountByArtist(ctx, sqlc.CountFlashesByArtistParams{
		ArtistID: artistID,
		Status:   statusFilter,
	})
	if err != nil {
		return FlashListResult{}, err
	}

	flashIDs := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		flashIDs[i] = row.ID
	}
	tiers, err := s.repo.ListPricingTiersForFlashes(ctx, flashIDs)
	if err != nil {
		return FlashListResult{}, fmt.Errorf("list pricing tiers: %w", err)
	}
	tiersByFlash := make(map[uuid.UUID][]sqlc.FlashPricingTier, len(rows))
	for _, tier := range tiers {
		tiersByFlash[tier.FlashID] = append(tiersByFlash[tier.FlashID], tier)
	}

	currency := s.artistCurrency(ctx, artistID)
	items := make([]Flash, 0, len(rows))
	for _, row := range rows {
		f, err := s.hydrateFlash(ctx, row, tiersByFlash[row.ID], currency)
		if err != nil {
			return FlashListResult{}, err
		}
		items = append(items, f)
	}

	return FlashListResult{
		Items:     items,
		Total:     counts.Total,
		Available: counts.Available,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

// requireArtist resolves the caller's artist profile, lazily provisioning it
// if missing. The flashbook routes already gate the caller to the artist role,
// so creating the row on demand here is safe (the net for artists whose row
// wasn't created at activation).
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

func (s *service) requireOwnership(ctx context.Context, userID uuid.UUID, row sqlc.Flash) error {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return err
	}
	if artist.ID != row.ArtistID {
		return ErrForbidden
	}
	return nil
}

// artistCurrency resolves the currency a flash should be priced in from the
// artist's settings, falling back to CAD when no settings row exists yet.
func (s *service) artistCurrency(ctx context.Context, artistID uuid.UUID) string {
	settings, err := s.repo.GetArtistSettings(ctx, artistID)
	if err != nil {
		return "CAD"
	}
	return settings.Currency
}

func (s *service) hydrateFlash(ctx context.Context, row sqlc.Flash, tiers []sqlc.FlashPricingTier, currency string) (Flash, error) {
	flash := flashFromRow(row, tiers)
	flash.Currency = currency

	if flash.S3Key != nil && *flash.S3Key != "" {
		url, err := s.s3.PresignGet(ctx, *flash.S3Key, presignViewTTL)
		if err != nil {
			return Flash{}, fmt.Errorf("presign image url: %w", err)
		}
		flash.ImageURL = &url
	}
	if flash.ReferenceS3Key != nil && *flash.ReferenceS3Key != "" {
		url, err := s.s3.PresignGet(ctx, *flash.ReferenceS3Key, presignViewTTL)
		if err != nil {
			return Flash{}, fmt.Errorf("presign reference image url: %w", err)
		}
		flash.ReferenceImageURL = &url
	}
	return flash, nil
}

func validateCreateInput(input CreateFlashInput) error {
	switch input.ColorType {
	case ColorBlackAndGrey, ColorColor, ColorBoth:
	default:
		return fmt.Errorf("%w: color_type must be black_and_grey, color, or both", ErrInvalidInput)
	}

	switch input.PricingMode {
	case PricingFlat:
		if input.FlatPriceCents == nil || *input.FlatPriceCents <= 0 {
			return fmt.Errorf("%w: flat pricing requires flat_price_cents > 0", ErrInvalidInput)
		}
		if input.FlatDurationMinutes == nil || *input.FlatDurationMinutes <= 0 {
			return fmt.Errorf("%w: flat pricing requires flat_duration_minutes > 0", ErrInvalidInput)
		}
	case PricingPerSize:
		if len(input.PricingTiers) == 0 {
			return fmt.Errorf("%w: per_size pricing requires at least one pricing tier", ErrInvalidInput)
		}
		for _, tier := range input.PricingTiers {
			if err := validatePricingTier(tier); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: pricing_mode must be per_size or flat", ErrInvalidInput)
	}

	if input.Publish && (input.S3Key == nil || *input.S3Key == "") {
		return fmt.Errorf("%w: publish requires an uploaded image (s3_key)", ErrInvalidInput)
	}
	return nil
}

func validatePricingTier(tier PricingTier) error {
	switch tier.SizeCode {
	case SizeXSmall, SizeSmall, SizeMedium, SizeLarge, SizeXLarge:
	default:
		return fmt.Errorf("%w: invalid size_code %q", ErrInvalidInput, tier.SizeCode)
	}
	if tier.PriceCents <= 0 {
		return fmt.Errorf("%w: tier %s needs price_cents > 0", ErrInvalidInput, tier.SizeCode)
	}
	if tier.DurationMinutes <= 0 {
		return fmt.Errorf("%w: tier %s needs duration_minutes > 0", ErrInvalidInput, tier.SizeCode)
	}
	return nil
}

func defaultStringSlice(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
