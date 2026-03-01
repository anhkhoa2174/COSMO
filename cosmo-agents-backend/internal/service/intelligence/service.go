package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	segDomain "github.com/rockship/cosmo-agents-go/internal/domain/segmentation"
	"github.com/rockship/cosmo-agents-go/internal/repository/contact"
	"github.com/rockship/cosmo-agents-go/internal/repository/interaction"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	"github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/internal/skills"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

// Service coordinates AI-style agents (enrichment, scoring) using the shared skills library.
type Service struct {
	contactRepo   *contact.ContactRepository
	contactSkill  *skills.ContactQuerySkill
	interaction   *skills.InteractionLogSkill
	segSkill      *skills.SegmentationLogicSkill
	scoreSkill    *skills.ScoringSkill
	embedSkill    *skills.EmbeddingSkill
	vectorSearch  *skills.VectorSearchSkill
	segmentations *segRepo.SegmentationRepository
	openAI        *ai.OpenAIClient
}

// NewService wires a new intelligence service.
func NewService(
	contactRepo *contact.ContactRepository,
	interactionRepo *interaction.Repository,
	segRepo *segRepo.SegmentationRepository,
	scoreRepo *segRepo.ScoreRepository,
	openAI *ai.OpenAIClient,
	vectorSearch *skills.VectorSearchSkill,
	vectorStore *vectorstore.RedisVectorStore,
) *Service {
	return &Service{
		contactRepo:   contactRepo,
		contactSkill:  skills.NewContactQuerySkill(contactRepo),
		interaction:   skills.NewInteractionLogSkill(interactionRepo),
		segSkill:      skills.NewSegmentationLogicSkill(segRepo),
		scoreSkill:    skills.NewScoringSkill(scoreRepo),
		embedSkill:    skills.NewEmbeddingSkill(openAI, vectorStore),
		vectorSearch:  vectorSearch,
		segmentations: segRepo,
		openAI:        openAI,
	}
}

// EnrichContact runs the contact enrichment agent synchronously.
func (s *Service) EnrichContact(ctx context.Context, userID, orgID, contactID uuid.UUID, forceRefresh bool) (*v1.ContactEnrichmentResponse, error) {
	contactModel, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil || contactModel == nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}
	if !s.canAccessContact(contactModel, userID, orgID) {
		return nil, errors.New("unauthorized contact access")
	}

	contactMap, err := s.contactSkill.GetContact(ctx, contactID)
	if err != nil {
		return nil, err
	}

	interactions, err := s.interaction.ListByContact(ctx, contactID, 20)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[EnrichContact] Building insights for contact %s...\n", contactID)
	insights := s.buildInsights(contactMap, interactions)
	fmt.Printf("[EnrichContact] Insights built successfully\n")

	if err := s.persistAIInsights(ctx, contactID, userID, orgID, insights); err != nil {
		return nil, err
	}

	embeddingText := buildEmbeddingText(contactMap, insights)
	embeddingCreated := false

	// Generate embedding metadata
	metadata := map[string]interface{}{
		"first_name":      getString(contactMap, "first_name"),
		"last_name":       getString(contactMap, "last_name"),
		"company":         getString(contactMap, "company"),
		"job_title":       getString(contactMap, "job_title"),
		"industry":        getString(contactMap, "industry"),
		"contact_channel": getString(contactMap, "contact_channel"),
		"lifecycle_stage": getString(contactMap, "lifecycle_stage"),
		"city":            getString(contactMap, "city"),
		"country":         getString(contactMap, "country"),
	}

	fmt.Printf("[EnrichContact] Generating embedding...\n")
	if _, err := s.embedSkill.GenerateAndStore(ctx, contactID, userID, embeddingText, metadata); err != nil {
		// Log error but don't fail the entire enrichment
		// This allows enrichment to work even without OpenAI API key
		fmt.Printf("Warning: embedding generation failed: %v\n", err)
	} else {
		embeddingCreated = true
		fmt.Printf("[EnrichContact] Embedding created successfully\n")
	}

	return &v1.ContactEnrichmentResponse{
		ContactID:         contactID,
		InsightsGenerated: len(insights.SuspectedPainPoints) + len(insights.SuspectedGoals),
		EmbeddingCreated:  embeddingCreated,
		ConfidenceAvg:     insights.AvgConfidence,
		AIInsights:        insights.ToMap(),
	}, nil
}

// EnrichContactWithContext runs enrichment with org/user context for better insights
func (s *Service) EnrichContactWithContext(ctx context.Context, userID, orgID, contactID uuid.UUID, forceRefresh bool, promptContext string) (*v1.ContactEnrichmentResponse, error) {
	contactModel, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil || contactModel == nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}
	if !s.canAccessContact(contactModel, userID, orgID) {
		return nil, errors.New("unauthorized contact access")
	}

	contactMap, err := s.contactSkill.GetContact(ctx, contactID)
	if err != nil {
		return nil, err
	}

	interactions, err := s.interaction.ListByContact(ctx, contactID, 20)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[EnrichContactWithContext] Building insights for contact %s with context...\n", contactID)

	// Build insights with context
	var insights EnrichmentInsights
	if s.openAI != nil && promptContext != "" {
		insights, err = s.buildAIInsightsWithContext(contactMap, interactions, promptContext)
		if err != nil {
			fmt.Printf("Warning: AI enrichment with context failed: %v, using fallback\n", err)
			insights = s.buildInsights(contactMap, interactions)
		}
	} else {
		insights = s.buildInsights(contactMap, interactions)
	}

	fmt.Printf("[EnrichContactWithContext] Insights built successfully\n")

	if err := s.persistAIInsights(ctx, contactID, userID, orgID, insights); err != nil {
		return nil, err
	}

	embeddingText := buildEmbeddingText(contactMap, insights)
	embeddingCreated := false

	metadata := map[string]interface{}{
		"first_name":      getString(contactMap, "first_name"),
		"last_name":       getString(contactMap, "last_name"),
		"company":         getString(contactMap, "company"),
		"job_title":       getString(contactMap, "job_title"),
		"industry":        getString(contactMap, "industry"),
		"contact_channel": getString(contactMap, "contact_channel"),
		"lifecycle_stage": getString(contactMap, "lifecycle_stage"),
		"city":            getString(contactMap, "city"),
		"country":         getString(contactMap, "country"),
	}

	if _, err := s.embedSkill.GenerateAndStore(ctx, contactID, userID, embeddingText, metadata); err != nil {
		fmt.Printf("Warning: embedding generation failed: %v\n", err)
	} else {
		embeddingCreated = true
	}

	return &v1.ContactEnrichmentResponse{
		ContactID:         contactID,
		InsightsGenerated: len(insights.SuspectedPainPoints) + len(insights.SuspectedGoals),
		EmbeddingCreated:  embeddingCreated,
		ConfidenceAvg:     insights.AvgConfidence,
		AIInsights:        insights.ToMap(),
	}, nil
}

// CalculateSegmentScores evaluates a contact against user segments and persists scores.
func (s *Service) CalculateSegmentScores(ctx context.Context, userID, orgID, contactID uuid.UUID, segmentationIDs []uuid.UUID) (*v1.CalculateScoresResponse, error) {
	contactModel, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil || contactModel == nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}
	if !s.canAccessContact(contactModel, userID, orgID) {
		return nil, errors.New("unauthorized contact access")
	}

	contactMap, err := s.contactSkill.GetContact(ctx, contactID)
	if err != nil {
		return nil, err
	}

	segments, err := s.selectSegments(ctx, userID, segmentationIDs)
	if err != nil {
		return nil, err
	}

	results := make([]v1.SegmentScoreResult, 0, len(segments))
	maxScore := 0

	for _, seg := range segments {
		fit, err := s.segSkill.EvaluateContactFit(ctx, contactMap, seg.ID)
		if err != nil || fit == nil {
			continue
		}
		if !fit.PassesFilters {
			continue
		}

		if err := s.scoreSkill.SaveScore(ctx, contactID, seg.ID, *fit); err != nil {
			return nil, err
		}

		if fit.FitScore > maxScore {
			maxScore = fit.FitScore
		}

		// Auto-enroll contact into segment if fit score meets threshold
		autoEnrolled := s.tryAutoEnrollContact(ctx, contactID, seg, fit.FitScore)

		result := v1.SegmentScoreResult{
			SegmentationID:   seg.ID,
			SegmentationName: seg.Name,
			FitScore:         fit.FitScore,
			ScoreBreakdown:   fit.ScoreBreakdown,
			PassesFilters:    fit.PassesFilters,
		}
		// Add auto_enrolled flag to score breakdown (1 = enrolled, 0 = not enrolled)
		if autoEnrolled {
			if result.ScoreBreakdown == nil {
				result.ScoreBreakdown = make(map[string]int)
			}
			result.ScoreBreakdown["auto_enrolled"] = 1
		}

		results = append(results, result)
	}

	if err := s.persistSegmentScores(ctx, contactID, userID, orgID, results, maxScore); err != nil {
		return nil, err
	}

	return &v1.CalculateScoresResponse{
		ContactID:         contactID,
		SegmentsEvaluated: len(segments),
		SegmentsMatched:   len(results),
		Scores:            results,
		PriorityScore:     maxScore,
	}, nil
}

// VectorSearchContacts searches for similar contacts using semantic search
func (s *Service) VectorSearchContacts(ctx context.Context, userID uuid.UUID, query string, limit int, threshold float64) (*v1.VectorSearchContactsResponse, error) {
	fmt.Printf("[VectorSearchContacts] Starting search - Query: '%s', UserID: %s, Limit: %d, Threshold: %.2f\n", query, userID, limit, threshold)

	// Use SearchContactsByText which includes enrichment with database data and filtering
	return s.SearchContactsByText(ctx, userID, query, limit, threshold)
}

// EmbedContact generates and stores an embedding for a contact without modifying ai_insights.
// Useful for auto-embedding right after contact creation or updates.
func (s *Service) EmbedContact(ctx context.Context, userID uuid.UUID, contactID uuid.UUID) error {
	if s.embedSkill == nil {
		return nil
	}

	contactMap, err := s.contactSkill.GetContact(ctx, contactID)
	if err != nil {
		return err
	}

	text := buildEmbeddingText(contactMap, EnrichmentInsights{})
	metadata := map[string]interface{}{
		"first_name":      getString(contactMap, "first_name"),
		"last_name":       getString(contactMap, "last_name"),
		"company":         getString(contactMap, "company"),
		"job_title":       getString(contactMap, "job_title"),
		"industry":        getString(contactMap, "industry"),
		"contact_channel": getString(contactMap, "contact_channel"),
		"lifecycle_stage": getString(contactMap, "lifecycle_stage"),
		"city":            getString(contactMap, "city"),
		"country":         getString(contactMap, "country"),
	}

	_, err = s.embedSkill.GenerateAndStore(ctx, contactID, userID, text, metadata)
	return err
}

func (s *Service) canAccessContact(contactModel *domain.Contact, userID, orgID uuid.UUID) bool {
	if contactModel.UserID == userID {
		return true
	}
	if contactModel.OrganizationID != nil && *contactModel.OrganizationID == orgID {
		return true
	}
	return false
}

func (s *Service) persistAIInsights(ctx context.Context, contactID, userID, orgID uuid.UUID, insights EnrichmentInsights) error {
	jb := marshalJSONB(insights.ToMap())
	_, err := s.contactRepo.UpdateFields(ctx, contactID, map[string]interface{}{
		"ai_insights": jb,
	}, userID, orgID)
	return err
}

func (s *Service) persistSegmentScores(ctx context.Context, contactID, userID, orgID uuid.UUID, scores []v1.SegmentScoreResult, priority int) error {
	payload := map[string]interface{}{
		"segments":       scores,
		"priority_score": priority,
	}
	_, err := s.contactRepo.UpdateFields(ctx, contactID, map[string]interface{}{
		"scores": marshalJSONB(payload),
	}, userID, orgID)
	return err
}

func (s *Service) selectSegments(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]*segDomain.Segmentation, error) {
	if len(ids) == 0 {
		return s.segmentations.List(ctx, userID, true)
	}
	out := make([]*segDomain.Segmentation, 0, len(ids))
	for _, id := range ids {
		seg, err := s.segmentations.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if seg != nil && seg.UserID == userID {
			out = append(out, seg)
		}
	}
	return out, nil
}

// tryAutoEnrollContact attempts to auto-enroll a contact into a segment if fit score meets threshold
func (s *Service) tryAutoEnrollContact(ctx context.Context, contactID uuid.UUID, segment *segDomain.Segmentation, fitScore int) bool {
	// Parse segment criteria to get auto_enroll_threshold
	var criteria map[string]interface{}
	if err := json.Unmarshal(segment.Criteria, &criteria); err != nil {
		return false
	}

	// Get auto_enroll_threshold from criteria (default 70 if not set)
	threshold := 70
	if val, ok := criteria["auto_enroll_threshold"]; ok {
		switch v := val.(type) {
		case float64:
			threshold = int(v)
		case int:
			threshold = v
		}
	}

	// Check if fit score meets threshold
	if fitScore < threshold {
		return false
	}

	// Update contact_segment_score with enrolled status
	// The scoreSkill.SaveScore already saved the score, now we need to mark it as enrolled
	if err := s.scoreSkill.MarkAsEnrolled(ctx, contactID, segment.ID); err != nil {
		// Log error but don't fail the entire calculation
		fmt.Printf("Failed to mark contact %s as enrolled in segment %s: %v\n", contactID, segment.ID, err)
		return false
	}

	fmt.Printf("Auto-enrolled contact %s into segment %s (fit_score: %d >= threshold: %d)\n",
		contactID, segment.Name, fitScore, threshold)

	return true
}

// EnrichmentInsights holds structured insights for contacts.
type EnrichmentInsights struct {
	SuspectedPainPoints []map[string]any `json:"suspected_pain_points"`
	SuspectedGoals      []map[string]any `json:"suspected_goals"`
	BuyingSignals       []map[string]any `json:"buying_signals"`
	DecisionStyle       map[string]any   `json:"decision_style"`
	CommunicationPrefs  map[string]any   `json:"communication_preferences"`
	AnticipatedObjs     []map[string]any `json:"anticipated_objections"`
	ResearchSuggestions []map[string]any `json:"research_suggestions"`
	AvgConfidence       float64          `json:"avg_confidence"`
}

// ToMap converts insights to a map usable for JSONB storage.
func (e EnrichmentInsights) ToMap() map[string]any {
	return map[string]any{
		"suspected_pain_points":     e.SuspectedPainPoints,
		"suspected_goals":           e.SuspectedGoals,
		"buying_signals":            e.BuyingSignals,
		"decision_style":            e.DecisionStyle,
		"communication_preferences": e.CommunicationPrefs,
		"anticipated_objections":    e.AnticipatedObjs,
		"research_suggestions":      e.ResearchSuggestions,
		"avg_confidence":            e.AvgConfidence,
	}
}

func (s *Service) buildInsights(contact map[string]any, interactions []*v1.InteractionResponse) EnrichmentInsights {
	// Use real AI if available, fallback to mock if OpenAI client is nil
	if s.openAI != nil {
		insights, err := s.buildAIInsights(contact, interactions)
		if err == nil {
			return insights
		}
		// If AI fails, log and fallback to mock
		fmt.Printf("Warning: AI enrichment failed: %v, using fallback logic\n", err)
	}

	// Fallback: Mock logic for development/offline mode
	return s.buildMockInsights(contact, interactions)
}

// buildAIInsights uses GPT-4 to generate personalized insights
func (s *Service) buildAIInsights(contact map[string]any, interactions []*v1.InteractionResponse) (EnrichmentInsights, error) {
	ctx := context.Background()
	now := time.Now().UTC()

	// Build context prompt
	prompt := s.buildEnrichmentPrompt(contact, interactions, now)

	// Call GPT using ChatCompletion
	fmt.Printf("[buildAIInsights] Calling OpenAI API...\n")
	response, err := s.openAI.ChatCompletion(ctx, ai.ChatCompletionRequest{
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		return EnrichmentInsights{}, fmt.Errorf("OpenAI API call failed: %w", err)
	}
	fmt.Printf("[buildAIInsights] OpenAI response received (length: %d)\n", len(response))
	fmt.Printf("[buildAIInsights] Full response:\n%s\n", response)

	// Parse structured JSON response
	insights, err := parseAIInsightsResponse(response, now)
	if err != nil {
		fmt.Printf("[buildAIInsights] Parse failed: %v\nResponse preview: %.200s...\n", err, response)
		return EnrichmentInsights{}, fmt.Errorf("failed to parse AI response: %w", err)
	}
	fmt.Printf("[buildAIInsights] Successfully parsed insights with %d pain points, %d goals\n",
		len(insights.SuspectedPainPoints), len(insights.SuspectedGoals))

	return insights, nil
}

// buildAIInsightsWithContext uses GPT with org/user context for better insights
func (s *Service) buildAIInsightsWithContext(contact map[string]any, interactions []*v1.InteractionResponse, orgContext string) (EnrichmentInsights, error) {
	ctx := context.Background()
	now := time.Now().UTC()

	// Build base prompt
	basePrompt := s.buildEnrichmentPrompt(contact, interactions, now)

	// Inject org/user context at the beginning
	prompt := fmt.Sprintf(`%s

=== CONTACT DATA ===
%s`, orgContext, basePrompt)

	// Call GPT using ChatCompletion
	fmt.Printf("[buildAIInsightsWithContext] Calling OpenAI API with context...\n")
	response, err := s.openAI.ChatCompletion(ctx, ai.ChatCompletionRequest{
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		return EnrichmentInsights{}, fmt.Errorf("OpenAI API call failed: %w", err)
	}
	fmt.Printf("[buildAIInsightsWithContext] OpenAI response received (length: %d)\n", len(response))

	// Parse structured JSON response
	insights, err := parseAIInsightsResponse(response, now)
	if err != nil {
		fmt.Printf("[buildAIInsightsWithContext] Parse failed: %v\n", err)
		return EnrichmentInsights{}, fmt.Errorf("failed to parse AI response: %w", err)
	}
	fmt.Printf("[buildAIInsightsWithContext] Successfully parsed insights with %d pain points, %d goals\n",
		len(insights.SuspectedPainPoints), len(insights.SuspectedGoals))

	return insights, nil
}

// buildMockInsights provides fallback insights using rule-based logic
func (s *Service) buildMockInsights(contact map[string]any, interactions []*v1.InteractionResponse) EnrichmentInsights {
	jobTitle := getString(contact, "job_title")
	company := getString(contact, "company")
	titleLower := strings.ToLower(jobTitle)
	now := time.Now().UTC()

	painPoint := fmt.Sprintf("Wants better automation for %s workflows", company)
	if strings.Contains(titleLower, "marketing") {
		painPoint = "Needs higher reply rates from outbound campaigns"
	} else if strings.Contains(titleLower, "sales") {
		painPoint = "Lacks clear visibility on deal momentum"
	}

	goal := fmt.Sprintf("Improve efficiency for %s team", company)
	if strings.Contains(titleLower, "ops") {
		goal = "Reduce manual data entry across ops stack"
	}

	insights := EnrichmentInsights{
		SuspectedPainPoints: []map[string]any{
			{
				"pain_point":  painPoint,
				"confidence":  0.74,
				"evidence":    []string{"Role and company context (mock)"},
				"detected_at": now.Format(time.RFC3339),
				"status":      "Unconfirmed",
			},
		},
		SuspectedGoals: []map[string]any{
			{
				"goal":        goal,
				"confidence":  0.72,
				"evidence":    []string{"Role responsibilities (mock)"},
				"detected_at": now.Format(time.RFC3339),
				"status":      "Unconfirmed",
			},
		},
		BuyingSignals: []map[string]any{
			{
				"signal":        "Recent engagement",
				"strength":      "Medium",
				"occurred_at":   now.Format(time.RFC3339),
				"recency_score": 65,
			},
		},
		DecisionStyle: map[string]any{
			"type":                 "Analytical",
			"confidence":           0.68,
			"evidence":             []string{"Role seniority and company size (mock)"},
			"key_decision_factors": []string{"ROI", "References"},
		},
		CommunicationPrefs: map[string]any{
			"preferred_channel":    "Email",
			"best_time_to_contact": "Tuesday 10am",
			"confidence":           0.7,
		},
		AnticipatedObjs: []map[string]any{
			{
				"objection":          "Migration risk from current tool",
				"likelihood":         0.62,
				"suggested_response": "Share migration case study",
			},
		},
		ResearchSuggestions: []map[string]any{
			{
				"category":      "Company Intelligence",
				"data_point":    fmt.Sprintf("Find %s's company size, funding stage, and growth trajectory", company),
				"why_important": "Helps tailor value proposition to company maturity level",
				"priority":      "High",
				"where_to_find": "LinkedIn company page, Crunchbase, company website",
			},
			{
				"category":      "Technical Stack",
				"data_point":    "Identify current tools and integrations they use",
				"why_important": "Position product as complementary or better alternative",
				"priority":      "High",
				"where_to_find": "BuiltWith, StackShare, job postings",
			},
			{
				"category":      "Budget & Authority",
				"data_point":    fmt.Sprintf("Confirm if %s has budget authority for %s", jobTitle, company),
				"why_important": "Avoid wasting time if not a decision maker",
				"priority":      "Medium",
				"where_to_find": "LinkedIn connections, mutual contacts, org chart research",
			},
		},
		AvgConfidence: 0.7,
	}

	if len(interactions) > 0 {
		insights.BuyingSignals[0]["signal"] = "Recently interacted"
		insights.BuyingSignals[0]["recency_score"] = 80
	}

	return insights
}

func buildEmbeddingText(contact map[string]any, insights EnrichmentInsights) string {
	// Build embedding text focusing on semantic signals (avoid name/email noise)
	parts := []string{}

	// Profile section (confirmed first)
	// Embed ALL fields from profile (including ALL system and custom fields)
	if profile, ok := contact["profile"].(map[string]interface{}); ok {
		// Fields to skip (will be handled separately below for proper parsing)
		skipFields := map[string]bool{
			"custom_fields":      true,
			"research_findings":  true,
			"ai_insights":        true,
			"confirmed_facts":    true,
			"insight_validation": true,
			"scores":             true,
			"first_name":         true,
			"last_name":          true,
			"email":              true,
			"phone":              true,
		}

		// Embed all profile fields except the ones we'll parse separately
		for fieldName, fieldValue := range profile {
			if skipFields[fieldName] {
				continue
			}

			// Convert field value to string
			if valueStr := fmt.Sprintf("%v", fieldValue); valueStr != "" && valueStr != "<nil>" && valueStr != "map[]" {
				// Convert snake_case to Title Case for readability
				displayName := strings.ReplaceAll(fieldName, "_", " ")
				displayName = strings.Title(displayName)
				parts = append(parts, fmt.Sprintf("%s: %s", displayName, valueStr))
			}
		}

		// Custom fields from profile.custom_fields (with nested value extraction)
		if customFields, ok := profile["custom_fields"].(map[string]interface{}); ok {
			for fieldName, fieldData := range customFields {
				// Extract value from {value, updated_at, source} structure
				if fieldDataMap, ok := fieldData.(map[string]interface{}); ok {
					if value, ok := fieldDataMap["value"]; ok {
						if valueStr := fmt.Sprintf("%v", value); valueStr != "" && valueStr != "<nil>" {
							parts = append(parts, fmt.Sprintf("%s: %s", fieldName, valueStr))
						}
					}
				}
			}
		}

		// Confirmed facts
		if confirmed, ok := profile["confirmed_facts"].(map[string]interface{}); ok {
			for k, v := range confirmed {
				if vs := fmt.Sprintf("%v", v); vs != "" && vs != "<nil>" && vs != "map[]" {
					parts = append(parts, fmt.Sprintf("Confirmed %s: %s", k, vs))
				}
			}
		}

		// AI insights already present - parse and extract meaningful text
		if ai, ok := profile["ai_insights"].(map[string]interface{}); ok {
			// Extract pain points
			if painPoints, ok := ai["suspected_pain_points"].([]interface{}); ok {
				for _, pp := range painPoints {
					if ppMap, ok := pp.(map[string]interface{}); ok {
						if painText, ok := ppMap["pain_point"].(string); ok && painText != "" {
							parts = append(parts, fmt.Sprintf("Pain point: %s", painText))
						}
					}
				}
			}

			// Extract goals
			if goals, ok := ai["suspected_goals"].([]interface{}); ok {
				for _, g := range goals {
					if gMap, ok := g.(map[string]interface{}); ok {
						if goalText, ok := gMap["goal"].(string); ok && goalText != "" {
							parts = append(parts, fmt.Sprintf("Goal: %s", goalText))
						}
					}
				}
			}

			// Extract buying signals
			if signals, ok := ai["buying_signals"].([]interface{}); ok {
				for _, s := range signals {
					if sMap, ok := s.(map[string]interface{}); ok {
						if signalText, ok := sMap["signal"].(string); ok && signalText != "" {
							parts = append(parts, fmt.Sprintf("Buying signal: %s", signalText))
						}
					}
				}
			}

			// Extract decision style
			if decisionStyle, ok := ai["decision_style"].(map[string]interface{}); ok {
				if styleType, ok := decisionStyle["type"].(string); ok && styleType != "" {
					parts = append(parts, fmt.Sprintf("Decision style: %s", styleType))
				}
			}

			// Extract communication preferences
			if commPrefs, ok := ai["communication_preferences"].(map[string]interface{}); ok {
				if channel, ok := commPrefs["preferred_channel"].(string); ok && channel != "" {
					parts = append(parts, fmt.Sprintf("Preferred communication: %s", channel))
				}
			}
		}

		// Research findings already collected
		if findings, ok := profile["research_findings"].([]interface{}); ok && len(findings) > 0 {
			for _, f := range findings {
				if s := fmt.Sprintf("%v", f); s != "" && s != "<nil>" {
					parts = append(parts, fmt.Sprintf("Research finding: %s", s))
				}
			}
		}
	}

	// Top-level contact fields (focus on role/company/industry; skip name/email/phone)
	// Important fields for semantic search
	topLevelFields := []string{
		"company",
		"job_title",
		"industry",          // e.g., Fintech, SaaS, Healthcare
		"contact_channel",   // e.g., LinkedIn, Email
		"lifecycle_stage",   // e.g., new, contacted, replied
		"context_level",     // LOW, MEDIUM, HIGH
		"outreach_decision", // INTRO, FOLLOW-UP, NURTURE, HOLD
		"scenario",          // Role-based, Post-reply, etc.
		"last_outcome",      // Result of last outreach
		"next_step",         // SEND, MEETING, CALL, WAIT, CLOSE
		"city",
		"country",
	}
	for _, fieldName := range topLevelFields {
		if value := getString(contact, fieldName); value != "" && value != "N/A" {
			displayName := strings.ReplaceAll(fieldName, "_", " ")
			displayName = strings.Title(displayName)
			parts = append(parts, fmt.Sprintf("%s: %s", displayName, value))
			// Boost important fields (company, job_title, industry) by repeating
			if fieldName == "company" || fieldName == "job_title" || fieldName == "industry" {
				parts = append(parts, fmt.Sprintf("%s: %s", displayName, value))
			}
		}
	}

	// Tags
	if tags, ok := contact["tags"].([]interface{}); ok && len(tags) > 0 {
		tagStrs := []string{}
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				tagStrs = append(tagStrs, tagStr)
			}
		}
		if len(tagStrs) > 0 {
			parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(tagStrs, ", ")))
		}
	}

	// Notes
	if notes := getString(contact, "notes"); notes != "" {
		parts = append(parts, fmt.Sprintf("Notes: %s", notes))
	}

	// AI Insights
	pains := flatten(insights.SuspectedPainPoints, "pain_point")
	if len(pains) > 0 {
		parts = append(parts, fmt.Sprintf("Pain points: %s", strings.Join(pains, ", ")))
	}

	goals := flatten(insights.SuspectedGoals, "goal")
	if len(goals) > 0 {
		parts = append(parts, fmt.Sprintf("Goals: %s", strings.Join(goals, ", ")))
	}

	return strings.Join(parts, ". ")
}

func flatten(items []map[string]any, key string) []string {
	values := []string{}
	for _, item := range items {
		if v, ok := item[key].(string); ok && v != "" {
			values = append(values, v)
		}
	}
	return values
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func marshalJSONB(m map[string]any) domain.JSONB {
	var jb domain.JSONB
	_ = jb.Marshal(m)
	return jb
}

// buildEnrichmentPrompt creates the GPT-4 prompt for contact enrichment
func (s *Service) buildEnrichmentPrompt(contact map[string]any, interactions []*v1.InteractionResponse, now time.Time) string {
	firstName := getString(contact, "first_name")
	lastName := getString(contact, "last_name")
	company := getString(contact, "company")
	jobTitle := getString(contact, "job_title")
	email := getString(contact, "email")

	// Profile context
	profile := map[string]interface{}{}
	if p, ok := contact["profile"].(map[string]interface{}); ok {
		profile = p
	}

	// Extract interests/tags/custom fields/confirmed facts/ai_insights/research findings
	interests := stringifyField(profile, "interests")
	tags := stringifySlice(contact["tags"])
	customFields := stringifyMap(profile, "custom_fields")
	confirmed := stringifyMap(profile, "confirmed_facts")
	existingAI := stringifyMap(profile, "ai_insights")
	researchFindings := stringifyField(profile, "research_findings")
	scores := stringifyMap(profile, "scores")

	// Build interaction summary
	interactionSummary := "No interactions yet."
	if len(interactions) > 0 {
		interactionSummary = fmt.Sprintf("%d recent interactions:", len(interactions))
		for i, inter := range interactions {
			if i >= 5 { // Limit to 5 most recent
				break
			}
			contentStr := ""
			if inter.Content != nil {
				contentStr = fmt.Sprintf("%v", inter.Content)
			}
			if len(contentStr) > 200 {
				contentStr = contentStr[:200] + "..."
			}
			interactionSummary += fmt.Sprintf("\n- %s via %s on %s (content: %s)",
				inter.InteractionType, inter.Channel, inter.OccurredAt, contentStr)
		}
	}

	prompt := fmt.Sprintf(`You are a sales intelligence AI analyzing a contact for a B2B sales team.

Contact Information:
- Name: %s %s
- Email: %s
- Company: %s
- Job Title: %s
- Confirmed facts (trust these most): %s
- Interests / tags / ICP hints: %s ; %s
- Custom fields (user-provided): %s
- Existing AI insights (do NOT repeat): %s
- Research findings already collected (avoid duplicates): %s
- Existing scores/segmentation signals: %s

%s

Your task: Generate structured insights to help sales reps engage this contact effectively.
You must ground every insight in the provided data. If no signal exists for a claim, skip it.
Prioritize confirmed_facts and interests; avoid generic boilerplate.
When proposing pain_points/goals/objections, explicitly reference combinations (e.g., role + custom_fields + research_findings) and avoid repeating existing_ai.

Analyze and provide insights in this EXACT JSON format:
{
  "suspected_pain_points": [
    {
      "pain_point": "Specific pain point description",
      "confidence": 0.75,
      "evidence": ["Evidence from data"],
      "detected_at": "%s",
      "status": "Unconfirmed"
    }
  ],
  "suspected_goals": [
    {
      "goal": "Specific goal description",
      "confidence": 0.70,
      "evidence": ["Evidence from data"],
      "detected_at": "%s",
      "status": "Unconfirmed"
    }
  ],
  "buying_signals": [
    {
      "signal": "Signal description",
      "strength": "High|Medium|Low",
      "occurred_at": "%s",
      "recency_score": 75
    }
  ],
  "decision_style": {
    "type": "Analytical|Collaborative|Decisive|Experimental",
    "confidence": 0.65,
    "evidence": ["Evidence from role/seniority"],
    "key_decision_factors": ["ROI", "References", "etc"]
  },
  "communication_preferences": {
    "preferred_channel": "Email|LinkedIn|Phone|Meeting",
    "best_time_to_contact": "Best time/day",
    "confidence": 0.60
  },
  "anticipated_objections": [
    {
      "objection": "Likely objection",
      "likelihood": 0.70,
      "suggested_response": "How to address it"
    }
  ],
  "research_suggestions": [
    {
      "category": "Company Intelligence|Technical Stack|Competitive Landscape|Budget & Authority|Timeline & Urgency",
      "data_point": "Specific information to research",
      "why_important": "Why this helps close the deal",
      "priority": "High|Medium|Low",
      "where_to_find": "LinkedIn, company website, etc."
    }
  ],
  "avg_confidence": 0.70
}

CRITICAL REQUIREMENTS:
1. You MUST include at least 3 items in "research_suggestions" array
2. You MUST include at least 2 items in "suspected_pain_points" array
3. You MUST include at least 2 items in "suspected_goals" array
4. Empty arrays are NOT acceptable

Guidelines:
- **CRITICAL**: Analyze ALL fields HOLISTICALLY as a complete picture. Look for patterns across job_title + company + interests + custom_fields + interactions together.
- Derive pain points/goals from COMBINATIONS of data: e.g., "Backend Developer" + "Startup" + custom field "Tech Stack: FastAPI" suggests they need scalable API solutions.
- Research suggestions MUST infer what the contact is looking for based on field combinations. Example: "Frontend Dev" + "Interest: Email campaigns" → suggest researching "email automation tools they might need", "current email marketing stack", "pain points with existing solutions".
- Do NOT propose research we already have in research_findings/custom_fields/confirmed_facts.
- Research suggestions should identify missing data to validate/extend what we have AND predict what solutions/products this contact likely needs.
- Be specific and actionable; tie evidence to provided fields/interactions.
- When contact has custom fields, RE-ANALYZE everything to generate fresh insights based on new data.
- Return ONLY valid JSON, no markdown

Respond with JSON only:`,
		firstName, lastName, email, company, jobTitle,
		confirmed,
		interests, tags,
		customFields,
		existingAI,
		researchFindings,
		scores,
		interactionSummary,
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339))

	return prompt
}

func stringifyField(profile map[string]interface{}, key string) string {
	if v, ok := profile[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func stringifyMap(profile map[string]interface{}, key string) string {
	if m, ok := profile[key].(map[string]interface{}); ok && len(m) > 0 {
		return fmt.Sprintf("%v", m)
	}
	return ""
}

func stringifySlice(v interface{}) string {
	if arr, ok := v.([]interface{}); ok && len(arr) > 0 {
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return strings.Join(out, ", ")
	}
	return ""
}

// parseAIInsightsResponse parses GPT-4 JSON response into EnrichmentInsights
func parseAIInsightsResponse(response string, now time.Time) (EnrichmentInsights, error) {
	// Clean response (remove markdown code blocks if present)
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var insights EnrichmentInsights
	if err := json.Unmarshal([]byte(response), &insights); err != nil {
		return EnrichmentInsights{}, fmt.Errorf("JSON unmarshal failed: %w. Response: %s", err, response)
	}

	return insights, nil
}
