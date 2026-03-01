package knowledge

// Handler handles knowledge-related HTTP requests following clean architecture
type Handler struct {
	knowledgeRepo KnowledgeRepository
	userRepo      UserRepository
	knowledgeSvc  KnowledgeService
}

// NewHandler creates a new knowledge handler
func NewHandler(
	knowledgeRepo KnowledgeRepository,
	userRepo UserRepository,
	knowledgeSvc KnowledgeService,
) *Handler {
	return &Handler{
		knowledgeRepo: knowledgeRepo,
		userRepo:      userRepo,
		knowledgeSvc:  knowledgeSvc,
	}
}
