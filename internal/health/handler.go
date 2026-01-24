package health

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Lelo88/catalog-api-golang/internal/httpx"
)

// Handler expone endpoints de salud.
// Incluye checks simples para liveness (/health) y readiness (/ready).
type Handler struct {
	db *pgxpool.Pool
}

// New crea un Handler. db puede ser nil si querés correr sin base en algún entorno,
// pero para este proyecto la DB es requerida (config.Load la exige).
func New(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

// Health indica si el proceso está vivo.
// No depende de base de datos.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, r, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

// Ready indica si el servicio está listo para atender tráfico.
// Acá sí verificamos dependencias críticas (por ahora, la base de datos).
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	// Timeout corto para readiness. Si la DB no responde rápido, consideramos que no está lista.
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if h.db == nil {
		httpx.Fail(w, r, http.StatusServiceUnavailable, "not_ready", "database pool not configured")
		return
	}

	if err := h.db.Ping(ctx); err != nil {
		httpx.Fail(w, r, http.StatusServiceUnavailable, "not_ready", "database is not reachable")
		return
	}

	httpx.OK(w, r, http.StatusOK, map[string]any{
		"status": "ready",
	})
}
