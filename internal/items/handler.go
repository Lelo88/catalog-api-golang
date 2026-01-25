package items

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Lelo88/catalog-api-golang/internal/httpx"
)

// Handler HTTP para items.
// Solo traduce HTTP <-> dominio (service).
type Handler struct {
	service *Service
}

// NewHandler crea un handler de items.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create maneja POST /items.
func (handler *Handler) Create(writer http.ResponseWriter, request *http.Request) {
	var itemInput CreateItemInput
	if err := json.NewDecoder(request.Body).Decode(&itemInput); err != nil {
		httpx.Fail(writer, request, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	item, err := handler.service.Create(request.Context(), itemInput)
	if err != nil {
		switch {
		case errors.Is(err, ErrorInvalidInput):
			httpx.Fail(writer, request, http.StatusBadRequest, "invalid_input", "invalid input data")
		case errors.Is(err, ErrorDuplicateName):
			httpx.Fail(writer, request, http.StatusConflict, "conflict", "item name already exists")
		default:
			// No filtramos detalles internos.
			httpx.Fail(writer, request, http.StatusInternalServerError, "internal_error", "unexpected error")
		}
		return
	}

	httpx.OK(writer, request, http.StatusCreated, item)
}
