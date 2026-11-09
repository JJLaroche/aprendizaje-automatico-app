package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain"
	domainMocks "github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/domain/mocks"
	"github.com/marielarodriguez/aprendizaje-automatico-app/internal/item/handler"
	"github.com/marielarodriguez/aprendizaje-automatico-app/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestItemHandler_Create(t *testing.T) {
	fixedTime := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	generatedID := uuid.New()

	tests := []struct {
		name           string
		body           string
		setupMocks     func(svc *domainMocks.MockItemService)
		expectedStatus int
		expectErrorCode string
	}{
		{
			name: "Éxito: Retorna 201 Created",
			body: `{"name":"Micrófono USB"}`,
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().Create(mock.Anything, "Micrófono USB").Return(&domain.Item{
					ID:        generatedID,
					Name:      "Micrófono USB",
					Status:    domain.StatusPending,
					CreatedAt: fixedTime,
					UpdatedAt: fixedTime,
				}, nil).Once()
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Error: Payload JSON malformado (400 Bad Request)",
			body: `{"name":`,
			setupMocks: func(svc *domainMocks.MockItemService) {
				// No debe llamar al service
			},
			expectedStatus: http.StatusBadRequest,
			expectErrorCode: "INVALID_PAYLOAD",
		},
		{
			name: "Error: Falla validación DTO nombre muy corto (400 Bad Request)",
			body: `{"name":"AB"}`,
			setupMocks: func(svc *domainMocks.MockItemService) {
				// No debe llamar al service
			},
			expectedStatus: http.StatusBadRequest,
			expectErrorCode: "VALIDATION_FAILED",
		},
		{
			name: "Error: Conflicto por nombre duplicado (409 Conflict)",
			body: `{"name":"Micrófono USB"}`,
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().Create(mock.Anything, "Micrófono USB").Return(nil, domain.ErrAlreadyExists).Once()
			},
			expectedStatus: http.StatusConflict,
			expectErrorCode: "CONFLICT",
		},
		{
			name: "Error: Fallo no controlado de backend (500 Internal Error)",
			body: `{"name":"Micrófono USB"}`,
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().Create(mock.Anything, "Micrófono USB").Return(nil, errors.New("db connection lost")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectErrorCode: "INTERNAL_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := domainMocks.NewMockItemService(t)
			tc.setupMocks(mockSvc)

			val := validator.New()
			h := handler.NewItemHandler(mockSvc, val)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/items", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Create(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectErrorCode != "" {
				var errResp handler.ErrorEnvelope
				err := json.NewDecoder(w.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectErrorCode, errResp.Error.Code)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestItemHandler_GetByID(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name           string
		idParam        string
		setupMocks     func(svc *domainMocks.MockItemService)
		expectedStatus int
		expectErrorCode string
	}{
		{
			name:    "Éxito: Retorna 200 OK con ítem",
			idParam: validID.String(),
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().GetByID(mock.Anything, validID).Return(&domain.Item{
					ID:     validID,
					Name:   "Teclado",
					Status: domain.StatusActive,
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "Error: UUID malformado (400 Bad Request)",
			idParam: "invalid-uuid",
			setupMocks: func(svc *domainMocks.MockItemService) {
				// No debe invocar al service
			},
			expectedStatus: http.StatusBadRequest,
			expectErrorCode: "BAD_REQUEST",
		},
		{
			name:    "Error: Ítem no encontrado (404 Not Found)",
			idParam: validID.String(),
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().GetByID(mock.Anything, validID).Return(nil, domain.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			expectErrorCode: "NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := domainMocks.NewMockItemService(t)
			tc.setupMocks(mockSvc)

			val := validator.New()
			h := handler.NewItemHandler(mockSvc, val)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/items/"+tc.idParam, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.idParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.GetByID(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectErrorCode != "" {
				var errResp handler.ErrorEnvelope
				_ = json.NewDecoder(w.Body).Decode(&errResp)
				assert.Equal(t, tc.expectErrorCode, errResp.Error.Code)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestItemHandler_UpdateStatus(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name           string
		idParam        string
		body           string
		setupMocks     func(svc *domainMocks.MockItemService)
		expectedStatus int
		expectErrorCode string
	}{
		{
			name:    "Éxito: Transiciona estado y responde 200 OK",
			idParam: validID.String(),
			body:    `{"status":"ACTIVE"}`,
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().UpdateStatus(mock.Anything, validID, domain.StatusActive).Return(&domain.Item{
					ID:     validID,
					Name:   "Notebook",
					Status: domain.StatusActive,
				}, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "Error: Violación de regla de negocio (422 Unprocessable Entity)",
			idParam: validID.String(),
			body:    `{"status":"PENDING"}`,
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().UpdateStatus(mock.Anything, validID, domain.StatusPending).Return(nil, domain.ErrInvalidStatus).Once()
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectErrorCode: "UNPROCESSABLE_ENTITY",
		},
		{
			name:    "Error: Status fuera del enum (400 Bad Request)",
			idParam: validID.String(),
			body:    `{"status":"UNKNOWN_STATUS"}`,
			setupMocks: func(svc *domainMocks.MockItemService) {
				// No debe invocar al service
			},
			expectedStatus: http.StatusBadRequest,
			expectErrorCode: "VALIDATION_FAILED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := domainMocks.NewMockItemService(t)
			tc.setupMocks(mockSvc)

			val := validator.New()
			h := handler.NewItemHandler(mockSvc, val)

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/items/"+tc.idParam+"/status", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.idParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.UpdateStatus(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectErrorCode != "" {
				var errResp handler.ErrorEnvelope
				_ = json.NewDecoder(w.Body).Decode(&errResp)
				assert.Equal(t, tc.expectErrorCode, errResp.Error.Code)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestItemHandler_Delete(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name           string
		idParam        string
		setupMocks     func(svc *domainMocks.MockItemService)
		expectedStatus int
	}{
		{
			name:    "Éxito: Soft delete responde 204 No Content",
			idParam: validID.String(),
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().Delete(mock.Anything, validID).Return(nil).Once()
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:    "Error: No encontrado responde 404 Not Found",
			idParam: validID.String(),
			setupMocks: func(svc *domainMocks.MockItemService) {
				svc.EXPECT().Delete(mock.Anything, validID).Return(domain.ErrNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := domainMocks.NewMockItemService(t)
			tc.setupMocks(mockSvc)

			val := validator.New()
			h := handler.NewItemHandler(mockSvc, val)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/items/"+tc.idParam, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.idParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.Delete(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}
