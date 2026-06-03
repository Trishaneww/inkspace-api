package flashes

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

func flashFromRow(row sqlc.Flash, tiers []sqlc.FlashPricingTier) Flash {
	return Flash{
		ID:                  row.ID.String(),
		ArtistID:            row.ArtistID.String(),
		Status:              FlashStatus(row.Status),
		Title:               row.Title,
		Description:         row.Description,
		S3Key:               row.S3Key,
		ReferenceS3Key:      row.ReferenceS3Key,
		ColorType:           ColorType(row.ColorType),
		Styles:              defaultStringSlice(row.Styles),
		Placements:          defaultStringSlice(row.Placements),
		PricingMode:         PricingMode(row.PricingMode),
		FlatPriceCents:      row.FlatPriceCents,
		FlatDurationMinutes: row.FlatDurationMinutes,
		DepositCents:        row.DepositCents,
		Currency:            row.Currency,
		Repeatable:          row.Repeatable,
		ClaimedAt:           timestampPtr(row.ClaimedAt),
		ClaimedByBookingID:  uuidStringPtr(row.ClaimedByBookingID),
		ArchivedAt:          timestampPtr(row.ArchivedAt),
		PublishedAt:         timestampPtr(row.PublishedAt),
		ViewCount:           row.ViewCount,
		SaveCount:           row.SaveCount,
		PricingTiers:        pricingTiersFromRows(tiers),
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
}

func pricingTiersFromRows(rows []sqlc.FlashPricingTier) []PricingTier {
	out := make([]PricingTier, 0, len(rows))
	for _, r := range rows {
		out = append(out, PricingTier{
			SizeCode:        SizeCode(r.SizeCode),
			DurationMinutes: r.DurationMinutes,
			PriceCents:      r.PriceCents,
		})
	}
	return out
}

func timestampPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func uuidStringPtr(id pgtype.UUID) *string {
	if !id.Valid {
		return nil
	}

	s := formatUUIDBytes(id.Bytes)
	return &s
}

func formatUUIDBytes(b [16]byte) string {
	const hex = "0123456789abcdef"

	out := make([]byte, 36)
	pos := 0
	for i, by := range b {
		out[pos] = hex[by>>4]
		out[pos+1] = hex[by&0x0f]
		pos += 2
		if i == 3 || i == 5 || i == 7 || i == 9 {
			out[pos] = '-'
			pos++
		}
	}
	return string(out)
}
