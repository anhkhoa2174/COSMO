package contact

// ContactCSVHeader represents CSV field mapping configuration
type ContactCSVHeader struct {
	NameImport *string `json:"name_import,omitempty"`
	FirstName  *string `json:"first_name,omitempty"`
	LastName   *string `json:"last_name,omitempty"`
	Email      *string `json:"email,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Company    *string `json:"company,omitempty"`
	JobTitle   *string `json:"job_title,omitempty"`
	Address    *string `json:"address,omitempty"`
	City       *string `json:"city,omitempty"`
	Country    *string `json:"country,omitempty"`
	State      *string `json:"state,omitempty"`
	Zip        *string `json:"zip,omitempty"`
}
