package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"

	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/bookings"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/flashes"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/openbook"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/payments"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/portfolios"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/settings"
)

func registerRoutes(engine *gin.Engine, cfg *config.Config, db *pgxpool.Pool, pub *events.Publisher, s3 *s3client.Client, rdb *redis.Client) {
	api := engine.Group("/v1")

	auth.New(cfg, db, pub, rdb).RegisterRoutes(api)
	portfolios.New(cfg, db, s3).RegisterRoutes(api)
	flashes.New(cfg, db, pub, s3).RegisterRoutes(api)
	settings.New(cfg, db, s3).RegisterRoutes(api)
	bookings.New(cfg, db, s3).RegisterRoutes(api)
	openbook.New(cfg, db, s3).RegisterRoutes(api)
	payments.New(cfg, db).RegisterRoutes(api)
}
