package role

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// RoleName represents possible role names.
type RoleName string

const (
	RoleNameAdmin  RoleName = "admin"
	RoleNameMember RoleName = "member"
)

// RoleStatus indicates the status of a role assignment.
type RoleStatus string

const (
	RoleStatusPending RoleStatus = "pending"
	RoleStatusActive  RoleStatus = "active"
)

// Role mirrors machine/models/role.py.
type Role struct {
	base.Base
	base.SoftDeleteMixin
	base.TimestampMixin

	UserID         uuid.UUID  `gorm:"type:uuid;not null;index:idx_roles_user_org,unique" json:"user_id"`
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null;index:idx_roles_user_org,unique" json:"organization_id"`
	Name           RoleName   `gorm:"type:text;not null;default:'admin'" json:"name"`
	JobTitle       string     `gorm:"type:text" json:"job_title"`
	Status         RoleStatus `gorm:"type:text;not null;default:'active'" json:"status"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.UserWithRoles
	// - relations.OrganizationWithRoles
}

// TableName specifies the table name.
func (Role) TableName() string {
	return "roles"
}
