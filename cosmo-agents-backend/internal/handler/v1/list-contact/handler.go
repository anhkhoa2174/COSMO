package listcontact

import (
	"context"

	"github.com/google/uuid"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Handler handles V1 list contact HTTP requests
type Handler struct {
	listContactRepo *contactRepo.ListContactRepository
	userRepo        *user.UserRepository
}

// NewHandler creates a new V1 list contact handler with injected dependencies
func NewHandler(
	listContactRepo *contactRepo.ListContactRepository,
	userRepo *user.UserRepository,
) *Handler {
	return &Handler{
		listContactRepo: listContactRepo,
		userRepo:        userRepo,
	}
}

// loadInboundLeadForms loads inbound lead forms for a list contact and converts to response format
func (h *Handler) loadInboundLeadForms(ctx context.Context, listContactID uuid.UUID) ([]v1schema.InboundLeadFormEntity, error) {
	// Load domain models
	domainForms, err := h.listContactRepo.GetInboundLeadForms(ctx, listContactID)
	if err != nil {
		return nil, err
	}

	// Convert to response entities
	forms := make([]v1schema.InboundLeadFormEntity, len(domainForms))
	for i, form := range domainForms {
		forms[i] = v1schema.InboundLeadFormEntity{
			ID:         form.ID,
			Name:       form.Name,
			Slug:       form.Slug,
			UIMetadata: form.UIMetadata,
			CreatedAt:  form.CreatedAt,
			UpdatedAt:  form.UpdatedAt,
		}
	}

	return forms, nil
}
