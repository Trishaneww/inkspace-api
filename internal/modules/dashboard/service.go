package dashboard

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Service interface {
	GetDashboard(ctx context.Context, userID uuid.UUID, rangeKey string) (Dashboard, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func rangeMonths(rangeKey string) (months int, comparable bool) {
	switch rangeKey {
	case "1m":
		return 1, true
	case "1y":
		return 12, true
	case "all":
		return 0, false
	default: // "6m"
		return 6, true
	}
}

func (s *service) GetDashboard(ctx context.Context, userID uuid.UUID, rangeKey string) (Dashboard, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return Dashboard{}, err
	}

	counts, err := s.repo.GetDashboardCounts(ctx, artist.ID)
	if err != nil {
		return Dashboard{}, err
	}

	currency := "CAD"
	if settings, err := s.repo.GetArtistSettings(ctx, artist.ID); err == nil {
		if settings.Currency != "" {
			currency = settings.Currency
		}
	}

	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	epoch := time.Unix(0, 0).UTC()
	months, comparable := rangeMonths(rangeKey)

	fetchSince := epoch
	if comparable {
		fetchSince = currentMonth.AddDate(0, -(2*months - 1), 0)
	}

	earningRows, err := s.repo.GetMonthlyEarnings(ctx, sqlc.GetMonthlyEarningsParams{
		ArtistID: artist.ID,
		Since:    pgtype.Timestamptz{Time: fetchSince, Valid: true},
	})
	if err != nil {
		return Dashboard{}, err
	}
	byMonth := indexEarnings(earningRows)

	chartStart := currentMonth.AddDate(0, -(months - 1), 0)
	prevStart := fetchSince
	mixCurrentStart := chartStart
	mixPrevStart := prevStart
	if !comparable {
		chartStart = earliestMonth(earningRows, currentMonth)
		months = monthsBetween(chartStart, currentMonth) + 1
		mixCurrentStart = epoch
		mixPrevStart = epoch
	}

	series := buildSeries(chartStart, months, byMonth)
	thisNet, thisCollected := sumSeries(chartStart, months, byMonth)
	var prevNet, prevCollected int64
	if comparable {
		prevNet, prevCollected = sumSeries(prevStart, months, byMonth)
	}

	mix, err := s.repo.GetBookingMixForRange(ctx, sqlc.GetBookingMixForRangeParams{
		ArtistID:     artist.ID,
		CurrentStart: pgtype.Timestamptz{Time: mixCurrentStart, Valid: true},
		PrevStart:    pgtype.Timestamptz{Time: mixPrevStart, Valid: true},
	})
	if err != nil {
		return Dashboard{}, err
	}

	upcomingRows, err := s.repo.ListUpcomingAppointments(ctx, artist.ID)
	if err != nil {
		return Dashboard{}, err
	}
	upcoming := make([]UpcomingAppt, 0, len(upcomingRows))
	for _, row := range upcomingRows {
		upcoming = append(upcoming, upcomingFromRow(row))
	}

	return Dashboard{
		Currency: currency,
		Pipeline: Pipeline{
			NewInquiries:    counts.NewInquiries,
			AwaitingDeposit: counts.AwaitingDeposit,
			Scheduled:       counts.Scheduled,
			LeadsThisMonth:  counts.LeadsThisMonth,
			LeadsLastMonth:  counts.LeadsLastMonth,
		},
		Mix: BookingMix{
			Custom:     mix.CurrentCustom,
			Flash:      mix.CurrentFlash,
			PrevPeriod: mix.PrevBookings,
		},
		Earnings: EarningsTrend{
			ThisPeriodNetCents:       thisNet,
			PrevPeriodNetCents:       prevNet,
			ThisPeriodCollectedCents: thisCollected,
			PrevPeriodCollectedCents: prevCollected,
			Months:                   series,
		},
		Upcoming: upcoming,
	}, nil
}
