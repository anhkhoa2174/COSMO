package playbook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/playbook"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	playbookRepo "github.com/rockship/cosmo-agents-go/internal/repository/playbook"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

// Service handles playbook business logic
type Service struct {
	playbookRepo     *playbookRepo.Repository
	automationRepo   *playbookRepo.AutomationRuleRepository
	enrollmentRepo   *playbookRepo.EnrollmentRepository
	approvalRepo     *playbookRepo.ApprovalRequestRepository
	contactRepo      *contactRepo.ContactRepository
	segmentationRepo *segRepo.SegmentationRepository
	openAI           *ai.OpenAIClient
}

// NewService creates a new playbook service
func NewService(
	playbookRepo *playbookRepo.Repository,
	automationRepo *playbookRepo.AutomationRuleRepository,
	enrollmentRepo *playbookRepo.EnrollmentRepository,
	approvalRepo *playbookRepo.ApprovalRequestRepository,
	contactRepo *contactRepo.ContactRepository,
	segmentationRepo *segRepo.SegmentationRepository,
	openAI *ai.OpenAIClient,
) *Service {
	return &Service{
		playbookRepo:     playbookRepo,
		automationRepo:   automationRepo,
		enrollmentRepo:   enrollmentRepo,
		approvalRepo:     approvalRepo,
		contactRepo:      contactRepo,
		segmentationRepo: segmentationRepo,
		openAI:           openAI,
	}
}

// CreatePlaybook creates a new playbook
func (s *Service) CreatePlaybook(ctx context.Context, req *v1schema.CreatePlaybookRequest) (*v1schema.PlaybookRead, error) {
	// Validate stages
	if len(req.Stages) == 0 {
		return nil, fmt.Errorf("playbook must have at least one stage")
	}

	// Build config JSONB
	config := v1schema.PlaybookConfig{
		Stages: req.Stages,
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Build initial performance JSONB
	performance := v1schema.PlaybookPerformance{
		ContactsEnrolled: 0,
		TotalSent:        0,
		TotalReplies:     0,
		TotalMeetings:    0,
		ReplyRate:        0.0,
		MeetingRate:      0.0,
	}

	perfBytes, err := json.Marshal(performance)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal performance: %w", err)
	}

	// Create playbook domain model
	pb := &playbook.Playbook{
		Name:         req.Name,
		Description:  req.Description,
		PlaybookType: string(req.PlaybookType),
		Config:       base.JSONB(configBytes),
		Performance:  base.JSONB(perfBytes),
		IsActive:     true,
	}

	// Insert into database
	if err := s.playbookRepo.Create(ctx, pb); err != nil {
		return nil, fmt.Errorf("failed to create playbook: %w", err)
	}

	// Convert to response
	return s.toPlaybookRead(pb)
}

// ListPlaybooks lists all playbooks
func (s *Service) ListPlaybooks(ctx context.Context) ([]*v1schema.PlaybookRead, error) {
	playbooks, err := s.playbookRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list playbooks: %w", err)
	}

	result := make([]*v1schema.PlaybookRead, 0, len(playbooks))
	for _, pb := range playbooks {
		playbookRead, err := s.toPlaybookRead(pb)
		if err != nil {
			continue // Skip invalid playbooks
		}
		result = append(result, playbookRead)
	}

	return result, nil
}

// GetPlaybook gets a playbook by ID
func (s *Service) GetPlaybook(ctx context.Context, playbookID uuid.UUID) (*v1schema.PlaybookRead, error) {
	pb, err := s.playbookRepo.GetByID(ctx, playbookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playbook: %w", err)
	}
	if pb == nil {
		return nil, fmt.Errorf("playbook not found")
	}

	return s.toPlaybookRead(pb)
}

// DeletePlaybook soft deletes a playbook
func (s *Service) DeletePlaybook(ctx context.Context, playbookID uuid.UUID) error {
	return s.playbookRepo.Delete(ctx, playbookID)
}

// GenerateContent generates AI content for a stage
func (s *Service) GenerateContent(ctx context.Context, playbookID, contactID uuid.UUID, stageID string, prompt string) (*v1schema.GenerateContentResponse, error) {
	// Fetch playbook
	pb, err := s.playbookRepo.GetByID(ctx, playbookID)
	if err != nil || pb == nil {
		return nil, fmt.Errorf("playbook not found")
	}

	// Fetch contact
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil || contact == nil {
		return nil, fmt.Errorf("contact not found")
	}

	// Parse contact profile
	var profile map[string]interface{}
	if err := json.Unmarshal(contact.Profile, &profile); err != nil {
		profile = make(map[string]interface{})
	}

	// Extract personalization data
	name := contact.Name
	company := contact.Company
	jobTitle := contact.JobTitle

	painPoints := extractPainPoints(profile)
	goals := extractGoals(profile)

	// Build AI prompt
	aiPrompt := fmt.Sprintf(`Generate personalized email content for a %s campaign.

**Contact Information:**
- Name: %s
- Title: %s
- Company: %s
- Pain Points: %s
- Goals: %s

**User Prompt:** %s

**Output Format (JSON):**
{
  "subject": "Email subject line (compelling, personalized)",
  "body": "Email body (2-3 paragraphs, mention pain points/goals, clear CTA)"
}

Return ONLY valid JSON, no markdown.`,
		pb.PlaybookType, name, jobTitle, company, painPoints, goals, prompt,
	)

	// Call AI
	response, err := s.openAI.ChatCompletion(ctx, ai.ChatCompletionRequest{
		Messages: []ai.Message{
			{Role: "user", Content: aiPrompt},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Parse response
	return parseContentResponse(response, name, company, painPoints)
}

// Helper functions

func (s *Service) toPlaybookRead(pb *playbook.Playbook) (*v1schema.PlaybookRead, error) {
	var config v1schema.PlaybookConfig
	if err := json.Unmarshal(pb.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	var performance v1schema.PlaybookPerformance
	if err := json.Unmarshal(pb.Performance, &performance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal performance: %w", err)
	}

	return &v1schema.PlaybookRead{
		PlaybookID:   pb.PlaybookID,
		Name:         pb.Name,
		Description:  pb.Description,
		PlaybookType: v1schema.PlaybookType(pb.PlaybookType),
		Config:       config,
		Performance:  performance,
		IsActive:     pb.IsActive,
		CreatedAt:    pb.CreatedAt,
		UpdatedAt:    pb.UpdatedAt,
	}, nil
}

func extractPainPoints(profile map[string]interface{}) string {
	aiInsights, ok := profile["ai_insights"].(map[string]interface{})
	if !ok {
		return "Unknown"
	}

	painPoints, ok := aiInsights["suspected_pain_points"].([]interface{})
	if !ok || len(painPoints) == 0 {
		return "Unknown"
	}

	if first, ok := painPoints[0].(map[string]interface{}); ok {
		if pp, ok := first["pain_point"].(string); ok {
			return pp
		}
	}

	return "Unknown"
}

func extractGoals(profile map[string]interface{}) string {
	aiInsights, ok := profile["ai_insights"].(map[string]interface{})
	if !ok {
		return "Unknown"
	}

	goals, ok := aiInsights["suspected_goals"].([]interface{})
	if !ok || len(goals) == 0 {
		return "Unknown"
	}

	if first, ok := goals[0].(map[string]interface{}); ok {
		if goal, ok := first["goal"].(string); ok {
			return goal
		}
	}

	return "Unknown"
}

func parseContentResponse(response, name, company, painPoint string) (*v1schema.GenerateContentResponse, error) {
	// Clean response
	response = cleanJSONResponse(response)

	var content struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := json.Unmarshal([]byte(response), &content); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &v1schema.GenerateContentResponse{
		Subject: content.Subject,
		Body:    content.Body,
		PersonalizationData: map[string]string{
			"name":       name,
			"company":    company,
			"pain_point": painPoint,
		},
	}, nil
}

func cleanJSONResponse(response string) string {
	response = trimPrefix(response, "```json")
	response = trimPrefix(response, "```")
	response = trimSuffix(response, "```")
	return trimSpace(response)
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func trimSpace(s string) string {
	// Simple trim spaces
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\t' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
