package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Olayori-X/stock-control-backend/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found (using system env)")
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

	log.Printf("Starting server on port %s...", port)
	err = http.ListenAndServe("0.0.0.0:"+port, r)
	if err != nil {
		log.Error(err)
		fmt.Println("Failed to start server:", err)
	} else {
		fmt.Printf("Server is running on port %s...", port)
	}
}
