package health

import (
	"net/http"
	"time"

	"github.com/Lelo88/catalog-api-golang/internal/httpx"
)

// Handler encapsula endpoints de health.
// Aunque hoy no tenga dependencias, lo dejamos como estructura para crecer (ready checks más adelante).
type Handler struct{}

// New crea un handler de health.
func New() *Handler {
	return &Handler{}
}

// Health indica si el proceso está vivo.
// NO chequea base de datos. Eso va en /ready más adelante.
func (handler *Handler) Health(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, r, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}
