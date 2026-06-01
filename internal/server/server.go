package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

type Server struct {
	cfg    *config.Config
	log    *slog.Logger
	http   *http.Server
	engine *gin.Engine
}

func New(cfg *config.Config, log *slog.Logger, db *pgxpool.Pool, pub *events.Publisher, s3 *s3client.Client) *Server {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(
		middleware.Recover(log),
		middleware.Logger(log),
		middleware.CORS(cfg.CORSAllowedOrigins),
	)

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	registerRoutes(engine, cfg, db, pub, s3)

	return &Server{
		cfg:    cfg,
		log:    log,
		engine: engine,
		http: &http.Server{
			Addr:              fmt.Sprintf(":%s", cfg.AppPort),
			Handler:           engine,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	s.log.Info("server_starting", "port", s.cfg.AppPort)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
