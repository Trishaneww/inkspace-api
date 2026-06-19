package bookings

import (
	"time"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

const defaultSlotStepMinutes = 30

type busyInterval struct {
	start time.Time
	end   time.Time
}

func buildSlots(
	day time.Time,
	loc *time.Location,
	windows []sqlc.ArtistAvailabilityWindow,
	busyRows []sqlc.ListAppointmentsByArtistInRangeRow,
	duration int32,
	settings sqlc.ArtistSetting,
	now time.Time,
) []SlotOption {
	step := settings.SlotIntervalMinutes
	if step <= 0 {
		step = defaultSlotStepMinutes
	}
	buffer := time.Duration(settings.BufferMinutes) * time.Minute
	minStart := now.Add(time.Duration(settings.MinNoticeMinutes) * time.Minute)

	var maxStart time.Time
	hasMax := settings.MaxAdvanceDays != nil && *settings.MaxAdvanceDays > 0
	if hasMax {
		maxStart = now.AddDate(0, 0, int(*settings.MaxAdvanceDays))
	}

	busy := make([]busyInterval, 0, len(busyRows))
	for _, b := range busyRows {
		if !b.ScheduledStart.Valid {
			continue
		}
		start := b.ScheduledStart.Time.Add(-buffer)
		end := b.ScheduledStart.Time.Add(time.Duration(b.DurationMinutes)*time.Minute + buffer)
		busy = append(busy, busyInterval{start: start, end: end})
	}

	weekday := int32(day.Weekday()) // Sunday = 0, matches the stored windows.
	slots := []SlotOption{}
	seen := map[string]bool{}

	for _, w := range windows {
		if w.Weekday != weekday {
			continue
		}
		for m := w.StartMinute; m+duration <= w.EndMinute; m += step {
			start := time.Date(day.Year(), day.Month(), day.Day(), int(m/60), int(m%60), 0, 0, loc)
			end := start.Add(time.Duration(duration) * time.Minute)

			if start.Before(minStart) {
				continue
			}
			if hasMax && start.After(maxStart) {
				continue
			}
			if overlapsBusy(start, end, busy) {
				continue
			}

			key := start.UTC().Format(time.RFC3339)
			if seen[key] {
				continue
			}
			seen[key] = true
			slots = append(slots, SlotOption{Start: key, Label: start.Format("3:04 PM")})
		}
	}
	return slots
}

func containsSlot(slots []SlotOption, start string) bool {
	for _, sl := range slots {
		if sl.Start == start {
			return true
		}
	}
	return false
}

func overlapsBusy(start, end time.Time, busy []busyInterval) bool {
	for _, b := range busy {
		if start.Before(b.end) && end.After(b.start) {
			return true
		}
	}
	return false
}

func loadLocation(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}
