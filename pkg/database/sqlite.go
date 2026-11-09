package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver (zero CGO)
)

// OpenSQLite valida y abre la conexión con SQLite aplicando parámetros de concurrencia segura.
func OpenSQLite(dbPath string) (*sql.DB, error) {
	trimmed := strings.TrimSpace(dbPath)
	if trimmed == "" {
		return nil, errors.New("la ruta de base de datos no puede estar vacía")
	}

	// Prevenir DSN parameter injection
	if strings.ContainsAny(trimmed, "?#&") {
		return nil, errors.New("la ruta contiene caracteres no permitidos (? # &) que podrían alterar los parámetros DSN")
	}

	cleanPath := filepath.Clean(trimmed)

	// Construcción segura del DSN mediante url.Values
	q := url.Values{}
	q.Set("_journal_mode", "WAL")
	q.Set("_busy_timeout", "5000")

	dsn := fmt.Sprintf("%s?%s", cleanPath, q.Encode())

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("error al abrir conexión sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("error al verificar ping a sqlite: %w", err)
	}

	return db, nil
}

// MigrateFromFS ejecuta los scripts de migración presentes en un fs.FS (ej. embed.FS).
func MigrateFromFS(ctx context.Context, db *sql.DB, fsys fs.FS, filename string) error {
	content, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return fmt.Errorf("error al leer archivo de migración %s: %w", filename, err)
	}

	if _, err := db.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("error al ejecutar migración sqlite: %w", err)
	}
	return nil
}
