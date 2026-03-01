package agents

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	interactionrepo "github.com/rockship/cosmo-agents-go/internal/repository/interaction"
)

// RelationshipScorerAgent computes relationship strength/health from interactions.
type RelationshipScorerAgent struct {
	contactRepo     *contactrepo.ContactRepository
	interactionRepo *interactionrepo.Repository
}

func NewRelationshipScorerAgent(contactRepo *contactrepo.ContactRepository, interactionRepo *interactionrepo.Repository) *RelationshipScorerAgent {
	return &RelationshipScorerAgent{
		contactRepo:     contactRepo,
		interactionRepo: interactionRepo,
	}
}

type RelationshipScore struct {
	ContactID          uuid.UUID `json:"contact_id"`
	StrengthScore      int       `json:"strength_score"`
	HealthScore        int       `json:"health_score"`
	LastInteractionAt  string    `json:"last_interaction_at,omitempty"`
	Interactions30Days int       `json:"interactions_30d"`
	Interactions90Days int       `json:"interactions_90d"`
	Replies90Days      int       `json:"replies_90d"`
	Meetings90Days     int       `json:"meetings_90d"`
}

func (a *RelationshipScorerAgent) Run(ctx context.Context, userID, orgID, contactID uuid.UUID) (*RelationshipScore, error) {
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

	interactions, err := a.interactionRepo.ListByContact(ctx, contactID, 200)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	limit30 := now.AddDate(0, 0, -30)
	limit90 := now.AddDate(0, 0, -90)

	var lastInteraction *time.Time
	interactions30 := 0
	interactions90 := 0
	replies90 := 0
	meetings90 := 0
	sent90 := 0
	opens90 := 0
	negative90 := 0

	for _, inter := range interactions {
		if inter == nil {
			continue
		}
		if lastInteraction == nil || inter.OccurredAt.After(*lastInteraction) {
			lastInteraction = &inter.OccurredAt
		}
		if inter.OccurredAt.After(limit30) {
			interactions30++
		}
		if inter.OccurredAt.After(limit90) {
			interactions90++
			switch inter.InteractionType {
			case "reply":
				replies90++
			case "meeting":
				meetings90++
			case "sent":
				sent90++
			case "open":
				opens90++
			case "bounce", "unsub":
				negative90++
			}
		}
	}

	score := float64(replies90*25 + meetings90*30 + sent90*4 + opens90*2 - negative90*25)
	if interactions90 == 0 {
		score = 0
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	strength := int(math.Round(score))
	result := &RelationshipScore{
		ContactID:          contactID,
		StrengthScore:      strength,
		HealthScore:        strength,
		Interactions30Days: interactions30,
		Interactions90Days: interactions90,
		Replies90Days:      replies90,
		Meetings90Days:     meetings90,
	}
	if lastInteraction != nil {
		result.LastInteractionAt = lastInteraction.UTC().Format(time.RFC3339Nano)
	}

	profile := map[string]interface{}{}
	_ = contactModel.Profile.Unmarshal(&profile)
	profile["relationship"] = map[string]interface{}{
		"strength_score":      result.StrengthScore,
		"health_score":        result.HealthScore,
		"last_interaction_at": result.LastInteractionAt,
		"interactions_30d":    result.Interactions30Days,
		"interactions_90d":    result.Interactions90Days,
		"replies_90d":         result.Replies90Days,
		"meetings_90d":        result.Meetings90Days,
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
