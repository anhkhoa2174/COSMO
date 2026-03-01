package campaign

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// parsePagination extracts pagination parameters from query string
func parsePagination(c fiber.Ctx, defaultOffset, defaultLimit int) (int, int) {
	offset := defaultOffset
	limit := defaultLimit
	offsetProvided := false

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil {
			offset = val
		}
		offsetProvided = true
	}

	// Allow alternate limit keys from FE (page_size, pageSize, per_page)
	if limit == defaultLimit {
		if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
			if val, err := strconv.Atoi(pageSizeStr); err == nil && val > 0 {
				limit = val
			}
		} else if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
			if val, err := strconv.Atoi(pageSizeStr); err == nil && val > 0 {
				limit = val
			}
		} else if perPageStr := c.Query("per_page"); perPageStr != "" {
			if val, err := strconv.Atoi(perPageStr); err == nil && val > 0 {
				limit = val
			}
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	// Allow page-based pagination for clients that pass ?page=N (1-indexed) or ?page_index=N (0-indexed).
	pageParamProvided := false
	if offset == defaultOffset {
		pageIndexStr := c.Query("page_index")
		if pageIndexStr == "" {
			pageIndexStr = c.Query("pageIndex")
		}
		if pageIndexStr != "" {
			if val, err := strconv.Atoi(pageIndexStr); err == nil && val >= 0 {
				offset = val * limit
				pageParamProvided = true
			}
		} else if pageStr := c.Query("page"); pageStr != "" {
			if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
				offset = (val - 1) * limit
				pageParamProvided = true
			}
		}
	}

	if limit <= 0 {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = defaultOffset
	}

	// Heuristic: some clients send offset as pageIndex while also sending limit.
	// If offset was explicitly provided, no page params, and offset looks like a page index (< limit),
	// treat it as pageIndex to avoid repeating page 1.
	if offsetProvided && !pageParamProvided && offset > 0 && limit > 0 && offset < limit {
		offset = offset * limit
	}

	return offset, limit
}

// deriveCampaignName generates a campaign name from the playbook
func deriveCampaignName(playbook string) string {
	normalized := strings.NewReplacer("-", " ", "_", " ").Replace(playbook)
	return strings.Title(strings.TrimSpace(normalized))
}

// campaignMetadataFromMap converts a map to CampaignMetadata
func campaignMetadataFromMap(data map[string]interface{}) domain.CampaignMetadata {
	if data == nil {
		return domain.CampaignMetadata{}
	}
	bytes, _ := json.Marshal(data)
	var metadata domain.CampaignMetadata
	_ = json.Unmarshal(bytes, &metadata)
	return metadata
}

// metadataToMap converts CampaignMetadata to a map
func metadataToMap(metadata domain.CampaignMetadata) map[string]interface{} {
	bytes, _ := json.Marshal(metadata)
	result := map[string]interface{}{}
	_ = json.Unmarshal(bytes, &result)
	return result
}

// mergeMaps recursively merges two maps
func mergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = map[string]interface{}{}
	}
	for k, v := range src {
		if existing, ok := dst[k]; ok {
			existingMap, exOK := existing.(map[string]interface{})
			valueMap, valOK := v.(map[string]interface{})
			if exOK && valOK {
				dst[k] = mergeMaps(existingMap, valueMap)
				continue
			}
		}
		dst[k] = v
	}
	return dst
}

// campaignAccessible checks if user has access to campaign
func campaignAccessible(campaign *domain.Campaign, userID uuid.UUID, roles []domain.Role) bool {
	if campaign == nil {
		return false
	}
	if campaign.UserID == userID {
		return true
	}
	if campaign.OrganizationID == nil {
		return false
	}
	return userInOrganization(*campaign.OrganizationID, roles)
}

// userInOrganization checks if user belongs to organization
func userInOrganization(target uuid.UUID, roles []domain.Role) bool {
	for _, role := range roles {
		if role.IsDeleted {
			continue
		}
		if role.OrganizationID == target {
			return true
		}
	}
	return false
}

// convertAssignConfig converts assignment request to campaign members
func convertAssignConfig(config []v1schema.AssignMemberRequest) ([]domain.CampaignMember, error) {
	seen := map[string]struct{}{}
	members := make([]domain.CampaignMember, len(config))
	for i, cfg := range config {
		intentKey := strings.ToLower(strings.TrimSpace(cfg.IntentType))
		if _, exists := seen[intentKey]; exists {
			return nil, fmt.Errorf("duplicate intent_type: %s", cfg.IntentType)
		}
		seen[intentKey] = struct{}{}

		handler := normalizeHandler(cfg.Who)
		if handler != domain.HandlerAI && handler != domain.HandlerHuman && handler != domain.HandlerDraft {
			return nil, fmt.Errorf("unsupported handler value: %s", cfg.Who)
		}

		intent := domain.IntentType(cfg.IntentType)
		if !intent.IsValid() {
			return nil, fmt.Errorf("invalid intent type: %s", cfg.IntentType)
		}

		payload := cfg.Payload
		members[i] = domain.CampaignMember{Who: handler, IntentType: intent, Payload: payload}
	}
	return members, nil
}

// normalizeHandler normalizes handler string values
func normalizeHandler(handler string) domain.Handler {
	switch strings.ToUpper(strings.TrimSpace(handler)) {
	case "AI", "LET AI REPLY":
		return domain.HandlerAI
	case "HUMAN", "ASSIGN TO A PERSON":
		return domain.HandlerHuman
	case "DRAFT", "DRAFT AN EMAIL":
		return domain.HandlerDraft
	default:
		return domain.Handler(handler)
	}
}

// Error response helpers
func badRequest(c fiber.Ctx, message string, err error) error {
	if err == nil {
		err = errors.New(message)
	}
	if c == nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s: %v", message, err))
	}
	return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, message, err.Error()))
}

func internalError(c fiber.Ctx, message string, err error) error {
	if c == nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("%s: %v", message, err))
	}
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, message, err.Error()))
}

func notFound(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(fiber.StatusNotFound, message, ""))
}

func unauthorizedResponse(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "unauthorized", ""))
}

func unauthorizedAccessResponse(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "You are not authorized to access this resource", ""))
}

func forbidden(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(fiber.StatusForbidden, message, ""))
}

// userIDFromContext extracts user ID from fiber context
func userIDFromContext(c fiber.Ctx) (uuid.UUID, bool) {
	userID := c.Locals("user_id")
	if userID == nil {
		return uuid.Nil, false
	}
	uid, ok := userID.(uuid.UUID)
	return uid, ok
}
