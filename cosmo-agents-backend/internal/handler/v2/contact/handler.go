package contact

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
	inboundLeadFormRepo "github.com/rockship/cosmo-agents-go/internal/repository/inbound_lead_form"
	integrationRepo "github.com/rockship/cosmo-agents-go/internal/repository/integration"
	operationRepo "github.com/rockship/cosmo-agents-go/internal/repository/operation"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	contactService "github.com/rockship/cosmo-agents-go/internal/service/contact"
	fieldValidationService "github.com/rockship/cosmo-agents-go/internal/service/field_validation"
)

// Handler handles V2 contact API requests with cleaner architecture
type Handler struct {
	contactRepo         *contactRepo.ContactRepository
	userRepo            *userRepo.UserRepository
	roleRepo            *roleRepo.RoleRepository
	listContactRepo     *contactRepo.ListContactRepository
	customFieldRepo     *customFieldRepo.CustomFieldRepository
	inboundLeadFormRepo *inboundLeadFormRepo.InboundLeadFormRepository
	operationRepo       *operationRepo.OperationRepository
	integrationRepo     *integrationRepo.IntegrationRepository

	// Reusable services
	authHelper     *middleware.AuthHelper
	fieldValidator *contactService.ContactFieldValidator
	fieldMapper    *fieldValidationService.FieldMapper
	csvImporter    *contactService.CSVImporter
	responseHelper *handler.ResponseHelper
	workerClient   workerClient
}

type workerClient interface {
	EnqueueLowPriorityTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// New creates a new refactored V2 ContactHandler
func New(
	contactRepo *contactRepo.ContactRepository,
	userRepo *userRepo.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	listContactRepo *contactRepo.ListContactRepository,
	customFieldRepo *customFieldRepo.CustomFieldRepository,
	inboundLeadFormRepo *inboundLeadFormRepo.InboundLeadFormRepository,
	operationRepo *operationRepo.OperationRepository,
	integrationRepo *integrationRepo.IntegrationRepository,
	workerClient workerClient,
) *Handler {
	return &Handler{
		contactRepo:         contactRepo,
		userRepo:            userRepo,
		roleRepo:            roleRepo,
		listContactRepo:     listContactRepo,
		customFieldRepo:     customFieldRepo,
		inboundLeadFormRepo: inboundLeadFormRepo,
		operationRepo:       operationRepo,
		integrationRepo:     integrationRepo,

		// Initialize services
		authHelper:     middleware.NewAuthHelper(userRepo, roleRepo),
		fieldValidator: contactService.NewContactFieldValidator(customFieldRepo),
		fieldMapper:    fieldValidationService.NewFieldMapper(),
		csvImporter:    contactService.NewCSVImporter(contactRepo),
		responseHelper: handler.NewResponseHelper(),
		workerClient:   workerClient,
	}
}
