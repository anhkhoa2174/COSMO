package intelligence

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/agents"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	interactionRepo "github.com/rockship/cosmo-agents-go/internal/repository/interaction"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	_ "github.com/rockship/cosmo-agents-go/internal/schema" // imported for swagger
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/internal/service/intelligence"
)

// Handler exposes AI intelligence endpoints (enrichment, scoring) backed by shared skills.
type Handler struct {
	svc               *intelligence.Service
	enrichmentAgent   *agents.ContactEnrichmentAgent
	segmentAgent      *agents.SegmentCalculatorAgent
	relationshipAgent *agents.RelationshipScorerAgent
	networkAgent      *agents.NetworkAnalyzerAgent
	campaignAgent     *agents.CampaignIntelligenceAgent
	authHelper        *middleware.AuthHelper
	responseHelper    *handler.ResponseHelper
}

// New constructs a new intelligence handler.
func New(
	svc *intelligence.Service,
	userRepo *userRepo.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	contactRepo *contactRepo.ContactRepository,
	interactionRepo *interactionRepo.Repository,
	campaignRepo *campaignRepo.CampaignRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	emailRepo *emailRepo.Repository,
) *Handler {
	return &Handler{
		svc:               svc,
		enrichmentAgent:   agents.NewContactEnrichmentAgent(svc),
		segmentAgent:      agents.NewSegmentCalculatorAgent(svc),
		relationshipAgent: agents.NewRelationshipScorerAgent(contactRepo, interactionRepo),
		networkAgent:      agents.NewNetworkAnalyzerAgent(contactRepo),
		campaignAgent:     agents.NewCampaignIntelligenceAgent(campaignRepo, conversationRepo, emailRepo),
		authHelper:        middleware.NewAuthHelper(userRepo, roleRepo),
		responseHelper:    handler.NewResponseHelper(),
	}
}

// EnrichContact triggers enrichment for a contact.
// @Summary Enrich contact with AI insights
// @Description Generates AI insights and embeddings for a contact
// @Tags Contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact ID"
// @Param body body v1schema.ContactEnrichmentRequest false "Enrichment options"
// @Success 200 {object} schema.APIResponse[v1schema.ContactEnrichmentResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/contacts/{id}/enrich [post]
func (h *Handler) EnrichContact(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "invalid contact id", err)
	}

	var req v1schema.ContactEnrichmentRequest
	_ = c.Bind().JSON(&req) // optional body

	resp, err := h.enrichmentAgent.Run(c.Context(), user.ID, orgID, contactID, req.ForceRefresh)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to enrich contact", err)
	}

	return h.responseHelper.Success(c, resp)
}

// CalculateScores recomputes segmentation fit scores for a contact.
// @Summary Calculate segmentation scores for a contact
// @Tags Contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact ID"
// @Param body body v1schema.CalculateScoresRequest false "Optional segmentation IDs"
// @Success 200 {object} schema.APIResponse[v1schema.CalculateScoresResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/contacts/{id}/calculate-scores [post]
func (h *Handler) CalculateScores(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "invalid contact id", err)
	}

	var req v1schema.CalculateScoresRequest
	_ = c.Bind().JSON(&req) // optional body

	resp, err := h.segmentAgent.Run(c.Context(), user.ID, orgID, contactID, req.SegmentationIDs)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to calculate scores", err)
	}

	return h.responseHelper.Success(c, resp)
}

// VectorSearchContacts performs semantic search on contacts
// @Summary Search contacts using vector similarity
// @Description Find similar contacts using AI embeddings and natural language
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param body body v1schema.VectorSearchContactsRequest true "Search query and parameters"
// @Success 200 {object} schema.APIResponse[v1schema.VectorSearchContactsResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/intelligence/vector-search/contacts [post]
func (h *Handler) VectorSearchContacts(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1schema.VectorSearchContactsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Threshold <= 0 {
		req.Threshold = 1.0 // Accept all results (distance can be 0-2)
	}

	fmt.Printf("[VectorSearch] Query: %s, Limit: %d, Threshold: %.2f, UserID: %s\n", req.Query, req.Limit, req.Threshold, user.ID)

	resp, err := h.svc.VectorSearchContacts(c.Context(), user.ID, req.Query, req.Limit, req.Threshold)
	if err != nil {
		fmt.Printf("[VectorSearch] ERROR: %v\n", err)
		return h.responseHelper.InternalServerError(c, "failed to perform vector search", err)
	}

	fmt.Printf("[VectorSearch] Found %d results\n", len(resp.Results))
	return h.responseHelper.Success(c, resp)
}

// FindSimilarContacts finds contacts similar to a given contact
// @Summary Find similar contacts
// @Description Find contacts similar to a given contact using vector embeddings
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param body body v1schema.FindSimilarContactRequest true "Contact ID and parameters"
// @Success 200 {object} schema.APIResponse[v1schema.VectorSearchContactsResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/intelligence/vector-search/similar [post]
func (h *Handler) FindSimilarContacts(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1schema.FindSimilarContactRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	contactID, err := uuid.Parse(req.ContactID)
	if err != nil {
		return h.responseHelper.BadRequest(c, "invalid contact_id", err)
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Threshold <= 0 {
		req.Threshold = 1.0
	}

	resp, err := h.svc.FindSimilarContacts(c.Context(), user.ID, contactID, req.Limit, req.Threshold)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to find similar contacts", err)
	}

	return h.responseHelper.Success(c, resp)
}

// SearchKnowledge performs semantic search on knowledge base
// @Summary Search knowledge base
// @Description Search knowledge documents using semantic similarity for RAG
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param body body v1schema.SearchKnowledgeRequest true "Search query and parameters"
// @Success 200 {object} schema.APIResponse[v1schema.SearchKnowledgeResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/intelligence/vector-search/knowledge [post]
func (h *Handler) SearchKnowledge(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1schema.SearchKnowledgeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Threshold <= 0 {
		req.Threshold = 1.0
	}

	resp, err := h.svc.SearchKnowledge(c.Context(), user.ID, req.Query, req.Limit, req.Threshold)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to search knowledge", err)
	}

	return h.responseHelper.Success(c, resp)
}

// SearchInteractions performs semantic search on interaction history
// @Summary Search interactions
// @Description Search interaction history using semantic similarity
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param body body v1schema.SearchInteractionsRequest true "Search query and parameters"
// @Success 200 {object} schema.APIResponse[v1schema.SearchInteractionsResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/intelligence/vector-search/interactions [post]
func (h *Handler) SearchInteractions(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1schema.SearchInteractionsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.7
	}

	resp, err := h.svc.SearchInteractions(c.Context(), user.ID, req.Query, req.Limit, req.Threshold)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to search interactions", err)
	}

	return h.responseHelper.Success(c, resp)
}

// FindSimilarSegments finds segments matching a query
// @Summary Find similar segments
// @Description Find segments that match a profile or query using semantic similarity
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param body body v1schema.FindSimilarSegmentsRequest true "Search query and parameters"
// @Success 200 {object} schema.APIResponse[v1schema.FindSimilarSegmentsResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/intelligence/vector-search/segments [post]
func (h *Handler) FindSimilarSegments(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1schema.FindSimilarSegmentsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.6
	}

	resp, err := h.svc.FindSimilarSegments(c.Context(), user.ID, req.Query, req.Limit, req.Threshold)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to find similar segments", err)
	}

	return h.responseHelper.Success(c, resp)
}

// HybridSearchContacts combines keyword and semantic search
// @Summary Hybrid search contacts
// @Description Combine keyword and semantic search for best results
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param body body v1schema.HybridSearchRequest true "Search query and parameters"
// @Success 200 {object} schema.APIResponse[v1schema.HybridSearchResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/intelligence/vector-search/hybrid [post]
func (h *Handler) HybridSearchContacts(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1schema.HybridSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}

	resp, err := h.svc.HybridSearchContacts(c.Context(), user.ID, req.Query, req.Keywords, req.Limit)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to perform hybrid search", err)
	}

	return h.responseHelper.Success(c, resp)
}

// ScoreRelationship computes and stores relationship strength for a contact.
// @Summary Score relationship strength
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param id path string true "Contact ID"
// @Success 200 {object} schema.APIResponse[agents.RelationshipScore]
// @Security BearerAuth
// @Router /v1/contacts/{id}/relationship-score [post]
func (h *Handler) ScoreRelationship(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "invalid contact id", err)
	}

	result, err := h.relationshipAgent.Run(c.Context(), user.ID, orgID, contactID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to score relationship", err)
	}
	return h.responseHelper.Success(c, result)
}

// AnalyzeNetwork updates contact network analysis placeholders.
// @Summary Analyze contact network
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param id path string true "Contact ID"
// @Success 200 {object} schema.APIResponse[agents.NetworkAnalysisResult]
// @Security BearerAuth
// @Router /v1/contacts/{id}/network-analysis [post]
func (h *Handler) AnalyzeNetwork(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "invalid contact id", err)
	}

	result, err := h.networkAgent.Run(c.Context(), user.ID, orgID, contactID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to analyze network", err)
	}
	return h.responseHelper.Success(c, result)
}

// CampaignIntelligence aggregates campaign metrics.
// @Summary Get campaign intelligence
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Success 200 {object} schema.APIResponse[agents.CampaignIntelligenceResult]
// @Security BearerAuth
// @Router /v1/campaigns/{id}/intelligence [get]
func (h *Handler) CampaignIntelligence(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "invalid campaign id", err)
	}

	result, err := h.campaignAgent.Run(c.Context(), user.ID, orgID, campaignID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to fetch campaign intelligence", err)
	}
	return h.responseHelper.Success(c, result)
}
