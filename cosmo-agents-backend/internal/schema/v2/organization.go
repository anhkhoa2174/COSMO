package v2

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// OrganizationReadResponse represents the response for GET /v2/organizations/{organization_id}
type OrganizationReadResponse struct {
	ID                      uuid.UUID `json:"id"`
	Name                    *string   `json:"name"`
	CompanyURL              *string   `json:"company_url"`
	CompanyDescription      *string   `json:"company_description"`
	CompanyTargetingPersona *[]string `json:"company_targeting_persona"`
	ValueOffering           *string   `json:"value_offering"`
}

// OrganizationMemberSearchRequest represents the request for POST /v2/organizations/{organization_id}/members/search
type OrganizationMemberSearchRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// OrganizationMemberDeleteRequest represents the request for DELETE /v2/organizations/{organization_id}/members/remove
type OrganizationMemberDeleteRequest struct {
	MemberIDs []uuid.UUID `json:"member_ids"`
}

// OrganizationRole represents a role in the organization
type OrganizationRole struct {
	ID             uuid.UUID                `json:"id"`
	OrganizationID uuid.UUID                `json:"organization_id"`
	Name           string                   `json:"name"`
	Status         domain.RoleStatus        `json:"status"`
	JobTitle       *string                  `json:"job_title"`
	Organization   OrganizationReadResponse `json:"organization"`
}

// OrganizationMemberEntity represents a member entity
type OrganizationMemberEntity struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Name        *string   `json:"name"`
	Picture     *string   `json:"picture"`
	PhoneNumber *string   `json:"phone_number"`
	IsDeleted   bool      `json:"is_deleted"`
}

// OrganizationListItem represents a single item in the member list
type OrganizationListItem struct {
	Entity OrganizationMemberEntity `json:"entity"`
	Role   OrganizationRole         `json:"role"`
}

// OrganizationMemberSearchResponse represents the response for POST /v2/organizations/{organization_id}/members/search
type OrganizationMemberSearchResponse = schema.PaginatedResponse[OrganizationListItem]

// OrganizationMemberCreateRequest represents the request for POST /v2/organizations/{organization_id}/members/invite
type OrganizationMemberCreateRequest struct {
	Email       string           `json:"email" validate:"required,email"`
	Name        *string          `json:"name"`
	JobTitle    *string          `json:"job_title"`
	Role        *domain.RoleName `json:"role"`
	RedirectURI *string          `json:"redirect_uri"`
}

// OrganizationMemberCreateResponse represents the response for POST /v2/organizations/{organization_id}/members/invite
type OrganizationMemberCreateResponse struct {
	// InvitedUser OrganizationMemberEntity `json:"invited_user"`
	// Role        OrganizationRole         `json:"role"`
	InvitedUser *domain.User `json:"invited_user"`
	Role        *domain.Role `json:"role"`
}

type InviteMemberPayload struct {
	MemberName string `json:"member_name"`
	Email      string `json:"email"`
	URL        string `json:"url"`
}
