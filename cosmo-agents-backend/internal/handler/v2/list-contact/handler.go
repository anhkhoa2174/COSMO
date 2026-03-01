package listcontact

import (
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
)

// Handler handles V2 list contact HTTP requests
type Handler struct {
	listContactRepo *contactRepo.ListContactRepository
	userRepo        *user.UserRepository
}

// NewHandler creates a new V2 list contact handler with injected dependencies
func NewHandler(
	listContactRepo *contactRepo.ListContactRepository,
	userRepo *user.UserRepository,
) *Handler {
	return &Handler{
		listContactRepo: listContactRepo,
		userRepo:        userRepo,
	}
}
