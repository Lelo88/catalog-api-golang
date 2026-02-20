package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// RequestIDFrom obtiene el ID de request generado por el middleware.
// Primero intenta leerlo del contexto (fuente de verdad). Como fallback, lee el header.
func RequestIDFrom(r *http.Request) string {
	if r == nil {
		return ""
	}

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		return reqID
	}

	return r.Header.Get("X-Request-Id")
}
