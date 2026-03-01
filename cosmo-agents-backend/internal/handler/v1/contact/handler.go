package contact

import (
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
	operationRepo "github.com/rockship/cosmo-agents-go/internal/repository/operation"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	contactService "github.com/rockship/cosmo-agents-go/internal/service/contact"
	fieldValidationService "github.com/rockship/cosmo-agents-go/internal/service/field_validation"
	"github.com/rockship/cosmo-agents-go/pkg/worker"

	feedbackRepo "github.com/rockship/cosmo-agents-go/internal/repository/feedback"
	intelService "github.com/rockship/cosmo-agents-go/internal/service/intelligence"
	scraperService "github.com/rockship/cosmo-agents-go/internal/service/scraper"
)

// Handler handles contact-related HTTP requests
type Handler struct {
	repo             *contactRepo.ContactRepository
	userRepo         *userRepo.UserRepository
	roleRepo         *roleRepo.RoleRepository
	organizationRepo *organization.OrganizationRepository
	customFieldRepo  *customFieldRepo.CustomFieldRepository
	listContactRepo  *contactRepo.ListContactRepository
	operationRepo    *operationRepo.OperationRepository
	feedbackRepo     *feedbackRepo.Repository
	intelSvc         *intelService.Service
	scraperSvc       *scraperService.Service
	workerClient     *worker.Client

	// Services
	authHelper     *middleware.AuthHelper
	fieldValidator *contactService.ContactFieldValidator
	fieldMapper    *fieldValidationService.FieldMapper
	csvImporter    *contactService.CSVImporter
	responseHelper *handler.ResponseHelper
}

// New creates a new contact handler with injected services
func New(
	repo *contactRepo.ContactRepository,
	userRepo *userRepo.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	organizationRepo *organization.OrganizationRepository,
	customFieldRepo *customFieldRepo.CustomFieldRepository,
	listContactRepo *contactRepo.ListContactRepository,
	operationRepo *operationRepo.OperationRepository,
	feedbackRepo *feedbackRepo.Repository,
	intelSvc *intelService.Service,
	scraperSvc *scraperService.Service,
	workerClient *worker.Client,
) *Handler {
	return &Handler{
		repo:             repo,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		organizationRepo: organizationRepo,
		customFieldRepo:  customFieldRepo,
		listContactRepo:  listContactRepo,
		operationRepo:    operationRepo,
		feedbackRepo:     feedbackRepo,
		intelSvc:         intelSvc,
		scraperSvc:       scraperSvc,
		workerClient:     workerClient,

		// Initialize services
		authHelper:     middleware.NewAuthHelper(userRepo, roleRepo),
		fieldValidator: contactService.NewContactFieldValidator(customFieldRepo),
		fieldMapper:    fieldValidationService.NewFieldMapper(),
		csvImporter:    contactService.NewCSVImporter(repo),
		responseHelper: handler.NewResponseHelper(),
	}
}
