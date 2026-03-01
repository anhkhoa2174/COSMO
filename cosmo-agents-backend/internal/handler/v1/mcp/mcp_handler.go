package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	agent "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	notificationRepo "github.com/rockship/cosmo-agents-go/internal/repository/notification"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
)

// MCPHandler handles MCP (Model Context Protocol) related endpoints
type MCPHandler struct {
	campaignRepo     *campaignRepo.CampaignRepository
	contactRepo      *contactRepo.ContactRepository
	listContactRepo  *contactRepo.ListContactRepository
	roleRepo         *roleRepo.RoleRepository
	notificationRepo *notificationRepo.NotificationRepository
	agentRepo        *agent.AgentRepository
	aiEmailService   *aiService.AIEmailService
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
) *MCPHandler {
	return &MCPHandler{
		campaignRepo:     campaignRepo,
		contactRepo:      contactRepo,
		listContactRepo:  listContactRepo,
		roleRepo:         roleRepo,
		notificationRepo: notificationRepo,
		agentRepo:        agentRepo,
		aiEmailService:   aiEmailService,
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
		if req.Phone != nil {
			var profile map[string]interface{}
			if len(existing.Profile) > 0 {
				_ = json.Unmarshal(existing.Profile, &profile)
			}
			if profile == nil {
				profile = make(map[string]interface{})
			}
			profile["phone"] = *req.Phone
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

	// Build profile with email and phone
	profile := make(map[string]interface{})
	if req.Email != "" {
		profile["email"] = req.Email
	}
	if req.Phone != nil && *req.Phone != "" {
		profile["phone"] = *req.Phone
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
