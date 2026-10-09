package mapper

import (
	"encoding/json"

	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	v1 "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// DailyActionToResponse maps a domain DailyAction to a response DTO.
func DailyActionToResponse(a *domain.DailyAction) v1.DailyActionResponse {
	resp := v1.DailyActionResponse{
		ID:              a.ID,
		Type:            string(a.Type),
		Priority:        a.Priority,
		Reasoning:       a.Reasoning,
		Status:          string(a.Status),
		StatusChangedAt: a.StatusChangedAt,
		SnoozeUntil:     a.SnoozeUntil,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}

	// Map contact snapshot
	var contact domain.ContactSnapshot
	if err := a.ContactSnapshot.Unmarshal(&contact); err == nil {
		resp.Contact = ContactSnapshotToResponse(&contact)
	}

	// Map priority factors
	var factors []domain.PriorityFactor
	if err := a.PriorityFactors.Unmarshal(&factors); err == nil {
		resp.PriorityFactors = make([]v1.PriorityFactorResponse, len(factors))
		for i, f := range factors {
			resp.PriorityFactors[i] = v1.PriorityFactorResponse{
				Factor:      f.Factor,
				Value:       f.Value,
				Description: f.Description,
			}
		}
	}

	// Map type-specific data
	switch a.Type {
	case domain.ActionTypeOutreach, domain.ActionTypeFollowup:
		resp.OutreachData = unmarshalOutreachData(a.OutreachData)
	case domain.ActionTypeRespond:
		resp.RespondData = unmarshalRespondData(a.RespondData)
	case domain.ActionTypeMeetingPrep:
		resp.MeetingData = unmarshalMeetingData(a.MeetingData)
	case domain.ActionTypeEnrich:
		resp.EnrichmentData = unmarshalEnrichmentData(a.EnrichmentData)
	}

	return resp
}

// DailyActionsToResponse maps a slice of domain DailyActions to response DTOs.
func DailyActionsToResponse(actions []domain.DailyAction) []v1.DailyActionResponse {
	result := make([]v1.DailyActionResponse, len(actions))
	for i := range actions {
		result[i] = DailyActionToResponse(&actions[i])
	}
	return result
}

// ContactSnapshotToResponse maps a domain ContactSnapshot to a response DTO.
func ContactSnapshotToResponse(c *domain.ContactSnapshot) v1.ActionContactResponse {
	return v1.ActionContactResponse{
		ID:             c.ID,
		Name:           c.Name,
		Email:          c.Email,
		Company:        c.Company,
		JobTitle:       c.JobTitle,
		LinkedinURL:    c.LinkedinURL,
		Source:         c.Source,
		Status:         c.Status,
		OutreachStage:  c.OutreachStage,
		LifecycleStage: c.LifecycleStage,
		AvatarURL:      c.AvatarURL,
	}
}

// AgentBriefingToResponse maps domain AgentBriefingData to a response DTO.
func AgentBriefingToResponse(b *domain.AgentBriefingData) *v1.AgentBriefingResponse {
	if b == nil {
		return nil
	}
	resp := &v1.AgentBriefingResponse{
		Greeting:           b.Greeting,
		StrategicReasoning: b.StrategicReasoning,
	}
	resp.MemoryReferences = make([]v1.MemoryReferenceResponse, len(b.MemoryReferences))
	for i, m := range b.MemoryReferences {
		resp.MemoryReferences[i] = v1.MemoryReferenceResponse{
			ContactID:      m.ContactID,
			ContactName:    m.ContactName,
			EventSummary:   m.EventSummary,
			EventTimestamp: m.EventTimestamp,
			Relevance:      m.Relevance,
		}
	}
	resp.CategoryCounts = make([]v1.CategoryCountResponse, len(b.CategoryCounts))
	for i, c := range b.CategoryCounts {
		resp.CategoryCounts[i] = v1.CategoryCountResponse{
			Category: c.Category,
			Label:    c.Label,
			Count:    c.Count,
			Icon:     c.Icon,
			Color:    c.Color,
		}
	}
	return resp
}

// --- Internal helpers for unmarshalling JSONB type-specific data ---

func unmarshalOutreachData(data []byte) *v1.OutreachActionDataResponse {
	if len(data) == 0 {
		return nil
	}
	var d domain.OutreachActionData
	if err := json.Unmarshal(data, &d); err != nil {
		return nil
	}
	resp := &v1.OutreachActionDataResponse{
		DraftMessage:             d.DraftMessage,
		Scenario:                 d.Scenario,
		ContextLevel:             d.ContextLevel,
		CompanyContext:            d.CompanyContext,
		FollowupNumber:           d.FollowupNumber,
		DaysSinceLastInteraction: d.DaysSinceLastInteraction,
		PreviousMessagesCount:    d.PreviousMessagesCount,
		LastSentDate:             d.LastSentDate,
		IsFinalFollowup:          d.IsFinalFollowup,
	}
	if d.OutreachState != nil {
		resp.OutreachState = &v1.OutreachStateSnapshotResponse{
			ConversationState: d.OutreachState.ConversationState,
			NextStep:          d.OutreachState.NextStep,
			FollowupCount:     d.OutreachState.FollowupCount,
			MaxFollowups:      d.OutreachState.MaxFollowups,
		}
	}
	return resp
}

func unmarshalRespondData(data []byte) *v1.RespondActionDataResponse {
	if len(data) == 0 {
		return nil
	}
	var d domain.RespondActionData
	if err := json.Unmarshal(data, &d); err != nil {
		return nil
	}
	resp := &v1.RespondActionDataResponse{
		ReplyPreview:      d.ReplyPreview,
		ReplyTimestamp:    d.ReplyTimestamp,
		ReplyChannel:      d.ReplyChannel,
		IntentAssessment:  d.IntentAssessment,
		IntentReasoning:   d.IntentReasoning,
		RecommendedAction: d.RecommendedAction,
		DraftResponse:     d.DraftResponse,
	}
	if d.ConversationContext != nil {
		resp.ConversationContext = &v1.ConversationContextResponse{
			TotalInteractions:          d.ConversationContext.TotalInteractions,
			DaysInConversation:         d.ConversationContext.DaysInConversation,
			LastOutgoingMessagePreview: d.ConversationContext.LastOutgoingMessagePreview,
			KeyTopicsDiscussed:         d.ConversationContext.KeyTopicsDiscussed,
		}
	}
	return resp
}

func unmarshalMeetingData(data []byte) *v1.MeetingActionDataResponse {
	if len(data) == 0 {
		return nil
	}
	var d domain.MeetingActionData
	if err := json.Unmarshal(data, &d); err != nil {
		return nil
	}
	resp := &v1.MeetingActionDataResponse{
		MeetingID:              d.MeetingID,
		MeetingTitle:           d.MeetingTitle,
		MeetingTime:            d.MeetingTime,
		MeetingDurationMinutes: d.MeetingDurationMinutes,
		MeetingChannel:         d.MeetingChannel,
		HoursUntilMeeting:      d.HoursUntilMeeting,
	}
	if d.Briefing != nil {
		resp.Briefing = &v1.MeetingBriefingResponse{
			ProspectProfileSummary: d.Briefing.ProspectProfileSummary,
			DiscoveryQuestions:     d.Briefing.DiscoveryQuestions,
			RecommendedNextSteps:   d.Briefing.RecommendedNextSteps,
			RiskFlags:              d.Briefing.RiskFlags,
		}
		if d.Briefing.ConversationSummary != nil {
			resp.Briefing.ConversationSummary = &v1.ConversationSummaryResponse{
				TouchpointCount: d.Briefing.ConversationSummary.TouchpointCount,
				DurationDays:    d.Briefing.ConversationSummary.DurationDays,
				ToneAssessment:  d.Briefing.ConversationSummary.ToneAssessment,
				KeyTopics:       d.Briefing.ConversationSummary.KeyTopics,
			}
		}
		resp.Briefing.PainPoints = make([]v1.PainPointResponse, len(d.Briefing.PainPoints))
		for i, p := range d.Briefing.PainPoints {
			resp.Briefing.PainPoints[i] = v1.PainPointResponse{
				PainPoint:  p.PainPoint,
				Confidence: p.Confidence,
				Evidence:   p.Evidence,
			}
		}
		resp.Briefing.SuggestedAgenda = make([]v1.AgendaItemResponse, len(d.Briefing.SuggestedAgenda))
		for i, a := range d.Briefing.SuggestedAgenda {
			resp.Briefing.SuggestedAgenda[i] = v1.AgendaItemResponse{
				Topic:           a.Topic,
				DurationMinutes: a.DurationMinutes,
				Notes:           a.Notes,
			}
		}
	}
	return resp
}

func unmarshalEnrichmentData(data []byte) *v1.EnrichmentActionDataResponse {
	if len(data) == 0 {
		return nil
	}
	var d domain.EnrichmentActionData
	if err := json.Unmarshal(data, &d); err != nil {
		return nil
	}
	resp := &v1.EnrichmentActionDataResponse{
		MissingFields: d.MissingFields,
		QualityImpact: d.QualityImpact,
		ContactStatus: d.ContactStatus,
	}
	resp.SuggestedSources = make([]v1.SuggestedSourceResponse, len(d.SuggestedSources))
	for i, s := range d.SuggestedSources {
		resp.SuggestedSources[i] = v1.SuggestedSourceResponse{
			Field:  s.Field,
			Source: s.Source,
			URL:    s.URL,
		}
	}
	return resp
}
