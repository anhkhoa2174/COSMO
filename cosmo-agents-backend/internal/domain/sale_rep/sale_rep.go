package sale_rep

import (
	"strings"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/organization"
	"github.com/rockship/cosmo-agents-go/internal/domain/user"

	"github.com/google/uuid"
)

// SaleRep represents a sales representative profile.
// Converted from machine/models/sale_rep.py
type SaleRep struct {
	base.Base
	base.TimestampMixin

	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Email          string    `gorm:"not null;index:idx_salereps_user_id_email,unique" json:"email"`
	CalendarLink   string    `json:"calendar_link"`
	Picture        string    `json:"picture"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index:idx_salereps_user_id_email,unique" json:"user_id"`
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index:idx_salereps_organization_id" json:"organization_id"`

	User         *user.User                 `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Organization *organization.Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
}

func (SaleRep) TableName() string {
	return "sale_reps"
}

var saleRepAllowedMergeTags = []string{"first_name", "last_name", "email", "calendar_link"}

// SaleRepAllowedMergeTags returns the merge tags available for sale reps.
func SaleRepAllowedMergeTags(prefix bool) []string {
	if !prefix {
		return append([]string{}, saleRepAllowedMergeTags...)
	}
	withPrefix := make([]string, len(saleRepAllowedMergeTags))
	for i, tag := range saleRepAllowedMergeTags {
		withPrefix[i] = "sale_rep_" + tag
	}
	return withPrefix
}

// ToEmailTemplateContext builds the merge tag payload for templates.
func (s *SaleRep) ToEmailTemplateContext() map[string]string {
	context := make(map[string]string)
	prefixed := SaleRepAllowedMergeTags(true)
	values := map[string]string{
		"first_name":    s.FirstName,
		"last_name":     s.LastName,
		"email":         s.Email,
		"calendar_link": s.CalendarLink,
	}
	for _, tag := range saleRepAllowedMergeTags {
		key := "sale_rep_" + tag
		val := values[tag]
		context[key] = strings.TrimSpace(val)
	}
	// ensure map includes keys even if empty
	for _, key := range prefixed {
		if _, ok := context[key]; !ok {
			context[key] = ""
		}
	}
	return context
}
