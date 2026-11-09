# ESPECIFICACIÓN TÉCNICA (SPEC TÉCNICO)

## 1. ARQUITECTURA Y ESTRUCTURA DE ARCHIVOS
Se utilizará una arquitectura por capas modularizada por dominio basada en Clean Architecture idiomática para Go (*Standard Go Project Layout*), incorporando un patrón **Registry** con inicialización perezosa / Singleton vía `sync.Once`, abstracción del tiempo (`pkg/clock`) para permitir testing determinista y mockeable, y generación de mocks mediante **Mockery**:

```
.
├── .mockery.yaml                   # Configuración declarativa de Mockery
├── cmd/
│   └── api/
│       └── main.go                 # Punto de entrada, registro explícito de rutas Chi y graceful shutdown
├── internal/
│   ├── item/                       # Módulo de Dominio: Item
│   │   ├── domain/                 # Entidades puras, errores de dominio e interfaces
│   │   │   ├── item.go
│   │   │   └── errors.go
│   │   ├── domain/mocks/           # Mocks autogenerados por Mockery
│   │   │   ├── item_repository.go
│   │   │   └── item_service.go
│   │   ├── handler/                # HTTP Handlers (métodos atómicos sin sub-router interno)
│   │   │   ├── item_handler.go
│   │   │   ├── item_handler_test.go# Tests unitarios de capa HTTP (Table-driven)
│   │   │   ├── request.go
│   │   │   └── response.go
│   │   ├── service/                # Casos de uso de negocio con Clock inyectado
│   │   │   ├── item_service.go
│   │   │   └── item_service_test.go# Tests unitarios de casos de uso (Table-driven)
│   │   ├── repository/             # Implementación de persistencia en SQLite
│   │   │   ├── sqlite_item_repository.go
│   │   │   └── sqlite_item_repository_test.go # Tests de integración/persistencia (Table-driven)
│   │   └── registry/               # Service Registry (Singleton con sync.Once)
│   │       └── registry.go
├── pkg/
│   ├── clock/                      # Abstracción de tiempo para testing y mockeo
│   │   ├── clock.go                # Interfaz Clock e implementación real RealClock
│   │   └── mocks/                  # Mock generado de Clock
│   │       └── clock.go
│   ├── database/                   # Conexión y configuración de SQLite
│   │   └── sqlite.go
│   └── validator/                  # Wrapper y configuración de go-playground/validator
├── migrations/                     # Archivos de migración SQL ordenados cronológicamente
│   └── 000001_create_items_table.up.sql
├── specs/                          # Especificaciones funcionales y técnicas del proyecto
│   ├── funcional.md
│   └── tecnico.md
├── Makefile                        # Comandos de build, run, test, lint y mock-gen
├── go.mod
└── go.sum
```

---

## 2. STACK TÉCNICO Y DEPENDENCIAS
- **Lenguaje:** Go 1.22+
- **HTTP Router:** [`github.com/go-chi/chi/v5`](https://github.com/go-chi/chi/v5)
- **Base de Datos:** SQLite local (sin CGO)
- **Driver de Persistencia:** [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) junto a `database/sql` nativo
- **Validación:** [`github.com/go-playground/validator/v10`](https://github.com/go-playground/validator)
- **Identificadores Únicos (UUID):** [`github.com/google/uuid`](https://github.com/google/uuid)
- **Sincronización y Concurrencia:** `sync.Once` estándar de Go
- **Testing & Aserciones:** `testing` estándar + [`github.com/stretchr/testify`](https://github.com/stretchr/testify)
- **Generación de Mocks:** [`github.com/vektra/mockery/v2`](https://github.com/vektra/mockery)

---

## 3. ABSTRACCIÓN DEL TIEMPO (`pkg/clock`)

Para evitar acoplar la lógica de negocio a llamadas directas no deterministas como `time.Now()`, se implementa una interfaz `Clock` inyectable:

### 3.1. Definición (`pkg/clock/clock.go`)
```go
package clock

import "time"

// Clock provee una abstracción para obtener la hora actual.
type Clock interface {
    Now() time.Time
}

// RealClock implementa Clock utilizando la hora del sistema en UTC.
type RealClock struct{}

func NewRealClock() Clock {
    return &RealClock{}
}

func (r *RealClock) Now() time.Time {
    return time.Now().UTC()
}
```

En pruebas unitarias se inyecta un mock de `Clock` o una implementación fija que retorna un timestamp predeterminado, permitiendo aserciones exactas sobre `created_at`, `updated_at` y `deleted_at`.

---

## 4. GENERACIÓN DE MOCKS CON MOCKERY

Los mocks de interfaces no se escriben manualmente; se gestionan de forma estandarizada y declarativa con **Mockery**.

### 4.1. Archivo de Configuración (`.mockery.yaml`)
```yaml
with-expecter: true
filename: "{{.InterfaceNameSnake}}.go"
dir: "{{.InterfaceDir}}/mocks"
mockname: "Mock{{.InterfaceName}}"
outpkg: "mocks"
packages:
  github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain:
    interfaces:
      ItemRepository:
      ItemService:
  github.com/marielarodriguez/aprendizaje-automatico-app/pkg/clock:
    interfaces:
      Clock:
```

### 4.2. Comando Makefile
```makefile
mocks:
	mockery
```

---

## 5. PATRÓN REGISTRY CON SINGLETON (`sync.Once`)

Cada dominio cuenta con un paquete `registry/` responsable de construir y proveer las dependencias requeridas (Repository, Service, Handler, Clock). La inicialización se realiza una única vez de manera thread-safe mediante `sync.Once`:

```go
package registry

import (
    "database/sql"
    "sync"

    "github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain"
    "github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/handler"
    "github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/repository"
    "github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/service"
    "github.com/marielarodriguez/aprendizaje-automatico-app/pkg/clock"
    "github.com/marielarodriguez/aprendizaje-automatico-app/pkg/validator"
)

type Registry struct {
    Handler    *handler.ItemHandler
    Service    domain.ItemService
    Repository domain.ItemRepository
    Clock      clock.Clock
}

var (
    instance *Registry
    once     sync.Once
)

func GetItemRegistry(db *sql.DB, val *validator.CustomValidator) *Registry {
    once.Do(func() {
        clk := clock.NewRealClock()
        repo := repository.NewSQLiteItemRepository(db, clk)
        svc := service.NewItemService(repo, clk)
        h := handler.NewItemHandler(svc, val)

        instance = &Registry{
            Handler:    h,
            Service:    svc,
            Repository: repo,
            Clock:      clk,
        }
    })
    return instance
}
```

---

## 6. REGISTRO CENTRALIZADO DE RUTAS EN `cmd/api/main.go`

Los handlers **no** definen métodos `Routes()` ni crean sub-enrutadores propios. Las rutas se mapean directamente en `main.go`:

```go
itemRegistry := registry.GetItemRegistry(db, customValidator)

r := chi.NewRouter()

r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(middleware.Timeout(5 * time.Second))

r.Get("/health", healthHandler)

r.Route("/api/v1/items", func(r chi.Router) {
    r.Post("/", itemRegistry.Handler.Create)
    r.Get("/", itemRegistry.Handler.List)
    r.Get("/{id}", itemRegistry.Handler.GetByID)
    r.Patch("/{id}/status", itemRegistry.Handler.UpdateStatus)
    r.Delete("/{id}", itemRegistry.Handler.Delete)
})
```

---

## 7. MODELO DE DATOS Y DEFINICIÓN DE STRUCTS

### 7.1. Dominio (`internal/item/domain/item.go`)
```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type Status string

const (
    StatusPending  Status = "PENDING"
    StatusActive   Status = "ACTIVE"
    StatusArchived Status = "ARCHIVED"
)

type Item struct {
    ID        uuid.UUID  `json:"id" db:"id"`
    Name      string     `json:"name" db:"name"`
    Status    Status     `json:"status" db:"status"`
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
    DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}
```

### 7.2. Errores de Dominio Requeridos (`internal/item/domain/errors.go`)
```go
package domain

import "errors"

var (
    ErrNotFound      = errors.New("recurso no encontrado")
    ErrAlreadyExists = errors.New("el recurso ya existe")
    ErrInvalidStatus = errors.New("transición de estado no permitida")
    ErrInvalidInput  = errors.New("datos de entrada inválidos")
)
```

### 7.3. DTOs de Entrada (`internal/item/handler/request.go`)
```go
type CreateItemRequest struct {
    Name string `json:"name" validate:"required,min=3,max=100"`
}

type UpdateStatusRequest struct {
    Status string `json:"status" validate:"required,oneof=PENDING ACTIVE ARCHIVED"`
}
```

---

## 8. CONTRATO DE ENDPOINTS (API RESTFUL)

### 8.1. `POST /api/v1/items`
- **Request Body:** JSON alineado a `CreateItemRequest`.
- **Criterio de Éxito:** Status `201 Created`. Devuelve `{"data": Item}`.
- **Errores:**
  - Status `400 Bad Request`: JSON malformado o validación de campos fallida.
  - Status `409 Conflict`: Nombre duplicado en ítems no eliminados.
  - Status `500 Internal Server Error`: Fallos no controlados de persistencia.

### 8.2. `GET /api/v1/items/{id}`
- **Path Parameter:** `id` (UUID v4 válido).
- **Criterio de Éxito:** Status `200 OK`. Devuelve `{"data": Item}`.
- **Errores:**
  - Status `400 Bad Request`: Formato de UUID no válido.
  - Status `404 Not Found`: Ítem inexistente o con baja lógica.

### 8.3. `PATCH /api/v1/items/{id}/status`
- **Path Parameter:** `id` (UUID v4).
- **Request Body:** JSON alineado a `UpdateStatusRequest`.
- **Criterio de Éxito:** Status `200 OK`. Devuelve `{"data": Item}` con estado actualizado.
- **Errores:**
  - Status `400 Bad Request`: UUID o body inválido.
  - Status `404 Not Found`: Ítem inexistente o eliminado.
  - Status `422 Unprocessable Entity`: Transición de estado prohibida (ej. `ARCHIVED` -> `PENDING`).

### 8.4. `GET /api/v1/items`
- **Query Parameters:** `page` (int, default 1), `limit` (int, default 10, max 100), `status` (opcional).
- **Criterio de Éxito:** Status `200 OK`. Devuelve arreglo paginado `{"data": PaginatedResult}`.

### 8.5. `DELETE /api/v1/items/{id}`
- **Path Parameter:** `id` (UUID v4).
- **Criterio de Éxito:** Status `204 No Content`.
- **Errores:**
  - Status `404 Not Found`: Ítem no encontrado o previamente eliminado.

---

## 9. ESTÁNDAR DE ERRORES Y MIDDLEWARES

### 9.1. Estructura JSON Unificada de Error
```json
{
  "error": {
    "code": "INVALID_PAYLOAD",
    "message": "Key: 'CreateItemRequest.Name' Error:Field validation for 'Name' failed on the 'min' tag",
    "timestamp": "2026-11-09T11:00:00Z"
  }
}
```

### 9.2. Middlewares en Router Chi
- **Logger:** `log/slog` estructurado JSON.
- **Recoverer:** Captura de panics devolviendo HTTP 500.
- **RequestID:** Inyección y propagación de `X-Request-Id`.
- **Timeout:** Inyección de `context.WithTimeout` (5s).

---

## 10. ESTRATEGIA DE TESTING INTEGRAL POR CAPAS

Se exige cobertura mediante pruebas estructuradas bajo el patrón **Table-Driven Tests** (`tt := range tests`) en las tres capas del dominio:

1. **Capa Handler (`internal/item/handler/item_handler_test.go`):**
   - Utiliza `httptest.NewRecorder()`, `httptest.NewRequest()` y el mock de `domain.ItemService` generado por Mockery.
   - Evalúa parseo de JSON, validaciones de DTO con `go-playground/validator`, respuestas exitosas y mapeo exacto de códigos HTTP (`201`, `200`, `204`, `400`, `404`, `409`, `422`, `500`).
   - Verifica explícitamente el cumplimiento de las expectativas con `mockService.AssertExpectations(t)`.

2. **Capa Service (`internal/item/service/item_service_test.go`):**
   - Utiliza mocks de `domain.ItemRepository` y `clock.Clock` generados por Mockery.
   - Evalúa reglas de negocio, máquina de estados de transición, unicidad, recorte de strings y validaciones de rango.
   - Verifica explícitamente las expectativas con `mockRepo.AssertExpectations(t)` y `mockClock.AssertExpectations(t)`.

3. **Capa Repository (`internal/item/repository/sqlite_item_repository_test.go`):**
   - Utiliza base de datos SQLite en memoria (`file::memory:?cache=shared`) ejecutando las migraciones reales embebidas (`migrations.Files`).
   - Inyecta `clock.Clock` (mock o real) y evalúa persistencia real, filtros de paginación, índice único condicional y soft delete (`deleted_at`).

---

## 11. INSTRUCCIONES DIRECTAS PARA DESARROLLO Y EVALUACIÓN (LLM-AS-A-JUDGE)
1. **Determinismo Temporal:** Prohibido el uso directo de `time.Now()` en lógica de negocio o repositorios; se debe inyectar y utilizar la interfaz `clock.Clock`.
2. **Generación Automatizada de Mocks:** Los mocks de testing deben ser generados exclusivamente con `mockery` basándose en `.mockery.yaml`. Prohibida la creación manual de structs mock en archivos de test.
3. **Table-Driven Tests Obligatorios:** Los tests en todas las capas (`handler`, `service`, `repository`) deben seguir la estructura idiomática de tablas de Go (`tests := []struct{...}`) con ejecución individual vía `t.Run(tc.name, func(t *testing.T) { ... })`.
4. **Aserción de Expectativas:** En todo test que involucre mocks, es obligatorio invocar `AssertExpectations(t)` al finalizar cada caso de prueba.
5. **Separación y Registro Limpio:** Los handlers solo exponen métodos `http.HandlerFunc`. No deben definir sub-routers ni acoplarse a rutas fijas; el registro se realiza exclusivamente en `main.go`.
6. **Uso de Registry y Singleton:** La inicialización de dependencias del dominio debe resolverse a través de `registry.GetItemRegistry()`, garantizando instancia única mediante `sync.Once`.
7. **Manejo Explícito de Errores y Wrapping:** Prohibido el uso de `_` para ignorar errores devueltos. Wrapping obligatorio con `%w`.
8. **Propagación de Contexto:** `ctx context.Context` siempre como primer parámetro en Repository, Service y Handler.
9. **Concurrencia Segura en SQLite:** Modo WAL, busy timeout y `SetMaxOpenConns(1)`.
10. **Graceful Shutdown:** Captura de `os.Interrupt` y `syscall.SIGTERM` con timeout para apagado limpio de conexiones y servidor.
