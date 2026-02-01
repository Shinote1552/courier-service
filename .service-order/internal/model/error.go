package model

import "errors"

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidInput     = errors.New("invalid input")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrInternal         = errors.New("internal error")
	ErrInvalidID        = errors.New("invalid id")
	ErrForeignOwnership = errors.New("foreign ownership")
)
