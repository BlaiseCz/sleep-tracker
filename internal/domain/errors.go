package domain

import "errors"

var (
	ErrNotFound           = errors.New("resource not found")
	ErrConflict           = errors.New("resource conflict")
	ErrOverlappingSleep   = errors.New("overlapping sleep period detected")
	ErrDuplicateRequest   = errors.New("duplicate client request")
	ErrInvalidInput       = errors.New("invalid input")
)
