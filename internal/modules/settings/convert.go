package settings

import (
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

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
	if u.Phone != nil {
		out.Phone = *u.Phone
	}
	if u.AvatarUrl != nil {
		out.AvatarURL = *u.AvatarUrl
	}
	if u.InstagramUrl != nil {
		out.InstagramURL = *u.InstagramUrl
	}
	return out
}

func settingsFromRow(r sqlc.ArtistSetting) ArtistSettings {
	out := ArtistSettings{
		StudioName:              r.StudioName,
		StudioAddress:           r.StudioAddress,
		StudioCity:              r.StudioCity,
		StudioProvince:          r.StudioProvince,
		StudioPostalCode:        r.StudioPostalCode,
		StudioCountry:           r.StudioCountry,
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
		NotifyByEmail:           r.NotifyByEmail,
		NotifyBySms:             r.NotifyBySms,
		Styles:                  r.Styles,
	}
	if out.Styles == nil {
		out.Styles = []string{}
	}
	if r.GoogleCalendarEmail != nil && *r.GoogleCalendarEmail != "" {
		out.GoogleCalendarConnected = true
		out.GoogleCalendarEmail = *r.GoogleCalendarEmail
	}
	if r.WaiverFileUrl != nil {
		out.WaiverFileURL = *r.WaiverFileUrl
	}
	return out
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
