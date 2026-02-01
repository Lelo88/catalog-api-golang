package docs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestRegisterRoutes_DocsRedirect(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusMovedPermanently, rec.Code)
	require.Equal(t, "/docs/", rec.Header().Get("Location"))
}

func TestRegisterRoutes_DocsAssets(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router)

	tests := []struct {
		name        string
		path        string
		contentType string
		file        string
	}{
		{
			name:        "swagger ui",
			path:        "/docs/",
			contentType: "text/html; charset=utf-8",
			file:        "swagger.html",
		},
		{
			name:        "openapi spec",
			path:        "/docs/openapi.yaml",
			contentType: "application/yaml; charset=utf-8",
			file:        "openapi.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, tt.contentType, rec.Header().Get("Content-Type"))
			require.Equal(t, expected, rec.Body.Bytes())
		})
	}
}
