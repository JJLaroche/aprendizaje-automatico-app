package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

// ResponseEnvelope representa la envoltura estándar para respuestas exitosas.
type ResponseEnvelope struct {
	Data interface{} `json:"data"`
}

// ErrorEnvelope representa la estructura estándar unificada para errores de la API.
type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contiene la información detallada del error ocurrido.
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// RespondJSON envía una respuesta exitosa formateada en JSON con el status code indicado.
func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if data != nil {
		_ = json.NewEncoder(w).Encode(ResponseEnvelope{Data: data})
	}
}

// RespondError envía una respuesta de error JSON con el código de error unificado.
func RespondError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	errPayload := ErrorEnvelope{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}

	_ = json.NewEncoder(w).Encode(errPayload)
}
