package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/validator"
)

// ItemHandler gestiona las peticiones HTTP del recurso Item mediante métodos atómicos.
// No contiene ruteo interno; las rutas son registradas centralizadamente en main.go.
type ItemHandler struct {
	service   domain.ItemService
	validator *validator.CustomValidator
}

// NewItemHandler inicializa el handler con sus dependencias requeridas.
func NewItemHandler(service domain.ItemService, val *validator.CustomValidator) *ItemHandler {
	return &ItemHandler{
		service:   service,
		validator: val,
	}
}

// Create implementa el endpoint POST /api/v1/items
func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "cuerpo de solicitud JSON malformado")
		return
	}

	if err := h.validator.ValidateStruct(req); err != nil {
		RespondError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	item, err := h.service.Create(r.Context(), req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			RespondError(w, http.StatusConflict, "CONFLICT", "ya existe un ítem activo con ese nombre")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			RespondError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error interno del servidor")
		return
	}

	RespondJSON(w, http.StatusCreated, item)
}

// GetByID implementa el endpoint GET /api/v1/items/{id}
func (h *ItemHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "BAD_REQUEST", "el identificador provisto no es un UUID válido")
		return
	}

	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "NOT_FOUND", "el recurso no fue encontrado")
			return
		}
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error interno del servidor")
		return
	}

	RespondJSON(w, http.StatusOK, item)
}

// List implementa el endpoint GET /api/v1/items
func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))

	var status *domain.Status
	if rawStatus := query.Get("status"); rawStatus != "" {
		st := domain.Status(rawStatus)
		status = &st
	}

	result, err := h.service.List(r.Context(), status, page, limit)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			RespondError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error interno del servidor")
		return
	}

	RespondJSON(w, http.StatusOK, result)
}

// UpdateStatus implementa el endpoint PATCH /api/v1/items/{id}/status
func (h *ItemHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "BAD_REQUEST", "el identificador provisto no es un UUID válido")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "cuerpo de solicitud JSON malformado")
		return
	}

	if err := h.validator.ValidateStruct(req); err != nil {
		RespondError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	updatedItem, err := h.service.UpdateStatus(r.Context(), id, domain.Status(req.Status))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "NOT_FOUND", "el recurso no fue encontrado")
			return
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			RespondError(w, http.StatusUnprocessableEntity, "UNPROCESSABLE_ENTITY", err.Error())
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			RespondError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error interno del servidor")
		return
	}

	RespondJSON(w, http.StatusOK, updatedItem)
}

// Delete implementa el endpoint DELETE /api/v1/items/{id}
func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "BAD_REQUEST", "el identificador provisto no es un UUID válido")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "NOT_FOUND", "el recurso no fue encontrado")
			return
		}
		RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "error interno del servidor")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
