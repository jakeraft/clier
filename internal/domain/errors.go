package domain

import "fmt"

// ErrNotFound indicates that an entity was not found.
type ErrNotFound struct {
	Entity string
	ID     string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Entity, e.ID)
}
