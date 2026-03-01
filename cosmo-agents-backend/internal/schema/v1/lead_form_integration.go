package v1

import "github.com/google/uuid"

type FacebookConfigSchema struct {
	FormID string `json:"form_id"`
	PageID string `json:"page_id"`
}

type ContactFieldMappingSchema struct {
	ExternalFieldName string `json:"external_field_name"`
	MappingType       string `json:"mapping_type"`
	ContactFieldName  string `json:"contact_field_name"`
}

type CustomFieldMappingSchema struct {
	ExternalFieldName string    `json:"external_field_name"`
	MappingType       string    `json:"mapping_type"`
	CustomFieldID     uuid.UUID `json:"custom_field_id"`
}

type LeadFormMappingRequest struct {
	ExternalFieldName string     `json:"external_field_name"`
	MappingType       string     `json:"mapping_type"`
	ContactFieldName  string     `json:"contact_field_name,omitempty"`
	CustomFieldID     *uuid.UUID `json:"custom_field_id,omitempty"`
}

type LeadFormIntegrationCreateRequest struct {
	CampaignID      uuid.UUID                `json:"campaign_id"`
	IntegrationType string                   `json:"integration_type"`
	Config          FacebookConfigSchema     `json:"config"`
	FieldMappings   []LeadFormMappingRequest `json:"field_mappings"`
}

type LeadFormIntegrationUpdateRequest struct {
	FieldMappings []LeadFormMappingRequest `json:"field_mappings"`
}

type LeadFormIntegrationResponse struct {
	ID              uuid.UUID                `json:"id"`
	CampaignID      uuid.UUID                `json:"campaign_id"`
	IntegrationType string                   `json:"integration_type"`
	Config          FacebookConfigSchema     `json:"config"`
	FieldMappings   []LeadFormMappingRequest `json:"field_mappings,omitempty"`
}
