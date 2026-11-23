package main

import (
	"fmt"
	"log"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Delcom Blog Backend on port %s...", cfg.AppPort)
	log.Printf("Connecting to database %s on %s:%s...", cfg.DBDatabase, cfg.DBHost, cfg.DBPort)

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := database.Seed(db, cfg); err != nil {
		log.Printf("Warning during seeding: %v", err)
	}

	r := server.SetupRouter(cfg, db)

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Delcom Blog Server running on http://127.0.0.1%s", addr)
	log.Printf("Swagger UI available at http://localhost%s/swagger/index.html", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
