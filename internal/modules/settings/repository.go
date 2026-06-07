package settings

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	// Account (users)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)
	GetUserByEmail(ctx context.Context, email string) (sqlc.User, error)
	GetUserByUsername(ctx context.Context, username string) (sqlc.User, error)
	UpdateUserProfile(ctx context.Context, params sqlc.UpdateUserProfileParams) (sqlc.User, error)
	UpdateUserEmail(ctx context.Context, params sqlc.UpdateUserEmailParams) (sqlc.User, error)
	UpdateUserPassword(ctx context.Context, params sqlc.UpdateUserPasswordParams) error

	// user_id → artist_id mapping
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
	EnsureArtist(ctx context.Context, userID uuid.UUID) error

	// Artist settings (scalar config)
	EnsureArtistSettings(ctx context.Context, artistID uuid.UUID) error
	GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error)
	UpdateArtistSettings(ctx context.Context, params sqlc.UpdateArtistSettingsParams) (sqlc.ArtistSetting, error)

	// Stripe Connect account linkage + onboarding status
	SetStripeAccount(ctx context.Context, params sqlc.SetStripeAccountParams) (sqlc.ArtistSetting, error)
	UpdateStripeAccountStatus(ctx context.Context, params sqlc.UpdateStripeAccountStatusParams) (sqlc.ArtistSetting, error)
	ClearStripeAccount(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error)
	GetArtistSettingsByStripeAccount(ctx context.Context, stripeAccountID *string) (sqlc.ArtistSetting, error)

	// Google Calendar connection (OAuth tokens stored encrypted)
	SetGoogleCalendarConnection(ctx context.Context, params sqlc.SetGoogleCalendarConnectionParams) (sqlc.ArtistSetting, error)
	ClearGoogleCalendarConnection(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error)

	// Availability windows
	ListAvailabilityWindows(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistAvailabilityWindow, error)
	DeleteAvailabilityWindows(ctx context.Context, artistID uuid.UUID) error
	InsertAvailabilityWindow(ctx context.Context, params sqlc.InsertAvailabilityWindowParams) (sqlc.ArtistAvailabilityWindow, error)

	// Session presets
	ListSessionPresets(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistSessionPreset, error)
	CreateSessionPreset(ctx context.Context, params sqlc.CreateSessionPresetParams) (sqlc.ArtistSessionPreset, error)
	UpdateSessionPreset(ctx context.Context, params sqlc.UpdateSessionPresetParams) (sqlc.ArtistSessionPreset, error)
	DeleteSessionPreset(ctx context.Context, params sqlc.DeleteSessionPresetParams) error

	// Days off
	ListDaysOff(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistDaysOff, error)
	AddDayOff(ctx context.Context, params sqlc.AddDayOffParams) error
	RemoveDayOff(ctx context.Context, params sqlc.RemoveDayOffParams) error

	// Blocklist
	ListBlocklist(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistBlocklist, error)
	AddBlocklistEntry(ctx context.Context, params sqlc.AddBlocklistEntryParams) (sqlc.ArtistBlocklist, error)
	RemoveBlocklistEntry(ctx context.Context, params sqlc.RemoveBlocklistEntryParams) error

	// RunInTx runs fn inside a transaction with a tx-bound Repository. Used to
	// replace an artist's availability windows atomically.
	RunInTx(ctx context.Context, fn func(Repository) error) error
}

type repository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db, q: sqlc.New(db)}
}

func (r *repository) withTx(tx pgx.Tx) *repository {
	return &repository{db: r.db, q: r.q.WithTx(tx)}
}

func (r *repository) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *repository) GetUserByUsername(ctx context.Context, username string) (sqlc.User, error) {
	return r.q.GetUserByUsername(ctx, &username)
}

func (r *repository) UpdateUserProfile(ctx context.Context, params sqlc.UpdateUserProfileParams) (sqlc.User, error) {
	return r.q.UpdateUserProfile(ctx, params)
}

func (r *repository) UpdateUserEmail(ctx context.Context, params sqlc.UpdateUserEmailParams) (sqlc.User, error) {
	return r.q.UpdateUserEmail(ctx, params)
}

func (r *repository) UpdateUserPassword(ctx context.Context, params sqlc.UpdateUserPasswordParams) error {
	return r.q.UpdateUserPassword(ctx, params)
}

func (r *repository) GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	return r.q.GetArtistByUserID(ctx, userID)
}

func (r *repository) EnsureArtist(ctx context.Context, userID uuid.UUID) error {
	return r.q.EnsureArtist(ctx, userID)
}

func (r *repository) EnsureArtistSettings(ctx context.Context, artistID uuid.UUID) error {
	return r.q.EnsureArtistSettings(ctx, artistID)
}

func (r *repository) GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error) {
	return r.q.GetArtistSettings(ctx, artistID)
}

func (r *repository) UpdateArtistSettings(ctx context.Context, params sqlc.UpdateArtistSettingsParams) (sqlc.ArtistSetting, error) {
	return r.q.UpdateArtistSettings(ctx, params)
}

func (r *repository) SetStripeAccount(ctx context.Context, params sqlc.SetStripeAccountParams) (sqlc.ArtistSetting, error) {
	return r.q.SetStripeAccount(ctx, params)
}

func (r *repository) UpdateStripeAccountStatus(ctx context.Context, params sqlc.UpdateStripeAccountStatusParams) (sqlc.ArtistSetting, error) {
	return r.q.UpdateStripeAccountStatus(ctx, params)
}

func (r *repository) ClearStripeAccount(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error) {
	return r.q.ClearStripeAccount(ctx, artistID)
}

func (r *repository) GetArtistSettingsByStripeAccount(ctx context.Context, stripeAccountID *string) (sqlc.ArtistSetting, error) {
	return r.q.GetArtistSettingsByStripeAccount(ctx, stripeAccountID)
}

func (r *repository) SetGoogleCalendarConnection(ctx context.Context, params sqlc.SetGoogleCalendarConnectionParams) (sqlc.ArtistSetting, error) {
	return r.q.SetGoogleCalendarConnection(ctx, params)
}

func (r *repository) ClearGoogleCalendarConnection(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error) {
	return r.q.ClearGoogleCalendarConnection(ctx, artistID)
}

func (r *repository) ListAvailabilityWindows(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistAvailabilityWindow, error) {
	return r.q.ListAvailabilityWindows(ctx, artistID)
}

func (r *repository) DeleteAvailabilityWindows(ctx context.Context, artistID uuid.UUID) error {
	return r.q.DeleteAvailabilityWindows(ctx, artistID)
}

func (r *repository) InsertAvailabilityWindow(ctx context.Context, params sqlc.InsertAvailabilityWindowParams) (sqlc.ArtistAvailabilityWindow, error) {
	return r.q.InsertAvailabilityWindow(ctx, params)
}

func (r *repository) ListSessionPresets(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistSessionPreset, error) {
	return r.q.ListSessionPresets(ctx, artistID)
}

func (r *repository) CreateSessionPreset(ctx context.Context, params sqlc.CreateSessionPresetParams) (sqlc.ArtistSessionPreset, error) {
	return r.q.CreateSessionPreset(ctx, params)
}

func (r *repository) UpdateSessionPreset(ctx context.Context, params sqlc.UpdateSessionPresetParams) (sqlc.ArtistSessionPreset, error) {
	return r.q.UpdateSessionPreset(ctx, params)
}

func (r *repository) DeleteSessionPreset(ctx context.Context, params sqlc.DeleteSessionPresetParams) error {
	return r.q.DeleteSessionPreset(ctx, params)
}

func (r *repository) ListDaysOff(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistDaysOff, error) {
	return r.q.ListDaysOff(ctx, artistID)
}

func (r *repository) AddDayOff(ctx context.Context, params sqlc.AddDayOffParams) error {
	return r.q.AddDayOff(ctx, params)
}

func (r *repository) RemoveDayOff(ctx context.Context, params sqlc.RemoveDayOffParams) error {
	return r.q.RemoveDayOff(ctx, params)
}

func (r *repository) ListBlocklist(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistBlocklist, error) {
	return r.q.ListBlocklist(ctx, artistID)
}

func (r *repository) AddBlocklistEntry(ctx context.Context, params sqlc.AddBlocklistEntryParams) (sqlc.ArtistBlocklist, error) {
	return r.q.AddBlocklistEntry(ctx, params)
}

func (r *repository) RemoveBlocklistEntry(ctx context.Context, params sqlc.RemoveBlocklistEntryParams) error {
	return r.q.RemoveBlocklistEntry(ctx, params)
}

func (r *repository) RunInTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(r.withTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
