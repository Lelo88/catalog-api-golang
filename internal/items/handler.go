package items

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Lelo88/catalog-api-golang/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ServiceAPI define lo que el handler necesita.
// Permite testear handlers con stubs sin tocar DB.
type ServiceAPI interface {
	Create(ctx context.Context, in CreateItemInput) (Item, error)
	List(ctx context.Context, page, limit int, query string) ([]Item, int, error)
	Get(ctx context.Context, id string) (Item, error)
	Update(ctx context.Context, id string, in UpdateItemInput) (Item, error)
	Delete(ctx context.Context, id string) error
}

// Handler HTTP para items.
// Solo traduce HTTP <-> dominio (service).
type Handler struct {
	service ServiceAPI
}

// NewHandler crea un handler de items.
func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

type pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
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

// List maneja GET /items con paginación y búsqueda.
func (handler *Handler) List(writer http.ResponseWriter, request *http.Request) {
	page, limit, err := parsePagination(request)
	if err != nil {
		httpx.Fail(writer, request, http.StatusBadRequest, "invalid_pagination", "invalid pagination parameters")
		return
	}

	query := strings.TrimSpace(request.URL.Query().Get("query"))

	items, total, err := handler.service.List(request.Context(), page, limit, query)
	if err != nil {
		switch {
		case errors.Is(err, ErrorInvalidInput):
			httpx.Fail(writer, request, http.StatusBadRequest, "invalid_input", "invalid input data")
		default:
			httpx.Fail(writer, request, http.StatusInternalServerError, "internal_error", "unexpected error")
		}
		return
	}

	httpx.OK(writer, request, http.StatusOK, map[string]any{
		"items": items,
		"pagination": pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// parsePagination parsea page y limit con defaults y límites razonables.
func parsePagination(request *http.Request) (int, int, error) {
	const (
		defaultPage  = 1
		defaultLimit = 20
		maxLimit     = 100
	)

	query := request.URL.Query()

	page := defaultPage
	limit := defaultLimit

	if value := strings.TrimSpace(query.Get("page")); value != "" {
		pageNumber, err := strconv.Atoi(value)
		if err != nil || pageNumber < 1 {
			return 0, 0, err
		}
		page = pageNumber
	}

	if value := strings.TrimSpace(query.Get("limit")); value != "" {
		limitNumber, err := strconv.Atoi(value)
		if err != nil || limitNumber < 1 {
			return 0, 0, err
		}
		if limitNumber > maxLimit {
			limitNumber = maxLimit
		}
		limit = limitNumber
	}

	return page, limit, nil
}

// GetByID maneja GET /items/{id}.
// Valida que el id sea UUID porque en DB es uuid; esto evita errores innecesarios.
func (handler *Handler) GetByID(writer http.ResponseWriter, request *http.Request) {
	id := chi.URLParam(request, "id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.Fail(writer, request, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	item, err := handler.service.Get(request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrorNotFound):
			httpx.Fail(writer, request, http.StatusNotFound, "not_found", "item not found")
		default:
			httpx.Fail(writer, request, http.StatusInternalServerError, "internal_error", "unexpected error")
		}
		return
	}

	httpx.OK(writer, request, http.StatusOK, item)
}

// Patch maneja PATCH /items/{id}.
func (handler *Handler) Patch(writer http.ResponseWriter, request *http.Request) {
	id := chi.URLParam(request, "id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.Fail(writer, request, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	// Primero leemos raw para saber qué campos vinieron.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(request.Body).Decode(&raw); err != nil {
		httpx.Fail(writer, request, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	// Re-encode y decode al struct para reutilizar tags y tipos.
	byteJson, _ := json.Marshal(raw)

	var itemInputUpdated UpdateItemInput
	if err := json.Unmarshal(byteJson, &itemInputUpdated); err != nil {
		httpx.Fail(writer, request, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	// Manejo explícito de description:
	// - Si el cliente envió "description": null => queremos setear NULL.
	// - Si NO envió "description" => no queremos tocar.
	_, descPresent := raw["description"]
	itemInputUpdated.DescriptionPresent = descPresent

	item, err := handler.service.Update(request.Context(), id, itemInputUpdated)
	if err != nil {
		switch {
		case errors.Is(err, ErrorInvalidInput):
			httpx.Fail(writer, request, http.StatusBadRequest, "invalid_input", "invalid input data")
		case errors.Is(err, ErrorNotFound):
			httpx.Fail(writer, request, http.StatusNotFound, "not_found", "item not found")
		case errors.Is(err, ErrorDuplicateName):
			httpx.Fail(writer, request, http.StatusConflict, "conflict", "item name already exists")
		default:
			httpx.Fail(writer, request, http.StatusInternalServerError, "internal_error", "unexpected error")
		}
		return
	}

	httpx.OK(writer, request, http.StatusOK, item)
}

// Delete maneja DELETE /items/{id}.
func (handler *Handler) Delete(writer http.ResponseWriter, request *http.Request) {
	id := chi.URLParam(request, "id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.Fail(writer, request, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	err := handler.service.Delete(request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrorNotFound):
			httpx.Fail(writer, request, http.StatusNotFound, "not_found", "item not found")
		default:
			httpx.Fail(writer, request, http.StatusInternalServerError, "internal_error", "unexpected error")
		}
		return
	}

	// 204 No Content: respuesta vacía.
	writer.WriteHeader(http.StatusNoContent)
}