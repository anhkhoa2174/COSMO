package contact

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Search handles POST /v1/contacts/search
// @Summary Search contacts
// @Description Searches contacts using legacy filter payloads and returns results matching the original Python API.
// @Tags Contacts
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit" default(25)
// @Param body body v1schema.ContactSearchRequest false "Search filters"
// @Success 200 {object} v1schema.ContactSearchResponse "Successfully retrieved contacts"
// @Failure 400 {object} any "Invalid request"
// @Failure 500 {object} any "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/search [post]
func (h *Handler) Search(c fiber.Ctx) error {
	_, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	offsetInt, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid offset value", err)
	}

	limitInt, err := strconv.Atoi(c.Query("limit", "25"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid limit value", err)
	}
	if limitInt <= 0 {
		limitInt = 25
	}
	if limitInt > 1000 {
		limitInt = 1000
	}

	var req v1schema.ContactSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	// Filter by organization_id only - all members can see all contacts in the org
	// The "Added By" field shows which BD added each contact.
	// The scope keys are set after the client's filter so it cannot override them.
	userFilter := req.Filter
	if userFilter == nil {
		userFilter = req.LegacyFilter
	}
	rawFilter := dropRawFilters(userFilter)
	rawFilter["is_deleted"] = false
	rawFilter["organization_id"] = organizationID

	// Normalize filter: convert text fields to ILIKE partial match
	filterMap := contactRepo.NormalizeContactFilter(rawFilter)

	pagination := &baseRepo.PaginationParams{
		Offset: offsetInt,
		Limit:  limitInt,
	}
	pagination.Validate()

	if segVal, ok := filterMap["segment_id"]; ok {
		delete(filterMap, "segment_id")
		segmentID, err := parseUUID(segVal)
		if err != nil {
			return h.responseHelper.BadRequest(c, "Invalid segment_id", err)
		}

		list, total, err := h.repo.FindBySegment(c.Context(), segmentID, filterMap, pagination)
		if err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to fetch contacts", err)
		}

		// Collect unique user IDs for "Added By" info
		segUserIDsMap := make(map[uuid.UUID]bool)
		for _, contact := range list {
			segUserIDsMap[contact.UserID] = true
		}
		var segUserIDs []uuid.UUID
		for uid := range segUserIDsMap {
			segUserIDs = append(segUserIDs, uid)
		}

		// Batch fetch users for "Added By" info
		segUserMap := make(map[uuid.UUID]struct{ Name, Email string })
		if len(segUserIDs) > 0 {
			users, err := h.userRepo.FindByIDs(c.Context(), segUserIDs)
			if err == nil {
				for _, u := range users {
					segUserMap[u.ID] = struct{ Name, Email string }{Name: u.Name, Email: u.Email}
				}
			}
		}

		items := make([]*v1schema.ContactListItem, len(list))
		for i := range list {
			items[i] = v1schema.ToContactListItem(list[i])
			// Add "Added By" info to entity
			if userInfo, ok := segUserMap[list[i].UserID]; ok {
				if userInfo.Name != "" {
					items[i].Entity["added_by_name"] = userInfo.Name
				}
				if userInfo.Email != "" {
					items[i].Entity["added_by_email"] = userInfo.Email
				}
			}
		}

		response := v1schema.ContactSearchResponse{
			List:   items,
			Total:  int64(total),
			Offset: pagination.Offset,
			Limit:  pagination.Limit,
		}

		return h.responseHelper.Success(c, response)
	}

	result, err := h.repo.FindAll(c.Context(), filterMap, pagination)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to fetch contacts", err)
	}

	// Collect unique user IDs for "Added By" info
	userIDsMap := make(map[uuid.UUID]bool)
	for i := range result.List {
		userIDsMap[result.List[i].UserID] = true
	}
	var userIDs []uuid.UUID
	for uid := range userIDsMap {
		userIDs = append(userIDs, uid)
	}

	// Batch fetch users for "Added By" info
	userMap := make(map[uuid.UUID]struct{ Name, Email string })
	if len(userIDs) > 0 {
		users, err := h.userRepo.FindByIDs(c.Context(), userIDs)
		if err == nil {
			for _, u := range users {
				userMap[u.ID] = struct{ Name, Email string }{Name: u.Name, Email: u.Email}
			}
		}
	}

	items := make([]*v1schema.ContactListItem, len(result.List))
	for i := range result.List {
		items[i] = v1schema.ToContactListItem(&result.List[i])
		// Add "Added By" info to entity
		if userInfo, ok := userMap[result.List[i].UserID]; ok {
			if userInfo.Name != "" {
				items[i].Entity["added_by_name"] = userInfo.Name
			}
			if userInfo.Email != "" {
				items[i].Entity["added_by_email"] = userInfo.Email
			}
		}
	}

	response := v1schema.ContactSearchResponse{
		List:   items,
		Total:  result.Total,
		Offset: result.Offset,
		Limit:  result.Limit,
	}

	return h.responseHelper.Success(c, response)
}

func parseUUID(value interface{}) (uuid.UUID, error) {
	switch v := value.(type) {
	case string:
		return uuid.Parse(v)
	case fmt.Stringer:
		return uuid.Parse(v.String())
	default:
		return uuid.Nil, fmt.Errorf("invalid UUID value")
	}
}

// dropRawFilters copies a client-supplied filter without the "$raw" operator,
// at any depth. FilterBuilder splices $raw into the WHERE clause verbatim, so
// accepting it from a client is SQL injection; the repository adds its own $raw
// expressions after this runs.
func dropRawFilters(filter map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(filter))
	for k, v := range filter {
		if k != string(baseRepo.OpRaw) {
			out[k] = dropRawValue(v)
		}
	}
	return out
}

func dropRawValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		return dropRawFilters(t)
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = dropRawValue(item)
		}
		return out
	}
	return v
}
