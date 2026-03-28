package main

import (
	"context"
	"dashboard/config"
	"dashboard/domain"
	"dashboard/infrastructure"
	boltrepo "dashboard/infrastructure/bolt"
	dockerrepo "dashboard/infrastructure/docker"
	gitrepo "dashboard/infrastructure/git"
	"dashboard/infrastructure/hosting"
	"dashboard/interfaces"
	"dashboard/interfaces/middleware"
	appusecase "dashboard/usecase/app"
	scanusecase "dashboard/usecase/scanner"
	settingsusecase "dashboard/usecase/settings"
	"dashboard/views"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func buildServer(
	cfg *config.Config,
	useCase domain.DashboardUseCase,
	appUseCase domain.AppUseCase,
	scanUseCase domain.ScannerUseCase,
	platformSettingsUseCase domain.PlatformSettingsUseCase,
	auth *middleware.SessionAuth,
	renderer *views.Renderer,
) *http.Server {
	mux := http.NewServeMux()
	handler := interfaces.NewDashboardHandler(useCase, renderer)
	handler.SetAppUseCase(appUseCase)
	handler.SetScanUseCase(scanUseCase)
	handler.SetPlatformSettingsUseCase(platformSettingsUseCase)
	handler.SetCertificateOperations(hosting.NewCertbotManager(), hosting.NewNginxHostManager())
	auth.SetAPIOnly(cfg.Mode == "api")
	if cfg.Mode == "api" {
		interfaces.RegisterAPIRoutes(mux, handler)
	} else {
		handler.SetLoginHandler(auth.LoginHandler())
		interfaces.RegisterRoutes(mux, handler)
	}

	next := http.Handler(mux)
	if !cfg.Auth.Disabled {
		next = auth.Middleware()(mux)
	}

	return &http.Server{Addr: cfg.GetServerAddress(), Handler: next}
}

func main() {
	// Load configuration
	cfg := config.Load()
	repo := infrastructure.NewDashboardRepository(cfg.Data.DashboardFile)
	service := domain.NewDashboardService(repo)
	if err := os.MkdirAll(cfg.Stacks.Dir, 0o755); err != nil {
		log.Fatalf("❌ Failed to create stacks directory: %v", err)
	}

	appRepo, err := boltrepo.NewAppRepository(cfg.Stacks.DBFile)
	if err != nil {
		log.Fatalf("❌ Failed to initialize BoltDB repository: %v", err)
	}
	defer func() {
		if closeErr := appRepo.Close(); closeErr != nil {
			log.Printf("⚠️  Failed to close BoltDB repository: %v", closeErr)
		}
	}()

	dockerRepository := dockerrepo.NewDockerRepository(cfg.Stacks.Dir)
	if err := dockerRepository.EnsureNetwork(context.Background()); err != nil {
		log.Fatalf("❌ Failed to ensure app network: %v", err)
	}
	gitRepository := gitrepo.NewGitRepository()
	platformSettingsRepository, err := boltrepo.NewPlatformSettingsRepository(cfg.Stacks.DBFile)
	if err != nil {
		log.Fatalf("❌ Failed to initialize platform settings repository: %v", err)
	}
	defer func() {
		if closeErr := platformSettingsRepository.Close(); closeErr != nil {
			log.Printf("⚠️  Failed to close platform settings repository: %v", closeErr)
		}
	}()

	platformSettingsService := settingsusecase.NewPlatformSettingsService(platformSettingsRepository, domain.DefaultPlatformSettings())
	if platformSettings, err := platformSettingsService.GetPlatformSettings(context.Background()); err != nil {
		log.Printf("⚠️  Failed to load platform settings: %v", err)
	} else {
		log.Printf("🧭 Platform settings loaded: certbot_enabled=%v staging=%v auto_renew=%v", platformSettings.CertbotEnabled, platformSettings.CertbotStaging, platformSettings.CertbotAutoRenew)
	}

	appService := appusecase.NewAppService(appRepo, dockerRepository, gitRepository, cfg.Stacks.Dir).
		WithImportTimeout(cfg.Import.Timeout).
		WithImportTempPath(cfg.Import.TempPath).
		WithPlatformSettingsUseCase(platformSettingsService).
		WithHostManagers(
			hosting.NewNginxHostManager(),
			hosting.NewCertbotManager(),
		)
	auth := middleware.NewSessionAuth(cfg.Auth.AdminUser, cfg.Auth.AdminPass)

	var renderer *views.Renderer
	if cfg.Mode != "api" {
		renderer, err = views.NewRenderer()
		if err != nil {
			log.Fatalf("❌ Failed to initialize view renderer: %v", err)
		}
	}

	scanService := scanusecase.NewScannerService(dockerRepository, appRepo, cfg.Stacks.Dir)
	server := buildServer(cfg, service, appService, scanService, platformSettingsService, auth, renderer)

	// Start server in goroutine
	go func() {
		if cfg.Mode == "api" {
			log.Printf("🚀 Starting Dashboard Server (API-only mode)...")
			log.Printf("📡 API URL: http://%s/api/", cfg.GetServerAddress())
		} else {
			log.Printf("🚀 Starting Dashboard Server (GUI mode)...")
			log.Printf("📡 Server URL: http://%s", cfg.GetServerAddress())
			log.Printf("🔐 Login page: http://%s/login", cfg.GetServerAddress())
		}
		log.Printf("📊 Data source: %s", cfg.Data.DashboardFile)
		log.Printf("🗂️  Stacks directory: %s", cfg.Stacks.Dir)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	log.Println("💡 Server ready! Press Ctrl+C to stop gracefully")

	<-quit
	log.Println("🛑 Received shutdown signal...")
	log.Println("⏳ Gracefully shutting down server (5s timeout)")

	// Give outstanding requests 5 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Shutdown error: %v", err)
		log.Println("💥 Forcing server shutdown")
		os.Exit(1)
	}

	log.Println("✅ Server shutdown completed successfully")
	log.Println("👋 Goodbye!")
}
