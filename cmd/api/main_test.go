package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lelo88/catalog-api-golang/internal/config"
	"github.com/Lelo88/catalog-api-golang/internal/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type fakePool struct {
	pingCalled  bool
	closeCalled bool
}

func (pool *fakePool) Ping(ctx context.Context) error {
	pool.pingCalled = true
	return nil
}

func (pool *fakePool) Close() {
	pool.closeCalled = true
}

func (pool *fakePool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

func (pool *fakePool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func TestMain_FatalOnError(t *testing.T) {
	originalLoad := loadConfigFn
	originalNewPool := newPoolFn
	originalListen := listenAndServeFn
	originalLogf := logfFn
	originalFatal := fatalf
	defer func() {
		loadConfigFn = originalLoad
		newPoolFn = originalNewPool
		listenAndServeFn = originalListen
		logfFn = originalLogf
		fatalf = originalFatal
	}()

	expectedErr := errors.New("config failed")
	loadConfigFn = func() (config.Config, error) {
		return config.Config{}, expectedErr
	}
	newPoolFn = func(ctx context.Context, url string) (appPool, error) {
		return nil, errors.New("should not be called")
	}
	listenAndServeFn = func(addr string, handler http.Handler) error {
		return nil
	}
	logfFn = func(format string, args ...any) {}

	fatalCalled := false
	var fatalArg any
	fatalf = func(args ...any) {
		fatalCalled = true
		if len(args) > 0 {
			fatalArg = args[0]
		}
	}

	main()

	require.True(t, fatalCalled)
	require.Equal(t, expectedErr, fatalArg)
}

func TestRun_ConfigError(t *testing.T) {
	deps := appDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, errors.New("load failed")
		},
		newPool: func(ctx context.Context, url string) (appPool, error) {
			return nil, errors.New("should not be called")
		},
		listenAndServe: func(addr string, handler http.Handler) error {
			return nil
		},
		logf: func(format string, args ...any) {},
	}

	err := run(context.Background(), deps)

	require.Error(t, err)
}

func TestRun_NewPoolError(t *testing.T) {
	deps := appDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{Port: "8080", DatabaseURL: "postgres://"}, nil
		},
		newPool: func(ctx context.Context, url string) (appPool, error) {
			return nil, errors.New("new pool failed")
		},
		listenAndServe: func(addr string, handler http.Handler) error {
			return nil
		},
		logf: func(format string, args ...any) {},
	}

	err := run(context.Background(), deps)

	require.Error(t, err)
}

func TestRun_ListenError(t *testing.T) {
	pool := &fakePool{}
	logged := ""
	deps := appDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{Port: "9090", DatabaseURL: "postgres://"}, nil
		},
		newPool: func(ctx context.Context, url string) (appPool, error) {
			return pool, nil
		},
		listenAndServe: func(addr string, handler http.Handler) error {
			return errors.New("listen failed")
		},
		logf: func(format string, args ...any) {
			logged = format
		},
	}

	err := run(context.Background(), deps)

	require.Error(t, err)
	require.True(t, pool.closeCalled)
	require.Equal(t, "listening on %s", logged)
}

func TestRun_Success(t *testing.T) {
	pool := &fakePool{}
	logCalled := false
	deps := appDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{Port: "7070", DatabaseURL: "postgres://"}, nil
		},
		newPool: func(ctx context.Context, url string) (appPool, error) {
			return pool, nil
		},
		listenAndServe: func(addr string, handler http.Handler) error {
			return nil
		},
		logf: func(format string, args ...any) {
			logCalled = true
		},
	}

	err := run(context.Background(), deps)

	require.NoError(t, err)
	require.True(t, pool.closeCalled)
	require.True(t, logCalled)
}

func TestBuildRouter_HealthReady(t *testing.T) {
	pool := &fakePool{}
	router := buildRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	data := asMap(t, resp.Data)
	require.Equal(t, "ok", data["status"])

	req = httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	resp = decodeResponse(t, rec)
	data = asMap(t, resp.Data)
	require.Equal(t, "ready", data["status"])
	require.True(t, pool.pingCalled)
}

func TestBuildRouter_NotFound(t *testing.T) {
	pool := &fakePool{}
	router := buildRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	require.Equal(t, "not_found", resp.Error.Code)
}

func TestBuildRouter_MethodNotAllowed(t *testing.T) {
	pool := &fakePool{}
	router := buildRouter(pool)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	require.Equal(t, "method_not_allowed", resp.Error.Code)
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
