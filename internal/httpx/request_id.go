package httpx

import "net/http"

// Chi guarda el request id en el header "X-Request-Id" y también lo propaga.
// Este helper lo lee desde el request para incluirlo en las respuestas.
func RequestIDFrom(request *http.Request) string {
	if request == nil {
		return ""
	}
	return request.Header.Get("X-Request-Id")
}
