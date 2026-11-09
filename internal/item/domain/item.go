package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status define los estados posibles en el ciclo de vida de un Item.
type Status string

const (
	StatusPending  Status = "PENDING"
	StatusActive   Status = "ACTIVE"
	StatusArchived Status = "ARCHIVED"
)

// IsValid comprueba si el valor del estado corresponde a uno de los permitidos.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusActive, StatusArchived:
		return true
	default:
		return false
	}
}

// CanTransitionTo valida las reglas de transición de estado:
// - PENDING -> ACTIVE o ARCHIVED
// - ACTIVE -> ARCHIVED
// - ARCHIVED es un estado final (no permite transicionar a ningún otro).
func (s Status) CanTransitionTo(target Status) bool {
	if s == target {
		return true
	}

	switch s {
	case StatusPending:
		return target == StatusActive || target == StatusArchived
	case StatusActive:
		return target == StatusArchived
	case StatusArchived:
		return false
	default:
		return false
	}
}

// Item representa la entidad principal del catálogo / inventario.
type Item struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	Status    Status     `json:"status" db:"status"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsDeleted verifica si el ítem fue eliminado lógicamente (soft delete).
func (i *Item) IsDeleted() bool {
	return i.DeletedAt != nil
}
