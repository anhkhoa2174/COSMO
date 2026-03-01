package inbound_lead_form

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/lead_form_integration"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	facebookTokenRepo "github.com/rockship/cosmo-agents-go/internal/repository/facebook_token"
	inboundLeadFormRepo "github.com/rockship/cosmo-agents-go/internal/repository/inbound_lead_form"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// LeadFormIntegrationService coordinates external lead integrations.
type LeadFormIntegrationService struct {
	campaignRepo        *campaignRepo.CampaignRepository
	integrationRepo     *inboundLeadFormRepo.LeadFormIntegrationRepository
	inboundLeadFormRepo *inboundLeadFormRepo.InboundLeadFormRepository
	facebookTokenRepo   *facebookTokenRepo.FacebookTokenRepository
}

func NewLeadFormIntegrationService(
	campaignRepo *campaignRepo.CampaignRepository,
	integrationRepo *inboundLeadFormRepo.LeadFormIntegrationRepository,
	inboundLeadFormRepo *inboundLeadFormRepo.InboundLeadFormRepository,
	facebookTokenRepo *facebookTokenRepo.FacebookTokenRepository,
) *LeadFormIntegrationService {
	return &LeadFormIntegrationService{
		campaignRepo:        campaignRepo,
		integrationRepo:     integrationRepo,
		inboundLeadFormRepo: inboundLeadFormRepo,
		facebookTokenRepo:   facebookTokenRepo,
	}
}

func (s *LeadFormIntegrationService) Create(ctx context.Context, userID uuid.UUID, req *v1schema.LeadFormIntegrationCreateRequest) (*v1schema.LeadFormIntegrationResponse, error) {
	if req.IntegrationType != string(lead_form_integration.IntegrationTypeFacebook) {
		return nil, fmt.Errorf("unsupported integration type: %s", req.IntegrationType)
	}

	campaign, err := s.campaignRepo.FindByID(ctx, req.CampaignID)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, fmt.Errorf("campaign not found")
	}

	integration := &domain.LeadFormIntegration{
		CampaignID:      req.CampaignID,
		IntegrationType: lead_form_integration.IntegrationType(req.IntegrationType),
	}

	config := domain.JSONB{}
	configMap := map[string]any{
		"type":    req.IntegrationType,
		"form_id": req.Config.FormID,
		"page_id": req.Config.PageID,
	}
	if err := config.Marshal(configMap); err != nil {
		return nil, err
	}
	integration.Config = config

	if err := s.integrationRepo.CreateIntegration(ctx, integration); err != nil {
		return nil, err
	}

	mappings, err := s.buildMappings(req.FieldMappings, integration.ID, req.CampaignID)
	if err != nil {
		return nil, err
	}

	if err := s.integrationRepo.CreateFieldMappings(ctx, mappings); err != nil {
		return nil, err
	}

	stored, err := s.integrationRepo.GetWithMappings(ctx, integration.ID)
	if err != nil {
		return nil, err
	}

	return s.buildResponse(stored), nil
}

func (s *LeadFormIntegrationService) Get(ctx context.Context, id uuid.UUID) (*v1schema.LeadFormIntegrationResponse, error) {
	integration, err := s.integrationRepo.GetWithMappings(ctx, id)
	if err != nil {
		return nil, err
	}
	if integration == nil {
		return nil, nil
	}
	return s.buildResponse(integration), nil
}

func (s *LeadFormIntegrationService) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	integration, err := s.integrationRepo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	if integration == nil {
		return false, nil
	}
	if err := s.integrationRepo.Delete(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

func (s *LeadFormIntegrationService) UpdateMappings(ctx context.Context, id uuid.UUID, mappingsReq []v1schema.LeadFormMappingRequest) (*v1schema.LeadFormIntegrationResponse, error) {
	integration, err := s.integrationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if integration == nil {
		return nil, nil
	}

	mappings, err := s.buildMappings(mappingsReq, id, integration.CampaignID)
	if err != nil {
		return nil, err
	}

	if err := s.integrationRepo.DeleteFieldMappings(ctx, id); err != nil {
		return nil, err
	}

	if err := s.integrationRepo.CreateFieldMappings(ctx, mappings); err != nil {
		return nil, err
	}

	stored, err := s.integrationRepo.GetWithMappings(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.buildResponse(stored), nil
}

func (s *LeadFormIntegrationService) buildMappings(req []v1schema.LeadFormMappingRequest, integrationID, campaignID uuid.UUID) ([]domain.LeadFieldMapping, error) {
	mappings := make([]domain.LeadFieldMapping, 0, len(req))
	for _, item := range req {
		mapping := domain.LeadFieldMapping{
			FormIntegrationID: integrationID,
			CampaignID:        campaignID,
			ExternalFieldName: item.ExternalFieldName,
			MappingType:       lead_form_integration.MappingType(item.MappingType),
		}
		if strings.EqualFold(item.MappingType, string(lead_form_integration.MappingTypeContactField)) {
			name := item.ContactFieldName
			mapping.ContactFieldName = &name
		} else if strings.EqualFold(item.MappingType, string(lead_form_integration.MappingTypeCustomField)) && item.CustomFieldID != nil {
			mapping.CustomFieldID = item.CustomFieldID
		}
		mappings = append(mappings, mapping)
	}
	return mappings, nil
}

func (s *LeadFormIntegrationService) buildResponse(integration *domain.LeadFormIntegration) *v1schema.LeadFormIntegrationResponse {
	resp := &v1schema.LeadFormIntegrationResponse{
		ID:              integration.ID,
		CampaignID:      integration.CampaignID,
		IntegrationType: string(integration.IntegrationType),
	}

	var config map[string]any
	if err := integration.Config.Unmarshal(&config); err == nil {
		resp.Config = v1schema.FacebookConfigSchema{
			FormID: fmt.Sprintf("%v", config["form_id"]),
			PageID: fmt.Sprintf("%v", config["page_id"]),
		}
	}

	if len(integration.FieldMappings) > 0 {
		resp.FieldMappings = make([]v1schema.LeadFormMappingRequest, len(integration.FieldMappings))
		for i, mapping := range integration.FieldMappings {
			item := v1schema.LeadFormMappingRequest{
				ExternalFieldName: mapping.ExternalFieldName,
				MappingType:       string(mapping.MappingType),
			}
			if mapping.ContactFieldName != nil {
				item.ContactFieldName = *mapping.ContactFieldName
			}
			if mapping.CustomFieldID != nil {
				item.CustomFieldID = mapping.CustomFieldID
			}
			resp.FieldMappings[i] = item
		}
	}

	return resp
}
