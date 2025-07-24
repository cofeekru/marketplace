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

	router.Post("/user/register", handlers.RegisterHandler(storage))
	router.Post("/user/login", handlers.LoginHandler(storage))

	router.Post("/item/add", handlers.AuthMiddleware(handlers.AddCardHandler(storage)))
	router.Get("/item/get", handlers.AuthMiddleware(handlers.GetAllCardsHandler(storage)))

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
