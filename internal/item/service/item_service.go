package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/clock"
)

type itemService struct {
	repo  domain.ItemRepository
	clock clock.Clock
}

// NewItemService crea una nueva instancia del servicio de dominio inyectando el repositorio y el reloj.
func NewItemService(repo domain.ItemRepository, clk clock.Clock) domain.ItemService {
	return &itemService{
		repo:  repo,
		clock: clk,
	}
}

// Create ejecuta el caso de uso CU-01 y CU-02.
func (s *itemService) Create(ctx context.Context, name string) (*domain.Item, error) {
	trimmedName := strings.TrimSpace(name)
	if len(trimmedName) < 3 || len(trimmedName) > 100 {
		return nil, fmt.Errorf("%w: el nombre debe tener entre 3 y 100 caracteres", domain.ErrInvalidInput)
	}

	existing, err := s.repo.GetByName(ctx, trimmedName)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("error al verificar unicidad de nombre: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrAlreadyExists
	}

	now := s.clock.Now()
	newItem := &domain.Item{
		ID:        uuid.New(),
		Name:      trimmedName,
		Status:    domain.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: nil,
	}

	if err := s.repo.Create(ctx, newItem); err != nil {
		return nil, fmt.Errorf("error al persistir ítem: %w", err)
	}

	return newItem, nil
}

// GetByID ejecuta el caso de uso CU-03 (consulta de ítem).
func (s *itemService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al buscar ítem por id: %w", err)
	}

	if item.IsDeleted() {
		return nil, domain.ErrNotFound
	}

	return item, nil
}

// List obtiene ítems paginados con filtro opcional de status.
func (s *itemService) List(ctx context.Context, status *domain.Status, page, limit int) (*domain.PaginatedResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	if status != nil && !status.IsValid() {
		return nil, fmt.Errorf("%w: status inválido", domain.ErrInvalidInput)
	}

	pagination := domain.Pagination{
		Page:  page,
		Limit: limit,
	}

	result, err := s.repo.List(ctx, status, pagination)
	if err != nil {
		return nil, fmt.Errorf("error al listar ítems: %w", err)
	}

	return result, nil
}

// UpdateStatus ejecuta los casos de uso CU-04 y CU-05 (transición de estados).
func (s *itemService) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus domain.Status) (*domain.Item, error) {
	if !newStatus.IsValid() {
		return nil, fmt.Errorf("%w: status '%s' no es válido", domain.ErrInvalidInput, newStatus)
	}

	item, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !item.Status.CanTransitionTo(newStatus) {
		return nil, fmt.Errorf("%w: no se permite transicionar de %s a %s", domain.ErrInvalidStatus, item.Status, newStatus)
	}

	item.Status = newStatus
	item.UpdatedAt = s.clock.Now()

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, fmt.Errorf("error al actualizar status del ítem: %w", err)
	}

	return item, nil
}

// Delete ejecuta el soft delete del ítem.
func (s *itemService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("error al realizar soft delete: %w", err)
	}

	return nil
}
