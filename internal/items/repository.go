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

// List devuelve items paginados. Si nameQuery no está vacío, filtra por name usando ILIKE.
// Nota: ILIKE con %...% puede no usar el índice btree. Para portfolio está perfecto;
// si luego querés optimizar, se puede migrar a trigram (pg_trgm) o búsqueda full-text.
func (repository *Repository) List(context context.Context, nameQuery string, limit, offset int) ([]Item, error) {
	const base = `
		SELECT id, name, description, price::text, stock, created_at, updated_at
		FROM items
	`
	const orderLimit = `
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`

	var rowsQuery string
	var args []any

	if nameQuery == "" {
		rowsQuery = base + orderLimit
		args = []any{limit, offset}
	} else {
		rowsQuery = base + `
			WHERE name ILIKE '%' || $3 || '%'
		` + orderLimit
		args = []any{limit, offset, nameQuery}
	}

	rows, err := repository.database.Query(context, rowsQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Item, 0, limit)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.Price, &it.Stock, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// Count devuelve la cantidad total de items según el filtro nameQuery.
// Se usa para calcular paginación (total pages, etc.).
func (repository *Repository) Count(context context.Context, nameQuery string) (int, error) {
	const base = `SELECT COUNT(*) FROM items`
	var query string
	var args []any

	if nameQuery == "" {
		query = base
		args = nil
	} else {
		query = base + ` WHERE name ILIKE '%' || $1 || '%'`
		args = []any{nameQuery}
	}

	var total int
	if err := repository.database.QueryRow(context, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
