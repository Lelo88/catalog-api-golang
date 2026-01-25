package items

import (
	"context"
	"errors"
	"strings"
)

// Errores de dominio (no HTTP). El handler los traduce a status codes.
var (
	ErrorInvalidInput  = errors.New("invalid input")
	ErrorDuplicateName = errors.New("duplicate item name")
)

// Service contiene reglas de negocio de items.
type Service struct {
	repo *Repository
}

// NewService crea un service de items.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
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
	item, err := service.repo.Insert(context, itemInput)
	if err != nil {
		// Si el repo detecta duplicado, lo exponemos como error de dominio.
		if errors.Is(err, ErrorDuplicateName) {
			return Item{}, ErrorDuplicateName
		}
		return Item{}, err
	}

	return item, nil
}
