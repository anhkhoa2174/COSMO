package v2

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// ContactResponse represents a contact payload returned by create/update endpoints.
type ContactResponse struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	OrganizationID *uuid.UUID             `json:"organization_id"`
	SourceID       *string                `json:"source_id"`
	Source         *string                `json:"source"`
	HubspotID      *string                `json:"hubspot_id"`
	Name           *string                `json:"name"`
	Email          *string                `json:"email"`
	Phone          *string                `json:"phone"`
	Company        *string                `json:"company"`
	JobTitle       *string                `json:"job_title"`
	Address        *string                `json:"address"`
	City           *string                `json:"city"`
	Country        *string                `json:"country"`
	State          *string                `json:"state"`
	Zip            *string                `json:"zip"`
	IsDeleted      bool                   `json:"is_deleted"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	Tags           map[string]interface{} `json:"tags,omitempty"`
}

// ContactListItem mirrors the legacy Python response shape for contact listings.
type ContactListItem struct {
	Entity map[string]interface{} `json:"entity"`
}

// ToContactResponse converts a domain contact to a response payload mirroring the Python API.
func ToContactResponse(contact *domain.Contact) *ContactResponse {
	tags := make(map[string]interface{})
	if len(contact.Tags) > 0 {
		if err := json.Unmarshal(contact.Tags, &tags); err != nil {
			tags = map[string]interface{}{"_unmarshal_error": err.Error()}
		}
	}

	// Parse profile to extract email and phone
	profile := make(map[string]interface{})
	if len(contact.Profile) > 0 {
		_ = json.Unmarshal(contact.Profile, &profile)
	}

	source := contact.Source
	sourceID := contact.SourceID

	response := &ContactResponse{
		ID:        contact.ID,
		UserID:    contact.UserID,
		IsDeleted: contact.IsDeleted,
		CreatedAt: contact.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: contact.UpdatedAt.Format(time.RFC3339Nano),
		Tags:      tags,
	}
	// Populate fields, treating domain.NOT_AVAILABLE ("N/A") as missing
	if contact.OrganizationID != nil {
		response.OrganizationID = contact.OrganizationID
	} else {
		response.OrganizationID = nil
	}

	if source != "" && source != domain.NOT_AVAILABLE {
		response.Source = &source
	} else {
		response.Source = nil
	}

	if sourceID != "" && sourceID != domain.NOT_AVAILABLE {
		response.SourceID = &sourceID
	} else {
		response.SourceID = nil
	}

	if contact.HubspotID != nil && *contact.HubspotID != domain.NOT_AVAILABLE {
		response.HubspotID = contact.HubspotID
	} else {
		response.HubspotID = nil
	}

	if contact.Name != "" && contact.Name != domain.NOT_AVAILABLE {
		response.Name = &contact.Name
	} else {
		response.Name = nil
	}
	// Email and Phone are now stored in profile JSONB
	if email, ok := profile["email"].(string); ok && email != "" && email != domain.NOT_AVAILABLE {
		response.Email = &email
	} else {
		response.Email = nil
	}
	if phone, ok := profile["phone"].(string); ok && phone != "" && phone != domain.NOT_AVAILABLE {
		response.Phone = &phone
	} else {
		response.Phone = nil
	}
	if contact.Company != "" && contact.Company != domain.NOT_AVAILABLE {
		response.Company = &contact.Company
	} else {
		response.Company = nil
	}
	if contact.JobTitle != "" && contact.JobTitle != domain.NOT_AVAILABLE {
		response.JobTitle = &contact.JobTitle
	} else {
		response.JobTitle = nil
	}
	if contact.Address != "" && contact.Address != domain.NOT_AVAILABLE {
		response.Address = &contact.Address
	} else {
		response.Address = nil
	}
	if contact.City != "" && contact.City != domain.NOT_AVAILABLE {
		response.City = &contact.City
	} else {
		response.City = nil
	}
	if contact.Country != "" && contact.Country != domain.NOT_AVAILABLE {
		response.Country = &contact.Country
	} else {
		response.Country = nil
	}
	if contact.State != "" && contact.State != domain.NOT_AVAILABLE {
		response.State = &contact.State
	} else {
		response.State = nil
	}
	if contact.Zip != "" && contact.Zip != domain.NOT_AVAILABLE {
		response.Zip = &contact.Zip
	} else {
		response.Zip = nil
	}

	return response
}

// ToContactListItem converts a domain contact to the legacy listing entity format.
func ToContactListItem(contact *domain.Contact) *ContactListItem {
	// Parse profile to extract email and phone
	profile := make(map[string]interface{})
	if len(contact.Profile) > 0 {
		_ = json.Unmarshal(contact.Profile, &profile)
	}

	entity := map[string]interface{}{
		"id":                contact.ID,
		"user_id":           contact.UserID,
		"organization_id":   contact.OrganizationID,
		"created_at":        contact.CreatedAt,
		"updated_at":        contact.UpdatedAt,
		"inbound_lead_form": nil,
		"tags":              map[string]interface{}{},
	}

	// Treat domain.NOT_AVAILABLE ("N/A") as missing and return nil
	if contact.SourceID != "" && contact.SourceID != domain.NOT_AVAILABLE {
		entity["source_id"] = contact.SourceID
	} else {
		entity["source_id"] = nil
	}

	if contact.Source != "" && contact.Source != domain.NOT_AVAILABLE {
		entity["source"] = contact.Source
	} else {
		entity["source"] = nil
	}

	if contact.HubspotID != nil && *contact.HubspotID != domain.NOT_AVAILABLE {
		entity["hubspot_id"] = contact.HubspotID
	} else {
		entity["hubspot_id"] = nil
	}

	if contact.Name != "" && contact.Name != domain.NOT_AVAILABLE {
		entity["name"] = contact.Name
	} else {
		entity["name"] = nil
	}
	// Email and Phone are now stored in profile JSONB
	if email, ok := profile["email"].(string); ok && email != "" && email != domain.NOT_AVAILABLE {
		entity["email"] = email
	} else {
		entity["email"] = nil
	}
	if phone, ok := profile["phone"].(string); ok && phone != "" && phone != domain.NOT_AVAILABLE {
		entity["phone"] = phone
	} else {
		entity["phone"] = nil
	}
	if contact.Company != "" && contact.Company != domain.NOT_AVAILABLE {
		entity["company"] = contact.Company
	} else {
		entity["company"] = nil
	}
	if contact.JobTitle != "" && contact.JobTitle != domain.NOT_AVAILABLE {
		entity["job_title"] = contact.JobTitle
	} else {
		entity["job_title"] = nil
	}
	if contact.Address != "" && contact.Address != domain.NOT_AVAILABLE {
		entity["address"] = contact.Address
	} else {
		entity["address"] = nil
	}
	if contact.City != "" && contact.City != domain.NOT_AVAILABLE {
		entity["city"] = contact.City
	} else {
		entity["city"] = nil
	}
	if contact.Country != "" && contact.Country != domain.NOT_AVAILABLE {
		entity["country"] = contact.Country
	} else {
		entity["country"] = nil
	}
	if contact.State != "" && contact.State != domain.NOT_AVAILABLE {
		entity["state"] = contact.State
	} else {
		entity["state"] = nil
	}
	if contact.Zip != "" && contact.Zip != domain.NOT_AVAILABLE {
		entity["zip"] = contact.Zip
	} else {
		entity["zip"] = nil
	}

	if len(contact.Profile) > 0 {
		var profile map[string]interface{}
		if err := json.Unmarshal(contact.Profile, &profile); err == nil {
			for k, v := range profile {
				entity[k] = v
			}
		} else {
			entity["profile_unmarshal_error"] = err.Error()
		}
	}

	if len(contact.Tags) > 0 {
		var tags interface{}
		if err := json.Unmarshal(contact.Tags, &tags); err == nil {
			entity["tags"] = tags
		} else {
			entity["tags_unmarshal_error"] = err.Error()
		}
	}

	// Note: Direct relationships removed to avoid circular imports
	// InboundLeadForm data should be loaded separately using relations package if needed

	return &ContactListItem{Entity: entity}
}

// CreateContactRequest represents create contact request
// Supports extra fields for custom fields (model_config = {"extra": "allow"})
type CreateContactRequest struct {
	FirstName      string                 `json:"first_name" validate:"required"`
	LastName       string                 `json:"last_name" validate:"required"`
	Email          string                 `json:"email" validate:"required,email"`
	Phone          string                 `json:"phone"`
	Company        string                 `json:"company"`
	JobTitle       string                 `json:"job_title"`
	Address        string                 `json:"address"`
	City           string                 `json:"city"`
	Country        string                 `json:"country"`
	State          string                 `json:"state"`
	Zip            string                 `json:"zip"`
	Profile        map[string]interface{} `json:"profile"`
	DoNotContact   bool                   `json:"do_not_contact"`
	OrganizationID *uuid.UUID             `json:"organization_id"`
	Tags           map[string]interface{} `json:"tags"`

	// ExtraFields captures any additional fields for custom field support
	ExtraFields map[string]interface{} `json:"-"`
}

// UnmarshalJSON custom unmarshaler to capture extra fields for custom field support
func (c *CreateContactRequest) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Define known fields
	knownFields := map[string]bool{
		"first_name": true, "last_name": true,
		"email": true, "phone": true, "company": true, "job_title": true,
		"address": true, "city": true, "country": true, "state": true, "zip": true,
		"profile": true, "do_not_contact": true, "organization_id": true, "tags": true,
	}

	// Extract extra fields
	c.ExtraFields = make(map[string]interface{})
	for key, value := range raw {
		if !knownFields[key] {
			c.ExtraFields[key] = value
		}
	}

	// Unmarshal into the struct normally (using a type alias to avoid recursion)
	type Alias CreateContactRequest
	aux := &struct{ *Alias }{Alias: (*Alias)(c)}
	return json.Unmarshal(data, aux)
}

// UpdateContactRequest represents update contact request
// Supports extra fields for custom fields (model_config = {"extra": "allow"})
type UpdateContactRequest struct {
	FirstName      *string                `json:"first_name,omitempty"`
	LastName       *string                `json:"last_name,omitempty"`
	Email          *string                `json:"email,omitempty" validate:"omitempty,email"`
	Phone          *string                `json:"phone,omitempty"`
	Company        *string                `json:"company,omitempty"`
	JobTitle       *string                `json:"job_title,omitempty"`
	Address        *string                `json:"address,omitempty"`
	City           *string                `json:"city,omitempty"`
	Country        *string                `json:"country,omitempty"`
	State          *string                `json:"state,omitempty"`
	Zip            *string                `json:"zip,omitempty"`
	Profile        map[string]interface{} `json:"profile,omitempty"`
	DoNotContact   *bool                  `json:"do_not_contact,omitempty"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty"`
	Tags           map[string]interface{} `json:"tags,omitempty"`

	// ExtraFields captures any additional fields for custom field support
	ExtraFields map[string]interface{} `json:"-"`
}

// UnmarshalJSON custom unmarshaler to capture extra fields for custom field support
func (u *UpdateContactRequest) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Define known fields
	knownFields := map[string]bool{
		"first_name": true, "last_name": true, "email": true, "phone": true,
		"company": true, "job_title": true, "address": true, "city": true,
		"country": true, "state": true, "zip": true, "profile": true,
		"do_not_contact": true, "organization_id": true, "tags": true,
	}

	// Extract extra fields
	u.ExtraFields = make(map[string]interface{})
	for key, value := range raw {
		if !knownFields[key] {
			u.ExtraFields[key] = value
		}
	}

	// Unmarshal into the struct normally (using a type alias to avoid recursion)
	type Alias UpdateContactRequest
	aux := &struct{ *Alias }{Alias: (*Alias)(u)}
	return json.Unmarshal(data, aux)
}

// BatchCreateContactsRequest represents batch create contacts request
type BatchCreateContactsRequest struct {
	Contacts []CreateContactRequest `json:"contacts" validate:"required,min=1,max=1000,dive"`
}

// BatchUpdateContactsRequest represents batch update contacts request
type BatchUpdateContactsRequest struct {
	Updates []ContactUpdate `json:"updates" validate:"required,min=1,max=1000,dive"`
}

// ContactUpdate represents a single contact update in batch
type ContactUpdate struct {
	ID      uuid.UUID            `json:"id" validate:"required"`
	Updates UpdateContactRequest `json:"updates" validate:"required"`
}

// BatchResponse represents batch operation response
type BatchResponse struct {
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// ListContactsRequest represents list contacts request with filters
type ListContactsRequest struct {
	Offset    int                    `query:"offset"`
	Limit     int                    `query:"limit"`
	Filter    map[string]interface{} `json:"filter"`
	SortBy    string                 `query:"sort_by"`
	SortOrder string                 `query:"sort_order"`
}

// ContactListResponse represents a paginated list of contacts
type ContactListResponse struct {
	Items      []ContactResponse `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// ContactDeleteRequest represents delete contacts request
type ContactDeleteRequest struct {
	IDs []string `json:"ids" validate:"required,min=1"`
}

// ContactFieldValuesRequest represents field values request (query params)
type ContactFieldValuesRequest struct {
	Fields []string `query:"fields" validate:"required,min=1"`
	Offset int      `query:"offset"`
	Limit  int      `query:"limit"`
}

// FieldValueItem represents a single field's distinct values
type FieldValueItem struct {
	List  []interface{} `json:"list"`
	Total int64         `json:"total"`
}

// ContactFieldValuesResponse represents field values response
type ContactFieldValuesResponse map[string]FieldValueItem

// ContactImportCSVResponse represents CSV import response
type ContactImportCSVResponse struct {
	Message string `json:"message"`
}

// ContactSearchRequest represents contact search payload supporting legacy filter keys.
type ContactSearchRequest struct {
	Filter map[string]interface{} `json:"filter,omitempty"`
}

// ContactSearchResponse mirrors the legacy Listing response for contact search.
type ContactSearchResponse = schema.PaginatedResponse[*ContactListItem]

// ContactImportHubspotMapping represents HubSpot field mapping
type ContactImportHubspotMapping struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Company   *string `json:"company,omitempty"`
	JobTitle  *string `json:"job_title,omitempty"`
	Address   *string `json:"address,omitempty"`
	City      *string `json:"city,omitempty"`
	Country   *string `json:"country,omitempty"`
	State     *string `json:"state,omitempty"`
	Zip       *string `json:"zip,omitempty"`
	// Support custom fields via map
	CustomFields map[string]string `json:"-"`
}

// ContactImportHubspotRequest represents HubSpot import request
type ContactImportHubspotRequest struct {
	ListIDs []string `json:"list_ids,omitempty"`
}

// ContactImportHubspotResponse represents HubSpot import response (operation)
type ContactImportHubspotResponse struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    interface{}            `json:"output,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type ContactImportHubspotTask struct {
	OperationID uuid.UUID `json:"operation_id"`
}
