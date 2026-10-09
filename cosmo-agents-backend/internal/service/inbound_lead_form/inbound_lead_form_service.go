package inbound_lead_form

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
	inboundLeadFormRepo "github.com/rockship/cosmo-agents-go/internal/repository/inbound_lead_form"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

type InboundLeadFormService struct {
	formsRepo        *inboundLeadFormRepo.InboundLeadFormRepository
	customFieldRepo  *customFieldRepo.CustomFieldRepository
	contactRepo      *contactRepo.ContactRepository
	listContactRepo  *contactRepo.ListContactRepository
	organizationRepo *organization.OrganizationRepository

	// orgResolver finds the organisation a form owner works in. Optional:
	// without it only an organisation the owner created is found.
	orgResolver primaryOrgFinder
}

// primaryOrgFinder is the slice of the role repository Submit needs.
type primaryOrgFinder interface {
	FindPrimaryOrganization(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
}

// WithOrgResolver lets Submit file leads under the organisation the owner is
// a member of, not only one they created.
func (s *InboundLeadFormService) WithOrgResolver(r primaryOrgFinder) *InboundLeadFormService {
	s.orgResolver = r
	return s
}

// Limits on what an anonymous visitor can store through a public form.
const (
	maxSubmitValueLen = 2000
	maxSubmitListLen  = 50
)

// ErrInvalidSubmission marks a submission rejected for its content, as
// opposed to a server failure.
var ErrInvalidSubmission = errors.New("invalid submission")

func invalidSubmission(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidSubmission, fmt.Sprintf(format, args...))
}

func NewInboundLeadFormService(
	formsRepo *inboundLeadFormRepo.InboundLeadFormRepository,
	customFieldRepo *customFieldRepo.CustomFieldRepository,
	contactRepo *contactRepo.ContactRepository,
	listContactRepo *contactRepo.ListContactRepository,
	organizationRepo *organization.OrganizationRepository,
) *InboundLeadFormService {
	return &InboundLeadFormService{
		formsRepo:        formsRepo,
		customFieldRepo:  customFieldRepo,
		contactRepo:      contactRepo,
		listContactRepo:  listContactRepo,
		organizationRepo: organizationRepo,
	}
}

var systemFieldDefinitions = map[string]struct {
	FieldType   string
	DisplayName string
	Required    bool
}{
	"email":       {FieldType: string(domain.CustomFieldDataTypeEmail), DisplayName: "Email Address", Required: true},
	"first_name":  {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "First Name", Required: true},
	"last_name":   {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "Last Name", Required: true},
	"phone":       {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "Phone Number", Required: false},
	"company":     {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "Company", Required: false},
	"job_title":   {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "Job Title", Required: false},
	"address":     {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "Address", Required: false},
	"city":        {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "City", Required: false},
	"state":       {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "State/Province", Required: false},
	"postal_code": {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "Postal Code", Required: false},
	"country":     {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "Country", Required: false},
	"website":     {FieldType: string(domain.CustomFieldDataTypeURL), DisplayName: "Website", Required: false},
	"notes":       {FieldType: string(domain.CustomFieldDataTypeText), DisplayName: "Notes", Required: false},
}

func (s *InboundLeadFormService) Create(ctx context.Context, userID uuid.UUID, req *v1schema.InboundLeadFormCreateRequest) (*v1schema.InboundLeadFormResponse, error) {
	if len(req.Fields) == 0 {
		return nil, fmt.Errorf("form must include at least one field")
	}

	// Build form entity
	form := &domain.InboundLeadForm{
		Name: req.Name,
		Slug: req.Slug,
	}
	form.UserID = &userID

	if req.UIMetadata != nil {
		var meta domain.JSONB
		if err := meta.Marshal(req.UIMetadata); err != nil {
			return nil, err
		}
		form.UIMetadata = meta
	}

	fields, err := s.prepareFormFields(ctx, userID, req.Fields)
	if err != nil {
		return nil, err
	}

	if err := s.formsRepo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.formsRepo.CreateWithFieldsTx(ctx, tx, form, fields); err != nil {
			return err
		}

		// If caller requested to attach this form to existing list contacts, create association rows
		// Note: AddInboundForms expects (listContactID, []formIDs). We must call it per list with the
		// newly created form ID to avoid inverting parameters.
		if len(req.ListContactIDs) > 0 && s.listContactRepo != nil {
			for _, listID := range req.ListContactIDs {
				if err := s.listContactRepo.AddInboundFormsWithOwnershipTx(ctx, tx, userID, listID, []uuid.UUID{form.ID}); err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return s.buildResponse(form, fields), nil
}

func (s *InboundLeadFormService) Get(ctx context.Context, userID uuid.UUID, identifier string) (*v1schema.InboundLeadFormResponse, error) {
	form, err := s.formsRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		return nil, err
	}

	if form == nil {
		return nil, nil
	}

	if form.UserID == nil || *form.UserID != userID {
		return nil, schema.ErrForbidden
	}

	return s.buildResponse(form, form.Fields), nil
}

// GetPublic returns what a visitor needs to render a form: its name, slug,
// fields and layout. It carries nothing about the owner, so it is safe to
// serve without authentication.
func (s *InboundLeadFormService) GetPublic(ctx context.Context, identifier string) (*v1schema.InboundLeadFormResponse, error) {
	form, err := s.formsRepo.GetByIdentifier(ctx, identifier)
	if err != nil || form == nil {
		return nil, err
	}
	return s.buildResponse(form, form.Fields), nil
}

func (s *InboundLeadFormService) List(ctx context.Context, userID uuid.UUID) ([]v1schema.InboundLeadFormListItem, error) {
	forms, err := s.formsRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]v1schema.InboundLeadFormListItem, len(forms))
	for i, form := range forms {
		items[i] = v1schema.InboundLeadFormListItem{
			ID:   form.ID,
			Name: form.Name,
			Slug: form.Slug,
		}
	}
	return items, nil
}

func (s *InboundLeadFormService) Update(ctx context.Context, userID uuid.UUID, identifier string, req *v1schema.InboundLeadFormUpdateRequest) (*v1schema.InboundLeadFormResponse, error) {
	form, err := s.formsRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if form == nil {
		return nil, nil
	}

	if form.UserID != nil && *form.UserID != userID {
		return nil, fmt.Errorf("unauthorized: cannot update form owned by another user")
	}

	// Validate the new fields before touching the stored ones: they used to be
	// deleted first, so an invalid edit left the form with no fields at all.
	fields, err := s.prepareFormFields(ctx, userID, req.Fields)
	if err != nil {
		return nil, err
	}

	attrs := map[string]interface{}{
		"name": req.Name,
	}
	if req.UIMetadata != nil {
		var meta domain.JSONB
		if err := meta.Marshal(req.UIMetadata); err != nil {
			return nil, err
		}
		attrs["ui_metadata"] = meta
	}

	if err := s.formsRepo.UpdateForm(ctx, form.ID, attrs); err != nil {
		return nil, err
	}

	if err := s.formsRepo.DeleteFormFields(ctx, form.ID); err != nil {
		return nil, err
	}

	for i := range fields {
		fields[i].FormID = form.ID
	}

	if err := s.formsRepo.CreateFormFields(ctx, fields); err != nil {
		return nil, err
	}

	form.Name = req.Name
	form.Fields = fields
	if req.UIMetadata != nil {
		var meta domain.JSONB
		meta.Marshal(req.UIMetadata)
		form.UIMetadata = meta
	}

	return s.buildResponse(form, fields), nil
}

func (s *InboundLeadFormService) Delete(ctx context.Context, userID uuid.UUID, identifier string) (bool, error) {
	form, err := s.formsRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		return false, err
	}
	if form == nil {
		return false, nil
	}

	if form.UserID != nil && *form.UserID != userID {
		return false, fmt.Errorf("unauthorized: cannot delete form owned by another user")
	}

	if err := s.formsRepo.Delete(ctx, form.ID); err != nil {
		return false, err
	}

	return true, nil
}

func (s *InboundLeadFormService) prepareFormFields(ctx context.Context, userID uuid.UUID, requests []v1schema.InboundLeadFormFieldRequest) ([]domain.FormField, error) {
	// Submit keeps only the form's own fields and requires an email (it is
	// the lead's identity and outreach address), so a form saved without an
	// email field could never be submitted.
	hasEmail := false
	for _, field := range requests {
		if strings.EqualFold(strings.TrimSpace(field.Name), "email") {
			hasEmail = true
			break
		}
	}
	if !hasEmail {
		return nil, fmt.Errorf("form must include an email field")
	}

	fields := make([]domain.FormField, 0, len(requests))

	// Collect custom field names
	lookup := make([]string, 0)
	for _, field := range requests {
		name := strings.ToLower(strings.TrimSpace(field.Name))
		if _, ok := systemFieldDefinitions[name]; !ok {
			lookup = append(lookup, name)
		}
	}

	customFields, err := s.customFieldRepo.FindByNormalizedNames(ctx, userID, lookup)
	if err != nil {
		return nil, err
	}

	for _, field := range requests {
		normalized := strings.ToLower(strings.TrimSpace(field.Name))
		def, isSystem := systemFieldDefinitions[normalized]

		formField := domain.FormField{}

		if isSystem {
			formField.IsSystemField = true
			formField.SystemFieldName = &normalized
			formField.DisplayName = firstNonEmpty(field.DisplayName, def.DisplayName)
			required := def.Required
			if field.IsRequired != nil {
				required = *field.IsRequired
			}
			formField.IsRequired = required
		} else {
			cf, exists := customFields[normalized]
			if !exists {
				return nil, fmt.Errorf("custom field '%s' not found", normalized)
			}
			formField.CustomFieldID = &cf.ID
			formField.DisplayName = firstNonEmpty(field.DisplayName, cf.Name)
			if field.IsRequired != nil {
				formField.IsRequired = *field.IsRequired
			}
		}

		if field.UIMetadata != nil {
			var meta domain.JSONB
			if err := meta.Marshal(field.UIMetadata); err != nil {
				return nil, err
			}
			formField.UIMetadata = meta
		}

		fields = append(fields, formField)
	}

	return fields, nil
}

func (s *InboundLeadFormService) buildResponse(form *domain.InboundLeadForm, fields []domain.FormField) *v1schema.InboundLeadFormResponse {
	resp := &v1schema.InboundLeadFormResponse{
		ID:   form.ID,
		Name: form.Name,
		Slug: form.Slug,
	}

	if len(form.UIMetadata) > 0 {
		var meta map[string]any
		if err := form.UIMetadata.Unmarshal(&meta); err == nil {
			resp.UIMetadata = meta
		}
	}

	resp.Fields = make([]v1schema.InboundLeadFormFieldResponse, len(fields))
	for i, field := range fields {
		item := v1schema.InboundLeadFormFieldResponse{
			DisplayName: field.DisplayName,
			IsRequired:  field.IsRequired,
		}

		if field.IsSystemField && field.SystemFieldName != nil {
			name := *field.SystemFieldName
			item.Name = name
			if def, ok := systemFieldDefinitions[name]; ok {
				item.FieldType = def.FieldType
			}
		} else if field.CustomFieldID != nil && field.CustomField != nil {
			item.Name = field.CustomField.NormalizedName
			item.FieldType = string(field.CustomField.DataType)
			if len(field.CustomField.Options) > 0 {
				item.SelectOptions = field.CustomField.Options
			}
			if field.CustomField.FallbackValue != nil {
				item.FallbackValue = *field.CustomField.FallbackValue
			}
		}

		if len(field.UIMetadata) > 0 {
			var meta map[string]any
			if err := field.UIMetadata.Unmarshal(&meta); err == nil {
				item.UIMetadata = meta
			}
		}

		resp.Fields[i] = item
	}

	return resp
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s *InboundLeadFormService) Submit(ctx context.Context, identifier string, payload map[string]any) (*v1schema.LeadFormSubmitResponse, error) {
	form, err := s.formsRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if form == nil {
		return nil, nil
	}
	if form.UserID == nil {
		return nil, fmt.Errorf("form owner not defined")
	}

	// Only the form's own fields are kept. The endpoint is public, so any
	// other key a visitor sends would otherwise be written into the owner's
	// contact profile.
	requiredFields := map[string]bool{}
	systemFields := map[string]bool{}
	customFields := map[string]bool{}
	for _, field := range form.Fields {
		var name string
		if field.IsSystemField && field.SystemFieldName != nil {
			name = *field.SystemFieldName
			systemFields[name] = true
		} else if field.CustomFieldID != nil && field.CustomField != nil {
			name = field.CustomField.NormalizedName
			customFields[name] = true
		} else {
			continue
		}
		if field.IsRequired {
			requiredFields[name] = true
		}
	}

	data := make(map[string]any)
	for key, value := range payload {
		key = strings.ToLower(strings.TrimSpace(key))
		if !systemFields[key] && !customFields[key] {
			continue
		}
		clean, err := cleanSubmitValue(key, value)
		if err != nil {
			return nil, err
		}
		if clean != nil {
			data[key] = clean
		}
	}

	for field := range requiredFields {
		if _, ok := data[field]; !ok {
			return nil, invalidSubmission("missing required field '%s'", field)
		}
	}

	emailVal, _ := data["email"].(string)
	if emailVal == "" {
		return nil, invalidSubmission("form submission must include email")
	}
	if addr, err := mail.ParseAddress(emailVal); err != nil || addr.Address != emailVal {
		return nil, invalidSubmission("'%s' is not a valid email address", emailVal)
	}

	orgID, err := s.ownerOrganization(ctx, *form.UserID)
	if err != nil {
		return nil, err
	}

	// Combine first_name and last_name into name
	// Read raw: stringFromMap's "N/A" for a missing part made "Ann N/A".
	firstName, _ := data["first_name"].(string)
	lastName, _ := data["last_name"].(string)
	name := strings.TrimSpace(firstName + " " + lastName)
	if name == "" {
		name = stringFromMap(data, "name")
	}

	contact := &domain.Contact{
		UserID:         *form.UserID,
		OrganizationID: orgID,
		Source:         string(domain.ContactSourceCosmoAgents),
		SourceID:       fmt.Sprintf("inbound:%s:%s", form.Slug, strings.ToLower(emailVal)),
		Name:           name,
		Company:        stringFromMap(data, "company"),
		JobTitle:       stringFromMap(data, "job_title"),
		Address:        stringFromMap(data, "address"),
		City:           stringFromMap(data, "city"),
		Country:        stringFromMap(data, "country"),
		State:          stringFromMap(data, "state"),
		Zip:            stringFromMap(data, "postal_code"),
	}

	// Build profile with email, phone, and custom fields
	profile := make(map[string]any)
	// Add email and phone to profile
	email := strings.TrimSpace(emailVal)
	if email != "" {
		profile["email"] = email
	}
	if phone, _ := data["phone"].(string); phone != "" {
		profile["phone"] = phone
	}
	// System fields without a contact column of their own live in profile.
	for _, key := range []string{"website", "notes"} {
		if v, ok := data[key]; ok {
			profile[key] = v
		}
	}
	for key, value := range data {
		if customFields[key] {
			profile[key] = value
		}
	}
	if len(profile) > 0 {
		var profileJSON domain.JSONB
		if err := profileJSON.Marshal(profile); err == nil {
			contact.Profile = profileJSON
		}
	}

	// As Create does: contact_information is the duplicate key and outreach
	// recipient; without it every form lead sat "pending" with nowhere to send.
	contact.ContactInformation = email
	contact.CalculateStatus()

	stored, err := s.contactRepo.UpsertBySource(ctx, contact)
	if err != nil {
		return nil, err
	}

	resp := &v1schema.LeadFormSubmitResponse{
		ContactID: stored.ID,
		Data:      data,
	}

	// If form is linked to list contacts, attach the newly created contact to those lists
	if s.listContactRepo != nil {
		listIDs, err := s.formsRepo.ListContactIDs(ctx, form.ID)
		if err != nil {
			return nil, err
		}
		for _, listID := range listIDs {
			if err := s.listContactRepo.AddContactsWithOwnership(ctx, *form.UserID, listID, []uuid.UUID{stored.ID}); err != nil {
				return nil, err
			}
		}
	}

	return resp, nil
}

// ownerOrganization is the organisation a form's leads belong to: the one the
// owner works in, else one they created. Picking the first of an unordered
// list filed leads under an arbitrary organisation, and a member who had not
// created one could not receive leads at all.
func (s *InboundLeadFormService) ownerOrganization(ctx context.Context, ownerID uuid.UUID) (*uuid.UUID, error) {
	if s.orgResolver != nil {
		orgID, err := s.orgResolver.FindPrimaryOrganization(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		if orgID != nil {
			return orgID, nil
		}
	}
	orgs, err := s.organizationRepo.FindByUserID(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, fmt.Errorf("owner has no organization configured")
	}
	oldest := orgs[0]
	for _, o := range orgs[1:] {
		if o.CreatedAt.Before(oldest.CreatedAt) {
			oldest = o
		}
	}
	return &oldest.ID, nil
}

// cleanSubmitValue accepts the shapes a form input produces (text, number,
// checkbox, multi-select) and trims and bounds them. It returns nil for an
// empty value, so a blank optional field is simply absent.
func cleanSubmitValue(key string, value any) (any, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil, nil
		}
		if len([]rune(v)) > maxSubmitValueLen {
			return nil, invalidSubmission("'%s' is longer than %d characters", key, maxSubmitValueLen)
		}
		return v, nil
	case float64, bool:
		return v, nil
	case []any:
		if len(v) > maxSubmitListLen {
			return nil, invalidSubmission("'%s' has more than %d values", key, maxSubmitListLen)
		}
		out := make([]string, 0, len(v))
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, invalidSubmission("'%s' must be a list of text values", key)
			}
			if str = strings.TrimSpace(str); str != "" {
				if len([]rune(str)) > maxSubmitValueLen {
					return nil, invalidSubmission("'%s' has a value longer than %d characters", key, maxSubmitValueLen)
				}
				out = append(out, str)
			}
		}
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	default:
		return nil, invalidSubmission("'%s' has an unsupported value", key)
	}
}

func stringFromMap(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			trimmed := strings.TrimSpace(str)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return domain.NOT_AVAILABLE
}
