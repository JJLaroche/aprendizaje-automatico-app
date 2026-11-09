package migrations

import "embed"

// Files contiene los scripts de migración SQL embebidos en el binario.
//
//go:embed *.sql
var Files embed.FS
