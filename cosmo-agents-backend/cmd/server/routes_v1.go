package main

import (
	"bufio"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// RegisterV1Routes registers all V1 API routes
func RegisterV1Routes(app *App, deps *Dependencies) {
	h := deps.V1Handlers
	authMiddleware := middleware.AuthMiddleware(app.JWTManager, deps.Repos.PersonalApiKey, deps.Repos.User)

	// Main v1 group
	v1 := app.Fiber.Group("/v1")
	// Align with Python contract: every request carries X-Auth-Provider (default google)
	v1.Use(middleware.AuthProviderMiddleware())

	// PubSub routes (NO AUTH - public endpoints matching Python)
	// Auth middleware will skip these paths automatically
	pubsub := v1.Group("/pubsub")
	// WebSocket endpoint for subscribing to topics
	pubsub.Get("/:topic/subscribe", h.PubSub.HandleWebSocketSubscribe)
	// Publish message to topic (URL params)
	pubsub.Post("/:topic/publish/:message", h.PubSub.PublishToRoom)
	// Delete topic and close all connections
	pubsub.Delete("/:topic", h.PubSub.DeleteTopic)
	// Legacy HTTP endpoints (for backwards compatibility)
	pubsub.Post("/publish", h.PubSub.PublishMessage)
	pubsub.Get("/subscribe/:topic", h.PubSub.SubscribeToTopic)
	pubsub.Get("/topic/:topic", h.PubSub.GetTopicInfo)

	// Auth routes (no auth required for login)
	// Auth middleware will skip these paths automatically
	auth := v1.Group("/auth")
	auth.Get("", h.Auth.Authorize)
	auth.Post("/login", h.Auth.Login)
	auth.Post("/refresh", h.Auth.RefreshToken)
	auth.Get("/oauth2callback", h.Auth.OAuth2Callback)
	auth.Post("/coze/token", authMiddleware, h.Auth.CozeToken)
	auth.Get("/me", authMiddleware, h.Auth.Me)
	auth.Post("/logout", authMiddleware, h.Auth.Logout)

	// Apply auth middleware to all v1 routes if enabled
	// Public routes (/v1/auth/*, /v1/pubsub/*) will be skipped automatically
	if app.Config.Auth.EnableAuth {
		v1.Use(authMiddleware)
		logger.Logger.Info().Msg("Authentication enabled for /v1 routes (excluding auth and pubsub)")
	} else {
		logger.Logger.Warn().Msg("Authentication DISABLED - for development only")
	}

	// User routes
	if app.Config.Auth.EnableAuth {
		v1.Get("/users/me", authMiddleware, h.User.GetCurrentUser)
		v1.Patch("/users/me", authMiddleware, h.User.UpdateCurrentUser)
		v1.Delete("/users/me", authMiddleware, h.User.DeleteCurrentUser)
	} else {
		v1.Get("/users/me", h.User.GetCurrentUser)
		v1.Patch("/users/me", h.User.UpdateCurrentUser)
		v1.Delete("/users/me", h.User.DeleteCurrentUser)
	}

	// Note: Debug route removed - DebugGetUser method doesn't exist

	// Personal API Keys routes
	v1.Post("/users/:user_id/personal-api-keys", h.User.CreatePersonalAPIKey)
	v1.Get("/users/:user_id/personal-api-keys", h.User.ListPersonalAPIKeys)
	v1.Delete("/users/:user_id/personal-api-keys/:key_id", h.User.DeletePersonalAPIKey)

	// Organization routes
	v1.Post("/organizations", h.Organization.CreateOrganization)
	// Backwards-compatible: accept both /v1/organization (singular) and /v1/organizations (plural) for listing
	v1.Get("/organization", h.Organization.ListOrganizations)
	v1.Get("/organizations", h.Organization.ListOrganizations)
	v1.Patch("/organizations/:id", h.Organization.UpdateOrganization)
	v1.Post("/organizations/:id/members", h.Organization.AddMember)
	v1.Post("/organizations/assign", h.Organization.AssignUser)

	// Contact routes
	v1.Post("/contacts", h.Contact.Create)
	v1.Post("/contacts/bulk", h.Contact.BulkCreate)
	v1.Post("/contacts/search", h.Contact.Search)
	v1.Get("/contact/:id", h.Contact.Get)
	v1.Get("/contact", h.Contact.List)
	v1.Get("/contacts/values", h.Contact.GetFieldValues)
	v1.Patch("/contacts/:id", h.Contact.Update)
	v1.Delete("/contacts", h.Contact.Delete)
	v1.Post("/contacts/import-csv", h.Contact.ImportCSV)
	v1.Post("/contacts/import-hubspot", h.Contact.ImportHubspot)
	v1.Post("/contacts/:id/research-findings", h.Contact.AddResearchFinding)
	v1.Post("/contacts/:id/extract-from-url", h.Contact.ExtractFromURL)
	v1.Post("/contacts/:id/extract-from-image", h.Contact.ExtractFromImage)
	v1.Post("/contacts/:id/extract-from-extension", h.Contact.ExtractFromExtension)
	v1.Post("/extract-from-image-preview", h.Contact.ExtractFromImagePreview)
	v1.Post("/contacts/:id/insights/validate", h.Contact.ValidateInsight)
	v1.Post("/contacts/:id/generate-linkedin-message", h.LinkedIn.GenerateMessage)
	v1.Post("/contacts/:id/enrich", h.Intelligence.EnrichContact)
	v1.Post("/contacts/:id/calculate-scores", h.Intelligence.CalculateScores)
	v1.Post("/contacts/:id/relationship-score", h.Intelligence.ScoreRelationship)
	v1.Post("/contacts/:id/network-analysis", h.Intelligence.AnalyzeNetwork)
	v1.Get("/campaigns/:id/intelligence", h.Intelligence.CampaignIntelligence)
	v1.Post("/contacts/:id/generate-meeting-brief", h.Intelligence.GenerateMeetingBrief)
	v1.Post("/contacts/re-embed-all", h.Contact.ReEmbedAll)
	v1.Post("/contacts/cleanup-vectors", h.Contact.CleanupDeletedVectors)
	v1.Post("/contacts/recalculate-status", h.Contact.RecalculateStatus)

	// Intelligence routes (grouped under /intelligence prefix)
	intelligence := v1.Group("/intelligence")
	intelligence.Post("/contacts/:id/enrich", h.Intelligence.EnrichContact)
	intelligence.Post("/contacts/:id/scores", h.Intelligence.CalculateScores)
	intelligence.Post("/contacts/:id/relationship-score", h.Intelligence.ScoreRelationship)
	intelligence.Post("/contacts/:id/network-analysis", h.Intelligence.AnalyzeNetwork)
	intelligence.Get("/campaigns/:id/intelligence", h.Intelligence.CampaignIntelligence)
	intelligence.Post("/vector-search/contacts", h.Intelligence.VectorSearchContacts)
	intelligence.Post("/vector-search/similar", h.Intelligence.FindSimilarContacts)
	intelligence.Post("/vector-search/knowledge", h.Intelligence.SearchKnowledge)
	intelligence.Post("/vector-search/interactions", h.Intelligence.SearchInteractions)
	intelligence.Post("/vector-search/segments", h.Intelligence.FindSimilarSegments)
	intelligence.Post("/vector-search/hybrid", h.Intelligence.HybridSearchContacts)

	// Segmentation routes
	segmentations := v1.Group("/segmentations")
	segmentations.Post("", h.Segmentation.Create)
	segmentations.Get("", h.Segmentation.List)
	segmentations.Get("/:id", h.Segmentation.Get)
	segmentations.Get("/:id/contacts", h.Segmentation.GetContacts)
	segmentations.Delete("/:id", h.Segmentation.Delete)
	segmentations.Put("/:segmentation_id/contacts/:contact_id/score", h.Segmentation.UpsertScore)
	segmentations.Get("/contacts/:contact_id/scores", h.Segmentation.ListScores)

	// Feedback routes
	feedback := v1.Group("/feedback")
	feedback.Post("/score-adjust", h.Feedback.ScoreAdjust)
	feedback.Post("/insight-validation", h.Feedback.InsightValidation)
	feedback.Post("/custom-fact", h.Feedback.CustomFact)
	feedback.Get("", h.Feedback.List)

	// Legacy Python-compatible /api/v1 contact routes
	apiV1 := app.Fiber.Group("/api/v1")
	if app.Config.Auth.EnableAuth {
		apiV1.Use(authMiddleware)
	}
	apiV1.Post("/contacts", h.Contact.Create)
	apiV1.Delete("/contacts", h.Contact.Delete)
	apiV1.Post("/contacts/search", h.Contact.Search)
	apiV1.Get("/contacts/values", h.Contact.GetFieldValues)
	apiV1.Patch("/contacts/:id", h.Contact.Update)
	apiV1.Post("/contacts/import-csv", h.Contact.ImportCSV)
	apiV1.Post("/contacts/import-hubspot", h.Contact.ImportHubspot)

	// Campaign routes
	campaigns := v1.Group("/campaigns")
	campaigns.Get("", h.Campaign.List)
	campaigns.Post("", h.Campaign.Create)
	campaigns.Post("/search", h.Campaign.SearchCampaigns)
	campaigns.Get("/:id", h.Campaign.GetByID)
	campaigns.Patch("/:id", h.Campaign.Update)
	campaigns.Patch("/:id/client-metadata", h.Campaign.UpdateClientMetadata)
	campaigns.Patch("/:id/save-outreach", h.Campaign.SaveOutreach)
	campaigns.Delete("/:id", h.Campaign.Delete)
	campaigns.Post("/:id/generate", h.Campaign.GenerateTemplates)
	campaigns.Post("/:id/assign", h.Campaign.AssignMember)
	campaigns.Post("/:id/follow-up-schedule", h.Campaign.SetFollowUpSchedule)
	campaigns.Delete("/:id/notifications", h.Campaign.DeleteNotifications)

	// MCP routes (Model Context Protocol integration)
	mcp := v1.Group("/mcp")
	mcp.Post("/campaigns", h.MCP.CreateCampaignWithContacts)
	mcp.Get("/contacts/pipeline", h.MCP.GetContactsPipeline)
	mcp.Get("/contacts/pipeline-summary", h.MCP.GetPipelineSummary)
	mcp.Get("/contacts/:contact_id/interactions", h.MCP.GetContactInteractions)
	mcp.Get("/outcome-metrics", h.MCP.GetOutcomeMetrics)
	mcp.Post("/daily-actions", h.MCP.CreateDailyActions)
	mcp.Get("/daily-actions/status", h.MCP.GetDailyActionsStatus)

	// Agent routes (pluralized prefix: /v1/agents)
	v1.Post("/agents", h.Agent.Create)
	v1.Get("/agents/:id", h.Agent.GetByID)
	v1.Post("/agents/search", h.Agent.Search)
	v1.Put("/agents/:id", h.Agent.Update)
	v1.Patch("/agents/:id", h.Agent.Update)
	v1.Delete("/agents/:id", h.Agent.Delete)

	// Conversations search route
	v1.Post("/agents/:id/conversations/search", h.Agent.GetConversations)

	// Task routes
	v1.Post("/task", h.Task.Create)
	v1.Get("/task/:id", h.Task.GetByID)
	v1.Get("/task", h.Task.List)
	v1.Put("/task/:id", h.Task.Update)

	// Template routes
	v1.Post("/template", h.Template.Create)
	v1.Get("/template/:id", h.Template.GetByID)
	v1.Get("/template", h.Template.List)
	v1.Put("/template/:id", h.Template.Update)
	v1.Delete("/template/:id", h.Template.Delete)

	// Email routes
	v1.Get("/emails/:id", h.Email.GetByID)
	v1.Get("/emails", h.Email.List)

	// New conversation routes matching Python API
	v1.Post("/conversations/search", h.Conversation.Search)
	v1.Get("/conversations/assignee", h.Conversation.GetAssignees)
	v1.Get("/conversations/:id", h.Conversation.GetConversationDetail)
	// Registered before the /:id routes so "trash" is not captured as an id.
	v1.Delete("/conversations/trash", h.Conversation.EmptyTrash)
	v1.Delete("/conversations/:id", h.Conversation.DeleteConversation)
	v1.Post("/conversations/:id/restore", h.Conversation.RestoreConversation)
	v1.Delete("/conversations/:id/permanent", h.Conversation.PurgeConversation)
	v1.Post("/conversations/:id/assign", h.Conversation.AssignConversation)
	v1.Post("/conversations/:id/intent", h.Conversation.CorrectIntent)

	// Personal conversation groups for the AI Inbox
	v1.Get("/conversation-groups", h.ConversationGroup.List)
	v1.Post("/conversation-groups", h.ConversationGroup.Create)
	v1.Patch("/conversation-groups/:id", h.ConversationGroup.Update)
	v1.Delete("/conversation-groups/:id", h.ConversationGroup.Delete)
	v1.Post("/conversation-groups/:id/conversations", h.ConversationGroup.AddConversation)
	v1.Delete("/conversation-groups/:id/conversations/:conversation_id", h.ConversationGroup.RemoveConversation)

	// Gmail routes - Temporarily disabled due to missing implementation
	// v1.Post("/gmail/auth/url", h.Gmail.GetAuthURL)
	// v1.Get("/gmail/auth/callback", h.Gmail.HandleCallback)
	// v1.Post("/gmail/auth/refresh", h.Gmail.RefreshToken)
	// v1.Get("/gmail/profile/:agent_id", h.Gmail.GetProfile)
	// v1.Post("/gmail/send", h.Gmail.SendEmail)

	// Task enqueue routes - Temporarily disabled due to missing implementation
	// tasks := v1.Group("/tasks/enqueue")
	// tasks.Post("/send-email", h.TaskEnqueue.EnqueueSendEmail)
	// tasks.Post("/execute-campaign", h.TaskEnqueue.EnqueueExecuteCampaign)
	// tasks.Post("/schedule-tasks", h.TaskEnqueue.EnqueueScheduleTasks)
	// tasks.Post("/sync-agent", h.TaskEnqueue.EnqueueSyncAgent)

	// Sale Rep routes
	v1.Post("/sale-reps", h.SaleRep.Create)
	v1.Get("/sale-reps/:id", h.SaleRep.GetByID)
	v1.Post("/sale-reps/search", h.SaleRep.Search)
	v1.Patch("/sale-reps/:id", h.SaleRep.Update)
	v1.Delete("/sale-reps/:id", h.SaleRep.Delete)

	// Custom Field routes
	v1.Post("/custom-fields", h.CustomField.Create)
	v1.Get("/custom-fields", h.CustomField.List)
	v1.Patch("/custom-fields/:id", h.CustomField.Update)
	v1.Delete("/custom-fields/:id", h.CustomField.Delete)

	// List Contact routes
	v1.Post("/list-contacts", h.ListContact.Create)
	v1.Get("/list-contacts/:id", h.ListContact.GetByID)
	v1.Post("/list-contacts/search", h.ListContact.Search)
	v1.Patch("/list-contacts/:id", h.ListContact.Update)
	v1.Delete("/list-contacts", h.ListContact.Delete)

	// Workflow routes
	v1.Post("/workflows", h.Workflow.Create)
	v1.Get("/workflows/:id", h.Workflow.GetByID)
	v1.Get("/workflows", h.Workflow.List)
	v1.Patch("/workflows/:id", h.Workflow.Update)
	v1.Delete("/workflows/:id", h.Workflow.Delete)

	// Knowledge routes (matches Python V1 API)
	v1.Get("/knowledge", h.Knowledge.List)           // List knowledge entries
	v1.Delete("/knowledge/:id", h.Knowledge.Delete)  // Delete knowledge entry
	v1.Post("/knowledge/upload", h.Knowledge.Upload) // Upload knowledge files
	v1.Post("/knowledge/search", h.Knowledge.Search) // Search knowledge base

	// AI routes
	aiGroup := v1.Group("/ai")
	aiCompanies := aiGroup.Group("/companies")
	aiCompanies.Get("/extract", h.AICompany.ExtractCompanyInfo)

	aiEmails := aiGroup.Group("/emails")
	aiEmails.Post("/generate", h.AIEmail.GenerateEmailTemplates)
	aiEmails.Post("/classify-intent", h.AIEmail.ClassifyIntent)
	aiEmails.Post("/reply", h.AIEmail.GenerateReply)

	// HubSpot routes
	hubspotGroup := v1.Group("/hubspot")
	hubspotGroup.Get("/authorize", h.Hubspot.Authorize)
	hubspotGroup.Get("/callback", h.Hubspot.Callback)
	hubspotGroup.Get("/users/me", h.Hubspot.GetUserInfo)

	// Outlook routes
	outlookGroup := v1.Group("/outlook")
	outlookGroup.Get("/authorize", h.Outlook.Authorize)
	outlookGroup.Get("/oauth2callback", h.Outlook.Callback)
	outlookGroup.Get("/refresh_token", h.Outlook.RefreshToken)
	outlookGroup.Get("/contacts", h.Outlook.GetContacts)
	outlookGroup.Get("/emails", h.Outlook.GetEmails)
	outlookGroup.Post("/send_email", h.Outlook.SendMail)
	outlookGroup.Post("/reply_email", h.Outlook.ReplyMail)
	outlookGroup.Post("/forward_email", h.Outlook.ForwardMail)

	// Inbound lead form routes
	inboundLeadForms := v1.Group("/inbound-lead-forms")
	inboundLeadForms.Post("", h.InboundLeadForm.Create)
	inboundLeadForms.Get("", h.InboundLeadForm.List)
	inboundLeadForms.Get("/:identifier", h.InboundLeadForm.Get)
	inboundLeadForms.Put("/:identifier", h.InboundLeadForm.Update)
	inboundLeadForms.Delete("/:identifier", h.InboundLeadForm.Delete)
	inboundLeadForms.Post("/:identifier/submit", h.InboundLeadForm.Submit)

	// Public lead form routes: visitors render and submit a form without an
	// account. /v1/public/ is exempt from auth, so these are rate limited per
	// IP to keep a script from filling the owner's contacts.
	publicLeadForms := v1.Group("/public/inbound-lead-forms")
	publicLeadForms.Get("/:identifier", middleware.PerIPRateLimit(60, time.Minute), h.InboundLeadForm.GetPublic)
	publicLeadForms.Post("/:identifier/submit", middleware.PerIPRateLimit(10, time.Minute), h.InboundLeadForm.Submit)

	// File routes (S3 operations - only if S3 is configured)
	if h.File != nil {
		filesGroup := v1.Group("/files")
		filesGroup.Get("", h.File.List)
		filesGroup.Post("/search", h.File.Search)
		filesGroup.Post("/s3", h.File.UploadToS3)
		filesGroup.Get("/s3", h.File.GetFromS3)
		filesGroup.Get("/s3/list", h.File.ListFromS3)
		filesGroup.Get("/s3/presigned-url", h.File.GetPresignedURL)
		filesGroup.Delete("/s3", h.File.DeleteFromS3)
	}

	// Lab routes (experimental)
	lab := v1.Group("/lab")
	lab.Post("/generate-sample-response", h.Lab.GenerateSampleResponse)
	lab.Get("/conversations/:conversation_id/emails", h.Lab.GetEmailsWithinConversation)

	// Playbook routes
	playbooks := v1.Group("/playbooks")
	playbooks.Post("", h.Playbook.Create)
	playbooks.Get("", h.Playbook.List)
	playbooks.Get("/:id", h.Playbook.Get)
	playbooks.Delete("/:id", h.Playbook.Delete)
	playbooks.Post("/:id/stages/:stage_id/generate-content", h.Playbook.GenerateContent)

	// Automation rule routes
	automationRules := v1.Group("/automation-rules")
	automationRules.Post("", h.AutomationRule.Create)
	automationRules.Get("", h.AutomationRule.List)
	automationRules.Get("/:id", h.AutomationRule.Get)
	automationRules.Patch("/:id/toggle", h.AutomationRule.Toggle)
	automationRules.Delete("/:id", h.AutomationRule.Delete)

	// Enrollment routes
	v1.Post("/contacts/:id/enroll", h.Enrollment.EnrollContact)
	v1.Get("/enrollments/:id", h.Enrollment.GetEnrollment)
	v1.Patch("/enrollments/:id/status", h.Enrollment.UpdateEnrollmentStatus)

	// Approval routes
	approvals := v1.Group("/enrollment-approvals")
	approvals.Get("/pending", h.Enrollment.ListPendingApprovals)
	approvals.Post("/:id/approve", h.Enrollment.ApproveEnrollment)
	approvals.Post("/:id/reject", h.Enrollment.RejectEnrollment)

	// Outreach routes (Phase 2 - Outreach Decision + Draft Engine)
	outreach := v1.Group("/outreach")
	outreach.Get("/suggest", h.Outreach.SuggestOutreach)
	outreach.Post("/contacts/:contact_id/draft", h.Outreach.GenerateDraft)
	outreach.Post("/contacts/:contact_id/update", h.Outreach.UpdateOutreach)
	outreach.Post("/batch-update", h.Outreach.BatchUpdateOutreach)
	outreach.Post("/batch-draft", h.Outreach.BatchGenerateDraft)
	outreach.Get("/contacts/:contact_id/state", h.Outreach.GetOutreachState)
	outreach.Get("/contacts/:contact_id/interactions", h.Outreach.GetInteractionHistory)
	outreach.Post("/contacts/:contact_id/interactions", h.Outreach.AddInteraction)
	outreach.Get("/contacts/:contact_id/meetings", h.Outreach.GetMeetings)
	outreach.Get("/meetings", h.Outreach.GetAllMeetings)
	outreach.Post("/meetings", h.Outreach.CreateMeeting)
	outreach.Patch("/meetings/:meeting_id", h.Outreach.UpdateMeeting)
	outreach.Delete("/meetings/:meeting_id", h.Outreach.DeleteMeeting)
	outreach.Post("/meetings/:meeting_id/generate-prep", h.Outreach.GenerateMeetingPrep)

	// Contact Notes routes (Team Conversation History)
	outreach.Post("/contacts/:contact_id/notes", h.Outreach.AddNote)
	outreach.Get("/contacts/:contact_id/notes", h.Outreach.GetNotes)
	outreach.Patch("/contacts/:contact_id/notes/:note_id", h.Outreach.UpdateNote)
	outreach.Delete("/contacts/:contact_id/notes/:note_id", h.Outreach.DeleteNote)

	// Feedback Loop routes (Task 8)
	outreach.Post("/feedback/action", h.Outreach.RecordFeedbackAction)
	outreach.Post("/feedback/outcome", h.Outreach.RecordFeedbackOutcome)
	outreach.Get("/feedback/stats", h.Outreach.GetFeedbackStats)
	outreach.Get("/feedback/scenarios", h.Outreach.GetScenarioStats)

	// Daily Actions routes (Daily BD Actions API)
	// Simple SSE test endpoint (no auth) — raw fasthttp streaming
	v1.Get("/sse-test", func(c fiber.Ctx) error {
		ctx := c.RequestCtx()
		ctx.Response.Header.Set("Content-Type", "text/event-stream")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")
		ctx.Response.Header.SetStatusCode(200)
		ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
			fmt.Fprintf(w, "retry: 3000\n\n")
			w.Flush()
			fmt.Fprintf(w, "data: hello from sse-test\n\n")
			w.Flush()
			time.Sleep(30 * time.Second)
		})
		return nil
	})

	dailyActions := v1.Group("/daily-actions")
	dailyActions.Post("/generate", h.DailyAction.Generate)
	dailyActions.Get("", h.DailyAction.GetDailyActions)
	dailyActions.Patch("/:action_id", h.DailyAction.UpdateAction)
	dailyActions.Get("/events", h.DailyAction.SSEStream)
	dailyActions.Get("/summary", h.DailyAction.GetSummary)
	dailyActions.Get("/categories/:category_id/actions", h.DailyAction.LoadMoreActions)
	dailyActions.Post("/chat", h.DailyAction.Chat)
	dailyActions.Get("/outcome-metrics", h.DailyAction.GetOutcomeMetrics)
	dailyActions.Post("/create-from-agent", h.DailyAction.CreateFromAgent)

	// Context routes (Agent Context Management)
	ctx := v1.Group("/context")
	ctx.Get("/org", h.Context.GetOrgContext)
	ctx.Patch("/org", h.Context.UpdateOrgContext)
	ctx.Get("/user", h.Context.GetUserContext)
	ctx.Patch("/user", h.Context.UpdateUserContext)
	ctx.Get("/merged", h.Context.GetMergedContext)
	ctx.Get("/history", h.Context.GetConversationHistory)
	ctx.Delete("/history", h.Context.ClearConversationHistory)
}
