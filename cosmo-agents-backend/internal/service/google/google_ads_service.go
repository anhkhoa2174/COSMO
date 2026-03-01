package google

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	inboundLeadFormRepo "github.com/rockship/cosmo-agents-go/internal/repository/inbound_lead_form"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

var (
	// ErrCampaignNotFound indicates the campaign is missing.
	ErrCampaignNotFound = errors.New("campaign not found")
	// ErrWebhookSlugExists indicates the slug is already in use.
	ErrWebhookSlugExists = errors.New("webhook slug already exists")
	// ErrCampaignInactive indicates campaign is not active/scheduled.
	ErrCampaignInactive = errors.New("campaign is not running")
	// ErrMissingAgent indicates campaign lacks an assigned agent.
	ErrMissingAgent = errors.New("campaign missing agent")
)

// GoogleAdsService handles webhook management and lead ingestion.
type GoogleAdsService struct {
	inboundRepo  *inboundLeadFormRepo.InboundLeadFormRepository
	contactRepo  *contactRepo.ContactRepository
	listRepo     *contactRepo.ListContactRepository
	campaignRepo *campaignRepo.CampaignRepository
	workerClient *worker.Client
	baseURL      string
	log          *zerolog.Logger
}

// NewGoogleAdsService creates a new service instance.
func NewGoogleAdsService(
	inboundRepo *inboundLeadFormRepo.InboundLeadFormRepository,
	contactRepo *contactRepo.ContactRepository,
	listRepo *contactRepo.ListContactRepository,
	campaignRepo *campaignRepo.CampaignRepository,
	workerClient *worker.Client,
	baseURL string,
) *GoogleAdsService {
	l := logger.Logger
	return &GoogleAdsService{
		inboundRepo:  inboundRepo,
		contactRepo:  contactRepo,
		listRepo:     listRepo,
		campaignRepo: campaignRepo,
		workerClient: workerClient,
		baseURL:      strings.TrimRight(baseURL, "/"),
		log:          &l,
	}
}

// CreateWebhookRequest captures webhook creation params.
type CreateWebhookRequest struct {
	CampaignID    uuid.UUID
	ContactListID uuid.UUID
	Name          string
	Slug          string
	UserID        uuid.UUID
}

// GenerateWebhook creates a webhook endpoint and persists lead form metadata.
func (s *GoogleAdsService) GenerateWebhook(ctx context.Context, req CreateWebhookRequest) (string, error) {
	// ensure campaign exists
	campaign, err := s.campaignRepo.FindWithRelations(ctx, req.CampaignID)
	if err != nil {
		return "", err
	}
	if campaign == nil {
		return "", ErrCampaignNotFound
	}

	// ensure slug unique
	existing, err := s.inboundRepo.FindBySlug(ctx, req.Slug)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return "", ErrWebhookSlugExists
	}

	// ensure list contact exists
	listContact, err := s.listRepo.FindByID(ctx, req.ContactListID)
	if err != nil {
		return "", err
	}
	if listContact == nil {
		return "", fmt.Errorf("contact list %s not found", req.ContactListID)
	}

	metadata := map[string]string{
		"campaign_id":     req.CampaignID.String(),
		"contact_list_id": req.ContactListID.String(),
	}

	var uiMetadata domain.JSONB
	if err := uiMetadata.Marshal(metadata); err != nil {
		return "", fmt.Errorf("marshal ui metadata: %w", err)
	}

	form := &domain.InboundLeadForm{
		Name:       req.Name,
		Slug:       req.Slug,
		UIMetadata: uiMetadata,
	}

	if req.UserID != uuid.Nil {
		form.UserID = &req.UserID
	}

	if err := s.inboundRepo.CreateWithListContact(ctx, form, []uuid.UUID{listContact.ID}); err != nil {
		return "", err
	}

	webhookURL := fmt.Sprintf("%s/v1/google-ads/%s/webhook", s.baseURL, form.Slug)
	return webhookURL, nil
}

// GoogleAdsLeadPayload represents webhook payload fields.
type GoogleAdsLeadPayload struct {
	LeadID         string                 `json:"lead_id"`
	UserColumnData []GoogleAdsColumnEntry `json:"user_column_data"`
}

// GoogleAdsColumnEntry maps Google Ads column data.
type GoogleAdsColumnEntry struct {
	ColumnID    string `json:"column_id"`
	StringValue string `json:"string_value"`
}

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

type executeCampaignPayload struct {
	CampaignID uuid.UUID   `json:"campaign_id"`
	UserID     uuid.UUID   `json:"user_id"`
	AgentID    uuid.UUID   `json:"agent_id"`
	ContactIDs []uuid.UUID `json:"contact_ids,omitempty"`
}

// ProcessWebhook ingests Google Ads leads and triggers campaign outreach.
func (s *GoogleAdsService) ProcessWebhook(ctx context.Context, slug string, payload GoogleAdsLeadPayload) error {
	form, err := s.inboundRepo.FindBySlug(ctx, slug)
	if err != nil {
		return err
	}
	if form == nil {
		s.log.Error().Str("slug", slug).Msg("google ads form not found")
		return nil
	}

	meta := map[string]string{}
	if len(form.UIMetadata) > 0 {
		if err := form.UIMetadata.Unmarshal(&meta); err != nil {
			s.log.Error().Err(err).Str("slug", slug).Msg("failed to parse form metadata")
		}
	}

	campaignID, err := uuid.Parse(meta["campaign_id"])
	if err != nil {
		s.log.Error().Err(err).Str("slug", slug).Msg("campaign_id missing in metadata")
		return nil
	}

	listContactID, err := uuid.Parse(meta["contact_list_id"])
	if err != nil {
		s.log.Error().Err(err).Str("slug", slug).Msg("contact_list_id missing in metadata")
		return nil
	}

	campaign, err := s.campaignRepo.FindWithRelations(ctx, campaignID)
	if err != nil {
		return err
	}
	if campaign == nil {
		s.log.Error().Str("campaign_id", campaignID.String()).Msg("campaign not found for google ads lead")
		return nil
	}

	if campaign.Status != domain.CampaignStatusActive && campaign.Status != domain.CampaignStatusScheduled {
		s.log.Warn().Str("campaign_id", campaign.ID.String()).Str("status", string(campaign.Status)).Msg("campaign not active for google ads lead")
		return ErrCampaignInactive
	}

	if campaign.AgentID == nil {
		s.log.Error().Str("campaign_id", campaign.ID.String()).Msg("campaign missing agent for google ads lead")
		return ErrMissingAgent
	}

	contactData := mapLeadData(payload)
	if contactData["email"] == "" {
		s.log.Warn().Str("lead_id", payload.LeadID).Msg("google ads payload missing email")
		return nil
	}

	// Combine first_name and last_name into name
	firstName := fallback(contactData["first_name"])
	lastName := fallback(contactData["last_name"])
	name := strings.TrimSpace(firstName + " " + lastName)
	if name == "" {
		name = fallback(contactData["name"])
	}

	// Build profile with email and phone
	email := strings.TrimSpace(contactData["email"])
	phone := fallback(contactData["phone"])
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
		SourceID:       payload.LeadID,
		Name:           name,
		Profile:        profileBytes,
		Company:        fallback(contactData["company"]),
		JobTitle:       fallback(contactData["job_title"]),
		Address:        fallback(contactData["address"]),
		City:           fallback(contactData["city"]),
		Country:        fallback(contactData["country"]),
		State:          fallback(contactData["state"]),
		Zip:            fallback(contactData["zip"]),
		OrganizationID: campaign.OrganizationID,
	}

	storedContact, err := s.contactRepo.UpsertBySource(ctx, contact)
	if err != nil {
		return err
	}

	if err := s.contactRepo.AddToList(ctx, storedContact.ID, listContactID); err != nil {
		s.log.Error().Err(err).Str("contact_id", storedContact.ID.String()).Msg("failed to associate contact with list")
	}

	// enqueue campaign execution
	payloadExec := executeCampaignPayload{
		CampaignID: campaign.ID,
		UserID:     campaign.UserID,
		AgentID:    *campaign.AgentID,
		ContactIDs: []uuid.UUID{storedContact.ID},
	}

	if _, err := s.workerClient.EnqueueCriticalTask(ctx, worker.TypeExecuteCampaign, payloadExec); err != nil {
		s.log.Error().Err(err).
			Str("campaign_id", campaign.ID.String()).
			Str("contact_id", storedContact.ID.String()).
			Msg("failed to enqueue campaign execution for google ads lead")
		return err
	}

	return nil
}

func mapLeadData(payload GoogleAdsLeadPayload) map[string]string {
	result := make(map[string]string)
	for _, column := range payload.UserColumnData {
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

func fallback(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return domain.NOT_AVAILABLE
	}
	return value
}
