package v1

// ExtractFromURLRequest represents request to extract contact info from URL
type ExtractFromURLRequest struct {
	URL string `json:"url" validate:"required,url"`
}

// ExtractFromURLResponse represents the extracted data
type ExtractFromURLResponse struct {
	URL            string                 `json:"url"`
	ExtractedData  map[string]interface{} `json:"extracted_data"`
	FieldsAdded    []string               `json:"fields_added"`
	Message        string                 `json:"message"`
	ContactUpdated bool                   `json:"contact_updated"`
}

// ExtractFromImageResponse represents the extracted data from a screenshot
type ExtractFromImageResponse struct {
	ExtractedData  map[string]interface{} `json:"extracted_data"`
	FieldsAdded    []string               `json:"fields_added"`
	Message        string                 `json:"message"`
	ContactUpdated bool                   `json:"contact_updated"`
}

// ExtractFromImagePreviewResponse represents extracted data for creating a new contact
type ExtractFromImagePreviewResponse struct {
	ExtractedData map[string]interface{} `json:"extracted_data"`
}

// ExtractFromExtensionRequest represents request from a browser extension
type ExtractFromExtensionRequest struct {
	SourceURL string                 `json:"source_url,omitempty"`
	Data      map[string]interface{} `json:"data"`
	RawText   string                 `json:"raw_text,omitempty"`
	UseAI     bool                   `json:"use_ai,omitempty"`
}

// ExtractFromExtensionResponse represents the extracted data from extension
type ExtractFromExtensionResponse struct {
	SourceURL      string                 `json:"source_url"`
	ExtractedData  map[string]interface{} `json:"extracted_data"`
	FieldsAdded    []string               `json:"fields_added"`
	Message        string                 `json:"message"`
	ContactUpdated bool                   `json:"contact_updated"`
}
