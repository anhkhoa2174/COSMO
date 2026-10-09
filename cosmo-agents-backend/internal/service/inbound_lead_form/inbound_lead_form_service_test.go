package inbound_lead_form

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/organization"
	contactRepository "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
	inboundLeadFormRepo "github.com/rockship/cosmo-agents-go/internal/repository/inbound_lead_form"
	orgRepo "github.com/rockship/cosmo-agents-go/internal/repository/organization"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

func newInboundLeadFormService(t *testing.T) (*InboundLeadFormService, uuid.UUID) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.InboundLeadForm{},
		&domain.FormField{},
		&domain.CustomField{},
		&organization.Organization{},
	))
	require.NoError(t, db.Exec(`ALTER TABLE inbound_lead_forms ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN DEFAULT false`).Error)

	require.NoError(t, db.AutoMigrate(&domain.Contact{}))

	userID := uuid.New()
	org := organization.Organization{
		Base: domain.Base{ID: uuid.New()},
		UserID: func() *uuid.UUID {
			v := userID
			return &v
		}(),
		Name: "Org",
	}
	require.NoError(t, db.Create(&org).Error)

	customRepo := customFieldRepo.NewCustomFieldRepository(db)
	formsRepo := inboundLeadFormRepo.NewInboundLeadFormRepository(db)
	contactRepo := contactRepository.NewContactRepository(db)
	listContactRepo := (*contactRepository.ListContactRepository)(nil) // skip list operations
	organizationRepo := orgRepo.NewOrganizationRepository(db)

	// custom field needed for non-system field mapping
	customField := domain.CustomField{
		Base:           domain.Base{ID: uuid.New()},
		OrganizationID: &org.ID,
		UserID:         userID,
		Name:           "Custom Field",
		NormalizedName: "custom_field",
		EntityType:     domain.CustomFieldEntityContact,
		DataType:       domain.CustomFieldDataTypeText,
	}
	require.NoError(t, db.Create(&customField).Error)

	service := NewInboundLeadFormService(formsRepo, customRepo, contactRepo, listContactRepo, organizationRepo)
	return service, userID
}

func TestInboundLeadFormService_CreateGetUpdate(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()

	createReq := &v1schema.InboundLeadFormCreateRequest{
		Name: "Demo Form",
		Slug: "demo-form",
		Fields: []v1schema.InboundLeadFormFieldRequest{
			{Name: "email"},
			{Name: "first_name"},
			{Name: "custom_field", DisplayName: "Custom", IsRequired: boolPtr(true)},
		},
		UIMetadata: map[string]any{"theme": "dark"},
	}

	created, err := service.Create(ctx, userID, createReq)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "Demo Form", created.Name)
	assert.Len(t, created.Fields, 3)

	fetched, err := service.Get(ctx, userID, "demo-form")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, created.ID, fetched.ID)

	updateReq := &v1schema.InboundLeadFormUpdateRequest{
		Name: "Updated Form",
		Fields: []v1schema.InboundLeadFormFieldRequest{
			{Name: "email"},
			{Name: "first_name", IsRequired: boolPtr(false)},
		},
	}
	updated, err := service.Update(ctx, userID, "demo-form", updateReq)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Updated Form", updated.Name)
	assert.Len(t, updated.Fields, 2)

	deleted, err := service.Delete(ctx, userID, "demo-form")
	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestInboundLeadFormService_Submit(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()

	req := &v1schema.InboundLeadFormCreateRequest{
		Name: "Lead Form",
		Slug: "lead-form",
		Fields: []v1schema.InboundLeadFormFieldRequest{
			{Name: "email"},
			{Name: "first_name"},
			{Name: "custom_field", IsRequired: boolPtr(false)},
		},
	}
	_, err := service.Create(ctx, userID, req)
	require.NoError(t, err)

	payload := map[string]any{
		"email":        "lead@example.com",
		"first_name":   "Lead",
		"custom_field": "extra",
	}

	resp, err := service.Submit(ctx, "lead-form", payload)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "lead@example.com", payload["email"])
	assert.NotEqual(t, uuid.Nil, resp.ContactID)

	// Missing required field should error
	_, err = service.Submit(ctx, "lead-form", map[string]any{"email": "lead@example.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field")
}

func TestInboundLeadFormService_ListAndErrors(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()

	req := &v1schema.InboundLeadFormCreateRequest{
		Name: "List Form",
		Slug: "list-form",
		Fields: []v1schema.InboundLeadFormFieldRequest{
			{Name: "email"},
		},
	}
	_, err := service.Create(ctx, userID, req)
	require.NoError(t, err)

	forms, err := service.List(ctx, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, forms)

	// Get missing returns nil
	form, err := service.Get(ctx, userID, "missing-form")
	assert.NoError(t, err)
	assert.Nil(t, form)

	// Delete missing returns false
	deleted, err := service.Delete(ctx, userID, "missing-form")
	assert.NoError(t, err)
	assert.False(t, deleted)

	// Submit owner without organization (remove orgs)
	svc, newUser := newInboundLeadFormService(t)
	ctx = context.Background()
	_, err = svc.Create(ctx, newUser, req)
	require.NoError(t, err)
	// wipe organizations for this user to trigger error
	require.NoError(t, svc.organizationRepo.GetDB().Where("user_id = ?", newUser).Delete(&organization.Organization{}).Error)
	_, err = svc.Submit(ctx, "list-form", map[string]any{"email": "a@example.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner has no organization configured")
}

func TestInboundLeadFormService_EdgeCases(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()

	_, err := service.Create(ctx, userID, &v1schema.InboundLeadFormCreateRequest{
		Name:   "NoFields",
		Slug:   "no-fields",
		Fields: []v1schema.InboundLeadFormFieldRequest{},
	})
	assert.Error(t, err)

	// Prepare form owned by another user to hit forbidden branches
	otherUser := uuid.New()
	createReq := &v1schema.InboundLeadFormCreateRequest{
		Name: "Other",
		Slug: "other-form",
		Fields: []v1schema.InboundLeadFormFieldRequest{
			{Name: "email"},
		},
	}
	_, err = service.Create(ctx, otherUser, createReq)
	require.NoError(t, err)

	form, err := service.Get(ctx, userID, "other-form")
	assert.Error(t, err)
	assert.Nil(t, form)

	updated, err := service.Update(ctx, userID, "other-form", &v1schema.InboundLeadFormUpdateRequest{
		Name:   "Updated",
		Fields: []v1schema.InboundLeadFormFieldRequest{{Name: "email"}},
	})
	assert.Error(t, err)
	assert.Nil(t, updated)

	deleted, err := service.Delete(ctx, userID, "other-form")
	assert.Error(t, err)
	assert.False(t, deleted)

	// Submit when form missing returns nil
	resp, err := service.Submit(ctx, "missing-form", map[string]any{})
	assert.NoError(t, err)
	assert.Nil(t, resp)

	// prepareFormFields should error on missing custom field
	_, err = service.prepareFormFields(ctx, userID, []v1schema.InboundLeadFormFieldRequest{
		{Name: "missing_custom"},
	})
	assert.Error(t, err)
}

func boolPtr(v bool) *bool { return &v }

// A lead who fills the form in again must not lose what was learned about
// them since, and must be reachable: contact_information carries the email.
func TestInboundLeadFormService_Resubmit_KeepsEnrichment(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()

	_, err := service.Create(ctx, userID, &v1schema.InboundLeadFormCreateRequest{
		Name: "Lead Form", Slug: "resubmit-form",
		Fields: []v1schema.InboundLeadFormFieldRequest{{Name: "email"}, {Name: "first_name"}, {Name: "company"}},
	})
	require.NoError(t, err)

	first, err := service.Submit(ctx, "resubmit-form", map[string]any{"email": "lead@example.com", "first_name": "Lead", "company": "Acme"})
	require.NoError(t, err)
	stored, err := service.contactRepo.GetByID(ctx, first.ContactID)
	require.NoError(t, err)
	assert.Equal(t, "lead@example.com", stored.ContactInformation)

	// Enrichment after the first submission.
	_, err = service.contactRepo.UpdateFields(ctx, first.ContactID, map[string]interface{}{
		"profile": domain.JSONB(`{"email":"lead@example.com","research_findings":[{"fact":"raised series A"}]}`),
	}, userID, *stored.OrganizationID)
	require.NoError(t, err)

	second, err := service.Submit(ctx, "resubmit-form", map[string]any{"email": "lead@example.com", "first_name": "Lead"})
	require.NoError(t, err)
	assert.Equal(t, first.ContactID, second.ContactID)

	after, err := service.contactRepo.GetByID(ctx, first.ContactID)
	require.NoError(t, err)
	assert.Equal(t, "Acme", after.Company, "a field left out of the resubmission is kept")
	assert.Contains(t, string(after.Profile), "raised series A", "profile keys added since are kept")
}

// The submit endpoint is public, so a visitor controls the whole payload.
func TestInboundLeadFormService_Submit_OnlyStoresTheFormsFields(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()
	_, err := service.Create(ctx, userID, &v1schema.InboundLeadFormCreateRequest{
		Name: "Lead Form", Slug: "public-form",
		Fields: []v1schema.InboundLeadFormFieldRequest{{Name: "email"}, {Name: "first_name"}, {Name: "notes"}, {Name: "custom_field"}},
	})
	require.NoError(t, err)

	resp, err := service.Submit(ctx, "public-form", map[string]any{
		"email": " lead@example.com ", "first_name": "Lead", "Notes": "call me", "custom_field": "blue",
		// Not on the form: dropped, not written into the owner's contact.
		"do_not_contact": true, "research_findings": []any{"fake"}, "status": "hot",
	})
	require.NoError(t, err)
	assert.NotContains(t, resp.Data, "status")

	stored, err := service.contactRepo.GetByID(ctx, resp.ContactID)
	require.NoError(t, err)
	assert.Equal(t, "lead@example.com", stored.ContactInformation)
	var profile map[string]any
	require.NoError(t, json.Unmarshal(stored.Profile, &profile))
	assert.Equal(t, "call me", profile["notes"])
	assert.Equal(t, "blue", profile["custom_field"])
	for _, key := range []string{"do_not_contact", "research_findings", "status"} {
		assert.NotContains(t, profile, key)
	}
	assert.False(t, stored.DoNotContact)
}

func TestInboundLeadFormService_Submit_RejectsBadValues(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()
	_, err := service.Create(ctx, userID, &v1schema.InboundLeadFormCreateRequest{
		Name: "Lead Form", Slug: "strict-form",
		Fields: []v1schema.InboundLeadFormFieldRequest{{Name: "email"}, {Name: "first_name"}, {Name: "notes"}},
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		payload map[string]any
	}{
		{"not an email", map[string]any{"email": "nope", "first_name": "A"}},
		{"email with a display name", map[string]any{"email": "Eve <eve@x.test>", "first_name": "A"}},
		{"required field left blank", map[string]any{"email": "a@x.test", "first_name": "   "}},
		{"value too long", map[string]any{"email": "a@x.test", "first_name": "A", "notes": strings.Repeat("x", maxSubmitValueLen+1)}},
		{"nested object", map[string]any{"email": "a@x.test", "first_name": "A", "notes": map[string]any{"a": 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Submit(ctx, "strict-form", tt.payload)
			assert.ErrorIs(t, err, ErrInvalidSubmission)
		})
	}
}

type fixedOrg struct{ id uuid.UUID }

func (f fixedOrg) FindPrimaryOrganization(context.Context, uuid.UUID) (*uuid.UUID, error) {
	return &f.id, nil
}

// A member who did not create their organisation still owns forms; their
// leads go to the organisation they work in.
func TestInboundLeadFormService_Submit_UsesTheOwnersOrganisation(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()
	memberOf := uuid.New()
	service.WithOrgResolver(fixedOrg{memberOf})
	_, err := service.Create(ctx, userID, &v1schema.InboundLeadFormCreateRequest{
		Name: "Lead Form", Slug: "member-form", Fields: []v1schema.InboundLeadFormFieldRequest{{Name: "email"}},
	})
	require.NoError(t, err)

	resp, err := service.Submit(ctx, "member-form", map[string]any{"email": "a@x.test"})
	require.NoError(t, err)
	stored, err := service.contactRepo.GetByID(ctx, resp.ContactID)
	require.NoError(t, err)
	require.NotNil(t, stored.OrganizationID)
	assert.Equal(t, memberOf, *stored.OrganizationID)
}

// A field the form does not ask for must not surface as "N/A".
func TestInboundLeadFormService_Submit_MissingPartsAreNotNA(t *testing.T) {
	service, userID := newInboundLeadFormService(t)
	ctx := context.Background()
	_, err := service.Create(ctx, userID, &v1schema.InboundLeadFormCreateRequest{
		Name: "Lead Form", Slug: "short-form", Fields: []v1schema.InboundLeadFormFieldRequest{{Name: "email"}, {Name: "first_name"}},
	})
	require.NoError(t, err)
	resp, err := service.Submit(ctx, "short-form", map[string]any{"email": "a@x.test", "first_name": "Ann"})
	require.NoError(t, err)
	stored, err := service.contactRepo.GetByID(ctx, resp.ContactID)
	require.NoError(t, err)
	assert.Equal(t, "Ann", stored.Name)
	assert.NotContains(t, string(stored.Profile), "phone")
}
