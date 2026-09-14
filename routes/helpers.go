package routes

import (
	"encoding/json"
	"net/http"
)

// writeJSON sends data back as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError keeps the error shape consistent with what the frontend
// already expects: { "error": "message" } rather than a raw stack trace.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
