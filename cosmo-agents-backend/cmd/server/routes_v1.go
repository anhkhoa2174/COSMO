package main

import (
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

	// Auth routes (no auth required for login)
	// Auth middleware will skip these paths automatically
	auth := v1.Group("/auth")
	auth.Post("/login", h.Auth.Login)
	auth.Post("/refresh", h.Auth.RefreshToken)
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
	v1.Post("/contacts/search", h.Contact.Search)
	v1.Get("/contact/:id", h.Contact.Get)
	v1.Get("/contact", h.Contact.List)
	v1.Get("/contacts/values", h.Contact.GetFieldValues)
	v1.Patch("/contacts/:id", h.Contact.Update)
	v1.Delete("/contacts", h.Contact.Delete)

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

	// Task enqueue routes
	tasks := v1.Group("/tasks/enqueue")
	tasks.Post("/send-email", h.TaskEnqueue.EnqueueSendEmail)
	tasks.Post("/execute-campaign", h.TaskEnqueue.EnqueueExecuteCampaign)
	tasks.Post("/schedule-tasks", h.TaskEnqueue.EnqueueScheduleTasks)
	tasks.Post("/sync-agent", h.TaskEnqueue.EnqueueSyncAgent)

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

	// Inbound lead form routes
	inboundLeadForms := v1.Group("/inbound-lead-forms")
	inboundLeadForms.Post("", h.InboundLeadForm.Create)
	inboundLeadForms.Get("", h.InboundLeadForm.List)
	inboundLeadForms.Get("/:identifier", h.InboundLeadForm.Get)
	inboundLeadForms.Put("/:identifier", h.InboundLeadForm.Update)
	inboundLeadForms.Delete("/:identifier", h.InboundLeadForm.Delete)
	inboundLeadForms.Post("/:identifier/submit", h.InboundLeadForm.Submit)
}
