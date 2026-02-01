package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type poolPinger interface {
	Ping(ctx context.Context) error
	Close()
}

var (
	newPool  = pgxpool.New
	pingPool = func(ctx context.Context, pool poolPinger) error {
		return pool.Ping(ctx)
	}
	closePool = func(pool poolPinger) {
		pool.Close()
	}
)

// NewPool crea un pool de conexiones a PostgreSQL.
// Se usa un timeout corto para evitar que el arranque quede colgado si la DB no responde.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := newPool(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	// Validación temprana: asegura que la app no arranca "a medias".
	if err := pingPool(ctx, pool); err != nil {
		closePool(pool)
		return nil, err
	}

	return pool, nil
}
