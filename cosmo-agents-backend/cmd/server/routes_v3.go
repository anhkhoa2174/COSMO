package main

import (
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// RegisterV3Routes registers all V3 API routes
func RegisterV3Routes(app *App, deps *Dependencies) {
	v3 := app.Fiber.Group("/v3")
	h := deps.V3Handlers
	authMiddleware := middleware.AuthMiddleware(app.JWTManager, deps.Repos.PersonalApiKey, deps.Repos.User)

	v3.Use(middleware.AuthProviderMiddleware())

	// Apply auth middleware to V3 routes
	if app.Config.Auth.EnableAuth {
		v3.Use(authMiddleware)
		logger.Logger.Info().Msg("Authentication enabled for /v3 routes")
	}

	// V3 Operation routes
	v3.Get("/operations/:operation_id", h.Operation.GetOperation)

	// V3 Contact routes
	v3.Post("/contacts/import", h.Contact.ImportCSV)

	// V3 Campaign routes (only if OpenAI is configured)
	if h.Campaign != nil {
		v3.Post("/campaigns/:campaign_id/templates", h.Campaign.GenerateTemplate)
		v3.Post("/campaigns/:campaign_id/templates/external", h.Campaign.SaveExternalTemplate)
		v3.Post("/campaigns/:campaign_id/templates/:template_id", h.Campaign.RegenerateTemplate)
		v3.Post("/campaigns/:campaign_id/generate-sample-response", h.Campaign.GenerateSampleResponse)
		v3.Post("/campaigns/:campaign_id/generate-reply", h.Campaign.GenerateReply)
	}
}
