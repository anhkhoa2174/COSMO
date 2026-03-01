package customfield

// Handler handles custom field-related HTTP requests
type Handler struct {
	customFieldRepo CustomFieldRepository
	userRepo        UserRepository
	roleRepo        RoleRepository
}

// NewHandler creates a new custom field handler
func NewHandler(customFieldRepo CustomFieldRepository, userRepo UserRepository, roleRepo RoleRepository) *Handler {
	return &Handler{
		customFieldRepo: customFieldRepo,
		userRepo:        userRepo,
		roleRepo:        roleRepo,
	}
}
