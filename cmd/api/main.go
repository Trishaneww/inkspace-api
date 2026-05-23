package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/database"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/server"
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

	srv := server.New(cfg, log, db, pub)

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
