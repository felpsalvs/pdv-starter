// Package service holds the business logic that used to live in
// src/services/*.js: validation, transaction orchestration and the rules
// tying orders, payments and the cash register together.
package service

// ValidationError mirrors services/errors.js's ValidationError — the HTTP
// layer maps it to a 400 with the message as-is, exactly like the Node
// backend's handleError() did.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}
