package skills

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
)

// ContactQuerySkill exposes basic contact lookups for agents.
type ContactQuerySkill struct {
	repo *contactRepo.ContactRepository
}

func NewContactQuerySkill(repo *contactRepo.ContactRepository) *ContactQuerySkill {
	return &ContactQuerySkill{repo: repo}
}

// GetContact fetches a contact by ID with JSONB fields unmarshaled.
func (s *ContactQuerySkill) GetContact(ctx context.Context, id uuid.UUID) (map[string]any, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("contact %s not found", id)
	}
	return marshalContact(c), nil
}

// marshalContact converts domain.Contact to a map for skill use.
func marshalContact(c *domain.Contact) map[string]any {
	// Get email from profile if available
	var profileEmail string
	var profile map[string]interface{}
	if err := json.Unmarshal(c.Profile, &profile); err == nil {
		if email, ok := profile["email"].(string); ok {
			profileEmail = email
		}
	}

	out := map[string]any{
		"id":        c.ID,
		"user_id":   c.UserID,
		"email":     profileEmail,
		"name":      c.Name,
		"company":   c.Company,
		"job_title": c.JobTitle,
		// Segment scoring and the embedding metadata both read industry from
		// this map; without it every contact scored as "industry unknown".
		"industry":        c.Industry,
		"do_not_contact":  c.DoNotContact,
		"created_at":      c.CreatedAt,
		"updated_at":      c.UpdatedAt,
		"organization_id": c.OrganizationID,
	}

	// helper to unmarshal JSONB safely
	unmarshal := func(src domain.JSONB) map[string]any {
		dst := map[string]any{}
		if len(src) == 0 {
			return dst
		}
		_ = json.Unmarshal(src, &dst)
		return dst
	}

	out["profile"] = unmarshal(c.Profile)
	out["tags"] = unmarshal(c.Tags)
	out["confirmed_facts"] = unmarshal(c.ConfirmedFacts)
	out["ai_insights"] = unmarshal(c.AIInsights)
	out["insight_validation"] = unmarshal(c.InsightValidation)
	out["scores"] = unmarshal(c.Scores)

	return out
}
