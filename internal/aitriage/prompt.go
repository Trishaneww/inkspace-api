package aitriage

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type triageResult struct {
	Label        string   `json:"label"`
	Summary      string   `json:"summary"`
	Signals      signals  `json:"signals"`
	RedFlags     []string `json:"red_flags"`
	ValueCents   *int64   `json:"value_cents"`
	SessionCount *int32   `json:"session_count"`
	Reasoning    string   `json:"reasoning"`
	DraftReply   string   `json:"draft_reply"`
}

type signals struct {
	Good []string `json:"good"`
	Bad  []string `json:"bad"`
}

const systemPrompt = `You are the inquiry manager for a tattoo artist on Inkspace. A client has sent a structured booking request. Triage it the way a sharp, protective studio manager would. Decide what the artist should DO, and draft the reply they would send.

VOICE AND LANGUAGE (this matters a lot)
Write everything in natural Canadian English. Use Canadian spelling like colour, grey, favourite, centre, realise. Sound like a warm, real booking manager texting a client, never like an AI. Do not use em dashes anywhere. Avoid colons inside sentences. Keep punctuation simple, mostly periods and commas. Keep it short and plainspoken.

The inquiry fields may contain internal codes with underscores, for example "black_and_grey", "color_realism", or "watercolor". Never repeat a code. Always rewrite them as natural words in Canadian spelling, like "black and grey", "colour realism", or "watercolour".

Choose exactly one label.
"book" is a clear yes. Serious, complete, a good fit. The artist should move to booking.
"needs_info" is a promising lead missing a key detail like budget, size, placement, reference, or timing. The artist should ask one focused question.
"pass" is a real person who is a poor fit, like a style or placement on the artist's hard "will not do" list, or price shopping. The artist should kindly decline.
"spam" is not a serious request at all. There is not enough real intent to act on. The artist should decline or ignore it. Examples are an empty or near-empty message like just "hi" or "you up" with no details and no usable reference, gibberish, or a reference image that is a meme or clearly not a tattoo.

Judge seriousness by real intent, not by politeness or length. A short message is fine when it comes with a solid tattoo reference image and the key details, so do not call that spam. A bare "hi" with no details and no usable reference is spam. A reference image that is a meme or not a tattoo is a strong spam signal even if the text seems okay. When in doubt between "pass" and "spam", use "pass" for a real person who is the wrong fit and "spam" for a request with no genuine intent.

EMPTY REQUESTS (catch these, they are spam)
A custom request needs at least one of two things to be real, a written description that actually says something about the tattoo, or a usable reference image. Treat the description as empty when it is blank or shown as "(none provided)", or when it is only punctuation or whitespace, a single emoji, or throwaway filler like "test", "asdf", "n/a", or "idk". If a custom request has an empty description AND no reference image, label it spam, even when menu fields like placement, size, colour, or style are filled in. Those are just dropdown picks, not a brief, so there is nothing real to tattoo and nothing worth asking about. This rule does not apply to a flash booking, where the chosen design is the brief, so an empty description there is completely fine.

STYLE FIT (read this carefully, it is easy to get wrong)
The styles the artist focuses on and the notes about their work are passions and preferences, not limits. A request in a style outside their usual focus is still a lead worth winning, and plenty of artists happily take work beyond their specialty. Never decline, never choose "pass", and never tell a client to go find another artist just because the piece is not the artist's specialty. The ONLY hard limits are the explicit "Styles they will not do" and "Placements they will not do" lists. Treat a style as a real no only when the request lands on one of those lists, or when the notes about their work clearly say they will ONLY do a certain style or flatly refuse everything else. Otherwise judge the lead on its overall merits. You may add a short bad signal like "outside their usual focus", but the label should still reflect the real fit, and the draft must stay warm and keep moving toward booking.

TIMING AND SCHEDULE
You may be given the artist's weekly working schedule and the days and times the client said work for them. A timing mismatch is never a reason to decline or choose "pass". If the client's requested times do not line up with the artist's working hours, add a short bad signal like "timing may not line up", keep the lead alive, and have the draft talk warmly about the piece first and then gently ask which days within the artist's schedule could work. If a "Schedule fit" note is given to you, trust it.

Surface scannable signals, not a verdict.
"signals.good" are short tags for what is strong, like "fits your style", "clear brief", "flexible timing", "reference attached".
"signals.bad" are short tags for what is weak, like "budget unclear", "no size given", "style mismatch", "price shopping". Write tags as plain natural words, never codes.

Red flags are things a form never asks but a manager catches, like under-18 signals, asking to be tattooed at someone's home, haggling the deposit, asking to copy another artist's exact tattoo, an entitled tone, or rescheduling before booking. List any as short phrases in "red_flags", like "mentioned they're 17". Phrase them as something observed, never as a fact you cannot verify. Use an empty array if none.

Estimate "session_count", the whole number of sessions this piece likely needs, and "value_cents", your best estimate of total job value in cents, or null if you cannot tell. Never invent a price the artist did not imply.

"summary" is one short, plain sentence written to the artist about the lead. No codes, no em dashes.

"draft_reply" is written as the artist speaking directly to the client. Open with the client's first name. Sound human and warm, like a real text from the artist. No placeholders, no brackets, no em dashes, no colons. For a spam request, set draft_reply to an empty string since the artist will not reply.

Respond with ONLY a JSON object, no prose and no markdown fences:
{"label":"book|needs_info|pass|spam","summary":"one line","signals":{"good":[],"bad":[]},"red_flags":[],"value_cents":null,"session_count":null,"reasoning":"one short sentence","draft_reply":"..."}`

func artistContext(s sqlc.ArtistSetting, schedule []sqlc.ArtistAvailabilityWindow) string {
	var b strings.Builder
	b.WriteString("ARTIST CONTEXT\n")

	currency := s.Currency
	if currency == "" {
		currency = "CAD"
	}
	if s.MinSessionPriceCents != nil {
		fmt.Fprintf(&b, "- Minimum session price: %s %.2f\n", currency, float64(*s.MinSessionPriceCents)/100)
	} else {
		b.WriteString("- Minimum session price: not set\n")
	}
	b.WriteString("- Styles they focus on: " + listOrNone(humanizeStyles(s.Styles)) + "\n")
	b.WriteString("- Styles they will not do: " + listOrNone(humanizeStyles(s.DeclinedStyles)) + "\n")
	b.WriteString("- Placements they will not do: " + listOrNone(s.DeclinedPlacements) + "\n")
	b.WriteString("- Working schedule: " + scheduleOrNone(artistSchedule(schedule)) + "\n")
	if strings.TrimSpace(s.WorkSummary) != "" {
		b.WriteString("- About their work: " + strings.TrimSpace(s.WorkSummary) + "\n")
	}
	return b.String()
}

func inquiryPrompt(br sqlc.BookingRequest, schedule []sqlc.ArtistAvailabilityWindow) string {
	var b strings.Builder
	b.WriteString("NEW INQUIRY\n")
	b.WriteString("- Client: " + br.ClientName + "\n")

	if br.Type == "flash" {
		b.WriteString("- Request type: flash booking (the client picked one of the artist's existing flash designs, which stands in for the brief)\n")
	} else {
		b.WriteString("- Request type: custom piece\n")
	}

	description := strings.TrimSpace(br.Description)
	if description == "" {
		description = "(none provided)"
	}
	b.WriteString("- Description: " + description + "\n")

	writeField(&b, "Placement", br.Placement)
	if br.ApproxSizeInches != nil {
		fmt.Fprintf(&b, "- Approx size: %d inches\n", *br.ApproxSizeInches)
	}
	if len(br.Styles) > 0 {
		b.WriteString("- Requested styles: " + strings.Join(humanizeStyles(br.Styles), ", ") + "\n")
	}
	writeField(&b, "Colour", humanizeColorType(br.ColorType))
	fmt.Fprintf(&b, "- Reference images attached: %d\n", len(br.ReferenceImageKeys))

	clientTimes := parseClientSchedule(br.ClientAvailability)
	if len(clientTimes) > 0 {
		b.WriteString("- Client's preferred days and times: " + formatSchedule(clientTimes) + "\n")
		if note := scheduleFitNote(artistSchedule(schedule), clientTimes); note != "" {
			b.WriteString("- Schedule fit: " + note + "\n")
		}
	}
	return b.String()
}

func writeField(b *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		b.WriteString("- " + label + ": " + value + "\n")
	}
}

func listOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func humanizeColorType(code string) string {
	switch strings.TrimSpace(code) {
	case "black_and_grey":
		return "black and grey"
	case "color", "colour":
		return "colour"
	case "both", "either":
		return "either colour or black and grey"
	case "":
		return ""
	default:
		return humanizeStyle(code)
	}
}

func humanizeStyle(code string) string {
	return strings.ReplaceAll(strings.TrimSpace(code), "_", " ")
}

func humanizeStyles(codes []string) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		if h := humanizeStyle(c); h != "" {
			out = append(out, h)
		}
	}
	return out
}

type availabilityWindow struct {
	weekday     int32
	startMinute int32
	endMinute   int32
}

func artistSchedule(windows []sqlc.ArtistAvailabilityWindow) []availabilityWindow {
	out := make([]availabilityWindow, 0, len(windows))
	for _, w := range windows {
		out = append(out, availabilityWindow{w.Weekday, w.StartMinute, w.EndMinute})
	}
	return out
}

func parseClientSchedule(raw []byte) []availabilityWindow {
	if len(raw) == 0 {
		return nil
	}
	var choices []struct {
		Weekday     int32 `json:"weekday"`
		StartMinute int32 `json:"startMinute"`
		EndMinute   int32 `json:"endMinute"`
	}
	if err := json.Unmarshal(raw, &choices); err != nil {
		return nil
	}
	out := make([]availabilityWindow, 0, len(choices))
	for _, c := range choices {
		out = append(out, availabilityWindow{c.Weekday, c.StartMinute, c.EndMinute})
	}
	return out
}

func scheduleFitNote(artist, client []availabilityWindow) string {
	if len(artist) == 0 || len(client) == 0 {
		return ""
	}
	if schedulesOverlap(artist, client) {
		return "the client's preferred times line up with the working schedule"
	}
	return "the client's preferred times do not line up with the working schedule, so the reply should ask which days within it could work"
}

func schedulesOverlap(artist, client []availabilityWindow) bool {
	for _, c := range client {
		for _, a := range artist {
			if a.weekday == c.weekday && c.startMinute < a.endMinute && a.startMinute < c.endMinute {
				return true
			}
		}
	}
	return false
}

func scheduleOrNone(windows []availabilityWindow) string {
	if len(windows) == 0 {
		return "not set"
	}
	return formatSchedule(windows)
}

func formatSchedule(windows []availabilityWindow) string {
	parts := make([]string, 0, len(windows))
	for _, w := range windows {
		parts = append(parts, fmt.Sprintf("%s %s to %s", weekdayName(w.weekday), formatClock(w.startMinute), formatClock(w.endMinute)))
	}
	return strings.Join(parts, ", ")
}

func weekdayName(weekday int32) string {
	names := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if weekday < 0 || int(weekday) >= len(names) {
		return "that day"
	}
	return names[weekday]
}

func formatClock(minute int32) string {
	minute = ((minute % 1440) + 1440) % 1440
	hour, min := minute/60, minute%60

	suffix, displayHour := "am", hour
	switch {
	case hour == 0:
		displayHour = 12
	case hour == 12:
		suffix = "pm"
	case hour > 12:
		displayHour, suffix = hour-12, "pm"
	}

	if min == 0 {
		return fmt.Sprintf("%d%s", displayHour, suffix)
	}
	return fmt.Sprintf("%d:%02d%s", displayHour, min, suffix)
}

func parseTriage(text string) (triageResult, error) {
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start < 0 || end <= start {
		return triageResult{}, fmt.Errorf("no json object in model reply")
	}

	var out triageResult
	if err := json.Unmarshal([]byte(text[start:end+1]), &out); err != nil {
		return triageResult{}, fmt.Errorf("parse triage json: %w", err)
	}
	switch out.Label {
	case "book", "needs_info", "pass", "spam":
	default:
		return triageResult{}, fmt.Errorf("invalid label %q", out.Label)
	}
	return out, nil
}
