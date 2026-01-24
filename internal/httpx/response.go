package httpx

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response es el sobre estándar que devuelve la API.
// Mantener un formato consistente hace que los clientes (frontend/tests) sean más simples.
type Response struct {
	Data  any        `json:"data,omitempty"`
	Error *ErrorBody `json:"error,omitempty"`
	Meta  *Meta      `json:"meta,omitempty"`
}

// Meta contiene información adicional útil para debugging y trazabilidad.
type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	TimeUTC   string `json:"time_utc,omitempty"`
}

// ErrorBody describe un error de forma estructurada.
// No exponer detalles internos (SQL, stacktrace, etc.) en producción.
type ErrorBody struct {
	Code    string `json:"code,omitempty"`    // ej: "invalid_input", "not_found"
	Message string `json:"message,omitempty"` // mensaje para humanos
}

// JSON escribe una respuesta JSON con headers correctos.
// Nota: en caso de error de encodeo, responde 500 de forma segura.
func JSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(resp); err != nil {
		// Último recurso: no se pudo serializar JSON.
		http.Error(w, `{"error":{"code":"internal","message":"internal server error"}}`, http.StatusInternalServerError)
	}
}

// OK devuelve una respuesta exitosa con data.
func OK(w http.ResponseWriter, r *http.Request, status int, data any) {
	JSON(w, status, Response{
		Data: data,
		Meta: &Meta{
			RequestID: RequestIDFrom(r),
			TimeUTC:   time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// Fail devuelve un error estructurado.
func Fail(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	JSON(w, status, Response{
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
		Meta: &Meta{
			RequestID: RequestIDFrom(r),
			TimeUTC:   time.Now().UTC().Format(time.RFC3339),
		},
	})
}
