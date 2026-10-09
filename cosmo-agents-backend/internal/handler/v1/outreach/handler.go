package outreach

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	outreachService "github.com/rockship/cosmo-agents-go/internal/service/outreach"
)

// Handler handles outreach HTTP requests
type Handler struct {
	outreachService *outreachService.OutreachService
	authHelper      *middleware.AuthHelper
	responseHelper  *handler.ResponseHelper
}

// NewHandler creates a new outreach handler
func NewHandler(
	outreachSvc *outreachService.OutreachService,
	userRepo *userRepo.UserRepository,
	roleRepo *roleRepo.RoleRepository,
) *Handler {
	return &Handler{
		outreachService: outreachSvc,
		authHelper:      middleware.NewAuthHelper(userRepo, roleRepo),
		responseHelper:  handler.NewResponseHelper(),
	}
}

// SuggestOutreach suggests contacts for outreach
// GET /v1/outreach/suggest?type=cold|followup|mixed&limit=10
func (h *Handler) SuggestOutreach(c fiber.Ctx) error {
	ctx := c.Context()
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	outreachType := c.Query("type", "mixed")
	limitStr := c.Query("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 10
	}

	// Admin sees all contacts in organization, member sees only their own
	isAdmin := h.authHelper.IsAdminInOrganization(ctx, user.ID, organizationID)

	result, err := h.outreachService.SuggestContacts(ctx, user.ID, organizationID, isAdmin, outreachType, limit)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to suggest contacts", err)
	}

	return h.responseHelper.Success(c, result)
}

// GenerateDraftRequest represents the request body for generating a draft
type GenerateDraftRequest struct {
	Language string `json:"language"` // "vi" (Vietnamese) or "en" (English), default "vi"
	Scenario string `json:"scenario"` // Optional scenario override
}

// GenerateDraft generates an outreach draft for a contact
// POST /v1/outreach/contacts/:contact_id/draft
func (h *Handler) GenerateDraft(c fiber.Ctx) error {
	ctx := c.Context()
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactIDStr := c.Params("contact_id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	// Parse optional request body for language
	var req GenerateDraftRequest
	_ = c.Bind().JSON(&req) // Ignore error - body is optional

	// Validate language (allow empty or "auto" for auto-detection)
	if req.Language != "" && req.Language != "vi" && req.Language != "en" && req.Language != "ja" && req.Language != "auto" {
		return h.responseHelper.BadRequest(c, "Invalid language. Must be 'vi', 'en', 'ja', or 'auto'", nil)
	}

	// Get user's display name for personalization
	userName := user.Name
	if userName == "" {
		userName = user.Email // Fallback to email if name not set
	}

	// Pass language to service - empty/"auto" means auto-detect from contact name
	language := req.Language
	if language == "auto" {
		language = "" // Let service auto-detect
	}

	result, err := h.outreachService.GenerateDraftForHandlerWithLanguage(ctx, user.ID, contactID, language, userName)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to generate draft", err)
	}

	return h.responseHelper.Success(c, result)
}

// UpdateOutreachRequest represents the request body for updating outreach
type UpdateOutreachRequest struct {
	Event     string `json:"event"`     // sent, replied, no_reply, meeting_booked, meeting_done, dropped
	Content   string `json:"content"`   // Message content (optional)
	Channel   string `json:"channel"`   // Channel (LinkedIn, Email, etc.)
	Sentiment string `json:"sentiment"` // positive, neutral, negative (for replies)
}

// UpdateOutreach updates outreach state after an event
// POST /v1/outreach/contacts/:contact_id/update
func (h *Handler) UpdateOutreach(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactIDStr := c.Params("contact_id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	var req UpdateOutreachRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if req.Event == "" {
		return h.responseHelper.BadRequest(c, "Event is required", nil)
	}

	result, err := h.outreachService.UpdateOutreachForHandler(ctx, userID, contactID, req.Event, req.Content, req.Channel, req.Sentiment)
	if errors.Is(err, outreachService.ErrUnknownEvent) {
		return h.responseHelper.BadRequest(c, "Unknown event", err)
	}
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to update outreach", err)
	}

	return h.responseHelper.Success(c, result)
}

// BatchUpdateOutreachItem represents a single item in a batch update request
type BatchUpdateOutreachItem struct {
	ContactID string `json:"contact_id"`
	Event     string `json:"event"`
	Content   string `json:"content"`
	Channel   string `json:"channel"`
	Sentiment string `json:"sentiment"`
}

// BatchUpdateOutreachRequest represents the request body for batch updating outreach
type BatchUpdateOutreachRequest struct {
	Updates []BatchUpdateOutreachItem `json:"updates"`
}

// BatchUpdateOutreachResultItem represents the result of a single batch update
type BatchUpdateOutreachResultItem struct {
	ContactID string      `json:"contact_id"`
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
	Result    interface{} `json:"result,omitempty"`
}

// BatchUpdateOutreach updates outreach state for multiple contacts at once
// POST /v1/outreach/batch-update
func (h *Handler) BatchUpdateOutreach(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req BatchUpdateOutreachRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if len(req.Updates) == 0 {
		return h.responseHelper.BadRequest(c, "Updates array is required and cannot be empty", nil)
	}

	if len(req.Updates) > 50 {
		return h.responseHelper.BadRequest(c, "Maximum 50 updates per batch", nil)
	}

	results := make([]BatchUpdateOutreachResultItem, 0, len(req.Updates))
	for _, item := range req.Updates {
		contactID, err := uuid.Parse(item.ContactID)
		if err != nil {
			results = append(results, BatchUpdateOutreachResultItem{
				ContactID: item.ContactID,
				Success:   false,
				Error:     "Invalid contact ID",
			})
			continue
		}

		if item.Event == "" {
			results = append(results, BatchUpdateOutreachResultItem{
				ContactID: item.ContactID,
				Success:   false,
				Error:     "Event is required",
			})
			continue
		}

		result, err := h.outreachService.UpdateOutreachForHandler(ctx, userID, contactID, item.Event, item.Content, item.Channel, item.Sentiment)
		if err != nil {
			results = append(results, BatchUpdateOutreachResultItem{
				ContactID: item.ContactID,
				Success:   false,
				Error:     err.Error(),
			})
			continue
		}

		results = append(results, BatchUpdateOutreachResultItem{
			ContactID: item.ContactID,
			Success:   true,
			Result:    result,
		})
	}

	return h.responseHelper.Success(c, results)
}

// BatchGenerateDraftRequest represents the request body for batch draft generation
type BatchGenerateDraftRequest struct {
	ContactIDs []string `json:"contact_ids"`
	Language   string   `json:"language"`  // "vi", "en", or "auto" (default)
	AutoSend   bool     `json:"auto_send"` // If true, also log as sent + update outreach stage
}

// BatchGenerateDraftResultItem represents the result of a single draft generation in a batch
type BatchGenerateDraftResultItem struct {
	ContactID          string      `json:"contact_id"`
	Success            bool        `json:"success"`
	Draft              string      `json:"draft,omitempty"`
	State              interface{} `json:"state,omitempty"`
	Scenario           string      `json:"scenario,omitempty"`
	ContactName        string      `json:"contact_name,omitempty"`
	ContactCompany     string      `json:"contact_company,omitempty"`
	ContactJobTitle    string      `json:"contact_job_title,omitempty"`
	ContactInformation string      `json:"contact_information,omitempty"`
	ContactChannel     string      `json:"contact_channel,omitempty"`
	Sent               bool        `json:"sent,omitempty"`
	Error              string      `json:"error,omitempty"`
}

// BatchGenerateDraftResponse represents the response for batch draft generation
type BatchGenerateDraftResponse struct {
	Total     int                            `json:"total"`
	Succeeded int                            `json:"succeeded"`
	Failed    int                            `json:"failed"`
	Results   []BatchGenerateDraftResultItem `json:"results"`
}

// BatchGenerateDraft generates outreach drafts for multiple contacts at once
// POST /v1/outreach/batch-draft
func (h *Handler) BatchGenerateDraft(c fiber.Ctx) error {
	ctx := c.Context()
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req BatchGenerateDraftRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if len(req.ContactIDs) == 0 {
		return h.responseHelper.BadRequest(c, "contact_ids array is required and cannot be empty", nil)
	}

	if len(req.ContactIDs) > 50 {
		return h.responseHelper.BadRequest(c, "Maximum 50 contacts per batch", nil)
	}

	// Validate language
	language := req.Language
	if language == "" || language == "auto" {
		language = "" // Let service auto-detect
	} else if language != "vi" && language != "en" && language != "ja" {
		return h.responseHelper.BadRequest(c, "Invalid language. Must be 'vi', 'en', or 'auto'", nil)
	}

	// Get user's display name for personalization
	userName := user.Name
	if userName == "" {
		userName = user.Email
	}

	// Parse all contact IDs first
	type contactIDPair struct {
		raw    string
		parsed uuid.UUID
	}
	var validContacts []contactIDPair
	results := make([]BatchGenerateDraftResultItem, len(req.ContactIDs))

	for i, idStr := range req.ContactIDs {
		contactID, err := uuid.Parse(idStr)
		if err != nil {
			results[i] = BatchGenerateDraftResultItem{
				ContactID: idStr,
				Success:   false,
				Error:     "Invalid contact ID",
			}
		} else {
			validContacts = append(validContacts, contactIDPair{raw: idStr, parsed: contactID})
		}
	}

	// Generate drafts in parallel using goroutines
	type draftResult struct {
		index  int
		result BatchGenerateDraftResultItem
	}

	ch := make(chan draftResult, len(validContacts))
	for _, vc := range validContacts {
		// Find original index
		idx := -1
		for i, id := range req.ContactIDs {
			if id == vc.raw {
				idx = i
				break
			}
		}

		go func(index int, contactID uuid.UUID, contactIDStr string, autoSend bool) {
			draft, err := h.outreachService.GenerateDraftForHandlerWithLanguage(ctx, user.ID, contactID, language, userName)
			if err != nil {
				ch <- draftResult{
					index: index,
					result: BatchGenerateDraftResultItem{
						ContactID: contactIDStr,
						Success:   false,
						Error:     err.Error(),
					},
				}
				return
			}

			item := BatchGenerateDraftResultItem{
				ContactID:          contactIDStr,
				Success:            true,
				Draft:              draft.Draft,
				State:              draft.State,
				Scenario:           string(draft.Scenario),
				ContactName:        draft.ContactName,
				ContactCompany:     draft.ContactCompany,
				ContactJobTitle:    draft.ContactJobTitle,
				ContactInformation: draft.ContactInformation,
				ContactChannel:     draft.ContactChannel,
			}

			// Auto-send: log as sent + update outreach stage
			if autoSend && draft.Draft != "" {
				channel := draft.ContactChannel
				if channel == "" {
					channel = "LinkedIn"
				}
				_, sendErr := h.outreachService.UpdateOutreachForHandler(ctx, user.ID, contactID, "sent", draft.Draft, channel, "")
				if sendErr != nil {
					item.Error = "Draft generated but failed to send: " + sendErr.Error()
				} else {
					item.Sent = true
				}
			}

			ch <- draftResult{
				index:  index,
				result: item,
			}
		}(idx, vc.parsed, vc.raw, req.AutoSend)
	}

	// Collect results
	for range validContacts {
		dr := <-ch
		results[dr.index] = dr.result
	}

	succeeded := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			succeeded++
		} else {
			failed++
		}
	}

	return h.responseHelper.Success(c, BatchGenerateDraftResponse{
		Total:     len(results),
		Succeeded: succeeded,
		Failed:    failed,
		Results:   results,
	})
}

// GetOutreachState gets the current outreach state for a contact
// GET /v1/outreach/contacts/:contact_id/state
func (h *Handler) GetOutreachState(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactIDStr := c.Params("contact_id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	state, err := h.outreachService.GetOutreachState(ctx, userID, contactID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get outreach state", err)
	}

	if state == nil {
		return h.responseHelper.NotFound(c, "Outreach state not found", nil)
	}

	return h.responseHelper.Success(c, state)
}

// GetInteractionHistory gets interaction history for a contact
// GET /v1/outreach/contacts/:contact_id/interactions?limit=20
func (h *Handler) GetInteractionHistory(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactIDStr := c.Params("contact_id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	}

	interactions, err := h.outreachService.GetInteractionHistory(ctx, userID, contactID, limit)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get interaction history", err)
	}

	return h.responseHelper.Success(c, interactions)
}

// AddInteractionRequest represents the request body for adding interaction history
type AddInteractionRequest struct {
	Content   string `json:"content"`
	Role      string `json:"role"`      // "me" (outgoing) or "client" (incoming)
	Channel   string `json:"channel"`   // LinkedIn, Email, Call, etc.
	Sentiment string `json:"sentiment"` // positive, neutral, negative (optional)
}

// AddInteraction adds a new interaction to conversation history
// POST /v1/outreach/contacts/:contact_id/interactions
func (h *Handler) AddInteraction(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactIDStr := c.Params("contact_id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	var req AddInteractionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if req.Content == "" {
		return h.responseHelper.BadRequest(c, "Content is required", nil)
	}

	if req.Role == "" {
		req.Role = "me" // Default to outgoing
	}

	// Map role to direction
	direction := "outgoing"
	if req.Role == "client" {
		direction = "incoming"
	}

	if req.Channel == "" {
		req.Channel = "LinkedIn"
	}

	interaction, err := h.outreachService.AddInteraction(ctx, userID, contactID, req.Content, direction, req.Channel, req.Sentiment)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to add interaction", err)
	}

	return h.responseHelper.Success(c, interaction)
}

// CreateMeetingRequest represents the request body for creating a meeting
type CreateMeetingRequest struct {
	ContactID       string `json:"contact_id"`
	Title           string `json:"title"`
	Time            string `json:"time"`
	DurationMinutes int    `json:"duration_minutes"`
	Channel         string `json:"channel"`
	Location        string `json:"location"`
	MeetingURL      string `json:"meeting_url"`
	Note            string `json:"note"`
}

// CreateMeeting creates a new meeting for a contact
// POST /v1/outreach/meetings
func (h *Handler) CreateMeeting(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req CreateMeetingRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if req.ContactID == "" {
		return h.responseHelper.BadRequest(c, "contact_id is required", nil)
	}

	contactID, err := uuid.Parse(req.ContactID)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	if req.Time == "" {
		return h.responseHelper.BadRequest(c, "time is required", nil)
	}

	meeting, err := h.outreachService.CreateMeetingForHandler(ctx, userID, contactID, req.Title, req.Time, req.DurationMinutes, req.Channel, req.Location, req.MeetingURL, req.Note)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to create meeting", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   meeting,
	})
}

// UpdateMeetingRequest represents the request body for updating a meeting
type UpdateMeetingRequest struct {
	Status         string `json:"status"`
	Note           string `json:"note"`
	Outcome        string `json:"outcome"`
	NextSteps      string `json:"next_steps"`
	MeetingContent string `json:"meeting_content"`
}

// UpdateMeeting updates a meeting
// PATCH /v1/outreach/meetings/:meeting_id
func (h *Handler) UpdateMeeting(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	meetingIDStr := c.Params("meeting_id")
	meetingID, err := uuid.Parse(meetingIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid meeting ID", err)
	}

	var req UpdateMeetingRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	meeting, err := h.outreachService.UpdateMeetingForHandler(ctx, userID, meetingID, req.Status, req.Note, req.Outcome, req.NextSteps, req.MeetingContent)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to update meeting", err)
	}

	return h.responseHelper.Success(c, meeting)
}

// GenerateMeetingPrepRequest represents the request body for generating meeting prep
type GenerateMeetingPrepRequest struct {
	Language string `json:"language"` // "vi" (Vietnamese) or "en" (English), default "vi"
}

// GenerateMeetingPrep generates meeting preparation document (talking points, discovery questions)
// POST /v1/outreach/meetings/:meeting_id/generate-prep
func (h *Handler) GenerateMeetingPrep(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 120*time.Second)
	defer cancel()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	meetingIDStr := c.Params("meeting_id")
	meetingID, err := uuid.Parse(meetingIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid meeting ID", err)
	}

	// Parse request body for language preference
	var req GenerateMeetingPrepRequest
	if err := c.Bind().JSON(&req); err != nil {
		// If no body provided, default to Vietnamese
		req.Language = "vi"
	}

	// Default to Vietnamese if not specified
	if req.Language == "" {
		req.Language = "vi"
	}

	meeting, err := h.outreachService.GenerateMeetingPrep(ctx, userID, meetingID, req.Language)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to generate meeting prep", err)
	}

	return h.responseHelper.Success(c, meeting)
}

// DeleteMeeting deletes a meeting
// DELETE /v1/outreach/meetings/:meeting_id
func (h *Handler) DeleteMeeting(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	meetingIDStr := c.Params("meeting_id")
	meetingID, err := uuid.Parse(meetingIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid meeting ID", err)
	}

	err = h.outreachService.DeleteMeetingForHandler(ctx, userID, meetingID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to delete meeting", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Meeting deleted"})
}

// GetMeetings gets meetings for a contact
// GET /v1/outreach/contacts/:contact_id/meetings
func (h *Handler) GetMeetings(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactIDStr := c.Params("contact_id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	meetings, err := h.outreachService.GetMeetingsForHandler(ctx, userID, contactID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get meetings", err)
	}

	return h.responseHelper.Success(c, meetings)
}

// GetAllMeetings gets all upcoming scheduled meetings for the authenticated user
// GET /v1/outreach/meetings
func (h *Handler) GetAllMeetings(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	meetings, err := h.outreachService.GetUpcomingMeetingsForHandler(ctx, userID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get meetings", err)
	}

	return h.responseHelper.Success(c, meetings)
}

// ============================================
// Feedback Loop Handlers (Task 8)
// ============================================

// RecordFeedbackActionRequest represents the request body for recording BD action
type RecordFeedbackActionRequest struct {
	FeedbackID    string `json:"feedback_id"`
	Action        string `json:"action"`         // used_draft, modified_draft, wrote_own, skipped
	ActualContent string `json:"actual_content"` // The actual message sent (if different)
}

// RecordFeedbackAction records what BD actually did with a suggestion
// POST /v1/outreach/feedback/action
func (h *Handler) RecordFeedbackAction(c fiber.Ctx) error {
	ctx := c.Context()
	_, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req RecordFeedbackActionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if req.FeedbackID == "" {
		return h.responseHelper.BadRequest(c, "feedback_id is required", nil)
	}

	feedbackID, err := uuid.Parse(req.FeedbackID)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid feedback ID", err)
	}

	if req.Action == "" {
		return h.responseHelper.BadRequest(c, "action is required (used_draft, modified_draft, wrote_own, skipped)", nil)
	}

	err = h.outreachService.RecordAction(ctx, feedbackID, req.Action, req.ActualContent)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to record action", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Action recorded successfully"})
}

// RecordFeedbackOutcomeRequest represents the request body for recording outcome
type RecordFeedbackOutcomeRequest struct {
	ContactID string  `json:"contact_id"`
	Outcome   string  `json:"outcome"`   // sent, replied, meeting_booked, meeting_done, dropped
	Sentiment *string `json:"sentiment"` // positive, neutral, negative (for replies)
}

// RecordFeedbackOutcome records the outcome of an outreach attempt
// POST /v1/outreach/feedback/outcome
func (h *Handler) RecordFeedbackOutcome(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req RecordFeedbackOutcomeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if req.ContactID == "" {
		return h.responseHelper.BadRequest(c, "contact_id is required", nil)
	}

	contactID, err := uuid.Parse(req.ContactID)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	if req.Outcome == "" {
		return h.responseHelper.BadRequest(c, "outcome is required", nil)
	}

	err = h.outreachService.RecordOutcome(ctx, contactID, userID, req.Outcome, req.Sentiment)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to record outcome", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Outcome recorded successfully"})
}

// GetFeedbackStats gets feedback statistics with insights
// GET /v1/outreach/feedback/stats
func (h *Handler) GetFeedbackStats(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	stats, err := h.outreachService.GetFeedbackStats(ctx, userID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get feedback stats", err)
	}

	return h.responseHelper.Success(c, stats)
}

// GetScenarioStats gets statistics grouped by scenario
// GET /v1/outreach/feedback/scenarios
func (h *Handler) GetScenarioStats(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	stats, err := h.outreachService.GetScenarioStats(ctx, userID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get scenario stats", err)
	}

	return h.responseHelper.Success(c, stats)
}

// ============================================
// Contact Notes Handlers (Team Conversation History)
// ============================================

// AddNoteRequest represents the request body for adding a note
type AddNoteRequest struct {
	Content string `json:"content" validate:"required"`
}

// AddNote adds an internal note for a contact (team conversation history)
// POST /v1/outreach/contacts/:contact_id/notes
// @Summary Add note to contact
// @Description Adds an internal team note for a contact
// @Tags Outreach
// @Accept json
// @Produce json
// @Param contact_id path string true "Contact ID"
// @Param body body AddNoteRequest true "Note content"
// @Success 201 {object} map[string]interface{} "Note created"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /v1/outreach/contacts/{contact_id}/notes [post]
func (h *Handler) AddNote(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactIDStr := c.Params("contact_id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	var req AddNoteRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if req.Content == "" {
		return h.responseHelper.BadRequest(c, "content is required", nil)
	}

	note, err := h.outreachService.AddNoteForHandler(ctx, userID, contactID, req.Content)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to add note", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   note,
	})
}

// GetNotes gets all internal notes for a contact
// GET /v1/outreach/contacts/:contact_id/notes?limit=50
// @Summary Get notes for contact
// @Description Gets all internal team notes for a contact
// @Tags Outreach
// @Produce json
// @Param contact_id path string true "Contact ID"
// @Param limit query int false "Maximum number of notes to return (default 50)"
// @Success 200 {object} map[string]interface{} "Notes list"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /v1/outreach/contacts/{contact_id}/notes [get]
func (h *Handler) GetNotes(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactIDStr := c.Params("contact_id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	limitStr := c.Query("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 50
	}

	notes, err := h.outreachService.GetNotesForHandler(ctx, userID, contactID, limit)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get notes", err)
	}

	return h.responseHelper.Success(c, notes)
}

// UpdateNoteRequest represents the request body for updating a note
type UpdateNoteRequest struct {
	Content string `json:"content" validate:"required"`
}

// UpdateNote updates an internal note
// PATCH /v1/outreach/contacts/:contact_id/notes/:note_id
// @Summary Update note
// @Description Updates an internal team note
// @Tags Outreach
// @Accept json
// @Produce json
// @Param contact_id path string true "Contact ID"
// @Param note_id path string true "Note ID"
// @Param body body UpdateNoteRequest true "Note content"
// @Success 200 {object} map[string]interface{} "Note updated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /v1/outreach/contacts/{contact_id}/notes/{note_id} [patch]
func (h *Handler) UpdateNote(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	noteIDStr := c.Params("note_id")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid note ID", err)
	}

	var req UpdateNoteRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if req.Content == "" {
		return h.responseHelper.BadRequest(c, "content is required", nil)
	}

	note, err := h.outreachService.UpdateNoteForHandler(ctx, userID, noteID, req.Content)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to update note", err)
	}

	return h.responseHelper.Success(c, note)
}

// DeleteNote deletes an internal note
// DELETE /v1/outreach/contacts/:contact_id/notes/:note_id
// @Summary Delete note
// @Description Deletes an internal team note
// @Tags Outreach
// @Produce json
// @Param contact_id path string true "Contact ID"
// @Param note_id path string true "Note ID"
// @Success 200 {object} map[string]interface{} "Note deleted"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /v1/outreach/contacts/{contact_id}/notes/{note_id} [delete]
func (h *Handler) DeleteNote(c fiber.Ctx) error {
	ctx := c.Context()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	noteIDStr := c.Params("note_id")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid note ID", err)
	}

	err = h.outreachService.DeleteNoteForHandler(ctx, userID, noteID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to delete note", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Note deleted"})
}
