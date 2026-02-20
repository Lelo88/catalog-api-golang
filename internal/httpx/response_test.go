package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestJSON_Success(t *testing.T) {
	rec := httptest.NewRecorder()

	JSON(rec, http.StatusCreated, Response{Data: map[string]any{"id": "1"}})

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

	resp := decodeResponse(t, rec)
	require.Nil(t, resp.Error)
	require.Nil(t, resp.Meta)
	data := asMap(t, resp.Data)
	require.Equal(t, "1", data["id"])
}

func TestJSON_EncodeError(t *testing.T) {
	rec := httptest.NewRecorder()

	JSON(rec, http.StatusTeapot, Response{Data: func() {}})

	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Contains(t, rec.Body.String(), "internal server error")
}

func TestOK(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "req-123")

	OK(rec, req, http.StatusOK, map[string]any{"ok": true})

	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeResponse(t, rec)
	require.Nil(t, resp.Error)
	require.NotNil(t, resp.Meta)
	require.Equal(t, "req-123", resp.Meta.RequestID)
	require.NotEmpty(t, resp.Meta.TimeUTC)
	_, err := time.Parse(time.RFC3339, resp.Meta.TimeUTC)
	require.NoError(t, err)

	data := asMap(t, resp.Data)
	require.Equal(t, true, data["ok"])
}

func TestFail(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "req-456")

	Fail(rec, req, http.StatusBadRequest, "invalid_input", "invalid input")

	require.Equal(t, http.StatusBadRequest, rec.Code)

	resp := decodeResponse(t, rec)
	require.Nil(t, resp.Data)
	require.NotNil(t, resp.Error)
	require.Equal(t, "invalid_input", resp.Error.Code)
	require.Equal(t, "invalid input", resp.Error.Message)
	require.NotNil(t, resp.Meta)
	require.Equal(t, "req-456", resp.Meta.RequestID)
	_, err := time.Parse(time.RFC3339, resp.Meta.TimeUTC)
	require.NoError(t, err)
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) Response {
	t.Helper()

	var response Response
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
