package main

import (
	"context"
	"dashboard/config"
	"dashboard/domain"
	"dashboard/interfaces"
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

func isPortInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true // port is in use
	}
	listener.Close()
	return false
}

func startServerOnPort(cfg *config.Config, port int) *http.Server {
	// Update config with the actual port
	cfg.Server.Port = port

	// Initialize dependencies (Clean Architecture layers)
	service := domain.NewDashboardService(cfg.Data.DashboardFile)
	handler := interfaces.NewDashboardHandler(service)

	// Setup HTTP routes using standard library
	mux := http.NewServeMux()

	// Web routes
	mux.HandleFunc("/", handler.Dashboard)

	// API routes
	mux.HandleFunc("/api/dashboard", handler.APIGetDashboard)
	mux.HandleFunc("/api/containers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			handler.APIUpdateContainer(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// Static files
	mux.Handle("/data/", http.StripPrefix("/data/", http.FileServer(http.Dir("data/"))))

	server := &http.Server{
		Addr:    cfg.GetServerAddress(),
		Handler: mux,
	}

	return server
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Use port from config (can be overridden by SERVER_PORT env var)
	preferredPort := cfg.Server.Port

	// Try to start server on preferred port first
	server := startServerOnPort(cfg, preferredPort)

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Starting Dashboard Server...")
		log.Printf("📡 Server URL: http://%s", cfg.GetServerAddress())
		log.Printf("📊 Data source: %s", cfg.Data.DashboardFile)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("⚠️  Port %d failed, trying alternative port...", preferredPort)

			// Find alternative port and restart server
			freePort, portErr := findFreePort(preferredPort + 1)
			if portErr != nil {
				log.Fatalf("❌ No free ports available in range %d-%d", preferredPort+1, preferredPort+99)
			}

			log.Printf("✅ Found free port: %d ✓ Using alternative port", freePort)
			server = startServerOnPort(cfg, freePort)

			// Try to start on alternative port
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("❌ Server failed to start on alternative port: %v", err)
			}
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
