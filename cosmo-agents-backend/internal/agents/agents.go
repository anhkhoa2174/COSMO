package agents

import "errors"

var (
	// ErrUnauthorized is returned when a user cannot access an entity.
	ErrUnauthorized = errors.New("unauthorized access")
	// ErrNotFound is returned when the entity is missing.
	ErrNotFound = errors.New("not found")
)
