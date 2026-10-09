package main

import (
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// RegisterV2Routes registers all V2 API routes
func RegisterV2Routes(app *App, deps *Dependencies) {
	v2 := app.Fiber.Group("/v2")
	h := deps.V2Handlers
	authMiddleware := middleware.AuthMiddleware(app.JWTManager, deps.Repos.PersonalApiKey, deps.Repos.User)

	v2.Use(middleware.AuthProviderMiddleware())

	// V2 Auth routes (no auth middleware required)
	v2Auth := v2.Group("/auth")
	v2Auth.Get("/oauth2callback", h.Auth.OAuth2Callback)
	v2Auth.Post("/refresh", h.Auth.RefreshToken)
	v2Auth.Post("/signout", authMiddleware, h.Auth.SignOut)
	// Parity endpoints with Python
	v2Auth.Get("/adapter/google", h.Auth.AdapterGoogle)
	v2Auth.Get("/members", h.Auth.MembersAuthURL)
	v2Auth.Get("/members/oauth2callback", h.Auth.MembersOAuth2Callback)
	v2Auth.Get("/members/invite-callback", h.Auth.MembersInviteCallback)

	// Apply auth middleware to other V2 routes
	if app.Config.Auth.EnableAuth {
		v2.Use(authMiddleware)
		logger.Logger.Info().Msg("Authentication enabled for /v2 routes")
	}

	// V2 User routes
	v2.Get("/users/me", h.User.WhoAmI)

	// V2 Draft Template routes
	v2.Get("/draft-templates/:draft_template_id", h.DraftTemplate.GetDraftTemplate)

	// V2 Organization routes
	v2.Get("/organizations/:organization_id", h.Organization.GetDetail)
	v2.Post("/organizations/:organization_id/members/invite", h.Organization.InviteMember)
	v2.Post("/organizations/:organization_id/members/search", h.Organization.SearchMembers)
	v2.Delete("/organizations/:organization_id/members/remove", h.Organization.RemoveMembers)
	v2.Get("/organizations/:organization_id/outreach-settings", h.Organization.GetOutreachSettings)
	v2.Put("/organizations/:organization_id/outreach-settings", h.Organization.UpdateOutreachSettings)
	v2.Get("/organizations/:organization_id/team-productivity", h.Organization.GetTeamProductivity)

	// Ask COSMO conversation history. The assistant streams from the front end;
	// these endpoints only remember what was said.
	v2.Get("/chat/sessions", h.Chat.ListSessions)
	v2.Post("/chat/sessions", h.Chat.CreateSession)
	v2.Get("/chat/sessions/:session_id/messages", h.Chat.GetMessages)
	v2.Post("/chat/sessions/:session_id/messages", h.Chat.AppendMessages)
	v2.Delete("/chat/sessions/:session_id", h.Chat.DeleteSession)

	// V2 List Contact routes
	v2.Post("/list-contacts/search", h.ListContact.Search)

	// V2 Knowledge routes
	v2.Get("/knowledge", h.Knowledge.List)
	v2.Delete("/knowledge/:knowledge_id", h.Knowledge.Delete)
	v2.Post("/knowledge/upload", h.Knowledge.Upload)

	// V2 Template routes
	v2.Get("/templates/:template_id", h.Template.Get)
	v2.Patch("/templates/:template_id", h.Template.Update)
	v2.Delete("/templates/:template_id", h.Template.Delete)
	v2.Post("/templates/:template_id/knowledges", h.Template.AddKnowledge)

	// V2 HubSpot routes
	v2.Get("/hubspot/callback", h.Hubspot.Callback)
	v2.Get("/hubspot/users/me", h.Hubspot.GetUser)
	v2.Patch("/hubspot/users/me", h.Hubspot.UpdateUser)

	// V2 Gmail routes - Temporarily disabled due to missing implementation
	v2Google := v2.Group("/google")
	v2Google.Get("/gmail", h.Gmail.GetGmailAuthorizationURL)
	v2Google.Get("/gmail/oauth2callback", h.Gmail.HandleGmailOAuth2Callback)
	v2Google.Post("/gmail/watch", h.Gmail.GmailWatch)
	v2Google.Post("/gmail/stop", h.Gmail.GmailStop)
	v2Google.Post("/gmail/notifications", h.Gmail.GmailNotifications)

	// Gmail agent management routes
	v2.Get("/gmail/agents/:agent_id/token-status", h.Gmail.CheckAgentTokenStatus)

	// V2 Email routes
	v2.Post("/emails/search", h.Email.Search)
	v2.Post("/emails/:email_id/reply", h.Email.Reply)

	// V2 Contact routes
	v2.Post("/contacts/search", h.Contact.Search)
	v2.Get("/contacts/values", h.Contact.GetFieldValues)
	v2.Post("/contacts/batch", h.Contact.BatchUpdate)
	v2.Post("/contacts/import", h.Contact.ImportCSV)
	v2.Post("/contacts/import/extract-csv-headers", h.Contact.ExtractCSVHeaders)
	v2.Get("/contacts/:id", h.Contact.Get)
	v2.Post("/contacts/import-hubspot", h.Contact.ImportHubspot)

	// V2 Conversation routes
	v2.Post("/conversations/search", h.Conversation.SearchConversations)

	// V2 Campaign routes
	// Note: AI-powered endpoints will return 503 if OpenAI is not configured
	v2.Post("/campaigns/search", h.Campaign.SearchCampaigns)
	v2.Get("/campaigns/:campaign_id/merge-tags", h.Campaign.GetMergeTags)
	v2.Post("/campaigns/:campaign_id/templates", h.Campaign.GenerateTemplate)
	v2.Post("/campaigns/:campaign_id/templates/:template_id", h.Campaign.RegenerateTemplate)
	v2.Get("/campaigns/:campaign_id/draft-templates", h.Campaign.CreateDraftTemplate)
	v2.Post("/campaigns/:campaign_id/generate-sample-response", h.Campaign.GenerateSampleResponse)
	v2.Post("/campaigns/:campaign_id/classify-sample-response", h.Campaign.ClassifySampleResponse)
	v2.Post("/campaigns/:campaign_id/generate-reply", h.Campaign.GenerateReply)
}
