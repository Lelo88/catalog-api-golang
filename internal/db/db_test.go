package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestNewPool_NewError(t *testing.T) {
	originalNewPool := newPool
	originalPingPool := pingPool
	originalClosePool := closePool
	defer func() {
		newPool = originalNewPool
		pingPool = originalPingPool
		closePool = originalClosePool
	}()

	expectedErr := errors.New("new pool failed")
	newPool = func(ctx context.Context, url string) (*pgxpool.Pool, error) {
		return nil, expectedErr
	}

	pingCalled := false
	pingPool = func(ctx context.Context, pool poolPinger) error {
		pingCalled = true
		return nil
	}

	closeCalled := false
	closePool = func(pool poolPinger) {
		closeCalled = true
	}

	pool, err := NewPool(context.Background(), "postgres://example")

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, pool)
	require.False(t, pingCalled)
	require.False(t, closeCalled)
}

func TestNewPool_PingError(t *testing.T) {
	originalNewPool := newPool
	originalPingPool := pingPool
	originalClosePool := closePool
	defer func() {
		newPool = originalNewPool
		pingPool = originalPingPool
		closePool = originalClosePool
	}()

	poolInstance := &pgxpool.Pool{}
	newPool = func(ctx context.Context, url string) (*pgxpool.Pool, error) {
		return poolInstance, nil
	}

	pingErr := errors.New("ping failed")
	pingPool = func(ctx context.Context, pool poolPinger) error {
		return pingErr
	}

	closeCalled := false
	closePool = func(pool poolPinger) {
		closeCalled = true
		require.Equal(t, poolInstance, pool)
	}

	pool, err := NewPool(context.Background(), "postgres://example")

	require.ErrorIs(t, err, pingErr)
	require.Nil(t, pool)
	require.True(t, closeCalled)
}

func TestNewPool_Success(t *testing.T) {
	originalNewPool := newPool
	originalPingPool := pingPool
	originalClosePool := closePool
	defer func() {
		newPool = originalNewPool
		pingPool = originalPingPool
		closePool = originalClosePool
	}()

	poolInstance := &pgxpool.Pool{}
	var capturedCtx context.Context
	var capturedURL string

	newPool = func(ctx context.Context, url string) (*pgxpool.Pool, error) {
		capturedCtx = ctx
		capturedURL = url
		return poolInstance, nil
	}

	pingCalled := false
	pingPool = func(ctx context.Context, pool poolPinger) error {
		pingCalled = true
		return nil
	}

	closeCalled := false
	closePool = func(pool poolPinger) {
		closeCalled = true
	}

	pool, err := NewPool(context.Background(), "postgres://example")

	require.NoError(t, err)
	require.Equal(t, poolInstance, pool)
	require.True(t, pingCalled)
	require.False(t, closeCalled)
	require.Equal(t, "postgres://example", capturedURL)

	deadline, ok := capturedCtx.Deadline()
	require.True(t, ok)
	require.True(t, time.Until(deadline) <= 5*time.Second)
	require.True(t, time.Until(deadline) > 0)
}

func TestDefaultPoolHooks(t *testing.T) {
	originalPingPool := pingPool
	originalClosePool := closePool
	defer func() {
		pingPool = originalPingPool
		closePool = originalClosePool
	}()

	fake := &fakePoolPinger{}

	err := originalPingPool(context.Background(), fake)
	require.NoError(t, err)

	originalClosePool(fake)

	require.True(t, fake.pingCalled)
	require.True(t, fake.closeCalled)
}

type fakePoolPinger struct {
	pingCalled  bool
	closeCalled bool
}

func (fake *fakePoolPinger) Ping(ctx context.Context) error {
	fake.pingCalled = true
	return nil
}

func (fake *fakePoolPinger) Close() {
	fake.closeCalled = true
}
