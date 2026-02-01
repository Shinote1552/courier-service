package courier

import "errors"

var (
	ErrMissingRequiredFields = errors.New("missing required fields")
	ErrInvalidCourierID      = errors.New("invalid courier id")
	ErrInvalidName           = errors.New("invalid name")
	ErrInvalidStatus         = errors.New("invalid status")
	ErrInvalidPhone          = errors.New("invalid phone")
	ErrInvalidTransport      = errors.New("invalid transport type")

	ErrCourierNotFound = errors.New("courier not found")
	ErrConflict        = errors.New("resource already exists")
)
