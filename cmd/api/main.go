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
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Contexto raíz del proceso.
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	r := chi.NewRouter()

	// Middlewares base para trazabilidad y estabilidad.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	// Errores de routing se manejan a nivel router.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(w, r, http.StatusNotFound, "not_found", "resource not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	healthHandler := health.New(pool)
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
