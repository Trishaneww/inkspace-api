package aitriage

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
	"github.com/trishaneupnexx/inkspace-api/internal/subscription"
)

const (
	TriageRequestedRoutingKey = "ai.triage.requested"
	queueName                 = "ai.triage"

	triageModel = anthropic.ModelClaudeHaiku4_5_20251001
	maxTokens   = 1500

	maxReferenceImages = 3
	referenceImageTTL  = 10 * time.Minute
)

type TriageRequestedEvent struct {
	BookingRequestID string `json:"bookingRequestId"`
}

type Publisher struct {
	pub *events.Publisher
}

func NewPublisher(pub *events.Publisher) *Publisher {
	return &Publisher{pub: pub}
}

func (p *Publisher) RequestTriage(ctx context.Context, bookingRequestID uuid.UUID) error {
	return p.pub.Publish(ctx, TriageRequestedRoutingKey, TriageRequestedEvent{
		BookingRequestID: bookingRequestID.String(),
	})
}

type worker struct {
	q      *sqlc.Queries
	s3     *s3client.Client
	client anthropic.Client
	log    *slog.Logger
}

func Run(ctx context.Context, rabbitURL string, db *pgxpool.Pool, s3 *s3client.Client, apiKey string, log *slog.Logger) {
	if apiKey == "" {
		log.Info("ai_triage_disabled", "reason", "no_api_key")
		return
	}

	consumer, err := events.NewConsumer(rabbitURL, queueName, []string{TriageRequestedRoutingKey})
	if err != nil {
		log.Warn("ai_triage_consumer_init_failed", "error", err)
		return
	}
	defer consumer.Close()

	w := &worker{
		q:      sqlc.New(db),
		s3:     s3,
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		log:    log,
	}
	if err := consumer.Run(ctx, w.handle); err != nil && ctx.Err() == nil {
		log.Warn("ai_triage_consumer_stopped", "error", err)
	}
}

func (w *worker) handle(ctx context.Context, routingKey string, body []byte) error {
	if routingKey != TriageRequestedRoutingKey {
		return nil
	}

	var ev TriageRequestedEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		w.log.Warn("ai_triage_bad_payload", "error", err)
		return nil
	}
	bookingID, err := uuid.Parse(ev.BookingRequestID)
	if err != nil {
		return nil
	}

	br, err := w.q.GetBookingRequestByID(ctx, bookingID)
	if err != nil {
		w.log.Warn("ai_triage_lead_missing", "booking_request_id", bookingID, "error", err)
		return nil
	}

	artist, err := w.q.GetArtistByID(ctx, br.ArtistID)
	if err != nil {
		return nil
	}
	if !subscription.IsPremium(artist.SubscriptionStatus) {
		w.setStatus(ctx, bookingID, "skipped")
		return nil
	}

	settings, err := w.q.GetArtistSettings(ctx, br.ArtistID)
	if err != nil {
		w.setStatus(ctx, bookingID, "failed")
		return nil
	}

	schedule, err := w.q.ListAvailabilityWindows(ctx, br.ArtistID)
	if err != nil {
		schedule = nil
	}

	result, err := w.runTriage(ctx, br, settings, schedule)
	if err != nil {
		w.log.Warn("ai_triage_failed", "booking_request_id", bookingID, "error", err)
		w.setStatus(ctx, bookingID, "failed")
		return nil
	}

	w.persist(ctx, br, settings, result)
	return nil
}

func (w *worker) runTriage(ctx context.Context, br sqlc.BookingRequest, settings sqlc.ArtistSetting, schedule []sqlc.ArtistAvailabilityWindow) (triageResult, error) {
	resp, err := w.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     triageModel,
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt, CacheControl: anthropic.NewCacheControlEphemeralParam()},
			{Text: artistContext(settings, schedule)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(w.inquiryContent(ctx, br, schedule)...),
		},
	})
	if err != nil {
		return triageResult{}, err
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(t.Text)
		}
	}
	return parseTriage(sb.String())
}

func (w *worker) inquiryContent(ctx context.Context, br sqlc.BookingRequest, schedule []sqlc.ArtistAvailabilityWindow) []anthropic.ContentBlockParamUnion {
	blocks := []anthropic.ContentBlockParamUnion{
		anthropic.NewTextBlock(inquiryPrompt(br, schedule)),
	}
	if w.s3 == nil {
		return blocks
	}
	for i, key := range br.ReferenceImageKeys {
		if i >= maxReferenceImages {
			break
		}
		url, err := w.s3.PresignGet(ctx, key, referenceImageTTL)
		if err != nil {
			w.log.Warn("ai_triage_presign_failed", "booking_request_id", br.ID, "error", err)
			continue
		}
		blocks = append(blocks, anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: url}))
	}
	return blocks
}

func (w *worker) persist(ctx context.Context, br sqlc.BookingRequest, settings sqlc.ArtistSetting, result triageResult) {
	signalsJSON, err := json.Marshal(result.Signals)
	if err != nil {
		signalsJSON = []byte("{}")
	}
	redFlagsJSON, err := json.Marshal(result.RedFlags)
	if err != nil || len(result.RedFlags) == 0 {
		redFlagsJSON = []byte("[]")
	}

	label := result.Label
	if err := w.q.UpdateBookingRequestTriage(ctx, sqlc.UpdateBookingRequestTriageParams{
		ID:             br.ID,
		AiLabel:        &label,
		AiSummary:      result.Summary,
		AiSignals:      signalsJSON,
		AiRedFlags:     redFlagsJSON,
		AiReasoning:    result.Reasoning,
		AiValueCents:   estimateValueCents(settings, result),
		AiSessionCount: result.SessionCount,
		AiDraftReply:   result.DraftReply,
	}); err != nil {
		w.log.Warn("ai_triage_persist_failed", "booking_request_id", br.ID, "error", err)
	}
}

func estimateValueCents(settings sqlc.ArtistSetting, result triageResult) *int64 {
	if settings.MinSessionPriceCents != nil && result.SessionCount != nil && *result.SessionCount > 0 {
		v := *settings.MinSessionPriceCents * int64(*result.SessionCount)
		return &v
	}
	return result.ValueCents
}

func (w *worker) setStatus(ctx context.Context, bookingID uuid.UUID, status string) {
	if err := w.q.SetBookingRequestTriageStatus(ctx, sqlc.SetBookingRequestTriageStatusParams{
		ID:       bookingID,
		AiStatus: status,
	}); err != nil {
		w.log.Warn("ai_triage_status_failed", "booking_request_id", bookingID, "status", status, "error", err)
	}
}
