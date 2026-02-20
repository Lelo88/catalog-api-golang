package items_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Lelo88/catalog-api-golang/internal/httpx"
	"github.com/Lelo88/catalog-api-golang/internal/items"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type stubService struct {
	createFn func(ctx context.Context, in items.CreateItemInput) (items.Item, error)
	listFn   func(ctx context.Context, page, limit int, query string) ([]items.Item, int, error)
	getFn    func(ctx context.Context, id string) (items.Item, error)
	updateFn func(ctx context.Context, id string, in items.UpdateItemInput) (items.Item, error)
	deleteFn func(ctx context.Context, id string) error

	createCalled bool
	createInput  items.CreateItemInput

	listCalled bool
	listPage   int
	listLimit  int
	listQuery  string

	getCalled bool
	getID     string

	updateCalled bool
	updateID     string
	updateInput  items.UpdateItemInput

	deleteCalled bool
	deleteID     string
}

func (service *stubService) Create(ctx context.Context, in items.CreateItemInput) (items.Item, error) {
	service.createCalled = true
	service.createInput = in
	if service.createFn != nil {
		return service.createFn(ctx, in)
	}
	return items.Item{}, nil
}

func (service *stubService) List(ctx context.Context, page, limit int, query string) ([]items.Item, int, error) {
	service.listCalled = true
	service.listPage = page
	service.listLimit = limit
	service.listQuery = query
	if service.listFn != nil {
		return service.listFn(ctx, page, limit, query)
	}
	return nil, 0, nil
}

func (service *stubService) Get(ctx context.Context, id string) (items.Item, error) {
	service.getCalled = true
	service.getID = id
	if service.getFn != nil {
		return service.getFn(ctx, id)
	}
	return items.Item{}, nil
}

func (service *stubService) Update(ctx context.Context, id string, in items.UpdateItemInput) (items.Item, error) {
	service.updateCalled = true
	service.updateID = id
	service.updateInput = in
	if service.updateFn != nil {
		return service.updateFn(ctx, id, in)
	}
	return items.Item{}, nil
}

func (service *stubService) Delete(ctx context.Context, id string) error {
	service.deleteCalled = true
	service.deleteID = id
	if service.deleteFn != nil {
		return service.deleteFn(ctx, id)
	}
	return nil
}

func TestHandler_Create(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		service := &stubService{}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Create(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_json", resp.Error.Code)
		require.False(t, service.createCalled)
	})

	t.Run("invalid input", func(t *testing.T) {
		service := &stubService{
			createFn: func(ctx context.Context, in items.CreateItemInput) (items.Item, error) {
				return items.Item{}, items.ErrorInvalidInput
			},
		}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"Phone","price":"10.00","stock":1}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Create(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_input", resp.Error.Code)
	})

	t.Run("duplicate name", func(t *testing.T) {
		service := &stubService{
			createFn: func(ctx context.Context, in items.CreateItemInput) (items.Item, error) {
				return items.Item{}, items.ErrorDuplicateName
			},
		}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"Phone","price":"10.00","stock":1}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Create(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "conflict", resp.Error.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		service := &stubService{
			createFn: func(ctx context.Context, in items.CreateItemInput) (items.Item, error) {
				return items.Item{}, errors.New("boom")
			},
		}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"Phone","price":"10.00","stock":1}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Create(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "internal_error", resp.Error.Code)
	})

	t.Run("success", func(t *testing.T) {
		service := &stubService{
			createFn: func(ctx context.Context, in items.CreateItemInput) (items.Item, error) {
				return items.Item{ID: "id-1", Name: in.Name, Price: in.Price, Stock: in.Stock}, nil
			},
		}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name":"Phone","price":"10.00","stock":1}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.Create(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		resp := decodeResponse(t, rec)
		data := asMap(t, resp.Data)
		require.Equal(t, "id-1", data["id"])
		require.True(t, service.createCalled)
		require.Equal(t, "Phone", service.createInput.Name)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("invalid pagination value", func(t *testing.T) {
		service := &stubService{}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/items?page=abc", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_pagination", resp.Error.Code)
		require.False(t, service.listCalled)
	})

	t.Run("invalid input from service", func(t *testing.T) {
		service := &stubService{
			listFn: func(ctx context.Context, page, limit int, query string) ([]items.Item, int, error) {
				return nil, 0, items.ErrorInvalidInput
			},
		}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/items?page=1&limit=10", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_input", resp.Error.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		service := &stubService{
			listFn: func(ctx context.Context, page, limit int, query string) ([]items.Item, int, error) {
				return nil, 0, errors.New("boom")
			},
		}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/items?page=1&limit=10", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "internal_error", resp.Error.Code)
	})

	t.Run("success with defaults and trimmed query", func(t *testing.T) {
		service := &stubService{
			listFn: func(ctx context.Context, page, limit int, query string) ([]items.Item, int, error) {
				return []items.Item{{ID: "id-1"}}, 1, nil
			},
		}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/items?query=+phone+", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, service.listCalled)
		require.Equal(t, 1, service.listPage)
		require.Equal(t, 20, service.listLimit)
		require.Equal(t, "phone", service.listQuery)

		resp := decodeResponse(t, rec)
		data := asMap(t, resp.Data)
		itemsList := asSlice(t, data["items"])
		require.Len(t, itemsList, 1)
		pagination := asMap(t, data["pagination"])
		require.Equal(t, json.Number("1"), pagination["page"])
		require.Equal(t, json.Number("20"), pagination["limit"])
		require.Equal(t, json.Number("1"), pagination["total"])
	})

	t.Run("limit capped", func(t *testing.T) {
		service := &stubService{
			listFn: func(ctx context.Context, page, limit int, query string) ([]items.Item, int, error) {
				return []items.Item{}, 0, nil
			},
		}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/items?page=1&limit=500", nil)
		rec := httptest.NewRecorder()

		handler.List(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, service.listCalled)
		require.Equal(t, 100, service.listLimit)
	})
}

func TestHandler_GetByID(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		service := &stubService{}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodGet, "/items/not-uuid", nil)
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", "not-uuid")

		handler.GetByID(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_id", resp.Error.Code)
		require.False(t, service.getCalled)
	})

	t.Run("not found", func(t *testing.T) {
		service := &stubService{
			getFn: func(ctx context.Context, id string) (items.Item, error) {
				return items.Item{}, items.ErrorNotFound
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodGet, "/items/"+id, nil)
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.GetByID(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "not_found", resp.Error.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		service := &stubService{
			getFn: func(ctx context.Context, id string) (items.Item, error) {
				return items.Item{}, errors.New("boom")
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodGet, "/items/"+id, nil)
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.GetByID(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "internal_error", resp.Error.Code)
	})

	t.Run("success", func(t *testing.T) {
		service := &stubService{
			getFn: func(ctx context.Context, id string) (items.Item, error) {
				return items.Item{ID: id, Name: "Phone"}, nil
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodGet, "/items/"+id, nil)
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.GetByID(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		resp := decodeResponse(t, rec)
		data := asMap(t, resp.Data)
		require.Equal(t, id, data["id"])
		require.True(t, service.getCalled)
		require.Equal(t, id, service.getID)
	})
}

func TestHandler_Patch(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		service := &stubService{}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodPatch, "/items/not-uuid", strings.NewReader(`{"name":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", "not-uuid")

		handler.Patch(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_id", resp.Error.Code)
		require.False(t, service.updateCalled)
	})

	t.Run("invalid json", func(t *testing.T) {
		service := &stubService{}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodPatch, "/items/"+id, strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Patch(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_json", resp.Error.Code)
		require.False(t, service.updateCalled)
	})

	t.Run("invalid json payload", func(t *testing.T) {
		service := &stubService{}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodPatch, "/items/"+id, strings.NewReader(`{"stock":"abc"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Patch(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_json", resp.Error.Code)
		require.False(t, service.updateCalled)
	})

	t.Run("invalid input", func(t *testing.T) {
		service := &stubService{
			updateFn: func(ctx context.Context, id string, in items.UpdateItemInput) (items.Item, error) {
				return items.Item{}, items.ErrorInvalidInput
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodPatch, "/items/"+id, strings.NewReader(`{"name":""}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Patch(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_input", resp.Error.Code)
	})

	t.Run("not found", func(t *testing.T) {
		service := &stubService{
			updateFn: func(ctx context.Context, id string, in items.UpdateItemInput) (items.Item, error) {
				return items.Item{}, items.ErrorNotFound
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodPatch, "/items/"+id, strings.NewReader(`{"name":"New"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Patch(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "not_found", resp.Error.Code)
	})

	t.Run("duplicate name", func(t *testing.T) {
		service := &stubService{
			updateFn: func(ctx context.Context, id string, in items.UpdateItemInput) (items.Item, error) {
				return items.Item{}, items.ErrorDuplicateName
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodPatch, "/items/"+id, strings.NewReader(`{"name":"New"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Patch(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "conflict", resp.Error.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		service := &stubService{
			updateFn: func(ctx context.Context, id string, in items.UpdateItemInput) (items.Item, error) {
				return items.Item{}, errors.New("boom")
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodPatch, "/items/"+id, strings.NewReader(`{"name":"New"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Patch(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "internal_error", resp.Error.Code)
	})

	t.Run("success with description null", func(t *testing.T) {
		service := &stubService{
			updateFn: func(ctx context.Context, id string, in items.UpdateItemInput) (items.Item, error) {
				return items.Item{ID: id, Name: "Updated"}, nil
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		body := `{"description":null,"price":"10.00"}`
		req := httptest.NewRequest(http.MethodPatch, "/items/"+id, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Patch(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, service.updateCalled)
		require.Equal(t, id, service.updateID)
		require.True(t, service.updateInput.DescriptionPresent)
		require.Nil(t, service.updateInput.Description)
		require.NotNil(t, service.updateInput.Price)
	})

	t.Run("success without description", func(t *testing.T) {
		service := &stubService{
			updateFn: func(ctx context.Context, id string, in items.UpdateItemInput) (items.Item, error) {
				return items.Item{ID: id, Name: "Updated"}, nil
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		body := `{"name":"Updated"}`
		req := httptest.NewRequest(http.MethodPatch, "/items/"+id, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Patch(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, service.updateCalled)
		require.False(t, service.updateInput.DescriptionPresent)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("invalid id", func(t *testing.T) {
		service := &stubService{}
		handler := items.NewHandler(service)

		req := httptest.NewRequest(http.MethodDelete, "/items/not-uuid", nil)
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", "not-uuid")

		handler.Delete(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "invalid_id", resp.Error.Code)
		require.False(t, service.deleteCalled)
	})

	t.Run("not found", func(t *testing.T) {
		service := &stubService{
			deleteFn: func(ctx context.Context, id string) error {
				return items.ErrorNotFound
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodDelete, "/items/"+id, nil)
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Delete(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "not_found", resp.Error.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		service := &stubService{
			deleteFn: func(ctx context.Context, id string) error {
				return errors.New("boom")
			},
		}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodDelete, "/items/"+id, nil)
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Delete(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		resp := decodeResponse(t, rec)
		require.Equal(t, "internal_error", resp.Error.Code)
	})

	t.Run("success", func(t *testing.T) {
		service := &stubService{}
		handler := items.NewHandler(service)

		id := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest(http.MethodDelete, "/items/"+id, nil)
		rec := httptest.NewRecorder()
		req = withURLParam(req, "id", id)

		handler.Delete(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.True(t, service.deleteCalled)
		require.Equal(t, id, service.deleteID)
		require.Empty(t, rec.Body.String())
	})
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) httpx.Response {
	t.Helper()

	var response httpx.Response
	decoder := json.NewDecoder(bytes.NewReader(recorder.Body.Bytes()))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&response))
	return response
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()

	out, ok := value.(map[string]any)
	require.True(t, ok, "expected map, got %T", value)
	return out
}

func asSlice(t *testing.T, value any) []any {
	t.Helper()

	out, ok := value.([]any)
	require.True(t, ok, "expected slice, got %T", value)
	return out
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}
