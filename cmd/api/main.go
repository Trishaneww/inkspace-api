package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/trishaneupnexx/inkspace-api/internal/aitriage"
	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/database"
	"github.com/trishaneupnexx/inkspace-api/internal/email"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/holds"
	"github.com/trishaneupnexx/inkspace-api/internal/maintenance"
	"github.com/trishaneupnexx/inkspace-api/internal/messaging"
	"github.com/trishaneupnexx/inkspace-api/internal/messagingnotify"
	"github.com/trishaneupnexx/inkspace-api/internal/redisclient"
	"github.com/trishaneupnexx/inkspace-api/internal/reminders"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
	"github.com/trishaneupnexx/inkspace-api/internal/server"
)

const (
	cleanupInterval   = time.Hour
	reminderInterval  = time.Hour
	holdSweepInterval = 10 * time.Minute
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("config_load_failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db_init_failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	pub, err := events.NewPublisher(cfg.RabbitMQURL)
	if err != nil {
		log.Error("rabbitmq_init_failed", "error", err)
		os.Exit(1)
	}
	defer pub.Close()

	s3, err := s3client.New(ctx, cfg.AWSRegion, cfg.AWSS3Bucket)
	if err != nil {
		log.Error("s3_init_failed", "error", err)
		os.Exit(1)
	}

	// Redis backs distributed rate limiting. Required in production (validated
	// in config); optional in dev, where limiting is simply skipped.
	var rdb *redis.Client
	if cfg.RedisURL != "" {
		rdb, err = redisclient.New(ctx, cfg.RedisURL)
		if err != nil {
			log.Error("redis_init_failed", "error", err)
			os.Exit(1)
		}
		defer func() { _ = rdb.Close() }()
	}

	srv := server.New(cfg, log, db, pub, s3, rdb)
	go maintenance.NewCleaner(db, log, cleanupInterval).Run(ctx)

	smsSender := messaging.NewSender(cfg, log)
	emailClient := email.NewClient(cfg.FrontendURL, cfg.InternalEmailSecret, log)
	go reminders.New(db, smsSender, emailClient, cfg.FrontendURL, log, reminderInterval).Run(ctx)
	go holds.New(db, emailClient, log, holdSweepInterval).Run(ctx)
	go messagingnotify.Run(ctx, cfg.RabbitMQURL, db, emailClient, cfg.FrontendURL, log)
	go aitriage.Run(ctx, cfg.RabbitMQURL, db, s3, cfg.AnthropicAPIKey, log)

	go func() {
		if err := srv.Start(); err != nil {
			log.Error("server_failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutdown_initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server_shutdown_error", "error", err)
	}
	log.Info("shutdown_complete")
}
