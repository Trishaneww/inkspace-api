package messaging

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
	"github.com/trishaneupnexx/inkspace-api/internal/ratelimit"
)

type Module struct {
	cfg     *config.Config
	rdb     *redis.Client
	Handler *Handler
	Service Service
	Repo    Repository
}

func New(cfg *config.Config, db *pgxpool.Pool, pub *events.Publisher, rdb *redis.Client) *Module {
	repo := NewRepository(db)
	svc := NewService(repo, pub)
	h := NewHandler(svc)
	return &Module{cfg: cfg, rdb: rdb, Handler: h, Service: svc, Repo: repo}
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	artistSend := m.sendLimiter("rl:msg:artist", ratelimit.Rule{RPS: 1, Burst: 15})
	clientSend := m.sendLimiter("rl:msg:client", ratelimit.Rule{RPS: 1, Burst: 15})
	guestSend := m.sendLimiter("rl:msg:guest", ratelimit.Rule{RPS: 0.2, Burst: 5})

	artist := rg.Group("/current-user/conversations")
	artist.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	artist.Use(middleware.RequireRole(string(auth.RoleArtist)))
	artist.GET("", m.Handler.ListArtistConversations)
	artist.GET("/:id", m.Handler.GetArtistConversation)
	artist.POST("/:id/messages", artistSend, m.Handler.SendArtistMessage)
	artist.POST("/:id/read", m.Handler.MarkArtistRead)

	client := rg.Group("/current-user/client/conversations")
	client.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	client.Use(middleware.RequireRole(string(auth.RoleUser)))
	client.GET("", m.Handler.ListClientConversations)
	client.GET("/:id", m.Handler.GetClientConversation)
	client.POST("/:id/messages", clientSend, m.Handler.SendClientMessage)
	client.POST("/:id/read", m.Handler.MarkClientRead)

	guest := rg.Group("/conversations/by-token/:token")
	guest.GET("", m.Handler.GetGuestConversation)
	guest.POST("/messages", guestSend, m.Handler.SendGuestMessage)
	guest.POST("/read", m.Handler.MarkGuestRead)
}

func (m *Module) sendLimiter(prefix string, rule ratelimit.Rule) gin.HandlerFunc {
	if m.rdb == nil {
		return func(c *gin.Context) {}
	}
	return middleware.RateLimit(ratelimit.NewRedisLimiter(m.rdb, rule), prefix, slog.Default())
}
