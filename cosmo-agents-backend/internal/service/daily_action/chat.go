package daily_action

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
)

// GenerateChatResponse builds a context-aware response based on the user's message
// and current pipeline state. This is a rule-based implementation; can be extended
// with OpenAI integration for more natural conversations.
func (s *Service) GenerateChatResponse(ctx context.Context, userID uuid.UUID, message string, gen *domain.DailyActionGeneration) string {
	lower := strings.ToLower(message)

	// Handle common intents
	switch {
	case containsAny(lower, "how many", "bao nhiêu", "total", "tổng"):
		return s.handlePipelineQuery(ctx, gen)
	case containsAny(lower, "summary", "tóm tắt", "overview", "tổng quan"):
		return s.handleSummaryQuery(ctx, gen)
	case containsAny(lower, "priority", "ưu tiên", "urgent", "quan trọng"):
		return s.handlePriorityQuery(ctx, gen)
	case containsAny(lower, "help", "giúp", "what can you", "bạn có thể"):
		return "I can help you with your daily BD actions! Try asking:\n- \"How many actions do I have today?\"\n- \"Give me a summary\"\n- \"What's my top priority?\"\n- \"Show my progress\""
	case containsAny(lower, "progress", "tiến độ", "status"):
		return s.handleProgressQuery(ctx, gen)
	default:
		return fmt.Sprintf("I received your message: \"%s\". I'm your BD assistant — I can help with action summaries, priorities, and pipeline insights. What would you like to know?", message)
	}
}

func (s *Service) handlePipelineQuery(ctx context.Context, gen *domain.DailyActionGeneration) string {
	if gen == nil {
		return "No actions generated for today yet. Would you like me to generate them?"
	}
	statusCounts, _ := s.actionRepo.CountByGenerationAndStatus(ctx, gen.ID)
	total, completed := 0, 0
	for _, cnt := range statusCounts {
		total += cnt
	}
	completed = statusCounts[domain.ActionStatusCompleted]
	// Skipped actions are decided, not remaining — same rule as the summary.
	remaining := total - completed - statusCounts[domain.ActionStatusSkipped]
	return fmt.Sprintf("You have %d actions today. %d completed, %d remaining.", total, completed, remaining)
}

func (s *Service) handleSummaryQuery(ctx context.Context, gen *domain.DailyActionGeneration) string {
	if gen == nil {
		return "No daily actions have been generated yet today."
	}
	statusCounts, _ := s.actionRepo.CountByGenerationAndStatus(ctx, gen.ID)
	total, completed, skipped, snoozed := 0, 0, 0, 0
	for st, cnt := range statusCounts {
		total += cnt
		switch st {
		case domain.ActionStatusCompleted:
			completed += cnt
		case domain.ActionStatusSkipped:
			skipped += cnt
		case domain.ActionStatusSnoozed:
			snoozed += cnt
		}
	}
	remaining := total - completed - skipped
	return fmt.Sprintf("Today's summary: %d total actions. %d completed, %d skipped, %d snoozed, %d remaining.",
		total, completed, skipped, snoozed, remaining)
}

func (s *Service) handlePriorityQuery(ctx context.Context, gen *domain.DailyActionGeneration) string {
	if gen == nil {
		return "No actions generated yet. Generate your daily actions first."
	}
	// Top 3 open actions by priority, across every category. (The per-category
	// query cannot be used here: an empty category matches nothing.)
	all, _ := s.actionRepo.FindByGenerationID(ctx, gen.ID)
	now := time.Now()
	var actions []domain.DailyAction
	for _, a := range all {
		if a.Status == domain.ActionStatusCompleted || a.Status == domain.ActionStatusSkipped ||
			(a.SnoozeUntil != nil && a.SnoozeUntil.After(now)) {
			continue
		}
		if actions = append(actions, a); len(actions) == 3 {
			break
		}
	}
	if len(actions) == 0 {
		return "All actions are completed! Great work."
	}
	var lines []string
	for i, a := range actions {
		var snapshot domain.ContactSnapshot
		_ = a.ContactSnapshot.Unmarshal(&snapshot)
		lines = append(lines, fmt.Sprintf("%d. %s — %s (priority: %d)", i+1, snapshot.Name, a.Reasoning, a.Priority))
	}
	return "Your top priorities:\n" + strings.Join(lines, "\n")
}

func (s *Service) handleProgressQuery(ctx context.Context, gen *domain.DailyActionGeneration) string {
	if gen == nil {
		return "No actions generated for today."
	}
	statusCounts, _ := s.actionRepo.CountByGenerationAndStatus(ctx, gen.ID)
	total, completed := 0, 0
	for st, cnt := range statusCounts {
		total += cnt
		if st == domain.ActionStatusCompleted {
			completed += cnt
		}
	}
	if total == 0 {
		return "No actions today."
	}
	pct := float64(completed) / float64(total) * 100
	return fmt.Sprintf("Progress: %d/%d (%.0f%%) completed.", completed, total, pct)
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
