package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Lelo88/catalog-api-golang/internal/httpx"
	"github.com/stretchr/testify/require"
)

type fakeDB struct {
	pingFn     func(ctx context.Context) error
	pingCalled bool
	lastCtx    context.Context
}

func (db *fakeDB) Ping(ctx context.Context) error {
	db.pingCalled = true
	db.lastCtx = ctx
	if db.pingFn != nil {
		return db.pingFn(ctx)
	}
	return nil
}

func TestHandler_Health(t *testing.T) {
	handler := New(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	data := asMap(t, resp.Data)
	require.Equal(t, "ok", data["status"])
}

func TestHandler_Ready(t *testing.T) {
	t.Run("db not configured", func(t *testing.T) {
		handler := New(nil)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()

		handler.Ready(rec, req)

		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		resp := decodeResponse(t, rec)
		require.NotNil(t, resp.Error)
		require.Equal(t, "not_ready", resp.Error.Code)
		require.Equal(t, "database pool not configured", resp.Error.Message)
	})

	t.Run("ping error", func(t *testing.T) {
		pingErr := errors.New("db down")
		db := &fakeDB{pingFn: func(ctx context.Context) error { return pingErr }}
		handler := New(db)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()

		handler.Ready(rec, req)

		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		resp := decodeResponse(t, rec)
		require.NotNil(t, resp.Error)
		require.Equal(t, "not_ready", resp.Error.Code)
		require.Equal(t, "database is not reachable", resp.Error.Message)
		require.True(t, db.pingCalled)
		deadline, ok := db.lastCtx.Deadline()
		require.True(t, ok)
		require.True(t, time.Until(deadline) <= 2*time.Second+100*time.Millisecond)
	})

	t.Run("ready", func(t *testing.T) {
		db := &fakeDB{}
		handler := New(db)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()

		handler.Ready(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		resp := decodeResponse(t, rec)
		data := asMap(t, resp.Data)
		require.Equal(t, "ready", data["status"])
		require.True(t, db.pingCalled)
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
