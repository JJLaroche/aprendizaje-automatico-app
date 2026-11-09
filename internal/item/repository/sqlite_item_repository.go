package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/clock"
)

type sqliteItemRepository struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSQLiteItemRepository inicializa una nueva instancia del repositorio SQLite para Items.
func NewSQLiteItemRepository(db *sql.DB, clk clock.Clock) domain.ItemRepository {
	return &sqliteItemRepository{
		db:    db,
		clock: clk,
	}
}

func (r *sqliteItemRepository) Create(ctx context.Context, item *domain.Item) error {
	query := `
		INSERT INTO items (id, name, status, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		item.ID.String(),
		item.Name,
		string(item.Status),
		item.CreatedAt.Format(time.RFC3339Nano),
		item.UpdatedAt.Format(time.RFC3339Nano),
		nil,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("error al insertar ítem en sqlite: %w", err)
	}

	return nil
}

func (r *sqliteItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	query := `
		SELECT id, name, status, created_at, updated_at, deleted_at
		FROM items
		WHERE id = ? AND deleted_at IS NULL
	`
	row := r.db.QueryRowContext(ctx, query, id.String())

	var (
		idStr        string
		name         string
		statusStr    string
		createdAtStr string
		updatedAtStr string
		deletedAtStr sql.NullString
	)

	err := row.Scan(&idStr, &name, &statusStr, &createdAtStr, &updatedAtStr, &deletedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al consultar ítem por id: %w", err)
	}

	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("id inválido en base de datos: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, createdAtStr)
	updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)

	var deletedAt *time.Time
	if deletedAtStr.Valid {
		parsedDeletedAt, err := time.Parse(time.RFC3339Nano, deletedAtStr.String)
		if err == nil {
			deletedAt = &parsedDeletedAt
		}
	}

	return &domain.Item{
		ID:        parsedID,
		Name:      name,
		Status:    domain.Status(statusStr),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}, nil
}

func (r *sqliteItemRepository) GetByName(ctx context.Context, name string) (*domain.Item, error) {
	query := `
		SELECT id, name, status, created_at, updated_at, deleted_at
		FROM items
		WHERE name = ? AND deleted_at IS NULL
	`
	row := r.db.QueryRowContext(ctx, query, name)

	var (
		idStr        string
		scannedName  string
		statusStr    string
		createdAtStr string
		updatedAtStr string
		deletedAtStr sql.NullString
	)

	err := row.Scan(&idStr, &scannedName, &statusStr, &createdAtStr, &updatedAtStr, &deletedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al consultar ítem por nombre: %w", err)
	}

	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("id inválido en base de datos: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, createdAtStr)
	updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)

	return &domain.Item{
		ID:        parsedID,
		Name:      scannedName,
		Status:    domain.Status(statusStr),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: nil,
	}, nil
}

func (r *sqliteItemRepository) List(ctx context.Context, status *domain.Status, pagination domain.Pagination) (*domain.PaginatedResult, error) {
	countQuery := `
		SELECT COUNT(*) 
		FROM items 
		WHERE deleted_at IS NULL
	`

	listQuery := `
		SELECT id, name, status, created_at, updated_at 
		FROM items 
		WHERE deleted_at IS NULL
	`

	var args []interface{}
	if status != nil {
		countQuery += " AND status = ?"
		listQuery += " AND status = ?"
		args = append(args, string(*status))
	}

	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("error al contar items: %w", err)
	}

	listQuery += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	listArgs := append(args, pagination.Limit, pagination.Offset())

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("error al listar items: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var items []domain.Item
	for rows.Next() {
		var (
			idStr        string
			name         string
			statusStr    string
			createdAtStr string
			updatedAtStr string
		)
		if err := rows.Scan(&idStr, &name, &statusStr, &createdAtStr, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("error al escanear fila de item: %w", err)
		}

		parsedID, _ := uuid.Parse(idStr)
		createdAt, _ := time.Parse(time.RFC3339Nano, createdAtStr)
		updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)

		items = append(items, domain.Item{
			ID:        parsedID,
			Name:      name,
			Status:    domain.Status(statusStr),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			DeletedAt: nil,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error durante la iteración de filas: %w", err)
	}

	return &domain.PaginatedResult{
		Items:      items,
		TotalCount: totalCount,
		Page:       pagination.Page,
		Limit:      pagination.Limit,
	}, nil
}

func (r *sqliteItemRepository) Update(ctx context.Context, item *domain.Item) error {
	query := `
		UPDATE items
		SET status = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(
		ctx,
		query,
		string(item.Status),
		item.UpdatedAt.Format(time.RFC3339Nano),
		item.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("error al actualizar ítem en sqlite: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error al verificar filas afectadas: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *sqliteItemRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := r.clock.Now().Format(time.RFC3339Nano)
	query := `
		UPDATE items
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, query, now, now, id.String())
	if err != nil {
		return fmt.Errorf("error al ejecutar soft delete en sqlite: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error al verificar filas afectadas: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
