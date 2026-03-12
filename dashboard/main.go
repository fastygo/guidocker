package main

import (
	"context"
	"fmt"
	"dashboard/config"
	"dashboard/domain"
	"dashboard/interfaces"
	"dashboard/infrastructure"
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

func buildServer(cfg *config.Config, useCase domain.DashboardUseCase) *http.Server {
	mux := http.NewServeMux()
	handler := interfaces.NewDashboardHandler(useCase)

	interfaces.RegisterRoutes(mux, handler)
	return &http.Server{Addr: cfg.GetServerAddress(), Handler: mux}
}

func main() {
	// Load configuration
	cfg := config.Load()
	requestedPort := cfg.Server.Port
	repo := infrastructure.NewDashboardRepository(cfg.Data.DashboardFile)
	service := domain.NewDashboardService(repo)
	freePort, err := resolvePort(cfg.Server.Port)
	if err != nil {
		log.Fatalf("❌ No free ports available in range %d-%d", cfg.Server.Port, cfg.Server.Port+99)
	}
	cfg.Server.Port = freePort

	if requestedPort != freePort {
		log.Printf("⚠️  Port %d was unavailable, using fallback port %d", requestedPort, freePort)
	}

	server := buildServer(cfg, service)

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Starting Dashboard Server...")
		log.Printf("📡 Server URL: http://%s", cfg.GetServerAddress())
		log.Printf("📊 Data source: %s", cfg.Data.DashboardFile)

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
