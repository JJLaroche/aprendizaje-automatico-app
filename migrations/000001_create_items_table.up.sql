CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('PENDING', 'ACTIVE', 'ARCHIVED')),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL
);

-- Índice único parcial: sólo para ítems activos (no eliminados lógicamente)
CREATE UNIQUE INDEX IF NOT EXISTS idx_items_unique_active_name 
ON items(name) 
WHERE deleted_at IS NULL;

-- Índice para acelerar filtros por status y soft delete
CREATE INDEX IF NOT EXISTS idx_items_status_deleted 
ON items(status, deleted_at);
