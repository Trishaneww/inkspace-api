package portfolios

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

type ColorType string

const (
	ColorBlackAndGrey ColorType = "black_and_grey"
	ColorFullColor    ColorType = "color"
)

const dateLayout = "2006-01-02"

type PortfolioItem struct {
	ID               string     `json:"id"`
	ArtistID         string     `json:"artistId"`
	Status           Status     `json:"status"`
	Title            string     `json:"title"`
	Description      *string    `json:"description,omitempty"`
	CompletionDate   *string    `json:"completionDate,omitempty"`
	ImageKeys        []string   `json:"imageKeys"`
	ImageURLs        []string   `json:"imageUrls"`
	Styles           []string   `json:"styles"`
	Placement        *string    `json:"placement,omitempty"`
	ColorType        *ColorType `json:"colorType,omitempty"`
	ApproxSizeInches *int32     `json:"approxSizeInches,omitempty"`
	Healed           bool       `json:"healed"`
	SessionCount     *int32     `json:"sessionCount,omitempty"`
	TotalMinutes     *int32     `json:"totalMinutes,omitempty"`
	PublishedAt      *string    `json:"publishedAt,omitempty"`
	ArchivedAt       *string    `json:"archivedAt,omitempty"`
	CreatedAt        string     `json:"createdAt"`
	UpdatedAt        string     `json:"updatedAt"`
}

type ListResult struct {
	Items     []PortfolioItem `json:"items"`
	Total     int64           `json:"total"`
	Published int64           `json:"published"`
	Limit     int32           `json:"limit"`
	Offset    int32           `json:"offset"`
}

type ListFilter struct {
	Status *Status
	Limit  int32
	Offset int32
}

// ── Request payloads ─────────────────────────────────────────────────────────
type PresignUploadInput struct {
	ContentType string `json:"contentType" binding:"required"`
}

type PresignUploadResponse struct {
	URL       string    `json:"url"`
	S3Key     string    `json:"s3Key"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type CreateInput struct {
	Title            string     `json:"title" binding:"required"`
	Description      *string    `json:"description,omitempty"`
	CompletionDate   *string    `json:"completionDate,omitempty"`
	ImageKeys        []string   `json:"imageKeys"`
	Styles           []string   `json:"styles"`
	Placement        *string    `json:"placement,omitempty"`
	ColorType        *ColorType `json:"colorType,omitempty"`
	ApproxSizeInches *int32     `json:"approxSizeInches,omitempty"`
	Healed           bool       `json:"healed"`
	SessionCount     *int32     `json:"sessionCount,omitempty"`
	TotalMinutes     *int32     `json:"totalMinutes,omitempty"`
	Publish          bool       `json:"publish"`
}

type UpdateInput struct {
	Title            string     `json:"title" binding:"required"`
	Description      *string    `json:"description,omitempty"`
	CompletionDate   *string    `json:"completionDate,omitempty"`
	ImageKeys        []string   `json:"imageKeys"`
	Styles           []string   `json:"styles"`
	Placement        *string    `json:"placement,omitempty"`
	ColorType        *ColorType `json:"colorType,omitempty"`
	ApproxSizeInches *int32     `json:"approxSizeInches,omitempty"`
	Healed           bool       `json:"healed"`
	SessionCount     *int32     `json:"sessionCount,omitempty"`
	TotalMinutes     *int32     `json:"totalMinutes,omitempty"`
}

// ── Conversions ──────────────────────────────────────────────────────────────
func itemFromRow(row sqlc.PortfolioItem) PortfolioItem {
	out := PortfolioItem{
		ID:               row.ID.String(),
		ArtistID:         row.ArtistID.String(),
		Status:           Status(row.Status),
		Title:            row.Title,
		Description:      row.Description,
		CompletionDate:   formatDate(row.CompletionDate),
		ImageKeys:        row.ImageKeys,
		ImageURLs:        []string{},
		Styles:           row.Styles,
		Placement:        row.Placement,
		ApproxSizeInches: row.ApproxSizeInches,
		Healed:           row.Healed,
		SessionCount:     row.SessionCount,
		TotalMinutes:     row.TotalMinutes,
		PublishedAt:      formatTimestampPtr(row.PublishedAt),
		ArchivedAt:       formatTimestampPtr(row.ArchivedAt),
		CreatedAt:        formatTimestamp(row.CreatedAt),
		UpdatedAt:        formatTimestamp(row.UpdatedAt),
	}
	if out.ImageKeys == nil {
		out.ImageKeys = []string{}
	}
	if out.Styles == nil {
		out.Styles = []string{}
	}
	if row.ColorType != nil {
		ct := ColorType(*row.ColorType)
		out.ColorType = &ct
	}
	return out
}

func formatDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format(dateLayout)
	return &s
}

func parseDate(s *string) (pgtype.Date, error) {
	if s == nil || *s == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func formatTimestamp(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.UTC().Format(time.RFC3339)
}

func formatTimestampPtr(ts pgtype.Timestamptz) *string {
	if !ts.Valid {
		return nil
	}
	s := ts.Time.UTC().Format(time.RFC3339)
	return &s
}
