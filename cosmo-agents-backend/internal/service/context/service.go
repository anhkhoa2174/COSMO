package context

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	baseDomain "github.com/rockship/cosmo-agents-go/internal/domain"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/context"
	contextRepo "github.com/rockship/cosmo-agents-go/internal/repository/context"
)

// Service manages context for agents - both org-level and user-level
type Service struct {
	repo *contextRepo.Repository
}

func NewService(repo *contextRepo.Repository) *Service {
	return &Service{repo: repo}
}

// MergedContext combines org and user context for agent use
type MergedContext struct {
	// Organization context
	ICPDefinition       string                   `json:"icp_definition,omitempty"`
	TargetCriteria      *domain.TargetCriteria   `json:"target_criteria,omitempty"`
	CompanyKnowledge    *domain.CompanyKnowledge `json:"company_knowledge,omitempty"`
	CommonPainPoints    []string                 `json:"common_pain_points,omitempty"`
	CommonGoals         []string                 `json:"common_goals,omitempty"`
	Competitors         []domain.Competitor      `json:"competitors,omitempty"`
	MessagingGuidelines string                   `json:"messaging_guidelines,omitempty"`
	OrgAIInstructions   string                   `json:"org_ai_instructions,omitempty"`

	// User context
	UserPreferences      *domain.UserPreferences `json:"user_preferences,omitempty"`
	CommunicationStyle   string                  `json:"communication_style,omitempty"`
	PreferredEmailLength string                  `json:"preferred_email_length,omitempty"`
	PersonalNotes        string                  `json:"personal_notes,omitempty"`
	RecentContacts       []domain.RecentContact  `json:"recent_contacts,omitempty"`
	UserAIInstructions   string                  `json:"user_ai_instructions,omitempty"`

	// Conversation context
	ConversationHistory []domain.ConversationHistory `json:"conversation_history,omitempty"`
}

// GetMergedContext retrieves both org and user context, merging them for agent use
func (s *Service) GetMergedContext(ctx context.Context, userID, orgID uuid.UUID, sessionID string) (*MergedContext, error) {
	merged := &MergedContext{}

	// Get org context (ignore not found - use defaults)
	orgCtx, err := s.repo.GetOrgContext(ctx, orgID)
	if err != nil && !errors.Is(err, contextRepo.ErrContextNotFound) {
		return nil, fmt.Errorf("failed to get org context: %w", err)
	}

	if orgCtx != nil {
		merged.ICPDefinition = orgCtx.ICPDefinition
		merged.MessagingGuidelines = orgCtx.MessagingGuidelines
		merged.OrgAIInstructions = orgCtx.AIInstructions

		// Parse JSON fields
		if orgCtx.TargetCriteria != nil {
			var tc domain.TargetCriteria
			if err := json.Unmarshal(orgCtx.TargetCriteria, &tc); err == nil {
				merged.TargetCriteria = &tc
			}
		}
		if orgCtx.CompanyKnowledge != nil {
			var ck domain.CompanyKnowledge
			if err := json.Unmarshal(orgCtx.CompanyKnowledge, &ck); err == nil {
				merged.CompanyKnowledge = &ck
			}
		}
		if orgCtx.CommonPainPoints != nil {
			_ = json.Unmarshal(orgCtx.CommonPainPoints, &merged.CommonPainPoints)
		}
		if orgCtx.CommonGoals != nil {
			_ = json.Unmarshal(orgCtx.CommonGoals, &merged.CommonGoals)
		}
		if orgCtx.Competitors != nil {
			_ = json.Unmarshal(orgCtx.Competitors, &merged.Competitors)
		}
	}

	// Get user context (ignore not found - use defaults)
	userCtx, err := s.repo.GetUserContext(ctx, userID)
	if err != nil && !errors.Is(err, contextRepo.ErrContextNotFound) {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	if userCtx != nil {
		merged.CommunicationStyle = userCtx.CommunicationStyle
		merged.PreferredEmailLength = userCtx.PreferredEmailLength
		merged.PersonalNotes = userCtx.PersonalNotes
		merged.UserAIInstructions = userCtx.AIInstructions

		if userCtx.Preferences != nil {
			var prefs domain.UserPreferences
			if err := json.Unmarshal(userCtx.Preferences, &prefs); err == nil {
				merged.UserPreferences = &prefs
			}
		}
		if userCtx.RecentContacts != nil {
			_ = json.Unmarshal(userCtx.RecentContacts, &merged.RecentContacts)
		}
	}

	// Get conversation history if session provided
	if sessionID != "" {
		history, err := s.repo.GetConversationHistory(ctx, userID, sessionID, 20)
		if err == nil {
			merged.ConversationHistory = history
		}
	}

	return merged, nil
}

// FormatContextForPrompt converts merged context into a string suitable for AI prompts
func (s *Service) FormatContextForPrompt(merged *MergedContext) string {
	var parts []string

	// Organization context section
	if merged.ICPDefinition != "" || merged.CompanyKnowledge != nil || len(merged.Competitors) > 0 {
		parts = append(parts, "=== COMPANY CONTEXT ===")

		if merged.ICPDefinition != "" {
			parts = append(parts, fmt.Sprintf("ICP (Ideal Customer Profile):\n%s", merged.ICPDefinition))
		}

		if merged.CompanyKnowledge != nil {
			ck := merged.CompanyKnowledge
			if ck.ProductName != "" {
				parts = append(parts, fmt.Sprintf("Product: %s", ck.ProductName))
			}
			if ck.ProductDescription != "" {
				parts = append(parts, fmt.Sprintf("Description: %s", ck.ProductDescription))
			}
			if len(ck.ValuePropositions) > 0 {
				parts = append(parts, fmt.Sprintf("Value Props: %s", strings.Join(ck.ValuePropositions, "; ")))
			}
		}

		if len(merged.CommonPainPoints) > 0 {
			parts = append(parts, fmt.Sprintf("Common Pain Points: %s", strings.Join(merged.CommonPainPoints, "; ")))
		}

		if len(merged.Competitors) > 0 {
			comps := make([]string, len(merged.Competitors))
			for i, c := range merged.Competitors {
				comps[i] = c.Name
			}
			parts = append(parts, fmt.Sprintf("Competitors: %s", strings.Join(comps, ", ")))
		}

		if merged.MessagingGuidelines != "" {
			parts = append(parts, fmt.Sprintf("Messaging Guidelines:\n%s", merged.MessagingGuidelines))
		}

		if merged.OrgAIInstructions != "" {
			parts = append(parts, fmt.Sprintf("Special Instructions:\n%s", merged.OrgAIInstructions))
		}
	}

	// User context section
	if merged.CommunicationStyle != "" || merged.PersonalNotes != "" || len(merged.RecentContacts) > 0 {
		parts = append(parts, "\n=== USER CONTEXT ===")

		if merged.CommunicationStyle != "" {
			parts = append(parts, fmt.Sprintf("Communication Style: %s", merged.CommunicationStyle))
		}

		if merged.PreferredEmailLength != "" {
			parts = append(parts, fmt.Sprintf("Preferred Email Length: %s", merged.PreferredEmailLength))
		}

		if len(merged.RecentContacts) > 0 {
			recent := make([]string, 0, len(merged.RecentContacts))
			for _, rc := range merged.RecentContacts[:min(5, len(merged.RecentContacts))] {
				recent = append(recent, fmt.Sprintf("%s (%s)", rc.ContactName, rc.Action))
			}
			parts = append(parts, fmt.Sprintf("Recent Contacts: %s", strings.Join(recent, ", ")))
		}

		if merged.PersonalNotes != "" {
			parts = append(parts, fmt.Sprintf("Personal Notes:\n%s", merged.PersonalNotes))
		}

		if merged.UserAIInstructions != "" {
			parts = append(parts, fmt.Sprintf("User Instructions:\n%s", merged.UserAIInstructions))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n")
}

// ============ Org Context Management ============

func (s *Service) GetOrgContext(ctx context.Context, orgID uuid.UUID) (*domain.OrgContext, error) {
	return s.repo.GetOrgContext(ctx, orgID)
}

func (s *Service) UpdateOrgContext(ctx context.Context, orgID uuid.UUID, updates map[string]interface{}) error {
	orgCtx, err := s.repo.GetOrgContext(ctx, orgID)
	if errors.Is(err, contextRepo.ErrContextNotFound) {
		orgCtx = &domain.OrgContext{
			OrganizationID: orgID,
		}
	} else if err != nil {
		return err
	}

	// Apply updates
	if v, ok := updates["icp_definition"].(string); ok {
		orgCtx.ICPDefinition = v
	}
	if v, ok := updates["messaging_guidelines"].(string); ok {
		orgCtx.MessagingGuidelines = v
	}
	if v, ok := updates["ai_instructions"].(string); ok {
		orgCtx.AIInstructions = v
	}
	if v, ok := updates["target_criteria"]; ok {
		jsonData, _ := json.Marshal(v)
		orgCtx.TargetCriteria = baseDomain.JSONB(jsonData)
	}
	if v, ok := updates["company_knowledge"]; ok {
		jsonData, _ := json.Marshal(v)
		orgCtx.CompanyKnowledge = baseDomain.JSONB(jsonData)
	}
	if v, ok := updates["common_pain_points"]; ok {
		jsonData, _ := json.Marshal(v)
		orgCtx.CommonPainPoints = baseDomain.JSONB(jsonData)
	}
	if v, ok := updates["common_goals"]; ok {
		jsonData, _ := json.Marshal(v)
		orgCtx.CommonGoals = baseDomain.JSONB(jsonData)
	}
	if v, ok := updates["competitors"]; ok {
		jsonData, _ := json.Marshal(v)
		orgCtx.Competitors = baseDomain.JSONB(jsonData)
	}

	return s.repo.UpsertOrgContext(ctx, orgCtx)
}

// ============ User Context Management ============

func (s *Service) GetUserContext(ctx context.Context, userID uuid.UUID) (*domain.UserContext, error) {
	return s.repo.GetUserContext(ctx, userID)
}

func (s *Service) UpdateUserContext(ctx context.Context, userID, orgID uuid.UUID, updates map[string]interface{}) error {
	userCtx, err := s.repo.GetUserContext(ctx, userID)
	if errors.Is(err, contextRepo.ErrContextNotFound) {
		userCtx = &domain.UserContext{
			UserID:         userID,
			OrganizationID: orgID,
		}
	} else if err != nil {
		return err
	}

	// Apply updates
	if v, ok := updates["communication_style"].(string); ok {
		userCtx.CommunicationStyle = v
	}
	if v, ok := updates["preferred_email_length"].(string); ok {
		userCtx.PreferredEmailLength = v
	}
	if v, ok := updates["personal_notes"].(string); ok {
		userCtx.PersonalNotes = v
	}
	if v, ok := updates["ai_instructions"].(string); ok {
		userCtx.AIInstructions = v
	}
	if v, ok := updates["preferences"]; ok {
		jsonData, _ := json.Marshal(v)
		userCtx.Preferences = baseDomain.JSONB(jsonData)
	}

	return s.repo.UpsertUserContext(ctx, userCtx)
}

func (s *Service) TrackRecentContact(ctx context.Context, userID, contactID uuid.UUID, contactName, action string) error {
	return s.repo.AddRecentContact(ctx, userID, domain.RecentContact{
		ContactID:   contactID,
		ContactName: contactName,
		LastTouched: time.Now(),
		Action:      action,
	})
}

// ============ Conversation History ============

func (s *Service) SaveMessage(ctx context.Context, userID, orgID uuid.UUID, contactID *uuid.UUID, sessionID, role, content string, toolsUsed []string) error {
	msg := &domain.ConversationHistory{
		UserID:         userID,
		OrganizationID: orgID,
		ContactID:      contactID,
		SessionID:      sessionID,
		Role:           role,
		Content:        content,
	}

	if len(toolsUsed) > 0 {
		jsonData, _ := json.Marshal(toolsUsed)
		msg.ToolsUsed = baseDomain.JSONB(jsonData)
	}

	return s.repo.SaveConversationMessage(ctx, msg)
}

func (s *Service) GetHistory(ctx context.Context, userID uuid.UUID, sessionID string, limit int) ([]domain.ConversationHistory, error) {
	return s.repo.GetConversationHistory(ctx, userID, sessionID, limit)
}

func (s *Service) ClearSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	return s.repo.ClearSessionHistory(ctx, userID, sessionID)
}

func (s *Service) CleanupOldHistory(ctx context.Context, olderThan time.Duration) (int64, error) {
	return s.repo.CleanupOldHistory(ctx, olderThan)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
