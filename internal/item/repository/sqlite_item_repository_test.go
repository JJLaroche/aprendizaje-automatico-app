package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/repository"
	"github.com/marielarodriguez/aprendizaje-automatico-app/migrations"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/clock"
	clockMocks "github.com/marielarodriguez/aprendizaje-automatico-app/pkg/clock/mocks"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// setupTestDB inicializa una base SQLite en memoria con el esquema de migraciones aplicado
func setupTestDB(t *testing.T) *sql.DB {
	// Base SQLite en memoria única por test
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = database.MigrateFromFS(ctx, db, migrations.Files, "000001_create_items_table.up.sql")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestSQLiteItemRepository_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	mockClock := clockMocks.NewMockClock(t)
	repo := repository.NewSQLiteItemRepository(db, mockClock)
	ctx := context.Background()

	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	existingID := uuid.New()

	// Pre-insertar un item para el test de unicidad
	preItem := &domain.Item{
		ID:        existingID,
		Name:      "Mouse Gamer",
		Status:    domain.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repo.Create(ctx, preItem))

	tests := []struct {
		name        string
		itemToSave  *domain.Item
		expectError bool
		expectedErr error
	}{
		{
			name: "Éxito: Inserta un nuevo ítem único",
			itemToSave: &domain.Item{
				ID:        uuid.New(),
				Name:      "Teclado Inalámbrico",
				Status:    domain.StatusPending,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectError: false,
		},
		{
			name: "Error: Conflicto de unicidad con ítem no eliminado",
			itemToSave: &domain.Item{
				ID:        uuid.New(),
				Name:      "Mouse Gamer",
				Status:    domain.StatusPending,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectError: true,
			expectedErr: domain.ErrAlreadyExists,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.Create(ctx, tc.itemToSave)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErr != nil {
					assert.ErrorIs(t, err, tc.expectedErr)
				}
			} else {
				assert.NoError(t, err)

				// Verificar lectura por ID
				fetched, err := repo.GetByID(ctx, tc.itemToSave.ID)
				assert.NoError(t, err)
				assert.NotNil(t, fetched)
				assert.Equal(t, tc.itemToSave.Name, fetched.Name)
				assert.Equal(t, tc.itemToSave.Status, fetched.Status)
			}

			mockClock.AssertExpectations(t)
		})
	}
}

func TestSQLiteItemRepository_ListAndFilter(t *testing.T) {
	db := setupTestDB(t)
	realClock := clock.NewRealClock()
	repo := repository.NewSQLiteItemRepository(db, realClock)
	ctx := context.Background()

	now := time.Now().UTC()

	// Cargar dataset de prueba
	items := []*domain.Item{
		{ID: uuid.New(), Name: "Item A", Status: domain.StatusPending, CreatedAt: now.Add(-3 * time.Minute), UpdatedAt: now},
		{ID: uuid.New(), Name: "Item B", Status: domain.StatusActive, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now},
		{ID: uuid.New(), Name: "Item C", Status: domain.StatusActive, CreatedAt: now.Add(-1 * time.Minute), UpdatedAt: now},
	}
	for _, it := range items {
		require.NoError(t, repo.Create(ctx, it))
	}

	activeStatus := domain.StatusActive
	pendingStatus := domain.StatusPending

	tests := []struct {
		name          string
		statusFilter  *domain.Status
		pagination    domain.Pagination
		expectedCount int
		expectedTotal int
	}{
		{
			name:          "Listar todos sin filtro de status",
			statusFilter:  nil,
			pagination:    domain.Pagination{Page: 1, Limit: 10},
			expectedCount: 3,
			expectedTotal: 3,
		},
		{
			name:          "Listar solo ítems con status ACTIVE",
			statusFilter:  &activeStatus,
			pagination:    domain.Pagination{Page: 1, Limit: 10},
			expectedCount: 2,
			expectedTotal: 2,
		},
		{
			name:          "Listar solo ítems con status PENDING",
			statusFilter:  &pendingStatus,
			pagination:    domain.Pagination{Page: 1, Limit: 10},
			expectedCount: 1,
			expectedTotal: 1,
		},
		{
			name:          "Paginación con limit 1",
			statusFilter:  nil,
			pagination:    domain.Pagination{Page: 1, Limit: 1},
			expectedCount: 1,
			expectedTotal: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := repo.List(ctx, tc.statusFilter, tc.pagination)
			assert.NoError(t, err)
			assert.NotNil(t, res)
			assert.Equal(t, tc.expectedTotal, res.TotalCount)
			assert.Len(t, res.Items, tc.expectedCount)
		})
	}
}

func TestSQLiteItemRepository_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	mockClock := clockMocks.NewMockClock(t)
	repo := repository.NewSQLiteItemRepository(db, mockClock)
	ctx := context.Background()

	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	deleteTime := now.Add(2 * time.Hour)

	item := &domain.Item{
		ID:        uuid.New(),
		Name:      "Monitor Curvo",
		Status:    domain.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, repo.Create(ctx, item))

	tests := []struct {
		name        string
		targetID    uuid.UUID
		setupClock  func()
		expectError bool
		expectedErr error
	}{
		{
			name:     "Éxito: Soft delete marca fecha y excluye de GetByID",
			targetID: item.ID,
			setupClock: func() {
				mockClock.EXPECT().Now().Return(deleteTime).Once()
			},
			expectError: false,
		},
		{
			name:     "Error: Soft delete sobre ítem inexistente",
			targetID: uuid.New(),
			setupClock: func() {
				mockClock.EXPECT().Now().Return(deleteTime).Once()
			},
			expectError: true,
			expectedErr: domain.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupClock()

			err := repo.SoftDelete(ctx, tc.targetID)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErr != nil {
					assert.ErrorIs(t, err, tc.expectedErr)
				}
			} else {
				assert.NoError(t, err)

				// Comprobar que GetByID ya no lo encuentra
				_, err = repo.GetByID(ctx, tc.targetID)
				assert.ErrorIs(t, err, domain.ErrNotFound)
			}

			mockClock.AssertExpectations(t)
		})
	}
}
