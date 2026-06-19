package payments

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/trishaneupnexx/inkspace-api/internal/clientaccount"
	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

var (
	ErrNotFound           = errors.New("payment request not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrAmountTooLow       = errors.New("amount is below the minimum")
	ErrNoDepositDefault   = errors.New("no default deposit configured")
	ErrStripeNotConnected = errors.New("stripe account not connected")
	ErrAlreadyRequested   = errors.New("a live payment request already exists")
	ErrNotPayable         = errors.New("payment request is not payable")
	ErrStripeAPI          = errors.New("stripe api error")
	ErrAccountExists      = errors.New("an account with this email already exists")
	ErrWeakPassword       = errors.New("password is too short")
	ErrPhoneRequired      = errors.New("a phone number is required")
	ErrPhoneTaken         = errors.New("that phone number is already in use")
	ErrNotRefundable      = errors.New("payment request is not refundable")
	ErrResendTooSoon      = errors.New("payment link was emailed too recently")
)

type Service interface {
	CreatePaymentRequest(ctx context.Context, userID, bookingID uuid.UUID, input CreatePaymentRequestInput) (PaymentRequest, error)
	CancelPaymentRequest(ctx context.Context, userID, paymentRequestID uuid.UUID) (PaymentRequest, error)
	ResendPaymentRequest(ctx context.Context, userID, paymentRequestID uuid.UUID) (PaymentRequest, error)
	RefundPaymentRequest(ctx context.Context, userID, paymentRequestID uuid.UUID) (PaymentRequest, error)
	GetEarnings(ctx context.Context, userID uuid.UUID) (Earnings, error)

	GetPublicPaymentRequest(ctx context.Context, token string) (PublicPaymentRequest, error)
	CreateCheckout(ctx context.Context, token string) (CheckoutResponse, error)
	CreateClientAccount(ctx context.Context, token string, input CreateClientAccountInput) (ClientSession, error)

	ProcessWebhook(ctx context.Context, payload []byte, signature string) error
}

type service struct {
	cfg  *config.Config
	repo Repository
	log  *slog.Logger
}

func NewService(cfg *config.Config, repo Repository) Service {
	return &service{cfg: cfg, repo: repo, log: slog.Default()}
}

func (s *service) CreatePaymentRequest(ctx context.Context, userID, bookingID uuid.UUID, input CreatePaymentRequestInput) (PaymentRequest, error) {
	if input.Type != typeDeposit && input.Type != typeFinal {
		return PaymentRequest{}, ErrInvalidInput
	}

	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return PaymentRequest{}, err
	}
	settings, err := s.repo.GetArtistSettings(ctx, artist.ID)
	if err != nil {
		return PaymentRequest{}, err
	}

	// The artist must have a connected, charge-ready Stripe account.
	if settings.StripeAccountID == nil || *settings.StripeAccountID == "" || !settings.StripeChargesEnabled {
		return PaymentRequest{}, ErrStripeNotConnected
	}

	booking, err := s.repo.GetBookingRequest(ctx, sqlc.GetBookingRequestParams{ID: bookingID, ArtistID: artist.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentRequest{}, ErrNotFound
		}
		return PaymentRequest{}, err
	}

	var amount int64
	switch {
	case input.AmountCents != nil:
		amount = *input.AmountCents
	case input.Type == typeDeposit && settings.DepositFlatFeeCents != nil:
		amount = *settings.DepositFlatFeeCents
	case input.Type == typeDeposit:
		return PaymentRequest{}, ErrNoDepositDefault
	default:
		return PaymentRequest{}, ErrInvalidInput
	}
	if amount < MinChargeCents {
		return PaymentRequest{}, ErrAmountTooLow
	}

	// A second outstanding request of the same type on one booking is not allowed.
	live, err := s.repo.CountLivePaymentRequests(ctx, sqlc.CountLivePaymentRequestsParams{
		BookingRequestID: bookingID,
		Type:             input.Type,
	})
	if err != nil {
		return PaymentRequest{}, err
	}
	if live > 0 {
		return PaymentRequest{}, ErrAlreadyRequested
	}

	fee := computeFee(amount, settings.PlatformFeePayer)
	token, err := newPublicToken()
	if err != nil {
		return PaymentRequest{}, err
	}

	row, err := s.repo.CreatePaymentRequest(ctx, sqlc.CreatePaymentRequestParams{
		ArtistID:          artist.ID,
		BookingRequestID:  bookingID,
		Type:              input.Type,
		Currency:          settings.Currency,
		AmountCents:       fee.AmountCents,
		PlatformFeeCents:  fee.PlatformFeeCents,
		ClientChargeCents: fee.ClientChargeCents,
		FeePayer:          settings.PlatformFeePayer,
		ClientEmail:       booking.ClientEmail,
		ClientName:        booking.ClientName,
		Description:       booking.Description,
		PublicToken:       token,
		ExpiresAt:         pgtype.Timestamptz{Time: time.Now().Add(PayLinkTTL), Valid: true},
	})
	if err != nil {
		return PaymentRequest{}, err
	}

	if input.Type == typeDeposit {
		_ = s.repo.SetBookingDepositStatus(ctx, sqlc.SetBookingDepositStatusParams{
			DepositStatus:    "pending",
			BookingRequestID: bookingID,
		})
	}

	s.sendPaymentRequestEmail(row, s.artistDisplayName(ctx, artist.ID))
	return paymentRequestFromRow(row, s.payLinkURL(row.PublicToken)), nil
}

func (s *service) CancelPaymentRequest(ctx context.Context, userID, paymentRequestID uuid.UUID) (PaymentRequest, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return PaymentRequest{}, err
	}
	row, err := s.repo.CancelPaymentRequest(ctx, sqlc.CancelPaymentRequestParams{
		ID:       paymentRequestID,
		ArtistID: artist.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentRequest{}, ErrNotFound
		}
		return PaymentRequest{}, err
	}
	if row.Type == typeDeposit {
		_ = s.repo.SetBookingDepositStatus(ctx, sqlc.SetBookingDepositStatusParams{
			DepositStatus:    "not_required",
			BookingRequestID: row.BookingRequestID,
		})
	}
	return paymentRequestFromRow(row, s.payLinkURL(row.PublicToken)), nil
}

func (s *service) ResendPaymentRequest(ctx context.Context, userID, paymentRequestID uuid.UUID) (PaymentRequest, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return PaymentRequest{}, err
	}
	row, err := s.repo.GetPaymentRequestForArtist(ctx, sqlc.GetPaymentRequestForArtistParams{
		ID:       paymentRequestID,
		ArtistID: artist.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentRequest{}, ErrNotFound
		}
		return PaymentRequest{}, err
	}
	if row.Status != "requested" && row.Status != "processing" {
		return PaymentRequest{}, ErrNotPayable
	}

	if time.Since(row.LastEmailedAt.Time) < ResendCooldown {
		return PaymentRequest{}, ErrResendTooSoon
	}

	updated, err := s.repo.MarkPaymentRequestEmailed(ctx, sqlc.MarkPaymentRequestEmailedParams{
		ID:       paymentRequestID,
		ArtistID: artist.ID,
	})
	if err != nil {
		return PaymentRequest{}, err
	}

	s.sendPaymentRequestEmail(updated, s.artistDisplayName(ctx, artist.ID))
	return paymentRequestFromRow(updated, s.payLinkURL(updated.PublicToken)), nil
}

func (s *service) RefundPaymentRequest(ctx context.Context, userID, paymentRequestID uuid.UUID) (PaymentRequest, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return PaymentRequest{}, err
	}
	row, err := s.repo.GetPaymentRequestForArtist(ctx, sqlc.GetPaymentRequestForArtistParams{
		ID:       paymentRequestID,
		ArtistID: artist.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentRequest{}, ErrNotFound
		}
		return PaymentRequest{}, err
	}
	if row.Status != "paid" || row.StripePaymentIntentID == nil {
		return PaymentRequest{}, ErrNotRefundable
	}

	params := &stripe.RefundParams{
		PaymentIntent:        row.StripePaymentIntentID,
		ReverseTransfer:      stripe.Bool(true),
		RefundApplicationFee: stripe.Bool(true),
	}
	params.Context = ctx
	if _, err := refund.New(params); err != nil {
		s.log.Warn("stripe_refund_failed", "payment_request_id", row.ID, "error", err)
		return PaymentRequest{}, ErrStripeAPI
	}

	updated, err := s.repo.RefundPaymentRequestByPaymentIntent(ctx, sqlc.RefundPaymentRequestByPaymentIntentParams{
		StripePaymentIntentID: row.StripePaymentIntentID,
		AmountRefundedCents:   row.ClientChargeCents,
	})
	if err != nil {
		return PaymentRequest{}, err
	}
	if updated.Type == typeDeposit {
		_ = s.repo.SetBookingDepositStatus(ctx, sqlc.SetBookingDepositStatusParams{
			DepositStatus:    "refunded",
			BookingRequestID: updated.BookingRequestID,
		})
	}
	return paymentRequestFromRow(updated, s.payLinkURL(updated.PublicToken)), nil
}

func (s *service) GetPublicPaymentRequest(ctx context.Context, token string) (PublicPaymentRequest, error) {
	row, err := s.repo.GetPaymentRequestByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PublicPaymentRequest{}, ErrNotFound
		}
		return PublicPaymentRequest{}, err
	}
	email := normalizeEmail(row.ClientEmail)
	_, userErr := s.repo.GetUserByEmail(ctx, email)

	return PublicPaymentRequest{
		Status:            row.Status,
		Type:              row.Type,
		Currency:          row.Currency,
		ClientChargeCents: row.ClientChargeCents,
		ArtistName:        s.artistDisplayName(ctx, row.ArtistID),
		ClientEmail:       email,
		ClientName:        row.ClientName,
		Description:       row.Description,
		Expired:           s.isExpired(row),
		HasAccount:        userErr == nil,
	}, nil
}

func (s *service) CreateClientAccount(ctx context.Context, token string, input CreateClientAccountInput) (ClientSession, error) {
	row, err := s.repo.GetPaymentRequestByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ClientSession{}, ErrNotFound
		}
		return ClientSession{}, err
	}

	session, err := clientaccount.Create(ctx, s.repo, s.cfg, row.ClientEmail, clientaccount.Input{
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Phone:          input.Phone,
		Password:       input.Password,
		MarketingOptIn: input.MarketingOptIn,
	}, "pay_page")
	switch {
	case errors.Is(err, clientaccount.ErrWeakPassword):
		return ClientSession{}, ErrWeakPassword
	case errors.Is(err, clientaccount.ErrAccountExists):
		return ClientSession{}, ErrAccountExists
	case errors.Is(err, clientaccount.ErrPhoneRequired):
		return ClientSession{}, ErrPhoneRequired
	case errors.Is(err, clientaccount.ErrPhoneTaken):
		return ClientSession{}, ErrPhoneTaken
	case err != nil:
		return ClientSession{}, err
	}
	return session, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *service) CreateCheckout(ctx context.Context, token string) (CheckoutResponse, error) {
	row, err := s.repo.GetPaymentRequestByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CheckoutResponse{}, ErrNotFound
		}
		return CheckoutResponse{}, err
	}
	if row.Status != "requested" && row.Status != "processing" {
		return CheckoutResponse{}, ErrNotPayable
	}
	if s.isExpired(row) {
		_, _ = s.repo.MarkPaymentRequestExpiredBySession(ctx, row.StripeCheckoutSessionID)
		return CheckoutResponse{}, ErrNotPayable
	}

	settings, err := s.repo.GetArtistSettings(ctx, row.ArtistID)
	if err != nil {
		return CheckoutResponse{}, err
	}
	if settings.StripeAccountID == nil || *settings.StripeAccountID == "" || !settings.StripeChargesEnabled {
		return CheckoutResponse{}, ErrStripeNotConnected
	}

	sess, err := s.createCheckoutSession(ctx, row, *settings.StripeAccountID)
	if err != nil {
		return CheckoutResponse{}, err
	}

	if _, err := s.repo.AttachCheckoutSession(ctx, sqlc.AttachCheckoutSessionParams{
		ID:                      row.ID,
		StripeCheckoutSessionID: &sess.ID,
		StripePaymentIntentID:   paymentIntentID(sess),
	}); err != nil {
		return CheckoutResponse{}, err
	}
	return CheckoutResponse{URL: sess.URL}, nil
}

func (s *service) ProcessWebhook(ctx context.Context, payload []byte, signature string) error {
	if s.cfg.StripePaymentsWebhookSecret == "" {
		s.log.Warn("payments_webhook_secret_not_configured")
		return ErrStripeAPI
	}

	event, err := webhook.ConstructEventWithOptions(
		payload, signature, s.cfg.StripePaymentsWebhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		s.log.Warn("payments_webhook_signature_invalid", "error", err)
		return ErrStripeAPI
	}

	inserted, err := s.repo.InsertStripeEvent(ctx, sqlc.InsertStripeEventParams{ID: event.ID, Type: string(event.Type)})
	if err != nil {
		return err
	}
	if inserted == 0 {
		return nil // already processed
	}

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event)
	case "payment_intent.succeeded":
		return s.handlePaymentIntentSucceeded(ctx, event)
	case "payment_intent.payment_failed":
		return s.handlePaymentIntentFailed(ctx, event)
	case "checkout.session.expired":
		return s.handleCheckoutExpired(ctx, event)
	case "charge.refunded":
		return s.handleChargeRefunded(ctx, event)
	default:
		s.log.Debug("payments_webhook_unhandled", "type", event.Type)
		return nil
	}
}

func (s *service) handleCheckoutCompleted(ctx context.Context, event stripe.Event) error {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return err
	}
	if sess.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return nil // unpaid (e.g. async method pending)
	}
	sessionID := sess.ID
	row, err := s.repo.MarkPaymentRequestPaidBySession(ctx, sqlc.MarkPaymentRequestPaidBySessionParams{
		StripeCheckoutSessionID: &sessionID,
		StripePaymentIntentID:   paymentIntentID(&sess),
	})
	if err != nil {
		return filterNotFound(err)
	}
	s.onPaymentPaid(ctx, row)
	return nil
}

func (s *service) handlePaymentIntentSucceeded(ctx context.Context, event stripe.Event) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return err
	}
	id := pi.ID
	row, err := s.repo.MarkPaymentRequestPaidByPaymentIntent(ctx, &id)
	if err != nil {
		return filterNotFound(err)
	}
	s.onPaymentPaid(ctx, row)
	return nil
}

func (s *service) handlePaymentIntentFailed(ctx context.Context, event stripe.Event) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return err
	}
	id := pi.ID
	_, err := s.repo.MarkPaymentRequestFailedByPaymentIntent(ctx, &id)
	return filterNotFound(err)
}

func (s *service) handleCheckoutExpired(ctx context.Context, event stripe.Event) error {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return err
	}
	id := sess.ID
	_, err := s.repo.MarkPaymentRequestExpiredBySession(ctx, &id)
	return filterNotFound(err)
}

func (s *service) handleChargeRefunded(ctx context.Context, event stripe.Event) error {
	var charge stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		return err
	}
	if charge.PaymentIntent == nil {
		return nil
	}
	id := charge.PaymentIntent.ID
	row, err := s.repo.RefundPaymentRequestByPaymentIntent(ctx, sqlc.RefundPaymentRequestByPaymentIntentParams{
		StripePaymentIntentID: &id,
		AmountRefundedCents:   charge.AmountRefunded,
	})
	if err != nil {
		return filterNotFound(err)
	}
	if row.Type == typeDeposit {
		_ = s.repo.SetBookingDepositStatus(ctx, sqlc.SetBookingDepositStatusParams{
			DepositStatus:    "refunded",
			BookingRequestID: row.BookingRequestID,
		})
	}
	return nil
}

func (s *service) onPaymentPaid(ctx context.Context, row sqlc.PaymentRequest) {
	if row.Type == typeDeposit {
		_ = s.repo.SetBookingDepositStatus(ctx, sqlc.SetBookingDepositStatusParams{
			DepositStatus:    "paid",
			BookingRequestID: row.BookingRequestID,
		})
	}
	s.sendPaymentReceiptEmail(row, s.artistDisplayName(ctx, row.ArtistID))
}

func (s *service) isExpired(row sqlc.PaymentRequest) bool {
	return row.ExpiresAt.Valid && time.Now().After(row.ExpiresAt.Time)
}

func (s *service) payLinkURL(token string) string {
	return s.cfg.FrontendURL + "/pay/" + token
}

func (s *service) GetEarnings(ctx context.Context, userID uuid.UUID) (Earnings, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return Earnings{}, err
	}

	totals, err := s.repo.GetArtistEarnings(ctx, artist.ID)
	if err != nil {
		return Earnings{}, err
	}
	rows, err := s.repo.ListRecentPaidPaymentsForArtist(ctx, artist.ID)
	if err != nil {
		return Earnings{}, err
	}

	currency := "CAD"
	if settings, err := s.repo.GetArtistSettings(ctx, artist.ID); err == nil && settings.Currency != "" {
		currency = settings.Currency
	}

	recent := make([]RecentPayment, 0, len(rows))
	for _, row := range rows {
		recent = append(recent, recentPaymentFromRow(row))
	}

	return Earnings{
		IssuerName:     s.artistDisplayName(ctx, artist.ID),
		Currency:       currency,
		AllTime:        earningsTotals(totals.CollectedCents, totals.FeeCents, totals.PaidCount),
		ThisMonth:      earningsTotals(totals.MonthCollectedCents, totals.MonthFeeCents, totals.MonthPaidCount),
		RecentPayments: recent,
	}, nil
}

func (s *service) artistDisplayName(ctx context.Context, artistID uuid.UUID) string {
	artist, err := s.repo.GetArtistByID(ctx, artistID)
	if err != nil {
		return ""
	}
	user, err := s.repo.GetUserByID(ctx, artist.UserID)
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(strings.Join([]string{stringOrEmpty(user.FirstName), stringOrEmpty(user.LastName)}, " "))
	if name != "" {
		return name
	}
	return stringOrEmpty(user.Username)
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func filterNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

func newPublicToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
