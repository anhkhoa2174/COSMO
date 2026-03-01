package ai

// CompanyInfo represents structured company insights extracted from AI.
type CompanyInfo struct {
	CompanyDescription      string   `json:"company_description"`
	CompanyTargetingPersona []string `json:"company_targeting_persona"`
	ValueOffering           string   `json:"value_offering"`
}
