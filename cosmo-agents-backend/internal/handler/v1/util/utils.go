package util

import (
	"github.com/google/uuid"
)

// containsUUID checks if a UUID exists in a list
func ContainsUUID(list []uuid.UUID, target uuid.UUID) bool {
	for _, id := range list {
		if id == target {
			return true
		}
	}
	return false
}
