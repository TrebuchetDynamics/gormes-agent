package httpjson

import (
	"encoding/json"
	"net/http"
)

// Write writes a JSON response with the adapter-standard content type.
func Write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
