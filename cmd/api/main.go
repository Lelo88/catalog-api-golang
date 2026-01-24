package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Lelo88/catalog-api-golang/internal/health"
	"github.com/Lelo88/catalog-api-golang/internal/httpx"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := chi.NewRouter()

	// Middlewares base:
	// - RequestID: genera/propaga un ID por request para trazabilidad.
	// - RealIP: obtiene IP real detrás de proxies (útil si deployás).
	// - Logger: logging básico por request.
	// - Recoverer: evita que un panic tumbe el proceso.
	// - Timeout: corta requests colgados (evita conexiones zombis).
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(10 * time.Second))

	healthHandler := health.New()
	router.Get("/health", healthHandler.Health)

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(
			w, r,
			http.StatusNotFound,
			"not_found",
			"resource not found",
		)
	})

	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.Fail(
			w, r,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
	})

	address := ":" + port
	log.Printf("listening on %s", address)
	if err := http.ListenAndServe(address, router); err != nil {
		log.Fatal(err)
	}
}
