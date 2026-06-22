package main

import (
	"context"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caiyuan0111/aicode/internal/config"
	"github.com/caiyuan0111/aicode/internal/handler"
	"github.com/caiyuan0111/aicode/internal/logging"
	"github.com/caiyuan0111/aicode/internal/service"
	"github.com/caiyuan0111/aicode/internal/store"
)

func main() {
	cfg := config.Load()

	// Init logger
	logger, err := logging.New(cfg.ServiceName)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Open database
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create stores
	userStore := store.NewUserStore(db)
	tokenStore := store.NewTokenStore(db)

	// Create services
	authService := service.NewAuthService(userStore, tokenStore, cfg)

	// Create handlers
	authHandler := handler.NewAuthHandler(authService)

	// Create middleware
	authMiddleware := handler.AuthMiddleware(cfg.JWTSecret)
	rateLimiter := handler.NewRateLimitMiddleware()

	// Build router (Go 1.22+ method+path patterns)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", authHandler.HandleRegister)
	mux.HandleFunc("POST /api/auth/login", rateLimiter.Handler(http.HandlerFunc(authHandler.HandleLogin)).ServeHTTP)
	mux.HandleFunc("POST /api/auth/refresh", authHandler.HandleRefresh)
	mux.HandleFunc("GET /api/me", authMiddleware(http.HandlerFunc(authHandler.HandleMe)).ServeHTTP)
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)

	// Wrap with request logging
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      logging.TraceMiddleware(logger)(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server starting on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server stopped")
}
