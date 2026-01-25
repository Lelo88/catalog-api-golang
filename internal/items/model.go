package items

import "time"

// Item representa un registro persistido en DB.
// Price se modela como string para evitar errores de precisión con float.
type Item struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Price       string    `json:"price"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateItemInput representa el payload para crear un item.
// Nota: Price es string por precisión (DB: numeric(10,2)).
type CreateItemInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Price       string  `json:"price"`
	Stock       int     `json:"stock"`
}
