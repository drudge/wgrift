package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type jsonResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
	Total *int   `json:"total,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(jsonResponse{Data: v})
}

func writeJSONList(w http.ResponseWriter, status int, v any, total int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(jsonResponse{Data: v, Total: &total})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(jsonResponse{Error: message})
}

// writeValidationError returns a 200 with the error in the body so that
// reverse proxies (nginx-ingress) don't intercept the response and replace
// the body with a generic error page.
func writeValidationError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonResponse{Error: message})
}

func readJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("empty request body")
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}
