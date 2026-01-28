package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	api "zckyachmd/lifeline/internal/api"
	"zckyachmd/lifeline/internal/auth"
	"zckyachmd/lifeline/internal/config"
	"zckyachmd/lifeline/internal/handlers"
	"zckyachmd/lifeline/internal/mode"
	"zckyachmd/lifeline/internal/security/audit"
	"zckyachmd/lifeline/internal/security/confirm"
	rl "zckyachmd/lifeline/internal/security/ratelimit"
	"zckyachmd/lifeline/internal/services"
	"zckyachmd/lifeline/pkg/jailer"
	"zckyachmd/lifeline/pkg/logger"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("fatal panic: %v", r)
		}
	}()

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logg := logger.New(cfg.Logging.Level)

	botAPI, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		log.Fatalf("telegram init: %v", err)
	}
	botAPI.Debug = false
	_, _ = botAPI.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})

	jail, err := jailer.New(cfg.Sandbox.Root)
	if err != nil {
		log.Fatalf("sandbox init: %v", err)
	}
	if _, err := jail.EnsureDir("inbox"); err != nil {
		log.Fatalf("inbox init: %v", err)
	}
	_, _ = jail.EnsureDir("snapshots")

	dsmClient := api.NewClient(cfg.DSM.BaseURL, cfg.DSM.APIToken)
	monitor := services.NewMonitoring(dsmClient)
	sys := &services.SystemService{}
	snap := services.NewSnapshot(monitor, sys)
	files := services.NewFileService(jail, cfg.Sandbox.MaxFileMB)

	authz := auth.New(cfg.Telegram.AdminChatIDs)
	limiter := rl.New(cfg.Security.RateLimitPerMin, time.Minute)
	confirmMgr := confirm.New(cfg.ConfirmTTL())

	initialMode := mode.ReadOnly
	switch cfg.Security.DefaultMode {
	case "emergency":
		initialMode = mode.Emergency
	case "lockdown":
		initialMode = mode.Lockdown
	}
	modes := mode.New(initialMode)

	auditPath := filepath.Join(cfg.Sandbox.Root, "audit.log")
	auditLog := audit.New(auditPath)

	bot := handlers.New(botAPI, authz, limiter, confirmMgr, modes, auditLog, monitor, files, sys, snap, logg, cfg.Sandbox.Root, cfg.ConfirmTTL(), cfg.Telegram.PollTimeout)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// DSM token rotation ticker (placeholder rotates to same token).
	go func() {
		ticker := time.NewTicker(cfg.TokenRefreshInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				dsmClient.RotateToken(cfg.DSM.APIToken)
			}
		}
	}()

	// health endpoint on localhost for container orchestration
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logg.Error().Interface("panic", r).Msg("health server panic")
			}
		}()
		startHealthServer()
	}()

	if err := bot.Start(ctx); err != nil {
		logg.Error().Err(err).Msg("bot stopped")
	}
}

func startHealthServer() {
	// simple health server on localhost:8080
	srv := &http.Server{Addr: "127.0.0.1:8080"}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	_ = srv.ListenAndServe()
}
