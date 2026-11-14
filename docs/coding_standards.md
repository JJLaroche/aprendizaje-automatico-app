# Estándares de Código y Buenas Prácticas (Golang)

Este documento define las reglas de ingeniería, diseño arquitectónico, seguridad y estilo de código para el backend en **Go (Golang)**. Estas reglas son indexadas en el sistema **RAG** y aplicadas estrictamente por el **LLM-as-a-Judge** al evaluar Pull Requests.

---

## 1. Arquitectura y Separación de Responsabilidades (Clean Architecture)

- **Regla de Dependencias:** El flujo de dependencias siempre apunta hacia adentro: `cmd/api` -> `handler` -> `service` -> `domain` <- `repository`.
- **Pureza del Dominio (`internal/<dominio>/domain`):**
  - No debe importar ningún paquete externo de infraestructura (ej. `database/sql`, drivers HTTP o SQLite).
  - Define entidades puras, enumeraciones con métodos de validación (`IsValid`, `CanTransitionTo`), errores centinela e interfaces (`Repository`, `Service`).
- **Aislamiento de la Capa HTTP (`internal/<dominio>/handler`):**
  - Los handlers **nunca** deben ejecutar queries SQL directamente ni acceder a la base de datos sin pasar por la capa de `service`.
  - Los handlers son métodos atómicos `http.HandlerFunc` que no crean sub-enrutadores propios. El ruteo de Chi se declara exclusivamente y de forma centralizada en `cmd/api/main.go`.
  - Toda respuesta exitosa debe utilizar el envelope estándar `{"data": ...}` y los errores deben formatearse con `{"error": {"code": "...", "message": "...", "timestamp": "..."}}`.
- **Patrón Registry y Singleton (`sync.Once`):**
  - Cada dominio debe contar con un paquete `registry/` con una función `Get<Dominio>Registry(...)` que instancie las capas de forma segura y perezosa usando `sync.Once`.

---

## 2. Manejo Idiomático de Errores

En Go **no existen excepciones ni bloques `try/catch`**. El manejo de errores debe ser explícito:

- **Prohibido Silenciar Errores:** Nunca usar el identificador blanco `_` para descartar un error retornado por I/O, base de datos o lógica de negocio (ej. `_ = repo.Create(...)` está estrictamente prohibido).
- **Control Inmediato:** Comprobar siempre `if err != nil` inmediatamente después de la llamada que genera el error.
- **Wrapping de Errores con `%w`:**
  - Al propagar un error hacia capas superiores, contextualizarlo usando `fmt.Errorf("contexto descriptivo: %w", err)` para preservar la cadena de errores.
- **Errores Centinela de Dominio:**
  - Usar errores tipados o variables centinela en `domain/errors.go` (ej. `domain.ErrNotFound`, `domain.ErrAlreadyExists`, `domain.ErrInvalidStatus`, `domain.ErrInvalidInput`).
  - Comparar con `errors.Is(err, target)` en lugar de comparar strings literales (`err.Error() == "..."`).

---

## 3. Seguridad y Persistencia en Base de Datos

- **Inyección SQL Estrictamente Prohibida:**
  - Todas las consultas SQL deben usar **placeholders parametrizados (`?`)**.
  - **Nunca** concatenar variables ni formatear strings de usuario dentro de queries (`fmt.Sprintf("SELECT ... WHERE id = '%s'", id)` está prohibido).
- **Sanitización de Configuración y DSN Injection:**
  - Las rutas de base de datos (`DB_PATH`) deben sanitizarse con `filepath.Clean` y rechazar caracteres de query (`?`, `#`, `&`).
  - Los parámetros DSN de SQLite (como `_journal_mode=WAL` y `_busy_timeout=5000`) deben construirse mediante `url.Values`.
- **Liberación Obligatoria de Recursos:**
  - Tras cualquier `QueryContext`, invocar inmediatamente `defer rows.Close()` y validar `rows.Err()` tras iterar.
- **Concurrencia Segura en SQLite:**
  - Configurar SQLite en modo `WAL` (*Write-Ahead Logging*) y restringir el pool de escritura a `db.SetMaxOpenConns(1)` para evitar bloqueos por concurrencia.
- **Variables de Entorno y Secretos:**
  - Las credenciales, tokens y variables de configuración deben leerse exclusivamente de variables de entorno (`os.Getenv`).
  - **Nunca** commitear archivos `.env` reales, tokens ni llaves de API al repositorio. Usar `.env.example` como plantilla.

---

## 4. Control del Tiempo y Determinismo (`pkg/clock`)

- **Prohibido el uso directo de `time.Now()` en Lógica de Negocio y Repositorios:**
  - Llamar a `time.Now()` directamente en services o repositorios rompe la reproducibilidad de los tests.
  - Se debe inyectar la interfaz `clock.Clock` (`pkg/clock/clock.go`) y obtener la hora actual invocando `s.clock.Now()`.
  - La implementación real debe trabajar en hora **UTC** (`time.Now().UTC()`).

---

## 5. Propagación de Contexto y Concurrencia

- **Propagación Obligatoria de `context.Context`:**
  - El primer parámetro de todas las funciones de `Handler`, `Service` y `Repository` debe ser `ctx context.Context`.
  - Usar siempre consultas preparadas con contexto (`ExecContext`, `QueryRowContext`, `QueryContext`, `PingContext`).
  - En los handlers, propagar `r.Context()`.
- **Timeouts y Cancelación:**
  - Los routers deben incluir middleware de timeout (máximo 5 segundos).
  - Los endpoints de inicialización y background deben usar `context.WithTimeout`.

---

## 6. Estándares de Testing

- **Table-Driven Tests Obligatorios:**
  - Todas las capas (`handler`, `service`, `repository`) deben organizar sus pruebas unitarias e integración usando tablas de casos (`tests := []struct{...}`).
  - Ejecutar cada caso mediante sub-tests nombrados: `t.Run(tc.name, func(t *testing.T) { ... })`.
- **Mocks con Mockery:**
  - Prohibido crear structs mock manuales dentro del código de prueba.
  - Los mocks de interfaces deben generarse automáticamente mediante `mockery` configurado en `.mockery.yaml`.
  - En todo test que involucre mocks, es obligatorio invocar `mock.AssertExpectations(t)` al finalizar cada caso.
- **Tests de Handlers:**
  - Usar `httptest.NewRecorder()` y `httptest.NewRequest()` simulando rutas Chi con `chi.RouteContext` para path params.
- **Tests de Repositorios:**
  - Probar persistencia real contra SQLite en memoria (`file:<uuid>?mode=memory&cache=shared`) ejecutando las migraciones DDL reales.

---

## 7. Estilo y Convenciones de Go

- **Nombres Concisos e Idiomáticos:**
  - En Go se prefieren nombres cortos y descriptivos (`ctx`, `req`, `svc`, `repo`, `w`, `r`).
  - Usar camelCase para variables y funciones no exportadas (`itemService`), y PascalCase para exportadas (`ItemService`).
  - En acrónimos, mantener mayúsculas uniformes: `ID`, `HTTP`, `URL`, `UUID`, `SQL` (ej. `GetByID`, no `GetById`).
- **Estructura de Retorno Anticipado (Early Return / Guard Clauses):**
  - Evitar anidaciones profundas con `else`. Retornar temprano ante condiciones de error:
    ```go
    // Bien:
    if err != nil {
        return nil, fmt.Errorf("error: %w", err)
    }
    // Continuar lógica principal al nivel base
    ```
