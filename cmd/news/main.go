package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/roman-zipp/news/internal/config"
	"github.com/roman-zipp/news/internal/database"
)

func main() {
	godotenv.Load()

	cfg := config.Load()

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, cfg.DBDriver); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}
