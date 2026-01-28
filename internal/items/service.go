package items

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Errores de dominio (no HTTP). El handler los traduce a status codes.
var (
	ErrorInvalidInput  = errors.New("invalid input")
	ErrorDuplicateName = errors.New("duplicate item name")
	ErrorNotFound      = errors.New("item not found")
)

// RepositoryAPI define lo que el service necesita. 
// Permite testear handlers con stubs sin tocar DB.
type RepositoryAPI interface {
	Insert(ctx context.Context, in CreateItemInput) (Item, error)
	List(ctx context.Context, query string, limit, offset int) ([]Item, error)
	Count(ctx context.Context, query string) (int, error)
	GetByID(ctx context.Context, id string) (Item, error)
	Update(ctx context.Context, id string, in UpdateItemInput) (Item, error)
	Delete(ctx context.Context, id string) error
}

// Service contiene reglas de negocio de items.
type Service struct {
	repository RepositoryAPI
}

// NewService crea un service de items.
func NewService(repository RepositoryAPI) *Service {
	return &Service{repository: repository}
}

// Create valida reglas y crea el item en DB.
func (service *Service) Create(context context.Context, itemInput CreateItemInput) (Item, error) {
	// Normalización mínima.
	itemInput.Name = strings.TrimSpace(itemInput.Name)
	itemInput.Price = strings.TrimSpace(itemInput.Price)

	// Validaciones de negocio (refuerzan constraints DB).
	if itemInput.Name == "" {
		return Item{}, ErrorInvalidInput
	}
	if itemInput.Price == "" {
		return Item{}, ErrorInvalidInput
	}
	if !isValidPrice(itemInput.Price) {
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
func (service *Service) Get(ctx context.Context, id string) (Item, error) {
	it, err := service.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, ErrorNotFound
		}
		return Item{}, err
	}
	return it, nil
}

// Update valida reglas y actualiza parcialmente un item.
// No valida UUID, eso es responsabilidad del handler (capa HTTP).
func (service *Service) Update(context context.Context, id string, itemInputUpdated UpdateItemInput) (Item, error) {
	// Debe venir al menos un campo.
	if itemInputUpdated.Name == nil && itemInputUpdated.Description == nil && itemInputUpdated.Price == nil && itemInputUpdated.Stock == nil {
		return Item{}, ErrorInvalidInput
	}

	// Validaciones de negocio (mínimas).
	if itemInputUpdated.Name != nil {
		name := strings.TrimSpace(*itemInputUpdated.Name)
		if name == "" {
			return Item{}, ErrorInvalidInput
		}
		itemInputUpdated.Name = &name
	}

	if itemInputUpdated.Price != nil {
		price := strings.TrimSpace(*itemInputUpdated.Price)
		if price == "" {
			return Item{}, ErrorInvalidInput
		}
		if !isValidPrice(price) {
			return Item{}, ErrorInvalidInput
		}
		itemInputUpdated.Price = &price
	}

	if itemInputUpdated.Stock != nil && *itemInputUpdated.Stock < 0 {
		return Item{}, ErrorInvalidInput
	}

	item, err := service.repository.Update(context, id, itemInputUpdated)
	if err != nil {
		switch {
		case errors.Is(err, ErrorNotFound):
			return Item{}, ErrorNotFound
		case errors.Is(err, ErrorDuplicateName):
			return Item{}, ErrorDuplicateName
		default:
			return Item{}, err
		}
	}

	return item, nil
}

// Delete elimina un item por ID.
func (service *Service) Delete(context context.Context, id string) error {
	return service.repository.Delete(context, id)
}

var pricePattern = regexp.MustCompile(`^\d+(\.\d{1,2})?$`)

func isValidPrice(value string) bool {
	price := strings.TrimSpace(value)
	if !pricePattern.MatchString(price) {
		return false
	}
	return isPositiveNonZero(price)
}

func isPositiveNonZero(price string) bool {
	// price ya viene validado con regex: \d+(\.\d{2})?
	// Entonces los únicos ceros posibles son "0" o "0.00" o "00.00" etc.
	for _, ch := range price {
		if ch >= '1' && ch <= '9' {
			return true
		}
	}
	return false
}
