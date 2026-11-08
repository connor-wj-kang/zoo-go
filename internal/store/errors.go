package store

import "fmt"

type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func validation(message string) error {
	return &AppError{Status: 400, Code: "VALIDATION_ERROR", Message: message}
}

func notFound(message string) error {
	return &AppError{Status: 404, Code: "NOT_FOUND", Message: message}
}

func conflict(message string) error {
	return &AppError{Status: 409, Code: "STATE_CONFLICT", Message: message}
}

func forbidden(message string) error {
	return &AppError{Status: 403, Code: "FORBIDDEN", Message: message}
}

func locationRequired(message string) error {
	return &AppError{Status: 422, Code: "LOCATION_REQUIRED", Message: message}
}
