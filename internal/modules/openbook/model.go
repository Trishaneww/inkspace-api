package openbook

import "encoding/json"

type Profile struct {
	Username          string               `json:"username"`
	ArtistID          string               `json:"artistId"`
	DisplayName       string               `json:"displayName"`
	AvatarURL         string               `json:"avatarUrl"`
	Location          string               `json:"location"`
	InstagramURL      string               `json:"instagramUrl"`
	AcceptingBookings bool                 `json:"acceptingBookings"`
	SchedulingMode    string               `json:"schedulingMode"`
	Styles            []string             `json:"styles"`
	CustomQuestions   []string             `json:"customQuestions"`
	Aftercare         string               `json:"aftercare"`
	FAQs              []FAQItem            `json:"faqs"`
	Availability      []AvailabilityWindow `json:"availability"`
	Locations         []PublicLocation     `json:"locations"`
	HasFlashes         bool                 `json:"hasFlashes"`
	HasPortfolio       bool                 `json:"hasPortfolio"`
	Theme              string               `json:"theme"`
	CustomTheme        *CustomTheme         `json:"customTheme,omitempty"`
	BackgroundImageURL string               `json:"backgroundImageUrl,omitempty"`
}

type CustomTheme struct {
	Background string `json:"background"`
	Card       string `json:"card"`
	Button     string `json:"button"`
	Text       string `json:"text"`
}

func parseCustomTheme(raw []byte) *CustomTheme {
	if len(raw) == 0 {
		return nil
	}
	var ct CustomTheme
	if err := json.Unmarshal(raw, &ct); err != nil {
		return nil
	}
	return &ct
}

type PublicLocation struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	IsPrimary bool    `json:"isPrimary"`
	IsCurrent bool    `json:"isCurrent"`
	StartDate *string `json:"startDate"`
	EndDate   *string `json:"endDate"`
}

type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type AvailabilityWindow struct {
	Weekday     int32 `json:"weekday"`
	StartMinute int32 `json:"startMinute"`
	EndMinute   int32 `json:"endMinute"`
}

type PresignReferenceInput struct {
	ContentType string `json:"contentType" binding:"required"`
}

type PresignReferenceResult struct {
	URL       string `json:"url"`
	Key       string `json:"key"`
	ExpiresAt string `json:"expiresAt"`
}

type CustomAnswer struct {
	Prompt string `json:"prompt"`
	Answer string `json:"answer"`
}

type AvailabilityChoice struct {
	Weekday     int32 `json:"weekday"`
	StartMinute int32 `json:"startMinute"`
	EndMinute   int32 `json:"endMinute"`
}

type CreateRequestInput struct {
	LocationID         string               `json:"locationId"`
	FlashID            *string              `json:"flashId"`
	SizeCode           string               `json:"sizeCode"`
	Description        string               `json:"description"`
	ReferenceImageKeys []string             `json:"referenceImageKeys"`
	Placement          string               `json:"placement"`
	ApproxSizeInches   *int32               `json:"approxSizeInches"`
	ColorType          string               `json:"colorType"`
	Styles             []string             `json:"styles"`
	CustomAnswers      []CustomAnswer       `json:"customAnswers"`
	ClientAvailability []AvailabilityChoice `json:"clientAvailability"`
	ClientName         string               `json:"clientName"`
	ClientEmail        string               `json:"clientEmail"`
	ClientPhone        string               `json:"clientPhone"`
}

type CreateRequestResult struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}
