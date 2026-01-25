package items

import "github.com/go-chi/chi/v5"

// RegisterRoutes registra rutas de items en el router.
// Mantener esto separado hace que main.go no crezca sin control.
func RegisterRoutes(route chi.Router, handler *Handler) {
	route.Route("/items", func(route chi.Router) {
		route.Post("/", handler.Create)
		route.Get("/", handler.List)
		route.Get("/{id}", handler.GetByID)
		route.Patch("/{id}", handler.Patch)
	})
}
