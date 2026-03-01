package agent

import (
	"github.com/gofiber/fiber/v3"
)

func (h *AgentHandler) RegisterRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	agentGroup := app.Group("/v1/agents")

	if authMiddleware != nil {
		agentGroup.Use(authMiddleware)
	}

	agentGroup.Post("/", h.Create)
	agentGroup.Get("/", h.List)
	agentGroup.Get("/:id", h.GetByID)
	agentGroup.Put("/:id", h.Update)
	agentGroup.Patch("/:id", h.Update)
	agentGroup.Delete("/:id", h.Delete)

	agentGroup.Post("/search", h.Search)
	agentGroup.Post("/:id/conversations/search", h.GetConversations)
}
