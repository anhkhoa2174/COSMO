package agents

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
)

// NetworkAnalyzerAgent creates warm-path analysis placeholders from existing contact data.
type NetworkAnalyzerAgent struct {
	contactRepo *contactrepo.ContactRepository
}

func NewNetworkAnalyzerAgent(contactRepo *contactrepo.ContactRepository) *NetworkAnalyzerAgent {
	return &NetworkAnalyzerAgent{contactRepo: contactRepo}
}

type NetworkAnalysisResult struct {
	ContactID        uuid.UUID              `json:"contact_id"`
	LinkedInURL      string                 `json:"linkedin_url,omitempty"`
	IntroProbability float64                `json:"intro_probability"`
	Status           string                 `json:"status"`
	AnalyzedAt       string                 `json:"analyzed_at"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

func (a *NetworkAnalyzerAgent) Run(ctx context.Context, userID, orgID, contactID uuid.UUID) (*NetworkAnalysisResult, error) {
	contactModel, err := a.contactRepo.GetByID(ctx, contactID)
	if err != nil {
		return nil, err
	}
	if contactModel == nil {
		return nil, ErrNotFound
	}
	if contactModel.UserID != userID {
		if contactModel.OrganizationID == nil || *contactModel.OrganizationID != orgID {
			return nil, ErrUnauthorized
		}
	}

	profile := map[string]interface{}{}
	_ = contactModel.Profile.Unmarshal(&profile)

	linkedinURL := extractLinkedIn(profile)
	status := "no_linkedin"
	if linkedinURL != "" {
		status = "unavailable"
	}

	result := &NetworkAnalysisResult{
		ContactID:        contactID,
		LinkedInURL:      linkedinURL,
		IntroProbability: 0,
		Status:           status,
		AnalyzedAt:       time.Now().UTC().Format(time.RFC3339Nano),
		Metadata: map[string]interface{}{
			"source": "placeholder",
		},
	}

	profile["network"] = map[string]interface{}{
		"status":            result.Status,
		"intro_probability": result.IntroProbability,
		"linkedin_url":      result.LinkedInURL,
		"analyzed_at":       result.AnalyzedAt,
		"metadata":          result.Metadata,
	}

	var profileJB domain.JSONB
	_ = profileJB.Marshal(profile)
	org := uuid.Nil
	if contactModel.OrganizationID != nil {
		org = *contactModel.OrganizationID
	}
	if _, err := a.contactRepo.UpdateFields(ctx, contactID, map[string]interface{}{
		"profile": profileJB,
	}, userID, org); err != nil {
		return nil, err
	}

	return result, nil
}

func extractLinkedIn(profile map[string]interface{}) string {
	if v, ok := profile["linkedin_url"].(string); ok && v != "" {
		return v
	}
	if v, ok := profile["linkedin"].(string); ok && v != "" {
		return v
	}
	if v, ok := profile["social_links"].(map[string]interface{}); ok {
		if link, ok := v["linkedin"].(string); ok {
			return link
		}
	}
	return ""
}
