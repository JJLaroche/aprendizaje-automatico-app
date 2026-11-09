# Items CRUD API (Backend en Go)

Aplicación backend desarrollada en **Go (Golang)** siguiendo **Clean Architecture** idiomática (*Standard Go Project Layout*), estructurada por dominios modulares con patrón **Registry (Singleton con `sync.Once`)** y registro centralizado de rutas.

Sirve como base de referencia para el curso de Aprendizaje Automático para la apertura de Pull Requests y su evaluación automática mediante un sistema **LLM-as-a-Judge**.

---

## 🚀 Arquitectura y Tecnologías
- **Lenguaje:** Go 1.22+
- **Router HTTP:** [Chi v5](https://github.com/go-chi/chi)
- **Base de Datos:** SQLite local con driver 100% puro en Go [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) (Zero CGO)
- **Patrón de Inyección:** Registry por dominio con Singleton (`sync.Once`)
- **Validación:** [go-playground/validator/v10](https://github.com/go-playground/validator)
- **Identificadores:** UUID v4 con [google/uuid](https://github.com/google/uuid)
- **Testing:** Unit testing nativo + [testify](https://github.com/stretchr/testify)

---

## 📁 Estructura del Proyecto
```
.
├── cmd/
│   └── api/
│       └── main.go                 # Punto de entrada, registro centralizado de rutas y graceful shutdown
├── internal/
│   └── item/                       # Módulo del dominio Item
│       ├── domain/                 # Entidades puras, reglas de transición de estado, interfaces y errores
│       │   ├── item.go
│       │   └── errors.go
│       ├── handler/                # HTTP Handlers (métodos atómicos sin sub-routers internos)
│       │   ├── item_handler.go
│       │   ├── request.go
│       │   └── response.go
│       ├── service/                # Casos de uso de negocio y orquestación
│       │   ├── item_service.go
│       │   └── item_service_test.go# Tests unitarios con mocks
│       ├── repository/             # Persistencia en SQLite (WAL mode, timeouts, consultas parametrizadas)
│       │   └── sqlite_item_repository.go
│       └── registry/               # Service Registry (Singleton con sync.Once)
│           └── registry.go
├── pkg/
│   ├── database/                   # Conexión y utilidades de SQLite
│   │   └── sqlite.go
│   └── validator/                  # Wrapper de validación desacoplado
│       └── validator.go
├── migrations/                     # DDL de base de datos con índices
│   └── 000001_create_items_table.up.sql
├── specs/                          # Especificaciones funcionales y técnicas de referencia
│   ├── funcional.md
│   └── tecnico.md
└── Makefile                        # Comandos de compilación, ejecución y tests
```

---

## ⚙️ Comandos Útiles

```bash
# Descargar dependencias
go mod tidy

# Correr los tests unitarios
make test
# O directamente:
go test -v ./...

# Iniciar la aplicación (DB_PATH es obligatoria, sin defaults hardcodeados)
make run
# O directamente:
DB_PATH=items.db go run cmd/api/main.go
```

El servidor iniciará por defecto en `http://localhost:8080`.

---

## 📡 Endpoints de la API

| Método | Endpoint | Descripción | Status Exitoso |
| :--- | :--- | :--- | :--- |
| `GET` | `/health` | Healthcheck del servicio | `200 OK` |
| `POST` | `/api/v1/items` | Crea un ítem (`{"name": "string"}`) | `201 Created` |
| `GET` | `/api/v1/items` | Lista ítems paginados (`?page=1&limit=10&status=PENDING`) | `200 OK` |
| `GET` | `/api/v1/items/{id}` | Obtiene el detalle de un ítem por UUID | `200 OK` |
| `PATCH`| `/api/v1/items/{id}/status` | Transiciona estado (`PENDING` -> `ACTIVE` -> `ARCHIVED`) | `200 OK` |
| `DELETE`| `/api/v1/items/{id}` | Soft delete del ítem (`deleted_at`) | `204 No Content` |
