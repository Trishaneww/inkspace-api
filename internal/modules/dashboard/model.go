package dashboard

import (
	"time"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Dashboard struct {
	Currency string         `json:"currency"`
	Pipeline Pipeline       `json:"pipeline"`
	Mix      BookingMix     `json:"mix"`
	Earnings EarningsTrend  `json:"earnings"`
	Upcoming []UpcomingAppt `json:"upcoming"`
}

type Pipeline struct {
	NewInquiries    int64 `json:"newInquiries"`
	AwaitingDeposit int64 `json:"awaitingDeposit"`
	Scheduled       int64 `json:"scheduled"`
	LeadsThisMonth  int64 `json:"leadsThisMonth"`
	LeadsLastMonth  int64 `json:"leadsLastMonth"`
}

type BookingMix struct {
	Custom     int64 `json:"custom"`
	Flash      int64 `json:"flash"`
	PrevPeriod int64 `json:"prevPeriod"`
}

type EarningsTrend struct {
	ThisPeriodNetCents       int64        `json:"thisPeriodNetCents"`
	PrevPeriodNetCents       int64        `json:"prevPeriodNetCents"`
	ThisPeriodCollectedCents int64        `json:"thisPeriodCollectedCents"`
	PrevPeriodCollectedCents int64        `json:"prevPeriodCollectedCents"`
	Months                   []MonthPoint `json:"months"`
}

type MonthPoint struct {
	Label          string `json:"label"`
	NetCents       int64  `json:"netCents"`
	CollectedCents int64  `json:"collectedCents"`
}

type UpcomingAppt struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ScheduledStart  string `json:"scheduledStart"`
	DurationMinutes int32  `json:"durationMinutes"`
	ClientName      string `json:"clientName"`
	ClientEmail     string `json:"clientEmail"`
	RequestType     string `json:"requestType"`
}

func indexEarnings(rows []sqlc.GetMonthlyEarningsRow) map[string]sqlc.GetMonthlyEarningsRow {
	byMonth := make(map[string]sqlc.GetMonthlyEarningsRow, len(rows))
	for _, row := range rows {
		byMonth[row.Month.Time.UTC().Format("2006-01")] = row
	}
	return byMonth
}

func buildSeries(start time.Time, months int, byMonth map[string]sqlc.GetMonthlyEarningsRow) []MonthPoint {
	points := make([]MonthPoint, 0, months)
	for i := 0; i < months; i++ {
		month := start.AddDate(0, i, 0)
		point := MonthPoint{Label: month.Format("Jan")}
		if row, ok := byMonth[month.Format("2006-01")]; ok {
			point.NetCents = row.NetCents
			point.CollectedCents = row.CollectedCents
		}
		points = append(points, point)
	}
	return points
}

func sumSeries(start time.Time, months int, byMonth map[string]sqlc.GetMonthlyEarningsRow) (netCents, collectedCents int64) {
	for i := 0; i < months; i++ {
		if row, ok := byMonth[start.AddDate(0, i, 0).Format("2006-01")]; ok {
			netCents += row.NetCents
			collectedCents += row.CollectedCents
		}
	}
	return netCents, collectedCents
}

func earliestMonth(rows []sqlc.GetMonthlyEarningsRow, fallback time.Time) time.Time {
	earliest := fallback
	for i, row := range rows {
		month := row.Month.Time.UTC()
		month = time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		if i == 0 || month.Before(earliest) {
			earliest = month
		}
	}
	return earliest
}

func monthsBetween(start, end time.Time) int {
	months := (end.Year()-start.Year())*12 + int(end.Month()) - int(start.Month())
	if months < 0 {
		return 0
	}
	return months
}

func upcomingFromRow(row sqlc.ListUpcomingAppointmentsRow) UpcomingAppt {
	out := UpcomingAppt{
		ID:              row.ID.String(),
		Type:            row.Type,
		DurationMinutes: row.DurationMinutes,
		ClientName:      row.ClientName,
		ClientEmail:     row.ClientEmail,
		RequestType:     row.RequestType,
	}
	if row.ScheduledStart.Valid {
		out.ScheduledStart = row.ScheduledStart.Time.UTC().Format(time.RFC3339)
	}
	return out
}
