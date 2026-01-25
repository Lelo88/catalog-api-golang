package items

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Errores de dominio (no HTTP). El handler los traduce a status codes.
var (
	ErrorInvalidInput  = errors.New("invalid input")
	ErrorDuplicateName = errors.New("duplicate item name")
	ErrorNotFound      = errors.New("item not found")
)

// Service contiene reglas de negocio de items.
type Service struct {
	repository *Repository
}

// NewService crea un service de items.
func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

// Create valida reglas y crea el item en DB.
func (service *Service) Create(context context.Context, itemInput CreateItemInput) (Item, error) {
	// Normalización mínima.
	itemInput.Name = strings.TrimSpace(itemInput.Name)

	// Validaciones de negocio (refuerzan constraints DB).
	if itemInput.Name == "" {
		return Item{}, ErrorInvalidInput
	}
	if strings.TrimSpace(itemInput.Price) == "" {
		return Item{}, ErrorInvalidInput
	}
	if itemInput.Stock < 0 {
		return Item{}, ErrorInvalidInput
	}

	// Delegamos persistencia al repo.
	item, err := service.repository.Insert(context, itemInput)
	if err != nil {
		// Si el repo detecta duplicado, lo exponemos como error de dominio.
		if errors.Is(err, ErrorDuplicateName) {
			return Item{}, ErrorDuplicateName
		}
		return Item{}, err
	}

	return item, nil
}

func (service *Service) List(context context.Context, page, limit int, nameQuery string) ([]Item, int, error) {
	// Validación mínima: paginación no puede ser absurda.
	if page < 1 || limit < 1 {
		return nil, 0, ErrorInvalidInput
	}

	// Normalizamos búsqueda.
	nameQuery = strings.TrimSpace(nameQuery)

	offset := (page - 1) * limit

	items, err := service.repository.List(context, nameQuery, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := service.repository.Count(context, nameQuery)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// Get obtiene un item por ID.
// Nota: el service no valida formato UUID; eso es más de HTTP/entrada (handler).
func (s *Service) Get(ctx context.Context, id string) (Item, error) {
	it, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, ErrorNotFound
		}
		return Item{}, err
	}
	return it, nil
}
