package v1

// CompanyExtractInfoResponse represents AI company enrichment response.
type CompanyExtractInfoResponse struct {
	CompanyDescription      string   `json:"company_description"`
	CompanyTargetingPersona []string `json:"company_targeting_persona"`
	ValueOffering           string   `json:"value_offering"`
}
