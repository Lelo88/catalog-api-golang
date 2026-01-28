package items

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// fakeRepo implementa RepositoryAPI para testing.
type fakeRepo struct {
	insertCalled bool
	updateCalled bool
	listCalled   bool
	countCalled  bool
	getCalled    bool

	insertCreatedInput CreateItemInput
	updateInput        UpdateItemInput
	insertErr          error

	listQuery  string
	listLimit  int
	listOffset int
	listErr    error
	listItems  []Item

	countQuery string
	countErr   error
	countTotal int

	getID   string
	getErr  error
	getItem Item

	updateID   string
	updateErr  error
	updateItem Item

	deleteCalled bool
	deleteID     string
	deleteErr    error
}

// Insert implementa RepositoryAPI.Insert
func (fakerepo *fakeRepo) Insert(ctx context.Context, itemInputCreated CreateItemInput) (Item, error) {
	fakerepo.insertCalled = true
	fakerepo.insertCreatedInput = itemInputCreated
	if fakerepo.insertErr != nil {
		return Item{}, fakerepo.insertErr
	}
	return Item{ID: "x", Name: itemInputCreated.Name, Price: itemInputCreated.Price, Stock: itemInputCreated.Stock}, nil
}

// List implementa RepositoryAPI.List
func (fakerepo *fakeRepo) List(ctx context.Context, query string, limit, offset int) ([]Item, error) {
	fakerepo.listCalled = true
	fakerepo.listQuery = query
	fakerepo.listLimit = limit
	fakerepo.listOffset = offset
	if fakerepo.listErr != nil {
		return nil, fakerepo.listErr
	}
	return fakerepo.listItems, nil
}

// Count implementa RepositoryAPI.Count
func (fakerepo *fakeRepo) Count(ctx context.Context, query string) (int, error) {
	fakerepo.countCalled = true
	fakerepo.countQuery = query
	if fakerepo.countErr != nil {
		return 0, fakerepo.countErr
	}
	return fakerepo.countTotal, nil
}

// GetByID implementa RepositoryAPI.GetByID
func (fakerepo *fakeRepo) GetByID(ctx context.Context, id string) (Item, error) {
	fakerepo.getCalled = true
	fakerepo.getID = id
	if fakerepo.getErr != nil {
		return Item{}, fakerepo.getErr
	}
	return fakerepo.getItem, nil
}

// Update implementa RepositoryAPI.Update
func (fakerepo *fakeRepo) Update(ctx context.Context, id string, in UpdateItemInput) (Item, error) {
	fakerepo.updateCalled = true
	fakerepo.updateInput = in
	fakerepo.updateID = id
	if fakerepo.updateErr != nil {
		return Item{}, fakerepo.updateErr
	}
	if fakerepo.updateItem.ID != "" {
		return fakerepo.updateItem, nil
	}
	return Item{ID: id, Name: "ok", Price: "1.00", Stock: 1}, nil
}

// Delete implementa RepositoryAPI.Delete
func (fakerepo *fakeRepo) Delete(ctx context.Context, id string) error {
	fakerepo.deleteCalled = true
	fakerepo.deleteID = id
	if fakerepo.deleteErr != nil {
		return fakerepo.deleteErr
	}
	return nil
}

// TestService_Create_InvalidInput prueba validaciones de Create
func TestService_Create_InvalidInput(t *testing.T) {
	t.Run("invalid name", func(t *testing.T) {
		repository := &fakeRepo{}
		service := NewService(repository)

		_, err := service.Create(context.Background(), CreateItemInput{
			Name:  "   ",
			Price: "100.00",
			Stock: 1,
		})
		require.ErrorIs(t, err, ErrorInvalidInput)
		require.False(t, repository.insertCalled, "repo.Insert should not be called on invalid input")
	})

	t.Run("price validation", func(t *testing.T) {
		// Casos diseñados para cubrir casos típicas:
		// - regex no matchea
		// - trim
		// - cero (bloqueo rápido)
		// - formatos válidos (siempre que tu regex los permita)
		tests := []struct {
			name    string
			price   string
			wantErr bool
		}{
			// No matchea regex / inválidos obvios
			{"letters", "aaa", true},
			{"mixed", "100a", true},
			{"blank", " ", true},
			{"comma", "10,00", true},
			{"dot-leading", ".50", true},
			{"negative", "-1.00", true},

			// Trim + formato
			{"trimmed valid", " 10.00 ", false},

			// Ceros (deben fallar si exigís > 0)
			{"zero int", "0", true},
			{"zero decimals", "0.00", true},

			// Decimales: ajustá según tu regex actual
			// Si usás `^\d+(\.\d{1,2})?$` estos pasan:
			{"one decimal", "10.5", false},
			{"two decimals", "10.50", false},
			{"int", "10", false},

			// Si cambiás a `^\d+(\.\d{2})?$`, entonces:
			// {"one decimal", "10.5", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repository := &fakeRepo{}
				service := NewService(repository)

				_, err := service.Create(context.Background(), CreateItemInput{
					Name:  "product",
					Price: tt.price,
					Stock: 1,
				})

				if tt.wantErr {
					require.ErrorIs(t, err, ErrorInvalidInput, "price=%q", tt.price)
					require.False(t, repository.insertCalled, "repo.Insert should not be called on invalid input (price=%q)", tt.price)
				} else {
					require.NoError(t, err, "price=%q", tt.price)
					require.True(t, repository.insertCalled, "repo.Insert should be called on valid input (price=%q)", tt.price)
				}
			})
		}
	})

	t.Run("invalid stock", func(t *testing.T) {
		testCases := []int{-1, -2}
		for _, stock := range testCases {
			repository := &fakeRepo{}
			service := NewService(repository)

			_, err := service.Create(context.Background(), CreateItemInput{
				Name:  "product",
				Price: "100",
				Stock: stock,
			})
			require.ErrorIs(t, err, ErrorInvalidInput, "stock=%d", stock)
			require.False(t, repository.insertCalled, "repo.Insert should not be called on invalid input")
		}
	})

	t.Run("insert product", func(t *testing.T) {
		errDB := errors.New("db down")
		tests := []struct {
			name       string
			insertErr  error
			wantErr    error
			expectSame bool
		}{
			{
				name:    "success",
				wantErr: nil,
			},
			{
				name:       "duplicate name error",
				insertErr:  fmt.Errorf("wrapped: %w", ErrorDuplicateName),
				wantErr:    ErrorDuplicateName,
				expectSame: false,
			},
			{
				name:       "unknown repo error",
				insertErr:  errDB,
				wantErr:    errDB,
				expectSame: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repository := &fakeRepo{insertErr: tt.insertErr}
				service := NewService(repository)

				input := CreateItemInput{
					Name:  "  product  ",
					Price: "10.00",
					Stock: 1,
				}

				product, err := service.Create(context.Background(), input)

				if tt.wantErr != nil {
					require.Error(t, err)
					require.ErrorIs(t, err, tt.wantErr)
					if tt.expectSame {
						require.True(t, err == tt.wantErr, "expected same error instance, got %v", err)
					}
					require.True(t, repository.insertCalled, "repo.Insert should be called")
					return
				}

				require.NoError(t, err)
				require.True(t, repository.insertCalled, "repo.Insert should be called")
				require.NotEmpty(t, product.ID, "expected created item, got empty ID")
				require.Equal(t, "product", repository.insertCreatedInput.Name, "expected trimmed name")
			})
		}
	})
}

// TestService_List prueba la lista de productos
func TestService_List(t *testing.T) {
	t.Run("invalid pagination", func(t *testing.T) {
		tests := []struct {
			name  string
			page  int
			limit int
		}{
			{"page zero", 0, 10},
			{"page negative", -1, 10},
			{"limit zero", 1, 0},
			{"limit negative", 1, -10},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repository := &fakeRepo{}
				service := NewService(repository)

				items, total, err := service.List(context.Background(), tt.page, tt.limit, "any")

				require.ErrorIs(t, err, ErrorInvalidInput)
				require.Nil(t, items)
				require.Zero(t, total)
				require.False(t, repository.listCalled, "repo.List should not be called")
				require.False(t, repository.countCalled, "repo.Count should not be called")
			})
		}
	})

	t.Run("list error", func(t *testing.T) {
		repository := &fakeRepo{listErr: errors.New("list failed")}
		service := NewService(repository)

		items, total, err := service.List(context.Background(), 1, 10, "  test  ")

		require.ErrorIs(t, err, repository.listErr)
		require.Nil(t, items)
		require.Zero(t, total)
		require.True(t, repository.listCalled, "repo.List should be called")
		require.False(t, repository.countCalled, "repo.Count should not be called on list error")
	})

	t.Run("count error", func(t *testing.T) {
		repository := &fakeRepo{
			listItems: []Item{{ID: "1", Name: "a"}},
			countErr:  errors.New("count failed"),
		}
		service := NewService(repository)

		items, total, err := service.List(context.Background(), 2, 5, "  test  ")

		require.ErrorIs(t, err, repository.countErr)
		require.Nil(t, items)
		require.Zero(t, total)
		require.True(t, repository.listCalled, "repo.List should be called")
		require.True(t, repository.countCalled, "repo.Count should be called")
		require.Equal(t, "test", repository.listQuery, "expected trimmed query")
		require.Equal(t, 5, repository.listLimit)
		require.Equal(t, 5, repository.listOffset)
		require.Equal(t, "test", repository.countQuery, "expected trimmed query")
	})

	t.Run("success", func(t *testing.T) {
		expectedItems := []Item{
			{ID: "1", Name: "a"},
			{ID: "2", Name: "b"},
		}
		repository := &fakeRepo{
			listItems: expectedItems,
			countTotal: 2,
		}
		service := NewService(repository)

		items, total, err := service.List(context.Background(), 3, 10, "  name  ")

		require.NoError(t, err)
		require.Equal(t, expectedItems, items)
		require.Equal(t, 2, total)
		require.True(t, repository.listCalled, "repo.List should be called")
		require.True(t, repository.countCalled, "repo.Count should be called")
		require.Equal(t, "name", repository.listQuery, "expected trimmed query")
		require.Equal(t, 10, repository.listLimit)
		require.Equal(t, 20, repository.listOffset)
		require.Equal(t, "name", repository.countQuery, "expected trimmed query")
	})
}

func TestService_Get(t *testing.T) {
	t.Run("not found maps to domain error", func(t *testing.T) {
		repository := &fakeRepo{
			getErr: fmt.Errorf("wrapped: %w", pgx.ErrNoRows),
		}
		service := NewService(repository)

		item, err := service.Get(context.Background(), "id-1")

		require.ErrorIs(t, err, ErrorNotFound)
		require.Equal(t, Item{}, item)
		require.True(t, repository.getCalled, "repo.GetByID should be called")
		require.Equal(t, "id-1", repository.getID)
	})

	t.Run("repo error is returned", func(t *testing.T) {
		errorDatabase := errors.New("db failed")
		repository := &fakeRepo{
			getErr: errorDatabase,
		}
		service := NewService(repository)

		item, err := service.Get(context.Background(), "id-2")

		require.ErrorIs(t, err, errorDatabase)
		require.True(t, err == errorDatabase, "expected same error instance")
		require.Equal(t, Item{}, item)
		require.True(t, repository.getCalled, "repo.GetByID should be called")
		require.Equal(t, "id-2", repository.getID)
	})

	t.Run("success", func(t *testing.T) {
		expected := Item{ID: "x", Name: "ok", Price: "1.00", Stock: 2}
		repository := &fakeRepo{
			getItem: expected,
		}
		service := NewService(repository)

		item, err := service.Get(context.Background(), "id-3")

		require.NoError(t, err)
		require.Equal(t, expected, item)
		require.True(t, repository.getCalled, "repo.GetByID should be called")
		require.Equal(t, "id-3", repository.getID)
	})
}

func TestService_Update(t *testing.T) {
	t.Run("requires at least one field", func(t *testing.T) {
		repository := &fakeRepo{}
		service := NewService(repository)

		_, err := service.Update(context.Background(), "id", UpdateItemInput{})
		require.ErrorIs(t, err, ErrorInvalidInput)
		require.False(t, repository.updateCalled, "repo.Update should not be called when payload has no fields")
	})

	t.Run("invalid name empty after trim", func(t *testing.T) {
		repository := &fakeRepo{}
		service := NewService(repository)

		_, err := service.Update(context.Background(), "id", UpdateItemInput{
			Name: stringPointer("   "),
		})
		require.ErrorIs(t, err, ErrorInvalidInput)
		require.False(t, repository.updateCalled, "repo.Update should not be called on invalid input")
	})

	t.Run("invalid price empty after trim", func(t *testing.T) {
		repository := &fakeRepo{}
		service := NewService(repository)

		_, err := service.Update(context.Background(), "id", UpdateItemInput{
			Price: stringPointer("   "),
		})
		require.ErrorIs(t, err, ErrorInvalidInput)
		require.False(t, repository.updateCalled, "repo.Update should not be called on invalid input")
	})

	t.Run("invalid price format", func(t *testing.T) {
		repository := &fakeRepo{}
		service := NewService(repository)

		_, err := service.Update(context.Background(), "id", UpdateItemInput{
			Price: stringPointer("0"),
		})
		require.ErrorIs(t, err, ErrorInvalidInput)
		require.False(t, repository.updateCalled, "repo.Update should not be called on invalid input")
	})

	t.Run("invalid stock negative", func(t *testing.T) {
		repository := &fakeRepo{}
		service := NewService(repository)

		_, err := service.Update(context.Background(), "id", UpdateItemInput{
			Stock: integerPointer(-1),
		})
		require.ErrorIs(t, err, ErrorInvalidInput)
		require.False(t, repository.updateCalled, "repo.Update should not be called on invalid input")
	})

	t.Run("repo not found maps to domain error", func(t *testing.T) {
		repository := &fakeRepo{
			updateErr: fmt.Errorf("wrapped: %w", ErrorNotFound),
		}
		service := NewService(repository)

		_, err := service.Update(context.Background(), "id", UpdateItemInput{
			Name: stringPointer("name"),
		})
		require.ErrorIs(t, err, ErrorNotFound)
		require.True(t, repository.updateCalled, "repo.Update should be called")
	})

	t.Run("repo duplicate name maps to domain error", func(t *testing.T) {
		repository := &fakeRepo{
			updateErr: fmt.Errorf("wrapped: %w", ErrorDuplicateName),
		}
		service := NewService(repository)

		_, err := service.Update(context.Background(), "id", UpdateItemInput{
			Name: stringPointer("name"),
		})
		require.ErrorIs(t, err, ErrorDuplicateName)
		require.True(t, repository.updateCalled, "repo.Update should be called")
	})

	t.Run("repo error is returned", func(t *testing.T) {
		errDB := errors.New("db failed")
		repository := &fakeRepo{
			updateErr: errDB,
		}
		service := NewService(repository)

		_, err := service.Update(context.Background(), "id", UpdateItemInput{
			Name: stringPointer("name"),
		})
		require.ErrorIs(t, err, errDB)
		require.True(t, err == errDB, "expected same error instance")
		require.True(t, repository.updateCalled, "repo.Update should be called")
	})

	t.Run("success trims fields and returns item", func(t *testing.T) {
		expected := Item{ID: "x", Name: "ok", Price: "1.00", Stock: 2}
		repository := &fakeRepo{
			updateItem: expected,
		}
		service := NewService(repository)

		item, err := service.Update(context.Background(), "id", UpdateItemInput{
			Name:  stringPointer("  name  "),
			Price: stringPointer(" 10.00 "),
			Stock: integerPointer(2),
		})
		require.NoError(t, err)
		require.Equal(t, expected, item)
		require.True(t, repository.updateCalled, "repo.Update should be called")
		require.Equal(t, "id", repository.updateID)
		require.NotNil(t, repository.updateInput.Name)
		require.Equal(t, "name", *repository.updateInput.Name)
		require.NotNil(t, repository.updateInput.Price)
		require.Equal(t, "10.00", *repository.updateInput.Price)
		require.NotNil(t, repository.updateInput.Stock)
		require.Equal(t, 2, *repository.updateInput.Stock)
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repository := &fakeRepo{}
		service := NewService(repository)

		err := service.Delete(context.Background(), "id")

		require.NoError(t, err)
		require.True(t, repository.deleteCalled, "repo.Delete should be called")
		require.Equal(t, "id", repository.deleteID)
	})

	t.Run("repo error is returned", func(t *testing.T) {
		errorFromDatabase := errors.New("delete failed")
		repository := &fakeRepo{deleteErr: errorFromDatabase}
		service := NewService(repository)

		err := service.Delete(context.Background(), "id-2")

		require.ErrorIs(t, err, errorFromDatabase)
		require.True(t, err == errorFromDatabase, "expected same error instance")
		require.True(t, repository.deleteCalled, "repo.Delete should be called")
		require.Equal(t, "id-2", repository.deleteID)
	})
}

func stringPointer(value string) *string {
	return &value
}

func integerPointer(value int) *int {
	return &value
}
