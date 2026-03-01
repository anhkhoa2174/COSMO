package google_ads

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	googleads "github.com/rockship/cosmo-agents-go/internal/handler/v1/google-ads"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	inboundLeadFormRepo "github.com/rockship/cosmo-agents-go/internal/repository/inbound_lead_form"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// GoogleAdsUseCase implements business logic for Google Ads operations
type GoogleAdsUseCase struct {
	inboundRepo  *inboundLeadFormRepo.InboundLeadFormRepository
	contactRepo  *contactRepo.ContactRepository
	listRepo     *contactRepo.ListContactRepository
	campaignRepo *campaignRepo.CampaignRepository
	workerClient *worker.Client
	baseURL      string
	log          *zerolog.Logger
}

// NewGoogleAdsUseCase creates a new GoogleAdsUseCase instance
func NewGoogleAdsUseCase(
	inboundRepo *inboundLeadFormRepo.InboundLeadFormRepository,
	contactRepo *contactRepo.ContactRepository,
	listRepo *contactRepo.ListContactRepository,
	campaignRepo *campaignRepo.CampaignRepository,
	workerClient *worker.Client,
	baseURL string,
) *GoogleAdsUseCase {
	l := logger.Logger
	return &GoogleAdsUseCase{
		inboundRepo:  inboundRepo,
		contactRepo:  contactRepo,
		listRepo:     listRepo,
		campaignRepo: campaignRepo,
		workerClient: workerClient,
		baseURL:      strings.TrimRight(baseURL, "/"),
		log:          &l,
	}
}

// GenerateWebhook creates a webhook endpoint and persists lead form metadata
func (uc *GoogleAdsUseCase) GenerateWebhook(
	ctx context.Context,
	req googleads.GenerateWebhookRequest,
) (*googleads.GenerateWebhookResponse, error) {
	// Ensure campaign exists
	campaign, err := uc.campaignRepo.FindWithRelations(ctx, req.CampaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to find campaign: %w", err)
	}
	if campaign == nil {
		return nil, googleads.ErrCampaignNotFound
	}

	// Ensure slug is unique
	existing, err := uc.inboundRepo.FindBySlug(ctx, req.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if existing != nil {
		return nil, googleads.ErrWebhookSlugExists
	}

	// Ensure contact list exists
	listContact, err := uc.listRepo.FindByID(ctx, req.ContactListID)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact list: %w", err)
	}
	if listContact == nil {
		return nil, googleads.ErrContactListNotFound
	}

	// Create metadata
	metadata := map[string]string{
		"campaign_id":     req.CampaignID.String(),
		"contact_list_id": req.ContactListID.String(),
	}

	var uiMetadata domain.JSONB
	if err := uiMetadata.Marshal(metadata); err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create inbound lead form
	form := &domain.InboundLeadForm{
		Name:       req.Name,
		Slug:       req.Slug,
		UIMetadata: uiMetadata,
	}

	if req.UserID != uuid.Nil {
		form.UserID = &req.UserID
	}

	if err := uc.inboundRepo.CreateWithListContact(ctx, form, []uuid.UUID{listContact.ID}); err != nil {
		return nil, fmt.Errorf("failed to create inbound lead form: %w", err)
	}

	// Generate webhook URL
	webhookURL := fmt.Sprintf("%s/v1/google-ads/%s/webhook", uc.baseURL, form.Slug)

	return &googleads.GenerateWebhookResponse{
		WebhookURL: webhookURL,
	}, nil
}

// ProcessWebhook ingests Google Ads leads and triggers campaign outreach
func (uc *GoogleAdsUseCase) ProcessWebhook(
	ctx context.Context,
	req googleads.ProcessWebhookRequest,
) error {
	// Find inbound form by slug
	form, err := uc.inboundRepo.FindBySlug(ctx, req.Slug)
	if err != nil {
		return fmt.Errorf("failed to find inbound form: %w", err)
	}
	if form == nil {
		uc.log.Error().Str("slug", req.Slug).Msg("google ads form not found")
		return googleads.ErrInboundFormNotFound
	}

	// Parse metadata
	meta := map[string]string{}
	if len(form.UIMetadata) > 0 {
		if err := form.UIMetadata.Unmarshal(&meta); err != nil {
			uc.log.Error().Err(err).Str("slug", req.Slug).Msg("failed to parse form metadata")
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	campaignID, err := uuid.Parse(meta["campaign_id"])
	if err != nil {
		uc.log.Error().Err(err).Str("slug", req.Slug).Msg("campaign_id missing in metadata")
		return fmt.Errorf("invalid campaign_id in metadata: %w", err)
	}

	listContactID, err := uuid.Parse(meta["contact_list_id"])
	if err != nil {
		uc.log.Error().Err(err).Str("slug", req.Slug).Msg("contact_list_id missing in metadata")
		return fmt.Errorf("invalid contact_list_id in metadata: %w", err)
	}

	// Find campaign
	campaign, err := uc.campaignRepo.FindWithRelations(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to find campaign: %w", err)
	}
	if campaign == nil {
		uc.log.Error().Str("campaign_id", campaignID.String()).Msg("campaign not found for google ads lead")
		return googleads.ErrCampaignNotFound
	}

	// Validate campaign status
	if campaign.Status != domain.CampaignStatusActive && campaign.Status != domain.CampaignStatusScheduled {
		uc.log.Warn().
			Str("campaign_id", campaign.ID.String()).
			Str("status", string(campaign.Status)).
			Msg("campaign not active for google ads lead")
		return googleads.ErrCampaignInactive
	}

	// Validate campaign has agent
	if campaign.AgentID == nil {
		uc.log.Error().Str("campaign_id", campaign.ID.String()).Msg("campaign missing agent for google ads lead")
		return googleads.ErrMissingAgent
	}

	// Map lead data
	contactData := uc.mapLeadData(req.UserColumnData)
	if contactData["email"] == "" {
		uc.log.Warn().Str("lead_id", req.LeadID).Msg("google ads payload missing email")
		return nil // Not an error, just skip
	}

	// Create contact - combine first_name and last_name into name
	firstName := uc.fallback(contactData["first_name"])
	lastName := uc.fallback(contactData["last_name"])
	name := strings.TrimSpace(firstName + " " + lastName)
	if name == "" || name == "N/A N/A" {
		name = uc.fallback(contactData["name"])
	}

	// Build profile with email and phone
	email := strings.TrimSpace(contactData["email"])
	phone := uc.fallback(contactData["phone"])
	profile := make(map[string]interface{})
	if email != "" && email != "N/A" {
		profile["email"] = email
	}
	if phone != "" && phone != "N/A" {
		profile["phone"] = phone
	}
	var profileBytes domain.JSONB
	if len(profile) > 0 {
		_ = profileBytes.Marshal(profile)
	}

	contact := &domain.Contact{
		UserID:         campaign.UserID,
		Source:         string(domain.ContactSourceGoogleAds),
		SourceID:       req.LeadID,
		Name:           name,
		Profile:        profileBytes,
		Company:        uc.fallback(contactData["company"]),
		JobTitle:       uc.fallback(contactData["job_title"]),
		Address:        uc.fallback(contactData["address"]),
		City:           uc.fallback(contactData["city"]),
		Country:        uc.fallback(contactData["country"]),
		State:          uc.fallback(contactData["state"]),
		Zip:            uc.fallback(contactData["zip"]),
		OrganizationID: campaign.OrganizationID,
	}

	// Upsert contact
	storedContact, err := uc.contactRepo.UpsertBySource(ctx, contact)
	if err != nil {
		return fmt.Errorf("failed to upsert contact: %w", err)
	}

	// Add contact to list
	if err := uc.contactRepo.AddToList(ctx, storedContact.ID, listContactID); err != nil {
		uc.log.Error().
			Err(err).
			Str("contact_id", storedContact.ID.String()).
			Msg("failed to associate contact with list")
	}

	// Enqueue campaign execution
	payload := executeCampaignPayload{
		CampaignID: campaign.ID,
		UserID:     campaign.UserID,
		AgentID:    *campaign.AgentID,
		ContactIDs: []uuid.UUID{storedContact.ID},
	}

	if _, err := uc.workerClient.EnqueueCriticalTask(ctx, worker.TypeExecuteCampaign, payload); err != nil {
		uc.log.Error().
			Err(err).
			Str("campaign_id", campaign.ID.String()).
			Str("contact_id", storedContact.ID.String()).
			Msg("failed to enqueue campaign execution for google ads lead")
		return fmt.Errorf("failed to enqueue campaign execution: %w", err)
	}

	return nil
}

// executeCampaignPayload represents the payload for campaign execution
type executeCampaignPayload struct {
	CampaignID uuid.UUID   `json:"campaign_id"`
	UserID     uuid.UUID   `json:"user_id"`
	AgentID    uuid.UUID   `json:"agent_id"`
	ContactIDs []uuid.UUID `json:"contact_ids,omitempty"`
}

// systemFieldMap maps Google Ads column IDs to contact fields
var systemFieldMap = map[string]string{
	"EMAIL":          "email",
	"FIRST_NAME":     "first_name",
	"FULL_NAME":      "first_name",
	"LAST_NAME":      "last_name",
	"PHONE_NUMBER":   "phone",
	"COMPANY_NAME":   "company",
	"JOB_TITLE":      "job_title",
	"STREET_ADDRESS": "address",
	"CITY":           "city",
	"COUNTRY":        "country",
	"REGION":         "state",
	"POSTAL_CODE":    "zip",
}

// mapLeadData converts Google Ads column data to contact field map
func (uc *GoogleAdsUseCase) mapLeadData(columns []googleads.ColumnEntry) map[string]string {
	result := make(map[string]string)
	for _, column := range columns {
		fieldName, ok := systemFieldMap[strings.ToUpper(column.ColumnID)]
		if !ok {
			continue
		}
		if column.StringValue == "" {
			continue
		}
		result[fieldName] = column.StringValue
	}
	return result
}

// fallback returns domain.NOT_AVAILABLE if value is empty
func (uc *GoogleAdsUseCase) fallback(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return domain.NOT_AVAILABLE
	}
	return value
}
