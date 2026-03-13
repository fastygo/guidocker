package main

import (
	"context"
	"dashboard/config"
	"dashboard/domain"
	"dashboard/infrastructure"
	boltrepo "dashboard/infrastructure/bolt"
	dockerrepo "dashboard/infrastructure/docker"
	"dashboard/interfaces"
	"dashboard/interfaces/middleware"
	appusecase "dashboard/usecase/app"
	"dashboard/views"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func findFreePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free ports found in range %d-%d", startPort, startPort+99)
}

func resolvePort(preferredPort int) (int, error) {
	if port, err := resolvePortIfFree(preferredPort); err == nil {
		return port, nil
	}
	return findFreePort(preferredPort + 1)
}

func resolvePortIfFree(port int) (int, error) {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return 0, err
	}
	listener.Close()
	return port, nil
}

func buildServer(cfg *config.Config, useCase domain.DashboardUseCase, appUseCase domain.AppUseCase, auth *middleware.SessionAuth, renderer *views.Renderer) *http.Server {
	mux := http.NewServeMux()
	handler := interfaces.NewDashboardHandler(useCase, renderer)
	handler.SetAppUseCase(appUseCase)
	handler.SetLoginHandler(auth.LoginHandler())

	interfaces.RegisterRoutes(mux, handler)
	return &http.Server{Addr: cfg.GetServerAddress(), Handler: auth.Middleware()(mux)}
}

func main() {
	// Load configuration
	cfg := config.Load()
	requestedPort := cfg.Server.Port
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
	appService := appusecase.NewAppService(appRepo, dockerRepository, cfg.Stacks.Dir)
	auth := middleware.NewSessionAuth(cfg.Auth.AdminUser, cfg.Auth.AdminPass)
	freePort, err := resolvePort(cfg.Server.Port)
	if err != nil {
		log.Fatalf("❌ No free ports available in range %d-%d", cfg.Server.Port, cfg.Server.Port+99)
	}
	cfg.Server.Port = freePort

	if requestedPort != freePort {
		log.Printf("⚠️  Port %d was unavailable, using fallback port %d", requestedPort, freePort)
	}

	renderer, err := views.NewRenderer()
	if err != nil {
		log.Fatalf("❌ Failed to initialize view renderer: %v", err)
	}

	server := buildServer(cfg, service, appService, auth, renderer)

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Starting Dashboard Server...")
		log.Printf("📡 Server URL: http://%s", cfg.GetServerAddress())
		log.Printf("📊 Data source: %s", cfg.Data.DashboardFile)
		log.Printf("🗂️  Stacks directory: %s", cfg.Stacks.Dir)
		log.Printf("🔐 Login page: http://%s/login", cfg.GetServerAddress())

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
