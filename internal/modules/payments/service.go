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

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/email"
	"github.com/trishaneupnexx/inkspace-api/internal/subscription"
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
	ListPayouts(ctx context.Context, userID uuid.UUID) ([]Payout, error)

	GetPublicPaymentRequest(ctx context.Context, token string) (PublicPaymentRequest, error)
	CreateClientCheckout(ctx context.Context, userID uuid.UUID, token string) (CheckoutResponse, error)

	// CreateDepositRequest is called by the bookings module when a session is
	// scheduled with a deposit. Satisfies bookings.DepositRequester.
	CreateDepositRequest(ctx context.Context, artistID, bookingID uuid.UUID, amountCents int64, scheduledStart *time.Time) (string, error)
	// SetDepositScheduledStart updates a live deposit's chosen time (client re-pick).
	SetDepositScheduledStart(ctx context.Context, bookingID uuid.UUID, scheduledStart time.Time) error
	// SetDepositConfirmer injects the bookings-side confirmer (wired in main).
	SetDepositConfirmer(confirmer DepositConfirmer)

	ProcessWebhook(ctx context.Context, payload []byte, signature string) error
}

type service struct {
	cfg              *config.Config
	repo             Repository
	log              *slog.Logger
	email            *email.Client
	depositConfirmer DepositConfirmer
}

func NewService(cfg *config.Config, repo Repository) Service {
	log := slog.Default()
	return &service{
		cfg:   cfg,
		repo:  repo,
		log:   log,
		email: email.NewClient(cfg.FrontendURL, cfg.InternalEmailSecret, log),
	}
}

func (s *service) SetDepositConfirmer(confirmer DepositConfirmer) {
	s.depositConfirmer = confirmer
}

// CreatePaymentRequest issues the standalone full ("final") payment. The artist
// enters the job total; any deposit already paid on the booking is subtracted so
// the client is only charged the remaining balance. Deposits are no longer
// created here — they ride along with scheduling (see CreateDepositRequest).
func (s *service) CreatePaymentRequest(ctx context.Context, userID, bookingID uuid.UUID, input CreatePaymentRequestInput) (PaymentRequest, error) {
	if input.Type != typeFinal {
		return PaymentRequest{}, ErrInvalidInput
	}
	if input.AmountCents == nil {
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

	jobTotal := *input.AmountCents
	depositPaid, err := s.repo.SumPaidDepositsForBooking(ctx, bookingID)
	if err != nil {
		return PaymentRequest{}, err
	}
	net := jobTotal - depositPaid
	if net < MinChargeCents {
		return PaymentRequest{}, ErrAmountTooLow
	}

	// A second outstanding final request on one booking is not allowed.
	live, err := s.repo.CountLivePaymentRequests(ctx, sqlc.CountLivePaymentRequestsParams{
		BookingRequestID: bookingID,
		Type:             typeFinal,
	})
	if err != nil {
		return PaymentRequest{}, err
	}
	if live > 0 {
		return PaymentRequest{}, ErrAlreadyRequested
	}

	fee := computeFee(net, settings.PlatformFeePayer, subscription.IsPremium(artist.SubscriptionStatus))
	token, err := newPublicToken()
	if err != nil {
		return PaymentRequest{}, err
	}

	row, err := s.repo.CreatePaymentRequest(ctx, sqlc.CreatePaymentRequestParams{
		ArtistID:          artist.ID,
		BookingRequestID:  bookingID,
		Type:              typeFinal,
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

	s.sendPaymentRequestEmail(row, s.artistDisplayName(ctx, artist.ID))
	out := paymentRequestFromRow(row, s.payLinkURL(row.PublicToken))
	out.JobTotalCents = jobTotal
	out.DepositAppliedCents = depositPaid
	return out, nil
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// CreateClientCheckout starts a Stripe checkout for a payment the signed-in
// client owns (matched by email). Account-first replacement for the old public
// guest checkout.
func (s *service) CreateClientCheckout(ctx context.Context, userID uuid.UUID, token string) (CheckoutResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return CheckoutResponse{}, err
	}
	row, err := s.repo.GetPaymentRequestByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CheckoutResponse{}, ErrNotFound
		}
		return CheckoutResponse{}, err
	}
	if !strings.EqualFold(strings.TrimSpace(row.ClientEmail), strings.TrimSpace(user.Email)) {
		return CheckoutResponse{}, ErrNotFound
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

		var chosenStart *time.Time
		if row.ScheduledStart.Valid {
			t := row.ScheduledStart.Time
			chosenStart = &t
		}
		if s.depositConfirmer != nil {
			confirmation, err := s.depositConfirmer.ConfirmDepositPaid(ctx, row.BookingRequestID, chosenStart)
			if err != nil {
				s.log.Warn("deposit_confirm_failed", "booking_request_id", row.BookingRequestID, "error", err)
			} else if confirmation.Confirmed {
				s.sendDepositPaidEmail(row, confirmation)
			}
		}
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

func (s *service) ListPayouts(ctx context.Context, userID uuid.UUID) ([]Payout, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	settings, err := s.repo.GetArtistSettings(ctx, artist.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []Payout{}, nil
		}
		return nil, err
	}
	if s.cfg.StripeSecretKey == "" ||
		settings.StripeAccountID == nil || *settings.StripeAccountID == "" {
		return []Payout{}, nil
	}

	raw, err := s.listStripePayouts(ctx, *settings.StripeAccountID)
	if err != nil {
		return nil, err
	}

	payouts := make([]Payout, 0, len(raw))
	for _, p := range raw {
		payouts = append(payouts, payoutFromStripe(p))
	}
	return payouts, nil
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
