package items

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type stubService struct{}

func (service *stubService) Create(ctx context.Context, in CreateItemInput) (Item, error) {
	return Item{ID: "id", Name: in.Name, Price: in.Price, Stock: in.Stock}, nil
}

func (service *stubService) List(ctx context.Context, page, limit int, query string) ([]Item, int, error) {
	return []Item{}, 0, nil
}

func (service *stubService) Get(ctx context.Context, id string) (Item, error) {
	return Item{ID: id}, nil
}

func (service *stubService) Update(ctx context.Context, id string, in UpdateItemInput) (Item, error) {
	return Item{ID: id}, nil
}

func (service *stubService) Delete(ctx context.Context, id string) error {
	return nil
}

func TestRegisterRoutes(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, NewHandler(&stubService{}))

	const id = "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "post items",
			method:     http.MethodPost,
			path:       "/items/",
			body:       `{"name":"Phone","price":"10.00","stock":2}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "get items",
			method:     http.MethodGet,
			path:       "/items/",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get item by id",
			method:     http.MethodGet,
			path:       "/items/" + id,
			wantStatus: http.StatusOK,
		},
		{
			name:       "patch item",
			method:     http.MethodPatch,
			path:       "/items/" + id,
			body:       `{"name":"Updated"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete item",
			method:     http.MethodDelete,
			path:       "/items/" + id,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			require.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}
