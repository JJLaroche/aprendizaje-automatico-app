package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	// ErrNotFound indica que el recurso solicitado no existe o fue eliminado lógicamente.
	ErrNotFound = errors.New("recurso no encontrado")

	// ErrAlreadyExists indica conflicto por unicidad (ej. nombre repetido).
	ErrAlreadyExists = errors.New("el recurso ya existe")

	// ErrInvalidStatus indica que la transición de estado solicitada no está permitida por reglas de negocio.
	ErrInvalidStatus = errors.New("transición de estado no permitida")

	// ErrInvalidInput indica error en los datos de entrada que no pasaron validación de negocio.
	ErrInvalidInput = errors.New("datos de entrada inválidos")
)

// Pagination define los parámetros para consultas paginadas.
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// Offset calcula el desplazamiento en SQL según la página y límite.
func (p Pagination) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}

// PaginatedResult envuelve una lista de items con sus metadatos de paginación.
type PaginatedResult struct {
	Items      []Item `json:"items"`
	TotalCount int    `json:"total_count"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
}

// ItemRepository define el contrato para la persistencia de Items.
type ItemRepository interface {
	Create(ctx context.Context, item *Item) error
	GetByID(ctx context.Context, id uuid.UUID) (*Item, error)
	GetByName(ctx context.Context, name string) (*Item, error)
	List(ctx context.Context, status *Status, pagination Pagination) (*PaginatedResult, error)
	Update(ctx context.Context, item *Item) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// ItemService define el contrato de casos de uso de negocio para Items.
type ItemService interface {
	Create(ctx context.Context, name string) (*Item, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Item, error)
	List(ctx context.Context, status *Status, page, limit int) (*PaginatedResult, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, newStatus Status) (*Item, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
