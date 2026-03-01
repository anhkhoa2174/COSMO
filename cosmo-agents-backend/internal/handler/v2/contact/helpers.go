package contact

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// toContactEntity converts domain.Contact to ContactEntity response
func (h *Handler) toContactEntitySingle(contact *domain.Contact) (ContactEntity, error) {
	var profile map[string]interface{}
	if err := json.Unmarshal(contact.Profile, &profile); err != nil {
		return ContactEntity{}, fmt.Errorf("Failed to parse profile: %w", err)
	}
	profileWrapper := map[string]interface{}{"profile": profile}

	// Extract email and phone from profile
	var email, phone string
	if e, ok := profile["email"].(string); ok {
		email = e
	}
	if p, ok := profile["phone"].(string); ok {
		phone = p
	}

	entity := ContactEntity{
		ID:             contact.ID,
		UserID:         contact.UserID,
		Name:           strPtrIfPresent(contact.Name),
		Email:          strPtrIfPresent(email),
		Phone:          strPtrIfPresent(phone),
		Company:        strPtrIfPresent(contact.Company),
		JobTitle:       strPtrIfPresent(contact.JobTitle),
		Address:        strPtrIfPresent(contact.Address),
		City:           strPtrIfPresent(contact.City),
		Country:        strPtrIfPresent(contact.Country),
		State:          strPtrIfPresent(contact.State),
		Zip:            strPtrIfPresent(contact.Zip),
		OrganizationID: contact.OrganizationID,
		CreatedAt:      contact.CreatedAt,
		UpdatedAt:      contact.UpdatedAt,
		Extra:          profileWrapper,
	}

	if contact.SourceID != "" {
		entity.SourceID = &contact.SourceID
	}
	if contact.Source != "" {
		entity.Source = &contact.Source
	}
	if contact.HubspotID != nil && *contact.HubspotID != "" {
		entity.HubspotID = contact.HubspotID
	}

	// Parse tags if present
	if len(contact.Tags) > 0 {
		var tags map[string]interface{}
		if err := json.Unmarshal(contact.Tags, &tags); err == nil {
			entity.Tags = tags
		}
	}

	// Note: Direct relationships removed to avoid circular imports
	// InboundLeadForm data should be loaded separately using relations package if needed
	entity.InboundLeadForm = nil

	return entity, nil
}

// toContactEntity converts domain.Contact to ContactEntity response
func (h *Handler) toContactEntityInList(contact *domain.Contact) (ContactEntity, error) {
	var profile map[string]interface{}
	if err := json.Unmarshal(contact.Profile, &profile); err != nil {
		return ContactEntity{}, fmt.Errorf("Failed to parse profile: %w", err)
	}

	// Extract email and phone from profile
	var email, phone string
	if e, ok := profile["email"].(string); ok {
		email = e
	}
	if p, ok := profile["phone"].(string); ok {
		phone = p
	}

	entity := ContactEntity{
		Extra:          profile,
		ID:             contact.ID,
		UserID:         contact.UserID,
		OrganizationID: contact.OrganizationID,
		Name:           strPtrIfPresent(contact.Name),
		Email:          strPtrIfPresent(email),
		Phone:          strPtrIfPresent(phone),
		Company:        strPtrIfPresent(contact.Company),
		JobTitle:       strPtrIfPresent(contact.JobTitle),
		Address:        strPtrIfPresent(contact.Address),
		City:           strPtrIfPresent(contact.City),
		Country:        strPtrIfPresent(contact.Country),
		State:          strPtrIfPresent(contact.State),
		Zip:            strPtrIfPresent(contact.Zip),
		CreatedAt:      contact.CreatedAt,
		UpdatedAt:      contact.UpdatedAt,
	}

	if contact.SourceID != "" {
		entity.SourceID = &contact.SourceID
	}
	if contact.Source != "" {
		entity.Source = &contact.Source
	}
	if contact.HubspotID != nil && *contact.HubspotID != "" {
		entity.HubspotID = contact.HubspotID
	}

	// Parse tags if present
	if len(contact.Tags) > 0 {
		var tags map[string]interface{}
		if err := json.Unmarshal(contact.Tags, &tags); err == nil {
			entity.Tags = tags
		}
	}

	// Note: Direct relationships removed to avoid circular imports
	// InboundLeadForm data should be loaded separately using relations package if needed
	entity.InboundLeadForm = nil

	return entity, nil
}

// strPtrIfPresent returns a *string unless the value is empty or equals domain.NOT_AVAILABLE
func strPtrIfPresent(s string) *string {
	if s == "" || s == domain.NOT_AVAILABLE {
		return nil
	}
	return &s
}

// parsePagination supports both offset/limit and page-based query params.
// It also normalizes small offset values that likely represent a page index.
func parsePagination(c fiber.Ctx, defaultOffset, defaultLimit int) (int, int) {
	offset := defaultOffset
	limit := defaultLimit
	offsetProvided := false

	rawOffset := c.Query("raw_offset") == "true" || c.Query("offset_mode") == "raw"

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
	pageIndexStr := c.Query("page_index")
	if pageIndexStr == "" {
		pageIndexStr = c.Query("pageIndex")
	}
	if pageIndexStr != "" {
		if val, err := strconv.Atoi(pageIndexStr); err == nil && val >= 0 {
			offset = val * limit
		}
	} else if pageStr := c.Query("page"); pageStr != "" {
		if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
			offset = (val - 1) * limit
		}
	}

	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = defaultOffset
	}

	// Treat offset as page index by default (unless raw offset is explicitly requested).
	if !rawOffset && offsetProvided && offset > 0 && limit > 0 {
		offset = offset * limit
	}

	return offset, limit
}

// deepCopyFilter creates a deep copy of filter map
func deepCopyFilter(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// sanitizeFilter removes sensitive fields from filter
func sanitizeFilter(filter map[string]interface{}) {
	delete(filter, "password")
	delete(filter, "token")
	delete(filter, "api_key")
}

func (c ContactEntity) MarshalJSON() ([]byte, error) {
	type Alias ContactEntity
	base, err := json.Marshal(Alias(c))
	if err != nil {
		return nil, err
	}

	if c.Extra == nil {
		return base, nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(base, &m); err != nil {
		return nil, err
	}

	for k, v := range c.Extra {
		m[k] = v
	}

	return json.Marshal(m)
}
