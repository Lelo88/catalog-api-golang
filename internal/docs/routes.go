package docs

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes monta las rutas de documentación (Swagger UI + OpenAPI YAML).
func RegisterRoutes(r chi.Router) {
	// Soporta /docs (sin slash) redirigiendo a /docs/
	r.Get("/docs", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/docs/", http.StatusMovedPermanently)
	})

	r.Route("/docs", func(r chi.Router) {
		// Swagger UI
		r.Get("/", SwaggerUIHandler())

		// Spec OpenAPI embebida (para que swagger.html la consuma por URL).
		r.Get("/openapi.yaml", OpenAPIHandler())
	})
}
