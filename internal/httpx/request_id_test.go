package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"
)

func TestRequestIDFrom(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		require.Equal(t, "", RequestIDFrom(nil))
	})

	t.Run("context request id has priority", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-Id", "header-id")

		ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "ctx-id")
		req = req.WithContext(ctx)

		require.Equal(t, "ctx-id", RequestIDFrom(req))
	})

	t.Run("header fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-Id", "header-id")

		require.Equal(t, "header-id", RequestIDFrom(req))
	})

	t.Run("empty when missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		require.Equal(t, "", RequestIDFrom(req))
	})
}
