package main

import (
	"log"
	"marketplace/internal/config"
	"marketplace/internal/handlers"
	sqlite "marketplace/internal/sqlite"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.MustLoad("./local.yaml")
	storage, err := sqlite.New(cfg.StoragePath)

	if err != nil {
		log.Fatalf("Failed to init storage: %s", err)
	}

	router := chi.NewRouter()

	router.Post("/register", handlers.RegisterHandler(storage))
	router.Post("/add-card", handlers.AuthMiddleware(handlers.AddCardHandler(storage)))
	router.Post("/login", handlers.LoginHandler(storage))

	router.Get("/all-cards", handlers.AuthMiddleware(handlers.GetAllCardsHandler(storage)))

	server := &http.Server{
		Addr:        cfg.Address,
		Handler:     router,
		ReadTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout: cfg.HTTPServer.IdleTimeout,
	}

	log.Println("Starting server...")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}

}
