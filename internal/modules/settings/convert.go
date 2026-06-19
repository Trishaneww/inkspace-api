package settings

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

const dateLayout = "2006-01-02"

func accountFromUser(u sqlc.User) Account {
	out := Account{
		ID:    u.ID.String(),
		Email: u.Email,
	}
	if u.FirstName != nil {
		out.FirstName = *u.FirstName
	}
	if u.LastName != nil {
		out.LastName = *u.LastName
	}
	if u.Username != nil {
		out.Username = *u.Username
	}
	out.Phone = u.Phone
	if u.AvatarURL != nil {
		out.AvatarURL = *u.AvatarURL
	}
	if u.InstagramURL != nil {
		out.InstagramURL = *u.InstagramURL
	}
	return out
}

func settingsFromRow(r sqlc.ArtistSetting) ArtistSettings {
	out := ArtistSettings{
		StripeConnected:         r.StripeAccountID != nil && *r.StripeAccountID != "",
		StripeChargesEnabled:    r.StripeChargesEnabled,
		StripePayoutsEnabled:    r.StripePayoutsEnabled,
		StripeDetailsSubmitted:  r.StripeDetailsSubmitted,
		PayoutFrequency:         r.PayoutFrequency,
		Currency:                r.Currency,
		DepositFlatFeeCents:     r.DepositFlatFeeCents,
		PlatformFeePayer:        r.PlatformFeePayer,
		DepositRefundPolicy:     r.DepositRefundPolicy,
		CancellationNoticeHours: r.CancellationNoticeHours,
		AcceptingBookings:       r.AcceptingBookings,
		Timezone:                r.Timezone,
		SlotIntervalMinutes:     r.SlotIntervalMinutes,
		BufferMinutes:           r.BufferMinutes,
		MinNoticeMinutes:        r.MinNoticeMinutes,
		MaxAdvanceDays:          r.MaxAdvanceDays,
		TermsText:               r.TermsText,
		TermsShowOnBooking:      r.TermsShowOnBooking,
		TermsShowAtDeposit:      r.TermsShowAtDeposit,
		WaiverRequired:          r.WaiverRequired,
		Aftercare:               r.Aftercare,
		FAQs:                    parseFAQs(r.Faqs),
		NotifyByEmail:           r.NotifyByEmail,
		NotifyBySMS:             r.NotifyBySMS,
		Styles:                  r.Styles,
	}
	if out.Styles == nil {
		out.Styles = []string{}
	}
	if r.GoogleCalendarEmail != nil && *r.GoogleCalendarEmail != "" {
		out.GoogleCalendarConnected = true
		out.GoogleCalendarEmail = *r.GoogleCalendarEmail
	}
	if r.WaiverFileURL != nil {
		out.WaiverFileURL = *r.WaiverFileURL
	}
	if r.CurrentLocationID.Valid {
		out.CurrentLocationID = uuid.UUID(r.CurrentLocationID.Bytes).String()
	}
	return out
}

func getLocationFromRow(r sqlc.ArtistLocation) Location {
	out := Location{
		ID:         r.ID.String(),
		Label:      r.Label,
		Address:    r.Address,
		City:       r.City,
		Province:   r.Province,
		PostalCode: r.PostalCode,
		Country:    r.Country,
		Timezone:   r.Timezone,
		IsPrimary:  r.IsPrimary,
	}
	if r.StartDate.Valid {
		s := r.StartDate.Time.Format(dateLayout)
		out.StartDate = &s
	}
	if r.EndDate.Valid {
		s := r.EndDate.Time.Format(dateLayout)
		out.EndDate = &s
	}
	return out
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

func availabilityFromRow(r sqlc.ArtistAvailabilityWindow) AvailabilityWindow {
	return AvailabilityWindow{
		ID:          r.ID.String(),
		Weekday:     r.Weekday,
		StartMinute: r.StartMinute,
		EndMinute:   r.EndMinute,
	}
}

func presetFromRow(r sqlc.ArtistSessionPreset) SessionPreset {
	return SessionPreset{
		ID:                    r.ID.String(),
		Name:                  r.Name,
		Description:           r.Description,
		ApproxDurationMinutes: r.ApproxDurationMinutes,
		Position:              r.Position,
	}
}

func blocklistFromRow(r sqlc.ArtistBlocklist) BlocklistEntry {
	out := BlocklistEntry{
		ID:   r.ID.String(),
		Note: r.Note,
	}
	if r.Email != nil {
		out.Email = *r.Email
	}
	if r.Phone != nil {
		out.Phone = *r.Phone
	}
	return out
}
