package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

// GenerateMeetingBrief generates a comprehensive pre-meeting brief for a contact
func (s *Service) GenerateMeetingBrief(ctx context.Context, contactID uuid.UUID) (*v1schema.MeetingBriefResponse, error) {
	// 1. Fetch contact data
	contactMap, err := s.contactSkill.GetContact(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	// 2. Fetch last 3 interactions
	interactions, err := s.interaction.ListByContact(ctx, contactID, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to get interactions: %w", err)
	}

	// 3. Fetch segment scores
	// Note: This requires accessing the score repository which isn't in the Service struct yet
	// For now, we'll extract from contact.profile.segmentation if available
	segments := extractSegments(contactMap)

	// 4. Build AI prompt for meeting prep
	prompt := s.buildMeetingBriefPrompt(contactMap, interactions, segments)

	// 5. Call Claude API to generate meeting brief
	response, err := s.openAI.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: "You are a sales assistant that generates concise, structured meeting briefs in JSON.",
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
		MaxTokens:   1200,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate meeting brief: %w", err)
	}

	// 6. Parse AI response
	brief, err := parseMeetingBriefResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse meeting brief: %w", err)
	}

	// 7. Add last interactions (from database)
	brief.LastInteractions = buildInteractionsList(interactions)

	// 8. Add confirmed facts (from contact profile)
	brief.ConfirmedFacts = extractConfirmedFacts(contactMap)

	// 9. Add segments
	brief.Segments = segments

	return brief, nil
}

// buildMeetingBriefPrompt creates the AI prompt for meeting preparation
func (s *Service) buildMeetingBriefPrompt(contactMap map[string]interface{}, interactions []*v1schema.InteractionResponse, segments []v1schema.MeetingBriefSegment) string {
	firstName := getString(contactMap, "first_name")
	lastName := getString(contactMap, "last_name")
	company := getString(contactMap, "company")
	jobTitle := getString(contactMap, "job_title")

	// Extract AI insights
	profile, _ := contactMap["profile"].(map[string]interface{})
	aiInsights, _ := profile["ai_insights"].(map[string]interface{})
	confirmedFacts, _ := profile["confirmed_facts"].(map[string]interface{})

	// Build interaction summary
	interactionSummary := ""
	for i, interaction := range interactions {
		summary := ""
		if interaction.Content != nil {
			if s, ok := interaction.Content["summary"].(string); ok {
				summary = s
			}
		}
		if summary == "" {
			summary = "(no summary)"
		}
		interactionSummary += fmt.Sprintf("%d. %s: %s\n", i+1, interaction.InteractionType, summary)
	}

	// Build segment summary
	segmentSummary := ""
	for _, seg := range segments {
		segmentSummary += fmt.Sprintf("- %s (Fit Score: %d%%)\n", seg.Name, seg.FitScore)
	}

	prompt := fmt.Sprintf(`You are preparing for a sales meeting with the following contact:

**Contact Information:**
- Name: %s %s
- Title: %s
- Company: %s

**Recent Interactions (Last 3):**
%s

**AI-Detected Insights:**
%s

**Confirmed Facts:**
%s

**Segment Fit:**
%s

**Your Task:**
Generate a comprehensive pre-meeting brief in JSON format with the following structure:

{
  "talking_points": [
    "3-5 key topics to discuss, referencing their pain points and goals"
  ],
  "discovery_questions": [
    "5-7 questions to uncover gaps in our understanding, validate assumptions, and deepen the relationship"
  ],
  "risk_flags": [
    "Any potential concerns, objections, or red flags to be aware of (empty array if none)"
  ]
}

**Guidelines:**
1. Talking Points: Be specific, reference their situation, show you've done research
2. Discovery Questions: Ask about decision process, timeline, budget, technical requirements, success criteria
3. Risk Flags: Only include if there are genuine concerns (low engagement, competitor evaluation, budget constraints, etc.)

Return ONLY valid JSON, no markdown formatting.`,
		firstName, lastName, jobTitle, company,
		interactionSummary,
		formatMap(aiInsights),
		formatMap(confirmedFacts),
		segmentSummary,
	)

	return prompt
}

// parseMeetingBriefResponse parses AI JSON response into MeetingBriefResponse
func parseMeetingBriefResponse(response string) (*v1schema.MeetingBriefResponse, error) {
	// Clean response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var brief struct {
		TalkingPoints      []string `json:"talking_points"`
		DiscoveryQuestions []string `json:"discovery_questions"`
		RiskFlags          []string `json:"risk_flags"`
	}

	if err := json.Unmarshal([]byte(response), &brief); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w. Response: %s", err, response)
	}

	return &v1schema.MeetingBriefResponse{
		TalkingPoints:      brief.TalkingPoints,
		DiscoveryQuestions: brief.DiscoveryQuestions,
		RiskFlags:          brief.RiskFlags,
		// LastInteractions, ConfirmedFacts, Segments will be added by caller
	}, nil
}

// buildInteractionsList converts interaction records to MeetingBriefInteraction
func buildInteractionsList(interactions []*v1schema.InteractionResponse) []v1schema.MeetingBriefInteraction {
	result := make([]v1schema.MeetingBriefInteraction, 0, len(interactions))

	for _, interaction := range interactions {
		occurredAt := interaction.OccurredAt
		summary := ""
		sentimentStr := ""
		if interaction.Content != nil {
			if s, ok := interaction.Content["summary"].(string); ok {
				summary = s
			}
			if s, ok := interaction.Content["sentiment"].(string); ok {
				sentimentStr = s
			}
		}

		var sentiment *string
		if sentimentStr != "" {
			sentiment = &sentimentStr
		}

		result = append(result, v1schema.MeetingBriefInteraction{
			Type:      interaction.InteractionType,
			Date:      occurredAt,
			Summary:   summary,
			Sentiment: sentiment,
		})
	}

	return result
}

// extractConfirmedFacts extracts confirmed facts from contact profile
func extractConfirmedFacts(contactMap map[string]interface{}) []v1schema.MeetingBriefFact {
	profile, ok := contactMap["profile"].(map[string]interface{})
	if !ok {
		return []v1schema.MeetingBriefFact{}
	}

	confirmedFacts, ok := profile["confirmed_facts"].(map[string]interface{})
	if !ok {
		return []v1schema.MeetingBriefFact{}
	}

	result := []v1schema.MeetingBriefFact{}

	// Extract pain points
	if painPoints, ok := confirmedFacts["pain_points"].([]interface{}); ok && len(painPoints) > 0 {
		facts := []string{}
		for _, p := range painPoints {
			if pm, ok := p.(map[string]interface{}); ok {
				if painPoint, ok := pm["pain_point"].(string); ok {
					facts = append(facts, painPoint)
				}
			}
		}
		if len(facts) > 0 {
			result = append(result, v1schema.MeetingBriefFact{
				Category: "Pain Points",
				Facts:    facts,
			})
		}
	}

	// Extract goals
	if goals, ok := confirmedFacts["goals"].([]interface{}); ok && len(goals) > 0 {
		facts := []string{}
		for _, g := range goals {
			if gm, ok := g.(map[string]interface{}); ok {
				if goal, ok := gm["goal"].(string); ok {
					facts = append(facts, goal)
				}
			}
		}
		if len(facts) > 0 {
			result = append(result, v1schema.MeetingBriefFact{
				Category: "Goals",
				Facts:    facts,
			})
		}
	}

	// Extract requirements
	if requirements, ok := confirmedFacts["requirements"].(map[string]interface{}); ok {
		facts := []string{}
		if budget, ok := requirements["budget_timeline"].(map[string]interface{}); ok {
			if amount, ok := budget["amount"].(float64); ok {
				facts = append(facts, fmt.Sprintf("Budget: $%.0f", amount))
			}
		}
		if decisionProcess, ok := requirements["decision_process"].(string); ok && decisionProcess != "" {
			facts = append(facts, fmt.Sprintf("Decision Process: %s", decisionProcess))
		}
		if len(facts) > 0 {
			result = append(result, v1schema.MeetingBriefFact{
				Category: "Budget & Timeline",
				Facts:    facts,
			})
		}
	}

	return result
}

// extractSegments extracts segment information from contact profile
func extractSegments(contactMap map[string]interface{}) []v1schema.MeetingBriefSegment {
	profile, ok := contactMap["profile"].(map[string]interface{})
	if !ok {
		return []v1schema.MeetingBriefSegment{}
	}

	segmentation, ok := profile["segmentation"].(map[string]interface{})
	if !ok {
		return []v1schema.MeetingBriefSegment{}
	}

	segments, ok := segmentation["segments"].([]interface{})
	if !ok {
		return []v1schema.MeetingBriefSegment{}
	}

	result := make([]v1schema.MeetingBriefSegment, 0, len(segments))
	for _, seg := range segments {
		segMap, ok := seg.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := segMap["segment_name"].(string)
		fitScore, _ := segMap["fit_score"].(float64)

		result = append(result, v1schema.MeetingBriefSegment{
			Name:     name,
			FitScore: int(fitScore),
		})
	}

	return result
}

// formatMap formats a map for display in prompt
func formatMap(m map[string]interface{}) string {
	if m == nil || len(m) == 0 {
		return "(None)"
	}

	bytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", m)
	}

	return string(bytes)
}
