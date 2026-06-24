package aitriage

import (
	"strings"
	"testing"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

func settingWithMin(min *int64) sqlc.ArtistSetting {
	return sqlc.ArtistSetting{MinSessionPriceCents: min}
}

func TestParseTriage(t *testing.T) {
	t.Run("plain json", func(t *testing.T) {
		out, err := parseTriage(`{"label":"book","summary":"Fineline forearm snake","signals":{"good":["fits your style"],"bad":[]},"red_flags":[],"value_cents":40000,"session_count":1,"reasoning":"clear brief","draft_reply":"Hey — love this one."}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Label != "book" || out.SessionCount == nil || *out.SessionCount != 1 {
			t.Errorf("bad parse: %+v", out)
		}
		if len(out.Signals.Good) != 1 {
			t.Errorf("expected one good signal, got %v", out.Signals.Good)
		}
	})

	t.Run("tolerates fences and prose", func(t *testing.T) {
		out, err := parseTriage("Here you go:\n```json\n{\"label\":\"pass\",\"summary\":\"s\",\"signals\":{\"good\":[],\"bad\":[\"style mismatch\"]},\"red_flags\":[\"mentioned they're 17\"],\"reasoning\":\"r\",\"draft_reply\":\"d\"}\n```")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Label != "pass" || len(out.RedFlags) != 1 {
			t.Errorf("bad parse: %+v", out)
		}
	})

	t.Run("rejects invalid label", func(t *testing.T) {
		if _, err := parseTriage(`{"label":"maybe"}`); err == nil {
			t.Error("expected error for invalid label")
		}
	})

	t.Run("rejects non-json", func(t *testing.T) {
		if _, err := parseTriage("no json here"); err == nil {
			t.Error("expected error for missing json")
		}
	})
}

func TestEstimateValueCents(t *testing.T) {
	floor := int64(20000)
	two := int32(2)

	// Deterministic floor × sessions wins over the model's number.
	withFloor := estimateValueCents(
		settingWithMin(&floor),
		triageResult{SessionCount: &two, ValueCents: ptrInt64(99999)},
	)
	if withFloor == nil || *withFloor != 40000 {
		t.Errorf("floor estimate = %v, want 40000", withFloor)
	}

	// No floor → fall back to the model's estimate.
	modelVal := ptrInt64(55000)
	noFloor := estimateValueCents(settingWithMin(nil), triageResult{SessionCount: &two, ValueCents: modelVal})
	if noFloor != modelVal {
		t.Errorf("expected model fallback, got %v", noFloor)
	}
}

func ptrInt64(v int64) *int64 { return &v }

func TestFormatClock(t *testing.T) {
	cases := map[int32]string{
		0:    "12am",
		540:  "9am",
		720:  "12pm",
		780:  "1pm",
		810:  "1:30pm",
		1020: "5pm",
		1439: "11:59pm",
	}
	for minute, want := range cases {
		if got := formatClock(minute); got != want {
			t.Errorf("formatClock(%d) = %q, want %q", minute, got, want)
		}
	}
}

func TestFormatSchedule(t *testing.T) {
	windows := []availabilityWindow{
		{weekday: 2, startMinute: 540, endMinute: 720},
		{weekday: 5, startMinute: 480, endMinute: 1020},
	}
	want := "Tuesday 9am to 12pm, Friday 8am to 5pm"
	if got := formatSchedule(windows); got != want {
		t.Errorf("formatSchedule = %q, want %q", got, want)
	}
}

func TestSchedulesOverlap(t *testing.T) {
	artist := []availabilityWindow{
		{weekday: 2, startMinute: 600, endMinute: 1080}, // Tue 10am–6pm
		{weekday: 4, startMinute: 600, endMinute: 1080}, // Thu 10am–6pm
	}

	t.Run("same day overlapping times", func(t *testing.T) {
		client := []availabilityWindow{{weekday: 2, startMinute: 540, endMinute: 720}} // Tue 9am–12pm
		if !schedulesOverlap(artist, client) {
			t.Error("expected overlap on Tuesday")
		}
	})

	t.Run("same day but disjoint times", func(t *testing.T) {
		client := []availabilityWindow{{weekday: 2, startMinute: 1080, endMinute: 1200}} // Tue 6pm–8pm
		if schedulesOverlap(artist, client) {
			t.Error("touching at the boundary is not an overlap")
		}
	})

	t.Run("different day", func(t *testing.T) {
		client := []availabilityWindow{{weekday: 0, startMinute: 600, endMinute: 1080}} // Sunday
		if schedulesOverlap(artist, client) {
			t.Error("expected no overlap on a day the artist does not work")
		}
	})
}

func TestScheduleFitNote(t *testing.T) {
	artist := []availabilityWindow{{weekday: 2, startMinute: 600, endMinute: 1080}}

	t.Run("no schedule set means no note", func(t *testing.T) {
		if note := scheduleFitNote(nil, artist); note != "" {
			t.Errorf("expected empty note, got %q", note)
		}
	})

	t.Run("mismatch asks to find days that work", func(t *testing.T) {
		client := []availabilityWindow{{weekday: 0, startMinute: 600, endMinute: 1080}}
		if note := scheduleFitNote(artist, client); !strings.Contains(note, "do not line up") {
			t.Errorf("expected a mismatch note, got %q", note)
		}
	})
}

func TestInquiryPromptSurfacesEmptyBrief(t *testing.T) {
	br := sqlc.BookingRequest{
		Type:        "custom",
		ClientName:  "Doseph",
		Description: ".",
		Placement:   "forearm",
	}
	got := inquiryPrompt(br, nil)

	if !strings.Contains(got, "Request type: custom") {
		t.Errorf("missing request type: %s", got)
	}
	
	if !strings.Contains(got, "Description: .") {
		t.Errorf("placeholder description not surfaced: %s", got)
	}
	if !strings.Contains(got, "Reference images attached: 0") {
		t.Errorf("zero references not stated: %s", got)
	}
}

func TestInquiryPromptMarksFlashAndMissingDescription(t *testing.T) {
	br := sqlc.BookingRequest{Type: "flash", ClientName: "Mia"}
	got := inquiryPrompt(br, nil)

	if !strings.Contains(got, "flash booking") {
		t.Errorf("flash booking not noted: %s", got)
	}
	if !strings.Contains(got, "Description: (none provided)") {
		t.Errorf("empty description not marked: %s", got)
	}
}

func TestParseClientSchedule(t *testing.T) {
	raw := []byte(`[{"weekday":2,"startMinute":540,"endMinute":720},{"weekday":5,"startMinute":480,"endMinute":1020}]`)
	got := parseClientSchedule(raw)
	if len(got) != 2 || got[0].weekday != 2 || got[1].endMinute != 1020 {
		t.Errorf("bad parse: %+v", got)
	}
	if parseClientSchedule([]byte("not json")) != nil {
		t.Error("expected nil on unparseable input")
	}
}
