package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain"
	domainMocks "github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain/mocks"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/service"
	clockMocks "github.com/marielarodriguez/aprendizaje-automatico-app/pkg/clock/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestItemService_Create(t *testing.T) {
	ctx := context.Background()
	fixedTime := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		inputName     string
		setupMocks    func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock)
		expectError   bool
		expectedErr   error
		validateItem  func(t *testing.T, item *domain.Item)
	}{
		{
			name:      "Éxito: Creación de ítem válido",
			inputName: "Teclado Mecánico",
			setupMocks: func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock) {
				repo.EXPECT().GetByName(ctx, "Teclado Mecánico").Return(nil, domain.ErrNotFound).Once()
				clk.EXPECT().Now().Return(fixedTime).Once()
				repo.EXPECT().Create(ctx, mock.MatchedBy(func(i *domain.Item) bool {
					return i.Name == "Teclado Mecánico" &&
						i.Status == domain.StatusPending &&
						i.CreatedAt.Equal(fixedTime) &&
						i.UpdatedAt.Equal(fixedTime) &&
						i.DeletedAt == nil
				})).Return(nil).Once()
			},
			expectError: false,
			validateItem: func(t *testing.T, item *domain.Item) {
				assert.NotNil(t, item)
				assert.Equal(t, "Teclado Mecánico", item.Name)
				assert.Equal(t, domain.StatusPending, item.Status)
				assert.Equal(t, fixedTime, item.CreatedAt)
				assert.NotEqual(t, uuid.Nil, item.ID)
			},
		},
		{
			name:      "Error: Nombre demasiado corto (< 3 caracteres)",
			inputName: "AB",
			setupMocks: func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock) {
				// No debe llamar ni al repo ni al reloj
			},
			expectError: true,
			expectedErr: domain.ErrInvalidInput,
		},
		{
			name:      "Error: Nombre duplicado (conflicto)",
			inputName: "Mouse Inalámbrico",
			setupMocks: func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock) {
				existing := &domain.Item{
					ID:        uuid.New(),
					Name:      "Mouse Inalámbrico",
					Status:    domain.StatusActive,
					CreatedAt: fixedTime,
					UpdatedAt: fixedTime,
				}
				repo.EXPECT().GetByName(ctx, "Mouse Inalámbrico").Return(existing, nil).Once()
			},
			expectError: true,
			expectedErr: domain.ErrAlreadyExists,
		},
		{
			name:      "Error: Fallo al persistir en repositorio",
			inputName: "Monitor UltraWide",
			setupMocks: func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock) {
				repo.EXPECT().GetByName(ctx, "Monitor UltraWide").Return(nil, domain.ErrNotFound).Once()
				clk.EXPECT().Now().Return(fixedTime).Once()
				repo.EXPECT().Create(ctx, mock.Anything).Return(errors.New("db disk failure")).Once()
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := domainMocks.NewMockItemRepository(t)
			mockClock := clockMocks.NewMockClock(t)
			svc := service.NewItemService(mockRepo, mockClock)

			tc.setupMocks(mockRepo, mockClock)

			item, err := svc.Create(ctx, tc.inputName)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErr != nil {
					assert.ErrorIs(t, err, tc.expectedErr)
				}
				assert.Nil(t, item)
			} else {
				assert.NoError(t, err)
				if tc.validateItem != nil {
					tc.validateItem(t, item)
				}
			}

			mockRepo.AssertExpectations(t)
			mockClock.AssertExpectations(t)
		})
	}
}

func TestItemService_GetByID(t *testing.T) {
	ctx := context.Background()
	itemID := uuid.New()
	fixedTime := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		id          uuid.UUID
		setupMocks  func(repo *domainMocks.MockItemRepository)
		expectError bool
		expectedErr error
	}{
		{
			name: "Éxito: Ítem encontrado activo",
			id:   itemID,
			setupMocks: func(repo *domainMocks.MockItemRepository) {
				repo.EXPECT().GetByID(ctx, itemID).Return(&domain.Item{
					ID:        itemID,
					Name:      "Pad Mouse",
					Status:    domain.StatusActive,
					CreatedAt: fixedTime,
					UpdatedAt: fixedTime,
					DeletedAt: nil,
				}, nil).Once()
			},
			expectError: false,
		},
		{
			name: "Error: Ítem inexistente",
			id:   itemID,
			setupMocks: func(repo *domainMocks.MockItemRepository) {
				repo.EXPECT().GetByID(ctx, itemID).Return(nil, domain.ErrNotFound).Once()
			},
			expectError: true,
			expectedErr: domain.ErrNotFound,
		},
		{
			name: "Error: Ítem eliminado lógicamente (soft deleted)",
			id:   itemID,
			setupMocks: func(repo *domainMocks.MockItemRepository) {
				deletedTime := fixedTime.Add(1 * time.Hour)
				repo.EXPECT().GetByID(ctx, itemID).Return(&domain.Item{
					ID:        itemID,
					Name:      "Pad Mouse",
					Status:    domain.StatusArchived,
					CreatedAt: fixedTime,
					UpdatedAt: fixedTime,
					DeletedAt: &deletedTime,
				}, nil).Once()
			},
			expectError: true,
			expectedErr: domain.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := domainMocks.NewMockItemRepository(t)
			mockClock := clockMocks.NewMockClock(t)
			svc := service.NewItemService(mockRepo, mockClock)

			tc.setupMocks(mockRepo)

			item, err := svc.GetByID(ctx, tc.id)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErr != nil {
					assert.ErrorIs(t, err, tc.expectedErr)
				}
				assert.Nil(t, item)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, item)
			}

			mockRepo.AssertExpectations(t)
			mockClock.AssertExpectations(t)
		})
	}
}

func TestItemService_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	itemID := uuid.New()
	fixedUpdateTime := time.Date(2026, 11, 9, 13, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		initialStatus domain.Status
		targetStatus  domain.Status
		setupMocks    func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock)
		expectError   bool
		expectedErr   error
	}{
		{
			name:          "Éxito: Transición PENDING -> ACTIVE",
			initialStatus: domain.StatusPending,
			targetStatus:  domain.StatusActive,
			setupMocks: func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock) {
				repo.EXPECT().GetByID(ctx, itemID).Return(&domain.Item{
					ID:        itemID,
					Name:      "Webcam HD",
					Status:    domain.StatusPending,
					CreatedAt: fixedUpdateTime,
					UpdatedAt: fixedUpdateTime,
				}, nil).Once()
				clk.EXPECT().Now().Return(fixedUpdateTime).Once()
				repo.EXPECT().Update(ctx, mock.MatchedBy(func(i *domain.Item) bool {
					return i.Status == domain.StatusActive && i.UpdatedAt.Equal(fixedUpdateTime)
				})).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name:          "Error: Transición prohibida ARCHIVED -> PENDING",
			initialStatus: domain.StatusArchived,
			targetStatus:  domain.StatusPending,
			setupMocks: func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock) {
				repo.EXPECT().GetByID(ctx, itemID).Return(&domain.Item{
					ID:        itemID,
					Name:      "Webcam HD",
					Status:    domain.StatusArchived,
					CreatedAt: fixedUpdateTime,
					UpdatedAt: fixedUpdateTime,
				}, nil).Once()
			},
			expectError: true,
			expectedErr: domain.ErrInvalidStatus,
		},
		{
			name:          "Error: Status inválido desconocido",
			initialStatus: domain.StatusPending,
			targetStatus:  domain.Status("INVALID_STATUS"),
			setupMocks: func(repo *domainMocks.MockItemRepository, clk *clockMocks.MockClock) {
				// No debe invocar repo
			},
			expectError: true,
			expectedErr: domain.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := domainMocks.NewMockItemRepository(t)
			mockClock := clockMocks.NewMockClock(t)
			svc := service.NewItemService(mockRepo, mockClock)

			tc.setupMocks(mockRepo, mockClock)

			updated, err := svc.UpdateStatus(ctx, itemID, tc.targetStatus)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErr != nil {
					assert.ErrorIs(t, err, tc.expectedErr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.targetStatus, updated.Status)
				assert.Equal(t, fixedUpdateTime, updated.UpdatedAt)
			}

			mockRepo.AssertExpectations(t)
			mockClock.AssertExpectations(t)
		})
	}
}

func TestItemService_Delete(t *testing.T) {
	ctx := context.Background()
	itemID := uuid.New()

	tests := []struct {
		name        string
		setupMocks  func(repo *domainMocks.MockItemRepository)
		expectError bool
		expectedErr error
	}{
		{
			name: "Éxito: Soft delete de ítem existente",
			setupMocks: func(repo *domainMocks.MockItemRepository) {
				repo.EXPECT().GetByID(ctx, itemID).Return(&domain.Item{
					ID:     itemID,
					Name:   "Cámara",
					Status: domain.StatusActive,
				}, nil).Once()
				repo.EXPECT().SoftDelete(ctx, itemID).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name: "Error: Soft delete de ítem no encontrado",
			setupMocks: func(repo *domainMocks.MockItemRepository) {
				repo.EXPECT().GetByID(ctx, itemID).Return(nil, domain.ErrNotFound).Once()
			},
			expectError: true,
			expectedErr: domain.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := domainMocks.NewMockItemRepository(t)
			mockClock := clockMocks.NewMockClock(t)
			svc := service.NewItemService(mockRepo, mockClock)

			tc.setupMocks(mockRepo)

			err := svc.Delete(ctx, itemID)

			if tc.expectError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockClock.AssertExpectations(t)
		})
	}
}
