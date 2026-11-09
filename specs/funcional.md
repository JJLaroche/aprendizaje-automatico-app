# ESPECIFICACIÓN FUNCIONAL (SPEC FUNCIONAL)

## 1. OBJETIVO DEL SISTEMA
Construir una API Backend sólida, eficiente y mantenible en **Go (Golang)** encargada de gestionar el ciclo de vida de **Ítems** de catálogo/inventario. El sistema servirá además como base de código de referencia contra la cual se abrirán Pull Requests para evaluar la efectividad y precisión de un evaluador automático basado en **LLM-as-a-Judge**.

---

## 2. REGLAS DE NEGOCIO Y COMPORTAMIENTO

### 2.1. Gestión de Ítems
1. **Creación de Ítem:**
   - El sistema debe recibir los datos obligatorios: `name` (longitud entre 3 y 100 caracteres).
   - Opcionalmente puede recibir campos descriptivos según evolucione el dominio.
   - Si falta algún campo obligatorio o no cumple el formato (ej. string vacío, longitud fuera de rango), el sistema debe rechazar la solicitud con una explicación clara del error (`400 Bad Request`).
   - El nombre del ítem debe ser único entre los ítems no eliminados. Si ya existe un ítem activo con el mismo nombre, el sistema debe responder con `409 Conflict`.
   - Todo ítem recién creado debe asignarse por defecto con el estado `PENDING`.

2. **Lectura / Consulta:**
   - Debe ser posible consultar un ítem por su identificador único (UUID v4).
   - Debe ser posible listar ítems con paginación (`page`, `limit`) y filtrado opcional por `status`.
   - Por defecto, los ítems con baja lógica (`deleted_at` no nulo) quedan excluidos de todas las consultas estándar.

3. **Actualización de Estado:**
   - Estados disponibles: `PENDING`, `ACTIVE`, `ARCHIVED`.
   - Transiciones de estado permitidas:
     - `PENDING` -> `ACTIVE`
     - `ACTIVE` -> `ARCHIVED`
     - `PENDING` -> `ARCHIVED` (cancelación directa)
   - Transiciones prohibidas:
     - No se permite revertir un estado de `ARCHIVED` a `PENDING` o `ACTIVE`.
     - Si se intenta una transición no válida, el sistema debe retornar un error de regla de negocio (`422 Unprocessable Entity`).

4. **Baja / Eliminación:**
   - La eliminación debe ser de tipo **Soft Delete** (marcado de fecha `deleted_at`), conservando el registro en la base de datos pero excluyéndolo de las búsquedas y lecturas estándar.
   - Si se intenta eliminar un ítem ya eliminado o inexistente, el sistema responde `404 Not Found`.

---

## 3. CASOS DE USO (HISTORIAS DE USUARIO)

### CU-01: Crear Ítem
- **Dado que** un cliente envía una petición válida con `{"name": "Teclado Mecánico"}`.
- **Cuando** la API procesa la solicitud.
- **Entonces** valida la entrada, verifica unicidad, guarda el registro en la base de datos, asigna un UUID v4 único, estado `PENDING`, timestamps UTC y retorna la entidad creada con código `201 Created`.

### CU-02: Intentar registrar Ítem duplicado
- **Dado que** se intenta registrar un ítem con un nombre que ya existe en el sistema y no está marcado como eliminado.
- **Cuando** la API valida la existencia previa en la persistencia.
- **Entonces** interrumpe el procesamiento y responde con un código `409 Conflict`.

### CU-03: Manejo de Recursos No Encontrados
- **Dado que** un cliente solicita un ítem especificando un ID inexistente o marcado como eliminado (`deleted_at != nil`).
- **Cuando** la API realiza la búsqueda en persistencia.
- **Entonces** responde inmediatamente con código `404 Not Found`.

### CU-04: Transición de Estado Válida
- **Dado que** un ítem existe en estado `PENDING`.
- **Cuando** el cliente envía una solicitud `PATCH /api/v1/items/{id}/status` con `{"status": "ACTIVE"}`.
- **Entonces** el sistema actualiza el estado a `ACTIVE`, renueva `updated_at` y devuelve el ítem actualizado con código `200 OK`.

### CU-05: Transición de Estado Inválida
- **Dado que** un ítem se encuentra en estado `ARCHIVED`.
- **Cuando** el cliente intenta cambiar su estado a `PENDING`.
- **Entonces** el sistema rechaza la operación con código `422 Unprocessable Entity` y el mensaje explicativo correspondiente.

---

## 4. FUERA DE ALCANCE (OUT OF SCOPE - MVP)
- Autenticación y autorización (JWT / OAuth2 / API Keys).
- Notificaciones vía email, webhooks o mensajería asíncrona.
- Integración con pasarelas de pago o proveedores de inventario externos.
- Generación de reportes o exportación masiva a CSV/PDF.
