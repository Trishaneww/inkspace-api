package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/events"

	"github.com/trishaneupnexx/inkspace-api/internal/modules/artists"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/messaging"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/notifications"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/payments"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/portfolios"
)

func registerRoutes(engine *gin.Engine, cfg *config.Config, db *pgxpool.Pool, pub *events.Publisher) {
	api := engine.Group("/v1")

	auth.New(cfg, db, pub).RegisterRoutes(api)
	artists.New(cfg, db, pub).RegisterRoutes(api)
	messaging.New(cfg, db, pub).RegisterRoutes(api)
	notifications.New(cfg, db, pub).RegisterRoutes(api)
	payments.New(cfg, db, pub).RegisterRoutes(api)
	portfolios.New(cfg, db, pub).RegisterRoutes(api)
}
