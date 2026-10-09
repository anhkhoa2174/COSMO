package customfield

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	cfdomain "github.com/rockship/cosmo-agents-go/internal/domain/custom_field"
	cfservice "github.com/rockship/cosmo-agents-go/internal/service/custom_field"
)

// reservedContactKeys are the top-level keys the contact create/update schemas
// (v1 and v2) bind to standard columns. A custom field with one of these keys
// could never hold its own value: the contact handlers route it to the column
// instead of profile.custom_fields.
var reservedContactKeys = map[string]struct{}{
	"id": {}, "name": {}, "first_name": {}, "last_name": {}, "email": {}, "phone": {},
	"company": {}, "job_title": {}, "address": {}, "city": {}, "country": {}, "state": {},
	"zip": {}, "profile": {}, "do_not_contact": {}, "organization_id": {}, "tags": {},
	"confirmed_facts": {}, "ai_insights": {}, "insight_validation": {}, "scores": {},
	"source": {}, "contact_information": {}, "linkedin_url": {}, "industry": {},
	"contact_channel": {}, "context_level": {}, "outreach_decision": {}, "scenario": {},
	"message_draft": {}, "last_outcome": {}, "next_step": {}, "meeting": {},
	"business_stage": {}, "custom_fields": {},
}

var errDuplicateName = errors.New("a custom field with this name already exists")

// validateDefinition checks a field definition before it is stored. Options
// are only meaningful for select fields; for other types they are dropped so
// that stale options left in a form do not get persisted.
func validateDefinition(cf *domain.CustomField) error {
	if !cf.DataType.IsValid() {
		return fmt.Errorf("invalid data_type %q", cf.DataType)
	}
	if !cf.EntityType.IsValid() {
		return fmt.Errorf("invalid entity_type %q", cf.EntityType)
	}

	options := make([]string, 0, len(cf.Options))
	if cf.DataType == domain.CustomFieldDataTypeSelect {
		for _, opt := range cf.Options {
			if opt = strings.TrimSpace(opt); opt != "" {
				options = append(options, opt)
			}
		}
		if len(options) == 0 {
			return errors.New("options is required for select data type")
		}
	}
	cf.Options = options

	svc := cfservice.NewCustomFieldService()
	optionValues := make([]interface{}, len(options))
	for i, opt := range options {
		optionValues[i] = opt
	}
	def := map[string]interface{}{"name": cf.Name, "data_type": string(cf.DataType), "options": optionValues}
	for label, value := range map[string]*string{"sample_data": cf.SampleData, "fallback_value": cf.FallbackValue} {
		if value == nil || strings.TrimSpace(*value) == "" {
			continue
		}
		if _, err := svc.ValidateDataType(def, *value); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}
	return nil
}

// validateNewKey rejects a storage key that collides with a standard contact
// column.
func validateNewKey(entityType domain.CustomFieldEntity, key string) error {
	if key == "" {
		return errors.New("custom field name cannot be empty")
	}
	if entityType == domain.CustomFieldEntityContact {
		if _, reserved := reservedContactKeys[key]; reserved {
			return fmt.Errorf("%q is a standard contact field and cannot be used as a custom field name", key)
		}
	}
	return nil
}

// checkDuplicate returns errDuplicateName when another field of the same
// entity in the same scope (organization, or user when there is none) already
// uses name's key, either as its stored key or as its current display name.
func (h *Handler) checkDuplicate(ctx context.Context, userID uuid.UUID, organizationID *uuid.UUID, entityType domain.CustomFieldEntity, name string, excludeID uuid.UUID) error {
	filter := map[string]interface{}{"entity_type": string(entityType)}
	if organizationID != nil {
		filter["organization_id"] = organizationID.String()
	} else {
		filter["user_id"] = userID.String()
	}

	existing, err := h.customFieldRepo.FindAll(ctx, filter, nil)
	if err != nil {
		return err
	}

	key := cfdomain.NormalizeName(name)
	for _, other := range existing.List {
		if other.ID == excludeID {
			continue
		}
		if other.NormalizedName == key || cfdomain.NormalizeName(other.Name) == key {
			return errDuplicateName
		}
	}
	return nil
}
