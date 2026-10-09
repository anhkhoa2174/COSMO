package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	dailyActionDomain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	agent "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	dailyActionRepo "github.com/rockship/cosmo-agents-go/internal/repository/daily_action"
	notificationRepo "github.com/rockship/cosmo-agents-go/internal/repository/notification"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
	dailyActionSvc "github.com/rockship/cosmo-agents-go/internal/service/daily_action"
	outreachSvc "github.com/rockship/cosmo-agents-go/internal/service/outreach"
)

// MCPHandler handles MCP (Model Context Protocol) related endpoints
type MCPHandler struct {
	campaignRepo       *campaignRepo.CampaignRepository
	contactRepo        *contactRepo.ContactRepository
	listContactRepo    *contactRepo.ListContactRepository
	roleRepo           *roleRepo.RoleRepository
	notificationRepo   *notificationRepo.NotificationRepository
	agentRepo          *agent.AgentRepository
	aiEmailService     *aiService.AIEmailService
	outreachService    *outreachSvc.Service
	dailyActionService *dailyActionSvc.Service
	interactionLogRepo *outreachRepo.InteractionLogRepository
	generationRepo     *dailyActionRepo.GenerationRepository
	actionRepo         *dailyActionRepo.ActionRepository
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(
	campaignRepo *campaignRepo.CampaignRepository,
	contactRepo *contactRepo.ContactRepository,
	listContactRepo *contactRepo.ListContactRepository,
	roleRepo *roleRepo.RoleRepository,
	notificationRepo *notificationRepo.NotificationRepository,
	agentRepo *agent.AgentRepository,
	aiEmailService *aiService.AIEmailService,
	outreachService *outreachSvc.Service,
	dailyActionService *dailyActionSvc.Service,
	interactionLogRepo *outreachRepo.InteractionLogRepository,
	generationRepo *dailyActionRepo.GenerationRepository,
	actionRepo *dailyActionRepo.ActionRepository,
) *MCPHandler {
	return &MCPHandler{
		campaignRepo:       campaignRepo,
		contactRepo:        contactRepo,
		listContactRepo:    listContactRepo,
		roleRepo:           roleRepo,
		notificationRepo:   notificationRepo,
		agentRepo:          agentRepo,
		aiEmailService:     aiEmailService,
		outreachService:    outreachService,
		dailyActionService: dailyActionService,
		interactionLogRepo: interactionLogRepo,
		generationRepo:     generationRepo,
		actionRepo:         actionRepo,
	}
}

// CreateCampaignWithContacts handles POST /v1/mcp/campaigns
// Creates a campaign with a list of contacts in a single API call
// @Summary Create Campaign with Contacts (MCP)
// @Description Create a new campaign with an associated contact list in a single operation. This endpoint is designed for Model Context Protocol integration.
// @Tags MCP
// @Accept json
// @Produce json
// @Param body body v1schema.MCPCampaignCreateRequest true "MCP Campaign Creation Request"
// @Security BearerAuth
// @Success 201 {object} schema.APIResponse[v1schema.MCPCampaignCreateResponse] "Campaign created successfully"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/mcp/campaigns [post]
func (h *MCPHandler) CreateCampaignWithContacts(c fiber.Ctx) error {
	// Extract user ID from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(
			schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", nil),
		)
	}

	ctx := c.Context()

	// Parse request
	var req v1schema.MCPCampaignCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid request body", err.Error()),
		)
	}

	// Validate request
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Validation failed", err.Error()),
		)
	}

	// Get user's organization
	organizationID, err := h.roleRepo.FindPrimaryOrganization(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to determine organization", err.Error()),
		)
	}

	if organizationID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "User must belong to an organization", nil),
		)
	}

	// Generate campaign name from playbook if not provided
	campaignName := req.Name
	if campaignName == nil {
		name := generateCampaignNameFromPlaybook(req.Playbook)
		campaignName = &name
	}

	// Step 1: Create contact list
	listContact, contactsCreated, err := h.createContactListWithContacts(ctx, userID, organizationID, *campaignName, req.Contacts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to create contact list", err.Error()),
		)
	}

	// Step 2: Create campaign
	campaign := &domain.Campaign{
		UserID:         userID,
		OrganizationID: organizationID,
		Name:           *campaignName,
		Playbook:       req.Playbook,
		ListContactID:  &listContact.ID,
		Status:         domain.CampaignStatusDraft,
	}

	created, err := h.campaignRepo.Create(ctx, campaign)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to create campaign", err.Error()),
		)
	}

	// Step 3: Create notification for user
	notification := &domain.Notification{
		UserID:     userID,
		CampaignID: created.ID,
	}
	if _, err := h.notificationRepo.Create(ctx, notification); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to create notification for campaign %s: %v\n", created.ID, err)
	}

	// Step 4: Trigger AI template generation (async)
	go h.generateCampaignTemplates(context.Background(), created.ID, userID, *organizationID)

	// Build campaign URL
	campaignURL := fmt.Sprintf("https://cosmoagents.ai/campaigns/%s", created.ID)

	response := v1schema.MCPCampaignCreateResponse{
		ID:              created.ID,
		CampaignURL:     campaignURL,
		Name:            &created.Name,
		ContactListID:   &listContact.ID,
		Playbook:        &created.Playbook,
		ContactsCreated: contactsCreated,
	}

	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(response))
}

// createContactListWithContacts creates a contact list and associates contacts with it
func (h *MCPHandler) createContactListWithContacts(
	ctx context.Context,
	userID uuid.UUID,
	organizationID *uuid.UUID,
	listName string,
	contactRequests []v1schema.MCPContactCreateRequest,
) (*domain.ListContact, int, error) {
	// Create contact list
	listContact := &domain.ListContact{
		UserID:         userID,
		OrganizationID: organizationID,
		Name:           listName,
		Source:         string(domain.ContactSourceCosmoAgents),
	}

	created, err := h.listContactRepo.Create(ctx, listContact)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create contact list: %w", err)
	}

	// Create or update contacts
	contactsCreated := 0
	for _, contactReq := range contactRequests {
		contact, err := h.createOrUpdateContact(ctx, userID, organizationID, contactReq)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create contact: %w", err)
		}

		// Associate contact with list
		if err := h.contactRepo.AddToList(ctx, contact.ID, created.ID); err != nil {
			return nil, 0, fmt.Errorf("failed to associate contact with list: %w", err)
		}

		contactsCreated++
	}

	return created, contactsCreated, nil
}

// createOrUpdateContact creates a new contact or updates if exists
func (h *MCPHandler) createOrUpdateContact(
	ctx context.Context,
	userID uuid.UUID,
	organizationID *uuid.UUID,
	req v1schema.MCPContactCreateRequest,
) (*domain.Contact, error) {
	// Check if contact already exists by email
	existing, err := h.contactRepo.FindByEmail(ctx, userID, req.Email)
	if err != nil {
		return nil, err
	}

	// Marshal tags to JSON
	var tagsJSON []byte
	if req.Tags != nil {
		tagsJSON, err = json.Marshal(req.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
	}

	if existing != nil {
		// Update existing contact
		existing.Name = req.Name
		// Update phone in profile (email and phone are now stored in profile JSONB)
		if req.Phone != nil || req.LinkedInURL != nil {
			var profile map[string]interface{}
			if len(existing.Profile) > 0 {
				_ = json.Unmarshal(existing.Profile, &profile)
			}
			if profile == nil {
				profile = make(map[string]interface{})
			}
			if req.Phone != nil {
				profile["phone"] = *req.Phone
			}
			if req.LinkedInURL != nil {
				profile["linkedin_url"] = *req.LinkedInURL
			}
			if profileBytes, err := json.Marshal(profile); err == nil {
				existing.Profile = profileBytes
			}
		}
		if req.Company != nil {
			existing.Company = *req.Company
		}
		if req.JobTitle != nil {
			existing.JobTitle = *req.JobTitle
		}
		if req.Address != nil {
			existing.Address = *req.Address
		}
		if req.City != nil {
			existing.City = *req.City
		}
		if req.Country != nil {
			existing.Country = *req.Country
		}
		if req.State != nil {
			existing.State = *req.State
		}
		if req.Zip != nil {
			existing.Zip = *req.Zip
		}
		if tagsJSON != nil {
			existing.Tags = tagsJSON
		}

		if err := h.contactRepo.Update(ctx, existing.ID, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// Build profile with email, phone, and linkedin
	profile := make(map[string]interface{})
	if req.Email != "" {
		profile["email"] = req.Email
	}
	if req.Phone != nil && *req.Phone != "" {
		profile["phone"] = *req.Phone
	}
	if req.LinkedInURL != nil && *req.LinkedInURL != "" {
		profile["linkedin_url"] = *req.LinkedInURL
	}
	var profileBytes domain.JSONB
	if len(profile) > 0 {
		_ = profileBytes.Marshal(profile)
	}

	// Create new contact - MCP contacts come from Apollo
	// For Apollo contacts, contact_information is the email
	contact := &domain.Contact{
		UserID:             userID,
		OrganizationID:     organizationID,
		Name:               req.Name,
		Profile:            profileBytes,
		Source:             string(domain.ContactSourceApollo),
		SourceID:           uuid.New().String(),
		ContactInformation: req.Email,
	}

	if req.Company != nil {
		contact.Company = *req.Company
	}
	if req.JobTitle != nil {
		contact.JobTitle = *req.JobTitle
	}
	if req.Address != nil {
		contact.Address = *req.Address
	}
	if req.City != nil {
		contact.City = *req.City
	}
	if req.Country != nil {
		contact.Country = *req.Country
	}
	if req.State != nil {
		contact.State = *req.State
	}
	if req.Zip != nil {
		contact.Zip = *req.Zip
	}
	if tagsJSON != nil {
		contact.Tags = tagsJSON
	}

	err = h.contactRepo.Create(ctx, contact)
	if err != nil {
		return nil, err
	}

	return contact, nil
}

// generateCampaignTemplates triggers AI template generation for the campaign
func (h *MCPHandler) generateCampaignTemplates(ctx context.Context, campaignID, userID, organizationID uuid.UUID) {
	// Add a small delay to ensure campaign is fully committed
	time.Sleep(5 * time.Second)

	campaign, err := h.campaignRepo.FindByID(ctx, campaignID)
	if err != nil {
		fmt.Printf("Error loading campaign %s for template generation: %v\n", campaignID, err)
		return
	}

	if campaign == nil {
		fmt.Printf("Campaign %s not found for template generation\n", campaignID)
		return
	}

	// Note: Campaign.Agent relationship removed to avoid circular imports
	// Agent data should be loaded separately using relations package if needed

	// Generate templates using AI service
	// Note: Placeholder for AI template generation
	// TODO: Implement actual AI template generation service method
	if h.aiEmailService != nil {
		fmt.Printf("AI template generation would be triggered for campaign %s\n", campaignID)
		// if err := h.aiEmailService.GenerateTemplates(ctx, campaign); err != nil {
		// 	fmt.Printf("Error generating templates for campaign %s: %v\n", campaignID, err)
		// }
	}
}

// ============================================
// AI-Native MCP Tools (Agentic Daily Actions)
// ============================================

// getMCPUserID extracts user_id from fiber context (consistent with existing MCP pattern).
func getMCPUserID(c fiber.Ctx) (uuid.UUID, error) {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("unauthorized")
	}
	return userID, nil
}

// GetPipelineSummary returns counts of the caller's contacts by outreach
// stage, follow-up depth and staleness.
// GET /v1/mcp/contacts/pipeline-summary
func (h *MCPHandler) GetPipelineSummary(c fiber.Ctx) error {
	userID, err := getMCPUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", nil))
	}

	summary, err := h.outreachService.PipelineSummary(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to summarise pipeline", err.Error()))
	}

	return c.JSON(schema.SuccessResponse(summary))
}

// GetContactsPipeline returns contacts with their outreach pipeline states for AI analysis.
// GET /v1/mcp/contacts/pipeline?limit=50&type=mixed
func (h *MCPHandler) GetContactsPipeline(c fiber.Ctx) error {
	userID, err := getMCPUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", nil))
	}

	orgID, err := h.roleRepo.FindPrimaryOrganization(c.Context(), userID)
	if err != nil || orgID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, "User must belong to an organization", nil))
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	suggestType := c.Query("type", "mixed")

	suggestions, err := h.outreachService.SuggestContacts(c.Context(), userID, *orgID, false, suggestType, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to fetch pipeline contacts", err.Error()))
	}

	items := make([]v1schema.MCPContactPipelineItem, 0, len(suggestions.Suggestions))
	for _, sc := range suggestions.Suggestions {
		item := v1schema.MCPContactPipelineItem{
			ContactID:     sc.Contact.ID,
			Name:          sc.Contact.Name,
			Email:         sc.Contact.ContactInformation,
			Company:       sc.Contact.Company,
			JobTitle:      sc.Contact.JobTitle,
			Industry:      sc.Contact.Industry,
			Source:        sc.Contact.Source,
			OutreachStage: sc.Contact.OutreachStage,
			BusinessStage: sc.Contact.BusinessStage,
			NextStep:      sc.NextStep,
			DaysSinceContact: sc.DaysSince,
			ContextLevel:  sc.Contact.ContextLevel,
			MessageDraft:  sc.MessageDraft,
			Type:          sc.Type,
		}
		if sc.State != nil {
			item.ConversationState = sc.State.ConversationState
			item.FollowupCount = sc.State.FollowupCount
		}
		items = append(items, item)
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(v1schema.MCPContactsPipelineResponse{
		Contacts: items,
		Total:    suggestions.Total,
	}))
}

// GetOutcomeMetrics returns performance metrics for AI to learn from.
// GET /v1/mcp/outcome-metrics?period=30d
func (h *MCPHandler) GetOutcomeMetrics(c fiber.Ctx) error {
	userID, err := getMCPUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", nil))
	}

	period := c.Query("period", "30d")

	metrics, err := h.dailyActionService.GetOutcomeMetrics(c.Context(), userID, period)
	if err != nil {
		// Return empty metrics if not computed yet
		return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(map[string]interface{}{
			"user_id":            userID,
			"period":             period,
			"total_sent":         0,
			"total_replied":      0,
			"total_meetings":     0,
			"reply_rate_overall": 0,
			"message":            "No metrics computed yet. Metrics are generated after outreach actions are completed.",
		}))
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(metrics))
}

// GetContactInteractions returns interaction history for a specific contact.
// GET /v1/mcp/contacts/:contact_id/interactions?limit=20
func (h *MCPHandler) GetContactInteractions(c fiber.Ctx) error {
	userID, err := getMCPUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", nil))
	}

	contactID, err := uuid.Parse(c.Params("contact_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, "Invalid contact_id", nil))
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Verify contact belongs to user
	contact, err := h.contactRepo.GetByID(c.Context(), contactID)
	if err != nil || contact == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(fiber.StatusNotFound, "Contact not found", nil))
	}
	if contact.UserID != userID {
		orgID, _ := h.roleRepo.FindPrimaryOrganization(c.Context(), userID)
		if orgID == nil || contact.OrganizationID == nil || *contact.OrganizationID != *orgID {
			return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(fiber.StatusForbidden, "Access denied", nil))
		}
	}

	logs, err := h.interactionLogRepo.FindByContactIDAndUserID(c.Context(), contactID, userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to fetch interactions", err.Error()))
	}

	items := make([]v1schema.MCPInteractionItem, 0, len(logs))
	for _, log := range logs {
		items = append(items, v1schema.MCPInteractionItem{
			ID:        log.ID,
			Channel:   log.Channel,
			Direction: log.Direction,
			Content:   log.Content,
			Subject:   log.Subject,
			Sentiment: log.Sentiment,
			Timestamp: log.Timestamp.Format(time.RFC3339),
		})
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(v1schema.MCPContactInteractionsResponse{
		ContactID:    contactID,
		ContactName:  contact.Name,
		Interactions: items,
		Total:        len(items),
	}))
}

// CreateDailyActions creates prioritized daily actions from AI-generated recommendations.
// POST /v1/mcp/daily-actions
func (h *MCPHandler) CreateDailyActions(c fiber.Ctx) error {
	userID, err := getMCPUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", nil))
	}

	var req dailyActionDomain.CreateFromAgentRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, "Invalid request body", err.Error()))
	}

	if len(req.Recommendations) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, "No recommendations provided", nil))
	}

	generation, err := h.dailyActionService.CreateFromAgentRecommendations(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to create daily actions", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(map[string]interface{}{
		"generation_id": generation.ID,
		"action_count":  generation.ActionCount,
		"status":        generation.Status,
		"message":       fmt.Sprintf("Successfully created %d daily actions from AI recommendations", generation.ActionCount),
	}))
}

// GetDailyActionsStatus returns the current daily actions briefing.
// GET /v1/mcp/daily-actions/status?language=vi
func (h *MCPHandler) GetDailyActionsStatus(c fiber.Ctx) error {
	userID, err := getMCPUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", nil))
	}

	today := time.Now().Format("2006-01-02")

	gen, err := h.generationRepo.FindActiveByUserAndDate(c.Context(), userID, today)
	if err != nil || gen == nil {
		return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(v1schema.MCPDailyActionsStatusResponse{
			Date:    today,
			Status:  "no_generation",
			Actions: []v1schema.MCPDailyActionsStatusItem{},
		}))
	}

	actions, err := h.actionRepo.FindByGenerationID(c.Context(), gen.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to fetch actions", err.Error()))
	}

	completed := 0
	pending := 0
	items := make([]v1schema.MCPDailyActionsStatusItem, 0, len(actions))
	for _, a := range actions {
		var snapshot dailyActionDomain.ContactSnapshot
		_ = a.ContactSnapshot.Unmarshal(&snapshot)

		items = append(items, v1schema.MCPDailyActionsStatusItem{
			ActionID:   a.ID,
			ContactID:  a.ContactID,
			Name:       snapshot.Name,
			Company:    snapshot.Company,
			ActionType: string(a.Type),
			CategoryID: string(a.CategoryID),
			Priority:   a.Priority,
			Status:     string(a.Status),
			Reasoning:  a.Reasoning,
		})

		switch a.Status {
		case dailyActionDomain.ActionStatusCompleted, dailyActionDomain.ActionStatusSkipped:
			completed++
		default:
			pending++
		}
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(v1schema.MCPDailyActionsStatusResponse{
		GenerationID: &gen.ID,
		Date:         today,
		Status:       string(gen.Status),
		Actions:      items,
		TotalActions: len(items),
		Completed:    completed,
		Pending:      pending,
	}))
}

// generateCampaignNameFromPlaybook converts playbook string to title case name
func generateCampaignNameFromPlaybook(playbook string) string {
	// Replace hyphens and underscores with spaces
	name := strings.ReplaceAll(playbook, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Convert to title case
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}
