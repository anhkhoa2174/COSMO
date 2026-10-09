package contact

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// CleanupDeletedVectors removes Redis vectors for contacts that have been soft-deleted.
// Route: POST /v1/contacts/cleanup-vectors
func (h *Handler) CleanupDeletedVectors(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	if h.intelSvc == nil {
		return h.responseHelper.InternalServerError(c, "intelligence service not available", fmt.Errorf("intel service nil"))
	}

	// Find all soft-deleted contacts for this user/org
	ctx := c.Context()
	const pageSize = 500
	offset := 0
	totalCleaned := 0

	for {
		contacts, err := h.repo.FindDeleted(ctx, user.ID, &orgID, offset, pageSize)
		if err != nil {
			return h.responseHelper.InternalServerError(c, "failed to list deleted contacts", err)
		}
		if len(contacts) == 0 {
			break
		}

		ids := make([]uuid.UUID, len(contacts))
		for i, ct := range contacts {
			ids[i] = ct.ID
		}
		h.intelSvc.DeleteContactVectors(ctx, ids)
		totalCleaned += len(contacts)

		offset += len(contacts)
		if len(contacts) < pageSize {
			break
		}
	}

	return h.responseHelper.Success(c, fiber.Map{
		"cleaned": totalCleaned,
	})
}
