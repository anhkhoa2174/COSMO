package contact

import (
	"github.com/rockship/cosmo-agents-go/internal/handler/common"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
	operationRepo "github.com/rockship/cosmo-agents-go/internal/repository/operation"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	contactService "github.com/rockship/cosmo-agents-go/internal/service/contact"
	fieldValidationService "github.com/rockship/cosmo-agents-go/internal/service/field_validation"
)

// Handler handles V3 contact API requests (enhanced CSV import with field mapping)
type Handler struct {
	contactRepo     *contactRepo.ContactRepository
	userRepo        *userRepo.UserRepository
	roleRepo        *roleRepo.RoleRepository
	listContactRepo *contactRepo.ListContactRepository
	customFieldRepo *customFieldRepo.CustomFieldRepository
	operationRepo   *operationRepo.OperationRepository

	// Common utilities for code reuse across API versions
	commonAuthHelper *common.AuthHelper
	fileHelper       *common.FileUploadHelper
	responseHelper   *common.ResponseHelper

	// V3 specific services
	fieldValidator *contactService.ContactFieldValidator
	fieldMapper    *fieldValidationService.FieldMapper
	csvImporter    *contactService.CSVImporter

	// runAsync starts a background import; tests swap it to run inline.
	runAsync func(func())
}

// New creates a new refactored V3 ContactHandler
func New(
	contactRepo *contactRepo.ContactRepository,
	userRepo *userRepo.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	listContactRepo *contactRepo.ListContactRepository,
	customFieldRepo *customFieldRepo.CustomFieldRepository,
	operationRepo *operationRepo.OperationRepository,
) *Handler {
	return &Handler{
		contactRepo:     contactRepo,
		userRepo:        userRepo,
		roleRepo:        roleRepo,
		listContactRepo: listContactRepo,
		customFieldRepo: customFieldRepo,
		operationRepo:   operationRepo,

		// Initialize common utilities
		commonAuthHelper: common.NewAuthHelper(userRepo, roleRepo),
		fileHelper:       common.NewFileUploadHelper(),
		responseHelper:   common.NewResponseHelper(),

		// V3 specific services
		fieldValidator: contactService.NewContactFieldValidator(customFieldRepo),
		fieldMapper:    fieldValidationService.NewFieldMapper(),
		csvImporter:    contactService.NewCSVImporter(contactRepo),
		runAsync:       func(f func()) { go f() },
	}
}
