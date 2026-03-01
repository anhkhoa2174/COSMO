package knowledge

// Handler handles V2 knowledge requests following clean architecture
type Handler struct {
	knowledgeRepo KnowledgeRepository
	userRepo      UserRepository
	knowledgeSvc  KnowledgeService
}

// NewHandler creates a new V2 knowledge handler
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
