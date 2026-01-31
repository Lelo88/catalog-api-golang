package items

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestRepository_Insert(t *testing.T) {
	t.Run("success with description", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		description := "High-end phone"
		input := CreateItemInput{
			Name:        "Phone X",
			Description: &description,
			Price:       "10.50",
			Stock:       3,
		}

		createdAt := time.Now().Add(-time.Minute)
		updatedAt := time.Now()
		expected := Item{
			ID:          "id-1",
			Name:        input.Name,
			Description: &description,
			Price:       input.Price,
			Stock:       input.Stock,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{values: []any{expected.ID, expected.Name, description, expected.Price, expected.Stock, expected.CreatedAt, expected.UpdatedAt}}
		}

		item, err := repository.Insert(context.Background(), input)

		require.NoError(t, err)
		require.Equal(t, expected, item)
		require.True(t, database.queryRowCalled)
		require.Contains(t, database.lastQuery, "INSERT INTO items")
		require.Equal(t, []any{input.Name, input.Description, input.Price, input.Stock}, database.lastArgs)
	})

	t.Run("success without description", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		input := CreateItemInput{
			Name:        "Keyboard",
			Description: nil,
			Price:       "20.00",
			Stock:       5,
		}

		createdAt := time.Now().Add(-2 * time.Minute)
		updatedAt := time.Now().Add(-time.Minute)
		expected := Item{
			ID:          "id-2",
			Name:        input.Name,
			Description: nil,
			Price:       input.Price,
			Stock:       input.Stock,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{values: []any{expected.ID, expected.Name, nil, expected.Price, expected.Stock, expected.CreatedAt, expected.UpdatedAt}}
		}

		item, err := repository.Insert(context.Background(), input)

		require.NoError(t, err)
		require.Equal(t, expected, item)
		require.True(t, database.queryRowCalled)
		require.Equal(t, []any{input.Name, input.Description, input.Price, input.Stock}, database.lastArgs)
	})

	t.Run("duplicate name returns domain error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: &pgconn.PgError{Code: "23505"}}
		}

		_, err := repository.Insert(context.Background(), CreateItemInput{
			Name:  "Repeated",
			Price: "15.00",
			Stock: 1,
		})

		require.ErrorIs(t, err, ErrorDuplicateName)
		require.True(t, database.queryRowCalled)
	})

	t.Run("other database errors are returned", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		dbErr := errors.New("db down")
		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: dbErr}
		}

		_, err := repository.Insert(context.Background(), CreateItemInput{
			Name:  "Invalid stock",
			Price: "5.00",
			Stock: -1,
		})

		require.ErrorIs(t, err, dbErr)
		require.True(t, err == dbErr, "expected same error instance")
	})
}

func TestRepository_List(t *testing.T) {
	t.Run("without query", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		createdAt := time.Now().Add(-time.Hour)
		updatedAt := time.Now().Add(-time.Minute)

		rows := &fakeRows{rows: [][]any{
			{"id-1", "Phone", "desc", "10.00", 1, createdAt, updatedAt},
			{"id-2", "Mouse", nil, "5.00", 2, createdAt, updatedAt},
		}}
		database.queryFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return rows, nil
		}

		items, err := repository.List(context.Background(), "", 10, 20)

		require.NoError(t, err)
		require.Len(t, items, 2)
		require.Equal(t, "id-1", items[0].ID)
		require.Equal(t, "id-2", items[1].ID)
		require.True(t, database.queryCalled)
		require.NotContains(t, database.lastQuery, "ILIKE")
		require.Equal(t, []any{10, 20}, database.lastArgs)
	})

	t.Run("with query", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		createdAt := time.Now().Add(-2 * time.Hour)
		updatedAt := time.Now().Add(-time.Minute)
		rows := &fakeRows{rows: [][]any{
			{"id-3", "Phone", "desc", "12.00", 3, createdAt, updatedAt},
		}}

		database.queryFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return rows, nil
		}

		items, err := repository.List(context.Background(), "phone", 5, 0)

		require.NoError(t, err)
		require.Len(t, items, 1)
		require.True(t, database.queryCalled)
		require.Contains(t, database.lastQuery, "ILIKE")
		require.Equal(t, []any{5, 0, "phone"}, database.lastArgs)
	})

	t.Run("query error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		queryErr := errors.New("query failed")
		database.queryFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, queryErr
		}

		items, err := repository.List(context.Background(), "", 1, 0)

		require.ErrorIs(t, err, queryErr)
		require.Nil(t, items)
	})

	t.Run("scan error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		rows := &fakeRows{rows: [][]any{{"id", "name", nil, "1.00", 1, time.Now(), time.Now()}}, scanErr: errors.New("scan")}
		database.queryFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return rows, nil
		}

		items, err := repository.List(context.Background(), "", 1, 0)

		require.Error(t, err)
		require.Nil(t, items)
	})

	t.Run("rows error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		rows := &fakeRows{rows: [][]any{}, err: errors.New("rows error")}
		database.queryFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return rows, nil
		}

		items, err := repository.List(context.Background(), "", 1, 0)

		require.Error(t, err)
		require.Nil(t, items)
	})
}

func TestRepository_Count(t *testing.T) {
	t.Run("without query", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{values: []any{5}}
		}

		count, err := repository.Count(context.Background(), "")

		require.NoError(t, err)
		require.Equal(t, 5, count)
		require.Equal(t, []any(nil), database.lastArgs)
	})

	t.Run("with query", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{values: []any{2}}
		}

		count, err := repository.Count(context.Background(), "phone")

		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, []any{"phone"}, database.lastArgs)
		require.Contains(t, database.lastQuery, "ILIKE")
	})

	t.Run("query row error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		queryErr := errors.New("query failed")
		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: queryErr}
		}

		count, err := repository.Count(context.Background(), "")

		require.ErrorIs(t, err, queryErr)
		require.Zero(t, count)
	})
}

func TestRepository_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		createdAt := time.Now().Add(-time.Minute)
		updatedAt := time.Now()
		expected := Item{
			ID:          "id-10",
			Name:        "Phone",
			Description: stringPointer("desc"),
			Price:       "10.00",
			Stock:       2,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{values: []any{expected.ID, expected.Name, "desc", expected.Price, expected.Stock, expected.CreatedAt, expected.UpdatedAt}}
		}

		item, err := repository.GetByID(context.Background(), "id-10")

		require.NoError(t, err)
		require.Equal(t, expected, item)
		require.Equal(t, []any{"id-10"}, database.lastArgs)
	})

	t.Run("query error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		dbErr := errors.New("query failed")
		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: dbErr}
		}

		item, err := repository.GetByID(context.Background(), "id-11")

		require.ErrorIs(t, err, dbErr)
		require.Equal(t, Item{}, item)
	})
}

func TestRepository_Update(t *testing.T) {
	t.Run("requires at least one field", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		_, err := repository.Update(context.Background(), "id", UpdateItemInput{})

		require.ErrorIs(t, err, ErrorInvalidInput)
		require.False(t, database.queryRowCalled)
	})

	t.Run("success with all fields", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		createdAt := time.Now().Add(-time.Hour)
		updatedAt := time.Now()
		description := "updated"
		expected := Item{
			ID:          "id-20",
			Name:        "New",
			Description: &description,
			Price:       "12.00",
			Stock:       5,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{values: []any{expected.ID, expected.Name, description, expected.Price, expected.Stock, expected.CreatedAt, expected.UpdatedAt}}
		}

		name := "New"
		price := "12.00"
		stock := 5
		item, err := repository.Update(context.Background(), "id-20", UpdateItemInput{
			Name:               &name,
			Description:        &description,
			DescriptionPresent: true,
			Price:              &price,
			Stock:              &stock,
		})

		require.NoError(t, err)
		require.Equal(t, expected, item)
		require.True(t, database.queryRowCalled)
		require.Contains(t, database.lastQuery, "UPDATE items")
		require.Equal(t, "id-20", database.lastArgs[len(database.lastArgs)-1])
		require.Len(t, database.lastArgs, 5)
	})

	t.Run("success with description null", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		createdAt := time.Now().Add(-time.Hour)
		updatedAt := time.Now()
		expected := Item{
			ID:          "id-21",
			Name:        "Name",
			Description: nil,
			Price:       "9.00",
			Stock:       1,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{values: []any{expected.ID, expected.Name, nil, expected.Price, expected.Stock, expected.CreatedAt, expected.UpdatedAt}}
		}

		price := "9.00"
		item, err := repository.Update(context.Background(), "id-21", UpdateItemInput{
			DescriptionPresent: true,
			Description:        nil,
			Price:              &price,
		})

		require.NoError(t, err)
		require.Equal(t, expected, item)
		require.Contains(t, normalizeSQL(database.lastQuery), "description = NULL")
		require.Equal(t, "id-21", database.lastArgs[len(database.lastArgs)-1])
		require.Len(t, database.lastArgs, 2)
	})

	t.Run("not found maps to domain error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: pgx.ErrNoRows}
		}

		_, err := repository.Update(context.Background(), "id-22", UpdateItemInput{
			Name: stringPointer("Name"),
		})

		require.ErrorIs(t, err, ErrorNotFound)
	})

	t.Run("duplicate name maps to domain error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: &pgconn.PgError{Code: "23505"}}
		}

		_, err := repository.Update(context.Background(), "id-23", UpdateItemInput{
			Name: stringPointer("Name"),
		})

		require.ErrorIs(t, err, ErrorDuplicateName)
	})

	t.Run("other error is returned", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		dbErr := errors.New("db failed")
		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: dbErr}
		}

		_, err := repository.Update(context.Background(), "id-24", UpdateItemInput{
			Name: stringPointer("Name"),
		})

		require.ErrorIs(t, err, dbErr)
		require.True(t, err == dbErr, "expected same error instance")
	})
}

func TestRepository_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{values: []any{"id-30"}}
		}

		err := repository.Delete(context.Background(), "id-30")

		require.NoError(t, err)
		require.Equal(t, []any{"id-30"}, database.lastArgs)
	})

	t.Run("not found maps to domain error", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: pgx.ErrNoRows}
		}

		err := repository.Delete(context.Background(), "id-31")

		require.ErrorIs(t, err, ErrorNotFound)
	})

	t.Run("other error is returned", func(t *testing.T) {
		database := &fakeDB{}
		repository := NewRepository(database)

		dbErr := errors.New("db failed")
		database.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &fakeRow{err: dbErr}
		}

		err := repository.Delete(context.Background(), "id-32")

		require.ErrorIs(t, err, dbErr)
		require.True(t, err == dbErr, "expected same error instance")
	})
}

type fakeDB struct {
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)

	lastQuery      string
	lastArgs       []any
	queryRowCalled bool
	queryCalled    bool
}

func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	db.queryRowCalled = true
	db.lastQuery = sql
	db.lastArgs = args
	if db.queryRowFn == nil {
		return &fakeRow{err: errors.New("unexpected QueryRow call")}
	}
	return db.queryRowFn(ctx, sql, args...)
}

func (db *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	db.queryCalled = true
	db.lastQuery = sql
	db.lastArgs = args
	if db.queryFn == nil {
		return nil, errors.New("unexpected Query call")
	}
	return db.queryFn(ctx, sql, args...)
}

type fakeRow struct {
	values []any
	err    error
}

func (row *fakeRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	return assignValues(dest, row.values)
}

type fakeRows struct {
	rows    [][]any
	idx     int
	closed  bool
	err     error
	scanErr error
}

func (rows *fakeRows) Close() {
	rows.closed = true
}

func (rows *fakeRows) Err() error {
	return rows.err
}

func (rows *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (rows *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (rows *fakeRows) Next() bool {
	if rows.closed {
		return false
	}
	if rows.idx >= len(rows.rows) {
		rows.closed = true
		return false
	}
	rows.idx++
	return true
}

func (rows *fakeRows) Scan(dest ...any) error {
	if rows.scanErr != nil {
		return rows.scanErr
	}
	if rows.idx == 0 || rows.idx > len(rows.rows) {
		return errors.New("scan called without next")
	}
	return assignValues(dest, rows.rows[rows.idx-1])
}

func (rows *fakeRows) Values() ([]any, error) {
	return nil, errors.New("not implemented")
}

func (rows *fakeRows) RawValues() [][]byte {
	return nil
}

func (rows *fakeRows) Conn() *pgx.Conn {
	return nil
}

func assignValues(dest []any, values []any) error {
	if len(dest) != len(values) {
		return fmt.Errorf("dest len %d does not match values len %d", len(dest), len(values))
	}
	for i, d := range dest {
		if d == nil {
			continue
		}
		if err := assignValue(d, values[i]); err != nil {
			return err
		}
	}
	return nil
}

func assignValue(dest any, value any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest is not pointer")
	}
	if value == nil {
		destValue.Elem().Set(reflect.Zero(destValue.Elem().Type()))
		return nil
	}
	valueValue := reflect.ValueOf(value)
	destElem := destValue.Elem()
	if destElem.Kind() == reflect.Ptr {
		ptrValue := reflect.New(destElem.Type().Elem())
		ptrValue.Elem().Set(valueValue.Convert(destElem.Type().Elem()))
		destElem.Set(ptrValue)
		return nil
	}
	destElem.Set(valueValue.Convert(destElem.Type()))
	return nil
}

func normalizeSQL(query string) string {
	return strings.Join(strings.Fields(query), " ")
}
