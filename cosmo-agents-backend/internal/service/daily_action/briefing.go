package daily_action

import (
	"fmt"
	"time"

	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
)

// GenerateAgentBriefing builds the agent briefing data from generated actions.
// This creates a deterministic briefing based on the action pipeline state.
// For AI-generated content, this can be extended with OpenAI integration.
func GenerateAgentBriefing(actions []domain.DailyAction, language string) domain.AgentBriefingData {
	categoryCounts := countByCategory(actions)

	// Build category count badges
	var counts []domain.CategoryCount
	categoryMeta := []struct {
		id    domain.CategoryID
		icon  string
		color string
		en    string
		vi    string
	}{
		{domain.CategoryReplied, "💬", "green", "Replied", "Đã trả lời"},
		{domain.CategoryFollowup, "🔄", "orange", "Follow-up", "Theo dõi"},
		{domain.CategoryNewOutreach, "🚀", "blue", "New Outreach", "Tiếp cận mới"},
		{domain.CategoryMeetingPrep, "📅", "purple", "Meeting Prep", "Chuẩn bị họp"},
		{domain.CategoryEnrichment, "🔍", "red", "Enrichment", "Bổ sung dữ liệu"},
	}

	for _, meta := range categoryMeta {
		cnt := categoryCounts[meta.id]
		if cnt == 0 {
			continue
		}
		label := meta.en
		if language == "vi" {
			label = meta.vi
		}
		counts = append(counts, domain.CategoryCount{
			Category: string(meta.id),
			Label:    label,
			Count:    cnt,
			Icon:     meta.icon,
			Color:    meta.color,
		})
	}

	// Build greeting
	greeting := buildGreeting(len(actions), language)

	// Build strategic reasoning
	reasoning := buildStrategicReasoning(actions, categoryCounts, language)

	// Build memory references from high-priority respond/followup actions
	var refs []domain.MemoryReference
	for _, a := range actions {
		if len(refs) >= 3 {
			break
		}
		if a.Type == domain.ActionTypeRespond || (a.Type == domain.ActionTypeFollowup && a.Priority <= 30) {
			var snapshot domain.ContactSnapshot
			if err := a.ContactSnapshot.Unmarshal(&snapshot); err == nil {
				refs = append(refs, domain.MemoryReference{
					ContactID:    snapshot.ID.String(),
					ContactName:  snapshot.Name,
					EventSummary: a.Reasoning,
					Relevance:    "high",
				})
			}
		}
	}

	return domain.AgentBriefingData{
		Greeting:           greeting,
		StrategicReasoning: reasoning,
		MemoryReferences:   refs,
		CategoryCounts:     counts,
	}
}

func buildGreeting(actionCount int, language string) string {
	hour := time.Now().Hour()
	var timeGreeting string

	if language == "vi" {
		switch {
		case hour < 12:
			timeGreeting = "Chào buổi sáng"
		case hour < 18:
			timeGreeting = "Chào buổi chiều"
		default:
			timeGreeting = "Chào buổi tối"
		}
		return fmt.Sprintf("%s! Hôm nay bạn có %d hành động cần thực hiện.", timeGreeting, actionCount)
	}

	switch {
	case hour < 12:
		timeGreeting = "Good morning"
	case hour < 18:
		timeGreeting = "Good afternoon"
	default:
		timeGreeting = "Good evening"
	}
	return fmt.Sprintf("%s! You have %d actions to review today.", timeGreeting, actionCount)
}

func buildStrategicReasoning(actions []domain.DailyAction, counts map[domain.CategoryID]int, language string) string {
	replied := counts[domain.CategoryReplied]
	followup := counts[domain.CategoryFollowup]
	outreach := counts[domain.CategoryNewOutreach]

	if language == "vi" {
		if replied > 0 {
			return fmt.Sprintf("Có %d liên hệ đã phản hồi cần trả lời. Ưu tiên phản hồi trước, sau đó theo dõi %d liên hệ và tiếp cận %d liên hệ mới.", replied, followup, outreach)
		}
		if followup > 0 {
			return fmt.Sprintf("Hôm nay tập trung theo dõi %d liên hệ và tiếp cận %d liên hệ mới.", followup, outreach)
		}
		return fmt.Sprintf("Tập trung tiếp cận %d liên hệ mới hôm nay.", outreach)
	}

	if replied > 0 {
		return fmt.Sprintf("%d contacts have replied and need responses. Prioritize replies first, then follow up with %d contacts and reach out to %d new prospects.", replied, followup, outreach)
	}
	if followup > 0 {
		return fmt.Sprintf("Focus on following up with %d contacts and reaching out to %d new prospects today.", followup, outreach)
	}
	return fmt.Sprintf("Focus on reaching out to %d new prospects today.", outreach)
}
