package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"

	"github.com/Lelo88/catalog-api-golang/internal/config"
	"github.com/Lelo88/catalog-api-golang/internal/db"
	"github.com/Lelo88/catalog-api-golang/internal/health"
	"github.com/Lelo88/catalog-api-golang/internal/httpx"
	"github.com/Lelo88/catalog-api-golang/internal/items"
	"github.com/Lelo88/catalog-api-golang/internal/docs"
)


type appPool interface {
	Ping(ctx context.Context) error
	Close()
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type appDeps struct {
	loadConfig      func() (config.Config, error)
	newPool         func(ctx context.Context, url string) (appPool, error)
	listenAndServe  func(addr string, handler http.Handler) error
	logf            func(format string, args ...any)
}

var (
	loadConfigFn     = config.Load
	newPoolFn        = func(ctx context.Context, url string) (appPool, error) { return db.NewPool(ctx, url) }
	listenAndServeFn = http.ListenAndServe
	logfFn           = log.Printf
	fatalf           = log.Fatal
)

// main carga dependencias reales y delega el arranque a run.
// Si run falla, finaliza el proceso con log.Fatal.
func main() {
	ctx := context.Background()
	deps := appDeps{
		loadConfig:     loadConfigFn,
		newPool:        newPoolFn,
		listenAndServe: listenAndServeFn,
		logf:           logfFn,
	}

	if err := run(ctx, deps); err != nil {
		fatalf(err)
	}
}

// run orquesta el inicio de la app: carga config, crea pool, arma el router y arranca el servidor.
func run(ctx context.Context, deps appDeps) error {
	configuration, err := deps.loadConfig()
	if err != nil {
		return err
	}

	pool, err := deps.newPool(ctx, configuration.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	router := buildRouter(pool)

	address := ":" + configuration.Port
	deps.logf("listening on %s", address)
	if err := deps.listenAndServe(address, router); err != nil {
		return err
	}

	return nil
}

// buildRouter construye el router HTTP con middlewares y rutas.
func buildRouter(pool appPool) http.Handler {
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

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.OK(w, r, http.StatusOK, map[string]any{
			"name":   "catalog-api-golang",
			"status": "ok",
			"docs":   "/docs/",
			"health": "/health",
			"ready":  "/ready",
			"openapi": "/openapi.yaml",
		})
	})

	healthHandler := health.New(pool)
	router.Get("/health", healthHandler.Health)
	router.Get("/ready", healthHandler.Ready)

	// Items
	itemsRepository := items.NewRepository(pool)
	itemsService := items.NewService(itemsRepository)
	itemsHandler := items.NewHandler(itemsService)
	items.RegisterRoutes(router, itemsHandler)

	// Docs
	docs.RegisterRoutes(router)
	router.Get("/openapi.yaml", docs.OpenAPIHandler())

	return router
}
