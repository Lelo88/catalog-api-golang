package items

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository accede a la tabla items.
// Contiene SQL y mapeo DB → modelo.
type Repository struct {
	database *pgxpool.Pool
}

// NewRepository crea un repositorio de items.
func NewRepository(database *pgxpool.Pool) *Repository {
	return &Repository{database: database}
}

// Insert crea un item y devuelve el registro persistido.
// Usamos RETURNING para obtener id y timestamps generados por DB.
func (repository *Repository) Insert(ctx context.Context, input CreateItemInput) (Item, error) {
	const query = `
		INSERT INTO items (name, description, price, stock)
		VALUES ($1, $2, $3::numeric, $4)
		RETURNING id, name, description, price::text, stock, created_at, updated_at;
	`

	var item Item
	err := repository.database.QueryRow(ctx, query, input.Name, input.Description, input.Price, input.Stock).
		Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		// Detectar conflicto por índice unique (ux_items_name).
		// Postgres: unique_violation = 23505
		var postgressError *pgconn.PgError
		if errors.As(err, &postgressError) && postgressError.Code == "23505" {
			return Item{}, ErrorDuplicateName
		}
		return Item{}, err
	}

	return item, nil
}
