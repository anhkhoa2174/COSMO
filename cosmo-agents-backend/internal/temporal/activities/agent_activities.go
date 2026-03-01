package activities

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"

	"github.com/rockship/cosmo-agents-go/internal/repository/contact"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	"github.com/rockship/cosmo-agents-go/internal/service/intelligence"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

// AgentActivities contains all agent-related activities
type AgentActivities struct {
	contactRepo      *contact.ContactRepository
	intelligenceSvc  *intelligence.Service
	segmentationRepo *segRepo.SegmentationRepository
	scoreRepo        *segRepo.ScoreRepository
	openAI           *ai.OpenAIClient
}

// NewAgentActivities creates activities with injected dependencies
func NewAgentActivities(
	contactRepo *contact.ContactRepository,
	intelligenceSvc *intelligence.Service,
	segmentationRepo *segRepo.SegmentationRepository,
	scoreRepo *segRepo.ScoreRepository,
	openAI *ai.OpenAIClient,
) *AgentActivities {
	return &AgentActivities{
		contactRepo:      contactRepo,
		intelligenceSvc:  intelligenceSvc,
		segmentationRepo: segmentationRepo,
		scoreRepo:        scoreRepo,
		openAI:           openAI,
	}
}

// ============ Research Agent Activities ============

type SearchContactsInput struct {
	Query     string            `json:"query"`
	Filters   map[string]string `json:"filters"`
	Limit     int               `json:"limit"`
	SegmentID string            `json:"segment_id,omitempty"`
	UserID    uuid.UUID         `json:"user_id"`
	OrgID     uuid.UUID         `json:"org_id"`
}

type SearchContactsOutput struct {
	Contacts []ContactInfo `json:"contacts"`
	Total    int           `json:"total"`
}

type ContactInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Company string `json:"company"`
	Title   string `json:"title"`
}

func (a *AgentActivities) SearchContacts(ctx context.Context, input SearchContactsInput) (*SearchContactsOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("SearchContacts activity started", "query", input.Query, "user_id", input.UserID)

	if a.contactRepo == nil {
		return &SearchContactsOutput{Contacts: []ContactInfo{}, Total: 0}, nil
	}

	// Build filter from input
	filter := make(map[string]interface{})
	for k, v := range input.Filters {
		filter[k] = v
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	// Search contacts using repository
	contacts, total, err := a.contactRepo.SearchWithFilter(ctx, input.UserID, []uuid.UUID{input.OrgID}, filter, nil)
	if err != nil {
		logger.Error("Failed to search contacts", "error", err)
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}

	// Convert to output format
	result := make([]ContactInfo, 0, len(contacts))
	for _, c := range contacts {
		if len(result) >= limit {
			break
		}
		// Get email from profile
		var email string
		var profile map[string]interface{}
		if err := json.Unmarshal(c.Profile, &profile); err == nil {
			if e, ok := profile["email"].(string); ok {
				email = e
			}
		}
		result = append(result, ContactInfo{
			ID:      c.ID.String(),
			Email:   email,
			Name:    c.Name,
			Company: c.Company,
			Title:   c.JobTitle,
		})
	}

	return &SearchContactsOutput{
		Contacts: result,
		Total:    total,
	}, nil
}

type GetContactInput struct {
	ContactID string    `json:"contact_id"`
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
}

type GetContactOutput struct {
	Contact    ContactInfo            `json:"contact"`
	AIInsights map[string]interface{} `json:"ai_insights,omitempty"`
	FitScores  []SegmentScore         `json:"fit_scores,omitempty"`
}

type SegmentScore struct {
	SegmentID   string  `json:"segment_id"`
	SegmentName string  `json:"segment_name"`
	FitScore    float64 `json:"fit_score"`
	Status      string  `json:"status"`
}

func (a *AgentActivities) GetContact(ctx context.Context, input GetContactInput) (*GetContactOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GetContact activity started", "contact_id", input.ContactID)

	if a.contactRepo == nil {
		return &GetContactOutput{Contact: ContactInfo{ID: input.ContactID}}, nil
	}

	contactID, err := uuid.Parse(input.ContactID)
	if err != nil {
		return nil, fmt.Errorf("invalid contact ID: %w", err)
	}

	// Get contact from repository
	contactModel, err := a.contactRepo.GetByID(ctx, contactID)
	if err != nil {
		logger.Error("Failed to get contact", "error", err)
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	// Get email from profile
	var contactEmail string
	var profile map[string]interface{}
	if err := json.Unmarshal(contactModel.Profile, &profile); err == nil {
		if e, ok := profile["email"].(string); ok {
			contactEmail = e
		}
	}

	output := &GetContactOutput{
		Contact: ContactInfo{
			ID:      contactModel.ID.String(),
			Email:   contactEmail,
			Name:    contactModel.Name,
			Company: contactModel.Company,
			Title:   contactModel.JobTitle,
		},
	}

	// Get AI insights if available
	if contactModel.AIInsights != nil {
		var insights map[string]interface{}
		if err := json.Unmarshal(contactModel.AIInsights, &insights); err == nil {
			output.AIInsights = insights
		}
	}

	return output, nil
}

// ============ Enrichment Agent Activities ============

type EnrichContactInput struct {
	ContactID    string    `json:"contact_id"`
	ForceRefresh bool      `json:"force_refresh"`
	UserID       uuid.UUID `json:"user_id"`
	OrgID        uuid.UUID `json:"org_id"`
}

type EnrichContactOutput struct {
	ContactID         string                 `json:"contact_id"`
	InsightsGenerated bool                   `json:"insights_generated"`
	AIInsights        map[string]interface{} `json:"ai_insights,omitempty"`
}

func (a *AgentActivities) EnrichContact(ctx context.Context, input EnrichContactInput) (*EnrichContactOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("EnrichContact activity started", "contact_id", input.ContactID)

	if a.intelligenceSvc == nil {
		return &EnrichContactOutput{
			ContactID:         input.ContactID,
			InsightsGenerated: false,
		}, nil
	}

	contactID, err := uuid.Parse(input.ContactID)
	if err != nil {
		return nil, fmt.Errorf("invalid contact ID: %w", err)
	}

	// Call intelligence service to enrich contact
	result, err := a.intelligenceSvc.EnrichContact(ctx, input.UserID, input.OrgID, contactID, input.ForceRefresh)
	if err != nil {
		logger.Error("Failed to enrich contact", "error", err)
		return nil, fmt.Errorf("failed to enrich contact: %w", err)
	}

	return &EnrichContactOutput{
		ContactID:         input.ContactID,
		InsightsGenerated: result.InsightsGenerated > 0,
		AIInsights:        result.AIInsights,
	}, nil
}

type CalculateScoresInput struct {
	ContactID  string    `json:"contact_id"`
	SegmentIDs []string  `json:"segment_ids,omitempty"`
	UserID     uuid.UUID `json:"user_id"`
	OrgID      uuid.UUID `json:"org_id"`
}

type CalculateScoresOutput struct {
	ContactID string         `json:"contact_id"`
	Scores    []SegmentScore `json:"scores"`
}

func (a *AgentActivities) CalculateSegmentScores(ctx context.Context, input CalculateScoresInput) (*CalculateScoresOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CalculateSegmentScores activity started", "contact_id", input.ContactID)

	if a.intelligenceSvc == nil {
		return &CalculateScoresOutput{ContactID: input.ContactID, Scores: []SegmentScore{}}, nil
	}

	contactID, err := uuid.Parse(input.ContactID)
	if err != nil {
		return nil, fmt.Errorf("invalid contact ID: %w", err)
	}

	// Parse segment IDs
	var segmentIDs []uuid.UUID
	for _, sid := range input.SegmentIDs {
		if id, err := uuid.Parse(sid); err == nil {
			segmentIDs = append(segmentIDs, id)
		}
	}

	// Calculate scores using intelligence service
	result, err := a.intelligenceSvc.CalculateSegmentScores(ctx, input.UserID, input.OrgID, contactID, segmentIDs)
	if err != nil {
		logger.Error("Failed to calculate scores", "error", err)
		return nil, fmt.Errorf("failed to calculate scores: %w", err)
	}

	// Convert to output format
	scores := make([]SegmentScore, 0, len(result.Scores))
	for _, r := range result.Scores {
		status := "low"
		if r.FitScore >= 80 {
			status = "high"
		} else if r.FitScore >= 50 {
			status = "medium"
		}
		scores = append(scores, SegmentScore{
			SegmentID:   r.SegmentationID.String(),
			SegmentName: r.SegmentationName,
			FitScore:    float64(r.FitScore),
			Status:      status,
		})
	}

	return &CalculateScoresOutput{
		ContactID: input.ContactID,
		Scores:    scores,
	}, nil
}

// ============ Analytics Agent Activities ============

type CountContactsInput struct {
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	SegmentID string    `json:"segment_id,omitempty"`
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
}

type CountContactsOutput struct {
	Count     int    `json:"count"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (a *AgentActivities) CountContactsCreated(ctx context.Context, input CountContactsInput) (*CountContactsOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CountContactsCreated activity started", "start", input.StartDate, "end", input.EndDate)

	// For now, return placeholder - this would need a specific repository method
	// TODO: Implement CountByDateRange in contact repository
	return &CountContactsOutput{
		Count:     0,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
	}, nil
}

type AnalyzeSegmentInput struct {
	SegmentID string    `json:"segment_id"`
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
}

type AnalyzeSegmentOutput struct {
	SegmentID       string   `json:"segment_id"`
	TotalContacts   int      `json:"total_contacts"`
	AverageFitScore float64  `json:"average_fit_score"`
	HealthScore     float64  `json:"health_score"`
	Recommendations []string `json:"recommendations"`
}

func (a *AgentActivities) AnalyzeSegment(ctx context.Context, input AnalyzeSegmentInput) (*AnalyzeSegmentOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("AnalyzeSegment activity started", "segment_id", input.SegmentID)

	// For now, return placeholder - segment analysis would need additional service methods
	// TODO: Implement AnalyzeSegmentHealth in intelligence service
	return &AnalyzeSegmentOutput{
		SegmentID:       input.SegmentID,
		TotalContacts:   0,
		AverageFitScore: 0,
		HealthScore:     0,
		Recommendations: []string{"Analysis not yet implemented"},
	}, nil
}

// ============ Outreach Agent Activities ============

type GenerateEmailInput struct {
	ContactID   string    `json:"contact_id"`
	EmailType   string    `json:"email_type"`
	Context     string    `json:"context,omitempty"`
	OrgContext  string    `json:"org_context,omitempty"`
	UserContext string    `json:"user_context,omitempty"`
	UserID      uuid.UUID `json:"user_id"`
	OrgID       uuid.UUID `json:"org_id"`
}

type GenerateEmailOutput struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Notes   string `json:"notes,omitempty"`
}

func (a *AgentActivities) GenerateEmail(ctx context.Context, input GenerateEmailInput) (*GenerateEmailOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GenerateEmail activity started", "contact_id", input.ContactID, "type", input.EmailType)

	if a.openAI == nil {
		return &GenerateEmailOutput{
			Subject: "Email subject placeholder",
			Body:    "Email body placeholder - OpenAI not configured",
		}, nil
	}

	contactID, err := uuid.Parse(input.ContactID)
	if err != nil {
		return nil, fmt.Errorf("invalid contact ID: %w", err)
	}

	// Get contact info
	var contactInfo string
	if a.contactRepo != nil {
		contactModel, err := a.contactRepo.GetByID(ctx, contactID)
		if err == nil && contactModel != nil {
			contactInfo = fmt.Sprintf("Name: %s\nTitle: %s\nCompany: %s",
				contactModel.Name, contactModel.JobTitle, contactModel.Company)
		}
	}

	// Build prompt for email generation
	prompt := fmt.Sprintf(`Generate a professional %s email for the following contact:

%s

Context: %s
Organization Context: %s
User Context: %s

Generate a subject line and email body. Keep it concise and professional.
Format your response as JSON with "subject" and "body" fields.`,
		input.EmailType, contactInfo, input.Context, input.OrgContext, input.UserContext)

	// Call OpenAI to generate email using ChatCompletion method
	response, err := a.openAI.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: "You are a professional email writer.",
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		logger.Error("Failed to generate email", "error", err)
		return nil, fmt.Errorf("failed to generate email: %w", err)
	}

	// Parse response
	var result GenerateEmailOutput
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If not valid JSON, use response as body
		result = GenerateEmailOutput{
			Subject: fmt.Sprintf("%s Email", input.EmailType),
			Body:    response,
		}
	}

	return &result, nil
}

// ============ Common Activities ============

type SaveConversationInput struct {
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	ToolsUsed []string  `json:"tools_used,omitempty"`
	ContactID string    `json:"contact_id,omitempty"`
}

func (a *AgentActivities) SaveConversation(ctx context.Context, input SaveConversationInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("SaveConversation activity", "session", input.SessionID, "role", input.Role)

	// TODO: Implement conversation persistence when context service is ready
	// For now, just log the conversation
	logger.Info("Conversation saved (placeholder)", "content_length", len(input.Content))
	return nil
}

type LoadContextInput struct {
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
	SessionID string    `json:"session_id,omitempty"`
}

type LoadContextOutput struct {
	OrgContext    map[string]interface{} `json:"org_context"`
	UserContext   map[string]interface{} `json:"user_context"`
	PromptContext string                 `json:"prompt_context"`
}

func (a *AgentActivities) LoadContext(ctx context.Context, input LoadContextInput) (*LoadContextOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("LoadContext activity", "user_id", input.UserID, "org_id", input.OrgID)

	// TODO: Implement context loading when context service is ready
	// For now, return empty context
	return &LoadContextOutput{
		OrgContext:    map[string]interface{}{},
		UserContext:   map[string]interface{}{},
		PromptContext: "",
	}, nil
}

// Helper to convert struct to JSON string
func ToJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// Helper to parse JSON string to struct
func FromJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
