package email

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1util "github.com/rockship/cosmo-agents-go/internal/handler/v1/util"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// EmailHandler handles email-related HTTP requests
type EmailHandler struct {
	taskRepo *taskRepo.TaskRepository
	roleRepo *roleRepo.RoleRepository
}

// NewEmailHandler creates a new EmailHandler
func NewEmailHandler(taskRepo *taskRepo.TaskRepository, roleRepo *roleRepo.RoleRepository) *EmailHandler {
	return &EmailHandler{
		taskRepo: taskRepo,
		roleRepo: roleRepo,
	}
}

// GetByID handles GET /v1/emails/:id
// @Summary Get email by ID
// @Description Retrieves a single email by its unique ID
// @Tags Emails
// @Accept json
// @Produce json
// @Param id path string true "Email ID (UUID)"
// @Success 200 {object} schema.APIResponse[v1schema.EmailDetailResponse] "Successfully retrieved email"
// @Failure 400 {object} schema.APIResponse[any] "Invalid email ID"
// @Failure 404 {object} schema.APIResponse[any] "Email not found"
// @Security BearerAuth
// @Router /v1/emails/{id} [get]
func (h *EmailHandler) GetByID(c fiber.Ctx) error {
	userVal := c.Locals("user_id")
	userUUID, ok := userVal.(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	idParam := c.Params("id")
	emailID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid email ID", err.Error(),
		))
	}

	task, err := h.taskRepo.FindEmailTaskByID(c.Context(), emailID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch email", err.Error(),
		))
	}
	if task == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Email not found", "",
		))
	}

	// Note: Direct relationships removed to avoid circular imports
	// Campaign and Contact data should be loaded separately using relations package if needed

	allowed, err := h.canAccessEmail(c.Context(), task, userUUID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to verify access", err.Error(),
		))
	}
	if !allowed {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Email not found", "",
		))
	}

	detail, err := buildEmailDetailResponse(task)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to build response", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(detail))
}

// List handles GET /v1/emails
// @Summary List emails
// @Description Retrieves a paginated list of emails with optional filtering by status, organization, or recipient address
// @Tags Emails
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit (max 100)" default(50)
// @Param organization_id query string false "Filter by organization ID (UUID)"
// @Param to_email query string false "Filter by recipient email (ILIKE match)"
// @Param status query string false "Filter by email status"
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v1schema.EmailResponse]] "Successfully retrieved emails"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/emails [get]
func (h *EmailHandler) List(c fiber.Ctx) error {
	userVal := c.Locals("user_id")
	userUUID, ok := userVal.(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsed, err := strconv.Atoi(offsetParam); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	limit := 25
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	filter := taskRepo.EmailTaskFilter{Status: c.Query("status"), ToEmail: c.Query("to_email"), UserIDs: []uuid.UUID{userUUID}}
	if orgID := c.Query("organization_id"); orgID != "" {
		parsed, err := uuid.Parse(orgID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid organization_id", err.Error(),
			))
		}
		orgIDs, err := h.getAccessibleOrganizationIDs(c.Context(), userUUID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError, "Failed to load organizations", err.Error(),
			))
		}
		if v1util.ContainsUUID(orgIDs, parsed) {
			filter.OrganizationID = &parsed
		}
	} else {
		orgIDs, err := h.getAccessibleOrganizationIDs(c.Context(), userUUID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError, "Failed to load organizations", err.Error(),
			))
		}
		filter.OrganizationIDs = orgIDs
	}

	tasks, total, err := h.taskRepo.FindEmailTasks(c.Context(), filter, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch emails", err.Error(),
		))
	}

	items := make([]v1schema.EmailResponse, 0, len(tasks))
	for _, task := range tasks {
		res, buildErr := buildEmailListItem(task)
		if buildErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError, "Failed to build response", buildErr.Error(),
			))
		}
		items = append(items, res)
	}

	listing := schema.PaginatedResponse[v1schema.EmailResponse]{
		List:   items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}

	return c.JSON(schema.SuccessResponse(listing))
}

// Helper utilities

func buildEmailListItem(task *domain.Task) (v1schema.EmailResponse, error) {
	response := v1schema.EmailResponse{
		ID:         task.ID,
		ScheduleAt: task.ScheduleAt,
		DoneAt:     task.DoneAt,
		Status:     task.Status,
		Error:      task.Error,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
	}

	// Note: Direct relationships removed to avoid circular imports
	// Contact data should be loaded separately using relations package if needed

	payload, err := task.GetPayload()
	if err != nil {
		return response, err
	}

	if from := normalizeValue(extractPayloadString(payload, "agent.email")); from != "" {
		response.FromEmail = stringPtr(from)
	}

	// Note: Campaign relationship should be loaded separately using relations package if needed
	// For now, we'll build basic response without campaign-specific subject formatting

	return response, nil
}

func buildEmailDetailResponse(task *domain.Task) (v1schema.EmailDetailResponse, error) {
	payload, err := task.GetPayload()
	if err != nil {
		return v1schema.EmailDetailResponse{}, err
	}

	clientData := prepareClientData(task, payload)

	response := v1schema.EmailDetailResponse{
		ID:         task.ID,
		ScheduleAt: task.ScheduleAt,
		DoneAt:     task.DoneAt,
		Status:     task.Status,
		Error:      task.Error,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
	}

	// Note: Contact data should be loaded separately using relations package if needed
	// For now, we'll build basic response without contact email

	if from := normalizeValue(extractPayloadString(payload, "agent.email")); from != "" {
		response.FromEmail = stringPtr(from)
	}

	if subjectTemplate := extractPayloadString(payload, "template.subject"); strings.TrimSpace(subjectTemplate) != "" {
		formatted := formatTemplateWithClientData(subjectTemplate, clientData)
		if val := normalizeValue(formatted); val != "" {
			response.Subject = stringPtr(val)
		}
	}

	if contentTemplate := extractPayloadString(payload, "template.content"); strings.TrimSpace(contentTemplate) != "" {
		formatted := formatTemplateWithClientData(contentTemplate, clientData)
		if val := normalizeValue(formatted); val != "" {
			response.Content = stringPtr(val)
		}
	}

	return response, nil
}

func prepareClientData(task *domain.Task, payload map[string]any) map[string]string {
	clientData := make(map[string]string)

	if task == nil {
		return clientData
	}

	// Note: Campaign and Contact relationships should be loaded separately using relations package if needed
	// For now, we'll only use payload data

	// Process contact data from payload if available
	if contactData, exists := payload["contact"]; exists {
		if contactMap, ok := contactData.(map[string]any); ok {
			contactFields := []string{"first_name", "last_name", "email", "phone", "company", "job_title", "address", "city", "state", "country", "zip"}
			for _, field := range contactFields {
				if value, exists := contactMap[field]; exists {
					if str, ok := toString(value); ok {
						if normalized := normalizeValue(str); normalized != "" {
							clientData["contact_"+field] = normalized
						}
					}
				}
			}

			// Process profile if available
			if profileData, exists := contactMap["profile"]; exists {
				if profileMap, ok := profileData.(map[string]any); ok {
					for key, value := range profileMap {
						if shouldSkipContactKey(key) {
							continue
						}
						if str, ok := toString(value); ok {
							if normalized := normalizeValue(str); normalized != "" {
								clientData["contact_"+key] = normalized
							}
						}
					}
				}
			}
		}
	}

	agent := extractPayloadMap(payload, "agent")
	if email, ok := agent["email"].(string); ok {
		if normalized := normalizeValue(email); normalized != "" {
			clientData["sender_email"] = normalized
		}
	}
	if name, ok := agent["name"].(string); ok {
		if normalized := normalizeValue(name); normalized != "" {
			clientData["sender_name"] = normalized
		}
	}

	defaultSignature := "\nBest regard,"

	if signature, ok := agent["signature"].(string); ok && strings.TrimSpace(signature) != "" {
		formatted := formatTemplateWithClientData(signature, clientData)
		if normalized := normalizeValue(formatted); normalized != "" {
			clientData["agent_signature"] = normalized
		} else {
			clientData["agent_signature"] = defaultSignature
		}
	} else {
		clientData["agent_signature"] = defaultSignature
	}

	return clientData
}

func (h *EmailHandler) getAccessibleOrganizationIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if h.roleRepo == nil {
		return nil, nil
	}

	roles, err := h.roleRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	seen := make(map[uuid.UUID]struct{})
	orgIDs := make([]uuid.UUID, 0, len(roles))

	for _, role := range roles {
		if role.IsDeleted {
			continue
		}
		if role.Status != domain.RoleStatusActive {
			continue
		}
		if role.OrganizationID == uuid.Nil {
			continue
		}
		if _, exists := seen[role.OrganizationID]; exists {
			continue
		}
		seen[role.OrganizationID] = struct{}{}
		orgIDs = append(orgIDs, role.OrganizationID)
	}

	return orgIDs, nil
}

func (h *EmailHandler) canAccessEmail(ctx context.Context, task *domain.Task, userID uuid.UUID) (bool, error) {
	if task == nil {
		return false, nil
	}

	// Note: Direct relationships removed to avoid circular imports
	// Campaign data should be loaded separately using relations package if needed
	if task.CampaignID == uuid.Nil {
		return false, nil
	}

	// TODO: Implement proper campaign loading and permission checking using RelationsHelper
	// For now, simplify access check
	return true, nil
}

func formatTemplateWithClientData(template string, data map[string]string) string {
	if template == "" || len(data) == 0 {
		return template
	}

	result := template
	for key, value := range data {
		single := fmt.Sprintf("{%s}", key)
		double := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, single, value)
		result = strings.ReplaceAll(result, double, value)
	}
	return result
}

func extractPayloadString(payload map[string]any, path string) string {
	if value, ok := extractPayloadValue(payload, path); ok {
		if str, ok := toString(value); ok {
			return str
		}
	}
	return ""
}

func extractPayloadMap(payload map[string]any, path string) map[string]any {
	if value, ok := extractPayloadValue(payload, path); ok {
		if m, ok := value.(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

func extractPayloadValue(payload map[string]any, path string) (any, bool) {
	if payload == nil || len(payload) == 0 {
		return nil, false
	}

	current := any(payload)
	for _, part := range strings.Split(path, ".") {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, exists := asMap[part]
		if !exists || next == nil {
			return nil, false
		}
		current = next
	}
	return current, true
}

func toString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	case float32, float64, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, bool:
		return fmt.Sprint(v), true
	default:
		return "", false
	}
}

func normalizeValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.EqualFold(trimmed, domain.NOT_AVAILABLE) {
		return ""
	}
	return trimmed
}

func shouldSkipContactKey(key string) bool {
	switch key {
	case "id", "created_at", "updated_at", "hubspot_id", "source_id", "is_deleted":
		return true
	default:
		return false
	}
}

func stringPtr(value string) *string {
	v := value
	return &v
}
