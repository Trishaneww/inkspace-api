package bookings

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

const presignViewTTL = 1 * time.Hour

var (
	ErrNotFound   = errors.New("inquiry not found")
	ErrNoOpenBook = errors.New("artist has no open book")
)

type Service interface {
	ListInquiries(ctx context.Context, userID uuid.UUID) (InquiryListResponse, error)
	GetInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error)
	AcceptInquiry(ctx context.Context, userID, inquiryID uuid.UUID, input AcceptInput) (Inquiry, error)
	DeclineInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error)
	ReopenInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error)
	SeedDevInquiries(ctx context.Context, userID uuid.UUID) (int, error)
}

type service struct {
	repo Repository
	s3   *s3client.Client
}

func NewService(repo Repository, s3 *s3client.Client) Service {
	return &service{repo: repo, s3: s3}
}

func (s *service) ListInquiries(ctx context.Context, userID uuid.UUID) (InquiryListResponse, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return InquiryListResponse{}, err
	}

	rows, err := s.repo.ListBookingRequestsByArtist(ctx, artist.ID)
	if err != nil {
		return InquiryListResponse{}, err
	}
	statsRow, err := s.repo.GetBookingStats(ctx, artist.ID)
	if err != nil {
		return InquiryListResponse{}, err
	}

	inquiries := make([]Inquiry, 0, len(rows))
	for _, row := range rows {
		inquiries = append(inquiries, inquiryFromRow(row))
	}
	return InquiryListResponse{Inquiries: inquiries, Stats: statsFromRow(statsRow)}, nil
}

func (s *service) GetInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	row, err := s.repo.GetBookingRequest(ctx, sqlc.GetBookingRequestParams{ID: inquiryID, ArtistID: artist.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}
	return s.enrichInquiry(ctx, row)
}

func (s *service) AcceptInquiry(ctx context.Context, userID, inquiryID uuid.UUID, input AcceptInput) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}

	depositStatus := "not_required"
	if err := s.repo.EnsureArtistSettings(ctx, artist.ID); err == nil {
		if settings, err := s.repo.GetArtistSettings(ctx, artist.ID); err == nil {
			if settings.DepositFlatFeeCents != nil && *settings.DepositFlatFeeCents > 0 {
				depositStatus = "pending"
			}
		}
	}

	return s.updateStatus(ctx, artist.ID, inquiryID, sqlc.UpdateBookingRequestStatusParams{
		Status:                 "accepted",
		SessionDurationMinutes: input.SessionDurationMinutes,
		DepositStatus:          &depositStatus,
	})
}

func (s *service) DeclineInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	return s.updateStatus(ctx, artist.ID, inquiryID, sqlc.UpdateBookingRequestStatusParams{Status: "declined"})
}

func (s *service) ReopenInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	row, err := s.repo.ReopenBookingRequest(ctx, sqlc.ReopenBookingRequestParams{ID: inquiryID, ArtistID: artist.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}
	return s.enrichInquiry(ctx, row)
}

func (s *service) updateStatus(ctx context.Context, artistID, inquiryID uuid.UUID, params sqlc.UpdateBookingRequestStatusParams) (Inquiry, error) {
	params.ID = inquiryID
	params.ArtistID = artistID
	row, err := s.repo.UpdateBookingRequestStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}
	return s.enrichInquiry(ctx, row)
}

func (s *service) enrichInquiry(ctx context.Context, row sqlc.BookingRequest) (Inquiry, error) {
	inquiry := inquiryFromRow(row)
	if s.s3 == nil {
		return inquiry, nil
	}
	for _, key := range inquiry.ReferenceImageKeys {
		url, err := s.s3.PresignGet(ctx, key, presignViewTTL)
		if err != nil {
			return Inquiry{}, fmt.Errorf("presign reference image: %w", err)
		}
		inquiry.ReferenceImageURLs = append(inquiry.ReferenceImageURLs, url)
	}
	return inquiry, nil
}

func (s *service) requireArtist(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err == nil {
		return artist, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Artist{}, err
	}
	if err := s.repo.EnsureArtist(ctx, userID); err != nil {
		return sqlc.Artist{}, err
	}
	return s.repo.GetArtistByUserID(ctx, userID)
}

// ── Dev seed ─────────────────────────────────────────────────────────────────
func (s *service) SeedDevInquiries(ctx context.Context, userID uuid.UUID) (int, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return 0, err
	}
	book, err := s.repo.GetOpenBookByArtist(ctx, artist.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNoOpenBook
		}
		return 0, err
	}

	availability := []byte(`[{"weekday":1,"ranges":[["600","1080"]]},{"weekday":4,"ranges":[["600","1080"]]}]`)

	samples := []struct {
		name, email, placement, description, pieceType string
		size                                           int32
	}{
		{"Maya Chen", "maya@example.com", "forearm", "Fine-line botanical half sleeve, black and grey.", "custom", 8},
		{"Devon Park", "devon@example.com", "ribs", "Loves the koi flash piece — slightly enlarged.", "flash", 6},
		{"Sam Rivera", "sam@example.com", "calf", "Traditional eagle, full colour.", "custom", 10},
		{"Jordan Lee", "jordan@example.com", "shoulder", "Small geometric mountain range.", "custom", 4},
		{"Avery Brooks", "avery@example.com", "wrist", "Matching script with partner.", "custom", 3},
		{"Riley Quinn", "riley@example.com", "thigh", "Ornamental mandala, dotwork.", "custom", 7},
	}

	customAnswers := []byte(`[{"prompt":"Is this your first tattoo?","answer":"No — this is my third."}]`)

	for i, sample := range samples {
		size := sample.size
		phone := "+1 (555) 010-00" + strconv.Itoa(10+i)

		styles := []string{}
		if sample.pieceType == string(RequestTypeCustom) {
			styles = []string{"fine_line", "blackwork"}
		}

		created, err := s.repo.CreateBookingRequest(ctx, sqlc.CreateBookingRequestParams{
			ArtistID:           artist.ID,
			OpenBookID:         book.ID,
			Type:               sample.pieceType,
			Description:        sample.description,
			ReferenceImageKeys: []string{},
			Placement:          sample.placement,
			ApproxSizeInches:   &size,
			Styles:             styles,
			ClientAvailability: availability,
			CustomAnswers:      customAnswers,
			ClientName:         sample.name,
			ClientEmail:        sample.email,
			ClientPhone:        &phone,
			Status:             "pending",
			DepositStatus:      "not_required",
			WaiverStatus:       "not_required",
		})
		if err != nil {
			return i, fmt.Errorf("seed inquiry %d: %w", i, err)
		}

		switch i {
		case 1, 2:
			depositPending := "pending"
			if _, err := s.repo.UpdateBookingRequestStatus(ctx, sqlc.UpdateBookingRequestStatusParams{
				ID:            created.ID,
				ArtistID:      artist.ID,
				Status:        "accepted",
				DepositStatus: &depositPending,
			}); err != nil {
				return i, err
			}
		case 5:
			if _, err := s.repo.UpdateBookingRequestStatus(ctx, sqlc.UpdateBookingRequestStatusParams{
				ID:       created.ID,
				ArtistID: artist.ID,
				Status:   "declined",
			}); err != nil {
				return i, err
			}
		}
	}

	return len(samples), nil
}
