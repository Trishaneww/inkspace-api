package portfolios

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
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

var (
	ErrNotFound     = errors.New("portfolio item not found")
	ErrForbidden    = errors.New("not allowed")
	ErrInvalidInput = errors.New("invalid input")
)

const (
	presignUploadTTL = 15 * time.Minute
	presignViewTTL   = 1 * time.Hour

	defaultListLimit = 50
	maxListLimit     = 100

	maxImages = 3
)

var contentTypeToExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

type Service interface {
	PresignImageUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error)

	Create(ctx context.Context, userID uuid.UUID, input CreateInput) (PortfolioItem, error)
	ListByArtist(ctx context.Context, artistID uuid.UUID, filter ListFilter) (ListResult, error)
	ListForCurrentUser(ctx context.Context, userID uuid.UUID, filter ListFilter) (ListResult, error)
	Update(ctx context.Context, userID, itemID uuid.UUID, input UpdateInput) (PortfolioItem, error)
	Publish(ctx context.Context, userID, itemID uuid.UUID) (PortfolioItem, error)
	Archive(ctx context.Context, userID, itemID uuid.UUID) (PortfolioItem, error)
	Unarchive(ctx context.Context, userID, itemID uuid.UUID) (PortfolioItem, error)
	Delete(ctx context.Context, userID, itemID uuid.UUID) error
}

type service struct {
	repo Repository
	s3   *s3client.Client
	log  *slog.Logger
}

func NewService(repo Repository, s3 *s3client.Client) Service {
	return &service{repo: repo, s3: s3, log: slog.Default()}
}

func (s *service) PresignImageUpload(ctx context.Context, userID uuid.UUID, contentType string) (PresignUploadResponse, error) {
	ext, ok := contentTypeToExt[contentType]
	if !ok {
		return PresignUploadResponse{}, fmt.Errorf("%w: unsupported contentType %q (allowed: image/jpeg, image/png, image/webp)", ErrInvalidInput, contentType)
	}

	key := fmt.Sprintf("portfolio/%s/%s.%s", userID, uuid.New(), ext)

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

func (s *service) Create(ctx context.Context, userID uuid.UUID, input CreateInput) (PortfolioItem, error) {
	if err := validateImages(input.ImageKeys); err != nil {
		return PortfolioItem{}, err
	}
	if err := validateColorType(input.ColorType); err != nil {
		return PortfolioItem{}, err
	}

	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return PortfolioItem{}, err
	}

	completionDate, err := parseDate(input.CompletionDate)
	if err != nil {
		return PortfolioItem{}, fmt.Errorf("%w: completionDate must be YYYY-MM-DD", ErrInvalidInput)
	}

	status := string(StatusDraft)
	publishedAt := pgtype.Timestamptz{}
	if input.Publish {
		status = string(StatusPublished)
		publishedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}

	row, err := s.repo.Create(ctx, sqlc.CreatePortfolioItemParams{
		ArtistID:         artist.ID,
		Status:           status,
		Title:            input.Title,
		Description:      input.Description,
		CompletionDate:   completionDate,
		ImageKeys:        input.ImageKeys,
		Styles:           defaultStringSlice(input.Styles),
		Placement:        input.Placement,
		ColorType:        colorTypePtr(input.ColorType),
		ApproxSizeInches: input.ApproxSizeInches,
		Healed:           input.Healed,
		SessionCount:     input.SessionCount,
		TotalMinutes:     input.TotalMinutes,
		PublishedAt:      publishedAt,
	})
	if err != nil {
		return PortfolioItem{}, fmt.Errorf("create portfolio item: %w", err)
	}
	return s.hydrate(ctx, row)
}

func (s *service) ListByArtist(ctx context.Context, artistID uuid.UUID, filter ListFilter) (ListResult, error) {
	// Public callers only ever see published pieces.
	published := StatusPublished
	filter.Status = &published
	return s.list(ctx, artistID, filter)
}

func (s *service) ListForCurrentUser(ctx context.Context, userID uuid.UUID, filter ListFilter) (ListResult, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return ListResult{}, err
	}
	return s.list(ctx, artist.ID, filter)
}

func (s *service) Update(ctx context.Context, userID, itemID uuid.UUID, input UpdateInput) (PortfolioItem, error) {
	if err := validateImages(input.ImageKeys); err != nil {
		return PortfolioItem{}, err
	}
	if err := validateColorType(input.ColorType); err != nil {
		return PortfolioItem{}, err
	}

	existing, err := s.requireOwned(ctx, userID, itemID)
	if err != nil {
		return PortfolioItem{}, err
	}

	completionDate, err := parseDate(input.CompletionDate)
	if err != nil {
		return PortfolioItem{}, fmt.Errorf("%w: completionDate must be YYYY-MM-DD", ErrInvalidInput)
	}

	updated, err := s.repo.Update(ctx, sqlc.UpdatePortfolioItemParams{
		ID:               itemID,
		Title:            input.Title,
		Description:      input.Description,
		CompletionDate:   completionDate,
		ImageKeys:        input.ImageKeys,
		Styles:           defaultStringSlice(input.Styles),
		Placement:        input.Placement,
		ColorType:        colorTypePtr(input.ColorType),
		ApproxSizeInches: input.ApproxSizeInches,
		Healed:           input.Healed,
		SessionCount:     input.SessionCount,
		TotalMinutes:     input.TotalMinutes,
	})
	if err != nil {
		return PortfolioItem{}, fmt.Errorf("update portfolio item: %w", err)
	}

	// Drop any images that were swapped out. Best-effort — an orphaned object
	// is harmless, and the row no longer references it.
	s.deleteOrphanedImages(ctx, existing.ImageKeys, input.ImageKeys)

	return s.hydrate(ctx, updated)
}

func (s *service) Publish(ctx context.Context, userID, itemID uuid.UUID) (PortfolioItem, error) {
	existing, err := s.requireOwned(ctx, userID, itemID)
	if err != nil {
		return PortfolioItem{}, err
	}
	if existing.Status != string(StatusDraft) {
		return PortfolioItem{}, fmt.Errorf("%w: only draft items can be published", ErrInvalidInput)
	}
	row, err := s.repo.Publish(ctx, itemID)
	if err != nil {
		return PortfolioItem{}, err
	}
	return s.hydrate(ctx, row)
}

func (s *service) Archive(ctx context.Context, userID, itemID uuid.UUID) (PortfolioItem, error) {
	existing, err := s.requireOwned(ctx, userID, itemID)
	if err != nil {
		return PortfolioItem{}, err
	}
	if existing.Status == string(StatusArchived) {
		return PortfolioItem{}, fmt.Errorf("%w: item is already archived", ErrInvalidInput)
	}
	row, err := s.repo.Archive(ctx, itemID)
	if err != nil {
		return PortfolioItem{}, err
	}
	return s.hydrate(ctx, row)
}

func (s *service) Unarchive(ctx context.Context, userID, itemID uuid.UUID) (PortfolioItem, error) {
	existing, err := s.requireOwned(ctx, userID, itemID)
	if err != nil {
		return PortfolioItem{}, err
	}
	if existing.Status != string(StatusArchived) {
		return PortfolioItem{}, fmt.Errorf("%w: only archived items can be unarchived", ErrInvalidInput)
	}
	row, err := s.repo.Unarchive(ctx, itemID)
	if err != nil {
		return PortfolioItem{}, err
	}
	return s.hydrate(ctx, row)
}

func (s *service) Delete(ctx context.Context, userID, itemID uuid.UUID) error {
	existing, err := s.requireOwned(ctx, userID, itemID)
	if err != nil {
		return err
	}
	for _, key := range existing.ImageKeys {
		if key == "" {
			continue
		}
		if err := s.s3.Delete(ctx, key); err != nil {
			s.log.Warn("portfolio_image_delete_failed", "item_id", itemID, "key", key, "error", err)
		}
	}
	return s.repo.Delete(ctx, itemID)
}

// ── Internals ────────────────────────────────────────────────────────────────
func (s *service) list(ctx context.Context, artistID uuid.UUID, filter ListFilter) (ListResult, error) {
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
		v := string(*filter.Status)
		statusFilter = &v
	}

	rows, err := s.repo.ListByArtist(ctx, sqlc.ListPortfolioItemsByArtistParams{
		ArtistID: artistID,
		Status:   statusFilter,
		Lim:      limit,
		Off:      offset,
	})
	if err != nil {
		return ListResult{}, err
	}
	counts, err := s.repo.CountByArtist(ctx, sqlc.CountPortfolioItemsByArtistParams{
		ArtistID: artistID,
		Status:   statusFilter,
	})
	if err != nil {
		return ListResult{}, err
	}

	items := make([]PortfolioItem, 0, len(rows))
	for _, row := range rows {
		item, err := s.hydrate(ctx, row)
		if err != nil {
			return ListResult{}, err
		}
		items = append(items, item)
	}

	return ListResult{
		Items:     items,
		Total:     counts.Total,
		Published: counts.Published,
		Limit:     limit,
		Offset:    offset,
	}, nil
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

// requireOwned loads the item and asserts the caller's artist owns it.
func (s *service) requireOwned(ctx context.Context, userID, itemID uuid.UUID) (sqlc.PortfolioItem, error) {
	row, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.PortfolioItem{}, ErrNotFound
		}
		return sqlc.PortfolioItem{}, err
	}
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return sqlc.PortfolioItem{}, err
	}
	if artist.ID != row.ArtistID {
		return sqlc.PortfolioItem{}, ErrForbidden
	}
	return row, nil
}

func (s *service) hydrate(ctx context.Context, row sqlc.PortfolioItem) (PortfolioItem, error) {
	item := itemFromRow(row)
	for _, key := range item.ImageKeys {
		if key == "" {
			continue
		}
		url, err := s.s3.PresignGet(ctx, key, presignViewTTL)
		if err != nil {
			return PortfolioItem{}, fmt.Errorf("presign image url: %w", err)
		}
		item.ImageURLs = append(item.ImageURLs, url)
	}
	return item, nil
}

func (s *service) deleteOrphanedImages(ctx context.Context, oldKeys, newKeys []string) {
	keep := make(map[string]bool, len(newKeys))
	for _, k := range newKeys {
		keep[k] = true
	}
	for _, key := range oldKeys {
		if key == "" || keep[key] {
			continue
		}
		if err := s.s3.Delete(ctx, key); err != nil {
			s.log.Warn("portfolio_image_delete_failed", "key", key, "error", err)
		}
	}
}

func validateImages(keys []string) error {
	if len(keys) == 0 {
		return fmt.Errorf("%w: at least one photo is required", ErrInvalidInput)
	}
	if len(keys) > maxImages {
		return fmt.Errorf("%w: at most %d photos are allowed", ErrInvalidInput, maxImages)
	}
	for _, k := range keys {
		if k == "" {
			return fmt.Errorf("%w: image keys must be non-empty", ErrInvalidInput)
		}
	}
	return nil
}

func validateColorType(ct *ColorType) error {
	if ct == nil {
		return nil
	}
	switch *ct {
	case ColorBlackAndGrey, ColorFullColor:
		return nil
	default:
		return fmt.Errorf("%w: colorType must be black_and_grey or color", ErrInvalidInput)
	}
}

func colorTypePtr(ct *ColorType) *string {
	if ct == nil {
		return nil
	}
	v := string(*ct)
	return &v
}

func defaultStringSlice(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
