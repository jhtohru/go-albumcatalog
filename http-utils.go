package catalog

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// decode decodes a T from r.
func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decoding json: %w", err)
	}
	return v, nil
}

// encode setup w, write statusCode as its status code and write v into its body.
func encode[T any](w http.ResponseWriter, statusCode int, v T) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}
	return nil
}

// encodeMessage write an HTTP response with the statusCode as its status code
// and writes msg into its body.
func encodeMessage(w http.ResponseWriter, statusCode int, msg string) error {
	data := struct {
		Message string `json:"message"`
	}{
		Message: msg,
	}
	return encode(w, statusCode, data)
}

// encodeproblems write an HTTP response with the statusCode as its status code
// and writes msg and problems into its body.
func encodeProblems(w http.ResponseWriter, statusCode int, msg string, problems map[string]string) error {
	data := struct {
		Message  string            `json:"message"`
		Problems map[string]string `json:"problems"`
	}{
		Message:  msg,
		Problems: problems,
	}
	return encode(w, statusCode, data)
}
