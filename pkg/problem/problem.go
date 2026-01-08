package problem

import (
	"encoding/json"
	"net/http"
)

const (
	ContentType = "application/problem+json"
	BaseURI     = "http://localhost:8080/problems"
)

// Problem represents an RFC 9457 problem+json response
type Problem struct {
	Type   string        `json:"type"`
	Title  string        `json:"title"`
	Status int           `json:"status"`
	Detail string        `json:"detail,omitempty"`
	Errors []FieldError  `json:"errors,omitempty"`
}

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// New creates a new Problem
func New(status int, problemType, title, detail string) *Problem {
	return &Problem{
		Type:   BaseURI + "/" + problemType,
		Title:  title,
		Status: status,
		Detail: detail,
	}
}

// WithErrors adds field errors to the problem
func (p *Problem) WithErrors(errors []FieldError) *Problem {
	p.Errors = errors
	return p
}

// Write writes the problem to the response
func (p *Problem) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p)
}

// Common problem constructors

func NotFound(detail string) *Problem {
	return New(http.StatusNotFound, "not-found", "Not Found", detail)
}

func BadRequest(detail string) *Problem {
	return New(http.StatusBadRequest, "bad-request", "Bad Request", detail)
}

func ValidationError(detail string, errors []FieldError) *Problem {
	return New(http.StatusUnprocessableEntity, "validation-error", "Validation Error", detail).WithErrors(errors)
}

func Conflict(detail string) *Problem {
	return New(http.StatusConflict, "conflict", "Conflict", detail)
}

func InternalError(detail string) *Problem {
	return New(http.StatusInternalServerError, "internal-error", "Internal Server Error", detail)
}
