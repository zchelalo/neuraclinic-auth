package uuidx

import "github.com/google/uuid"

// New returns a UUIDv7 and falls back to UUIDv4 if the generator errors.
func New() uuid.UUID {
	id, err := uuid.NewV7()
	if err == nil {
		return id
	}
	return uuid.New()
}

func NewString() string {
	return New().String()
}
