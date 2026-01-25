package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Lelo88/catalog-api-golang/internal/config"
	"github.com/Lelo88/catalog-api-golang/internal/db"
	"github.com/Lelo88/catalog-api-golang/internal/health"
	"github.com/Lelo88/catalog-api-golang/internal/httpx"
	// + imports:
	"github.com/Lelo88/catalog-api-golang/internal/items"
)

func main() {
	configuration, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Contexto raíz del proceso.
	context := context.Background()

	pool, err := db.NewPool(context, configuration.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	router := chi.NewRouter()

	// Middlewares base para trazabilidad y estabilidad.
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(10 * time.Second))

	// Errores de routing se manejan a nivel router.
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(w, r, http.StatusNotFound, "not_found", "resource not found")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	healthHandler := health.New(pool)
	router.Get("/health", healthHandler.Health)
	router.Get("/ready", healthHandler.Ready)

	// Items
	itemsRepository := items.NewRepository(pool)
	itemsService := items.NewService(itemsRepository)
	itemsHandler := items.NewHandler(itemsService)
	items.RegisterRoutes(router, itemsHandler)

	address := ":" + configuration.Port
	log.Printf("listening on %s", address)
	if err := http.ListenAndServe(address, router); err != nil {
		log.Fatal(err)
	}
}
