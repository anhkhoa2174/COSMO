package contact

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ReEmbedAll triggers best-effort re-embedding for all contacts of the current user/org.
// Route: POST /v1/contacts/re-embed-all
// Response: {"started": true, "total": N} (does not wait for completion)
func (h *Handler) ReEmbedAll(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	if h.intelSvc == nil {
		return h.responseHelper.InternalServerError(c, "embedding service not available", fmt.Errorf("intel service nil"))
	}

	// Fetch contacts for this user/org (simple pagination)
	ctx := c.Context()
	const pageSize = 200
	offset := 0
	totalStarted := 0

	for {
		contacts, total, err := h.repo.FindByUserIDWithPagination(ctx, user.ID, offset, pageSize)
		if err != nil {
			// best-effort: return partial count
			return h.responseHelper.InternalServerError(c, "failed to list contacts", err)
		}
		for _, ct := range contacts {
			// Quick org check
			if ct.OrganizationID != nil && *ct.OrganizationID != orgID {
				continue
			}
			totalStarted++
			go func(contactID uuid.UUID, userID uuid.UUID) {
				_ = h.intelSvc.EmbedContact(context.Background(), userID, contactID)
			}(ct.ID, user.ID)
		}
		offset += len(contacts)
		if offset >= total || len(contacts) == 0 {
			break
		}
	}

	return h.responseHelper.Success(c, fiber.Map{
		"started":       true,
		"total_started": totalStarted,
	})
}
