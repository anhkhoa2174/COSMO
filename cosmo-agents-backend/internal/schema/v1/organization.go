package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// OrganizationResponse represents organization API response
type OrganizationResponse struct {
	ID                      uuid.UUID  `json:"id"`
	UserID                  *uuid.UUID `json:"user_id"`
	Name                    string     `json:"name"`
	CompanyURL              string     `json:"company_url"`
	CompanyDescription      string     `json:"company_description"`
	CompanyTargetingPersona []string   `json:"company_targeting_persona"`
	ValueOffering           string     `json:"value_offering"`
	CreatedAt               string     `json:"created_at"`
	UpdatedAt               string     `json:"updated_at"`
}

// ToOrganizationResponse converts domain.Organization to response DTO
func ToOrganizationResponse(org *domain.Organization) *OrganizationResponse {
	return &OrganizationResponse{
		ID:                      org.ID,
		UserID:                  org.UserID,
		Name:                    org.Name,
		CompanyURL:              org.CompanyURL,
		CompanyDescription:      org.CompanyDescription,
		CompanyTargetingPersona: org.CompanyTargetingPersona,
		ValueOffering:           org.ValueOffering,
		CreatedAt:               org.CreatedAt.Format("2006-01-02T15:04:05.999999Z07:00"),
		UpdatedAt:               org.UpdatedAt.Format("2006-01-02T15:04:05.999999Z07:00"),
	}
}

// CreateOrganizationRequest represents create organization request
type CreateOrganizationRequest struct {
	Name                    string   `json:"name" validate:"required"`
	CompanyURL              string   `json:"company_url"`
	CompanyDescription      string   `json:"company_description"`
	CompanyTargetingPersona []string `json:"company_targeting_persona"`
	ValueOffering           string   `json:"value_offering"`
}

// UpdateOrganizationRequest represents update organization request
type UpdateOrganizationRequest struct {
	Name                    *string  `json:"name"`
	CompanyURL              *string  `json:"company_url"`
	CompanyDescription      *string  `json:"company_description"`
	CompanyTargetingPersona []string `json:"company_targeting_persona"`
	ValueOffering           *string  `json:"value_offering"`
}

// RoleResponse represents role API response
type RoleResponse struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	JobTitle       string    `json:"job_title"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ToRoleResponse converts domain.Role to response DTO
func ToRoleResponse(role *domain.Role) *RoleResponse {
	return &RoleResponse{
		ID:             role.ID,
		UserID:         role.UserID,
		OrganizationID: role.OrganizationID,
		Name:           string(role.Name),
		JobTitle:       role.JobTitle,
		Status:         string(role.Status),
		CreatedAt:      role.CreatedAt,
		UpdatedAt:      role.UpdatedAt,
	}
}

// RoleCreateRequest represents create role request
type RoleCreateRequest struct {
	Email          string    `json:"email" validate:"required,email"`
	OrganizationID uuid.UUID `json:"organization_id" validate:"required"`
	Name           string    `json:"name" validate:"required,oneof=admin member"`
	JobTitle       string    `json:"job_title"`
}

// OrganizationMemberCreateRequest represents add member request
type OrganizationMemberCreateRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	JobTitle string `json:"job_title" validate:"required"`
	Role     string `json:"role" validate:"omitempty,oneof=admin member" default:"member"`
}
