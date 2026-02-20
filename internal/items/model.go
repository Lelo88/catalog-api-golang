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

// UpdateItemInput representa el payload para actualizar un item.
type UpdateItemInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Price       *string `json:"price,omitempty"`
	Stock       *int    `json:"stock,omitempty"`
	// DescriptionPresent indica si el cliente envió el campo "description".
	// No se serializa, solo sirve para diferenciar "no tocar" vs "set null".
	DescriptionPresent bool `json:"-"`
}

