package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Olayori-X/stock-control-backend/internal/handlers"
	"github.com/Olayori-X/stock-control-backend/internal/tools/scheduler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Info("No .env file found (using system env)")
	}

	log.SetReportCaller(true)
	r := chi.NewRouter()

	// ✅ Add CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://scs-kappa.vercel.app"}, // Frontend domains
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "userid", "username"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by browsers
	}))

	// ✅ Register your app handlers
	handlers.Handler(r)

	// Server port setup
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default for local dev
	}

	srv := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: r,
	}

	// ── Background jobs ─────────────────────────────────────────────────
	schedulerCtx, cancelScheduler := context.WithCancel(context.Background())
	go scheduler.StartOverdueInvoiceCheck(schedulerCtx)

	// ── Start server ─────────────────────────────────────────────────────
	go func() {
		log.Infof("Starting server on port %s...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	// ── Graceful shutdown on SIGINT/SIGTERM ─────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("Shutdown signal received, stopping...")
	cancelScheduler()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shut down: ", err)
	} else {
		log.Info("Server shut down cleanly")
	}
}
