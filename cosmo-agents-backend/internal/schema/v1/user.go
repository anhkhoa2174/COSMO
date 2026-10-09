package v1

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// NotificationResponse represents a user notification in API responses
type NotificationResponse struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	CampaignID uuid.UUID `json:"campaign_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UserResponse represents the API response for a user
type UserResponse struct {
	ID                    uuid.UUID       `json:"id"`
	Email                 string          `json:"email"`
	Name                  string          `json:"name"`
	Picture               string          `json:"picture"`
	LastHistoryID         string          `json:"last_history_id"`
	Provider              string          `json:"provider"`
	PhoneNumber           []string        `json:"phone_number"`
	StaffEmails           []string        `json:"staff_emails"`
	JobTitle              string          `json:"job_title"`
	HubspotCredentials json.RawMessage `json:"hubspot_credentials"`
	HubspotFieldMapping   json.RawMessage `json:"hubspot_field_mapping"`
	UIMetadata            json.RawMessage `json:"ui_metadata"`
	IsDeleted             bool            `json:"is_deleted"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`

	// Relationships
	Organizations []OrganizationResponse `json:"organizations"`
	Roles         []RoleResponse         `json:"roles"`
	Notifications []NotificationResponse `json:"notifications"`
}

// ListUsersRequest represents the request for listing users
type ListUsersRequest struct {
	Offset int `query:"offset" validate:"gte=0"`
	Limit  int `query:"limit" validate:"gte=1,lte=100"`
}

// CreateUserRequest represents creating a new user
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Name     string `json:"name" validate:"required,min=1,max=255"`
	Picture  string `json:"picture,omitempty" validate:"omitempty,url"`
	JobTitle string `json:"job_title,omitempty" validate:"omitempty,max=255"`
}

// UpdateUserRequest represents updating a user
type UpdateUserRequest struct {
	Email       *string         `json:"email,omitempty" validate:"omitempty,email"`
	Name        *string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Picture     *string         `json:"picture,omitempty" validate:"omitempty,url"`
	Credentials *map[string]any `json:"credentials,omitempty"`
	Provider    *string         `json:"provider,omitempty"`
	UIMetadata  *map[string]any `json:"ui_metadata,omitempty"`
}

// StaffEmailRequest represents a request to upsert staff emails
type StaffEmailRequest struct {
	StaffEmails []string `json:"staff_emails" validate:"dive,email"`
}

// ToUserResponse converts a domain User to UserResponse
func ToUserResponse(user *domain.User) UserResponse {
	// Convert phone numbers and staff emails from pq.StringArray to []string
	// Always return empty array instead of nil
	phoneNumbers := []string{}
	if user.PhoneNumber != nil && len(user.PhoneNumber) > 0 {
		phoneNumbers = user.PhoneNumber
	}

	staffEmails := []string{}
	if user.StaffEmails != nil && len(user.StaffEmails) > 0 {
		staffEmails = user.StaffEmails
	}

	// Convert JSON fields to raw JSON bytes
	var hubspotCredentials json.RawMessage
	if len(user.HubspotCredentials) > 0 {
		hubspotCredentials = json.RawMessage(user.HubspotCredentials)
	} else {
		hubspotCredentials = json.RawMessage("{}")
	}

	var hubspotFieldMapping json.RawMessage
	if len(user.HubspotFieldMapping) > 0 {
		hubspotFieldMapping = json.RawMessage(user.HubspotFieldMapping)
	} else {
		hubspotFieldMapping = json.RawMessage("{}")
	}

	var uiMetadata json.RawMessage
	if len(user.UIMetadata) > 0 {
		uiMetadata = json.RawMessage(user.UIMetadata)
	} else {
		uiMetadata = json.RawMessage("{}")
	}

	// Note: Direct relationships removed to avoid circular imports
	// These fields should be loaded separately using relations package if needed
	orgs := []OrganizationResponse{}
	roles := []RoleResponse{}
	notifications := []NotificationResponse{}

	return UserResponse{
		ID:                    user.ID,
		Email:                 user.Email,
		Name:                  user.Name,
		Picture:               user.Picture,
		LastHistoryID:         user.LastHistoryID,
		Provider:              user.Provider,
		PhoneNumber:           phoneNumbers,
		StaffEmails:           staffEmails,
		JobTitle:              user.JobTitle,
		HubspotCredentials: hubspotCredentials,
		HubspotFieldMapping:   hubspotFieldMapping,
		UIMetadata:            uiMetadata,
		IsDeleted:             user.IsDeleted,
		CreatedAt:             user.CreatedAt,
		UpdatedAt:             user.UpdatedAt,
		Organizations:         orgs,
		Roles:                 roles,
		Notifications:         notifications,
	}
}
