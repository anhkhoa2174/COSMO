package inbound_lead_form

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
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
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.InboundLeadForm{},
		&domain.FormField{},
		&domain.CustomField{},
		&organization.Organization{},
	))
	if err := db.Exec(`ALTER TABLE inbound_lead_forms ADD COLUMN is_deleted BOOLEAN DEFAULT false`).Error; err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			require.NoError(t, err)
		}
	}

	// Minimal contacts table for SQLite (avoids Postgres-specific GIN index)
	createContactSQL := `
	CREATE TABLE contacts (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		source_id TEXT,
		hubspot_id TEXT,
		source TEXT,
		name TEXT,
		first_name TEXT,
		last_name TEXT,
		email TEXT,
		phone TEXT,
		company TEXT,
		job_title TEXT,
		address TEXT,
		city TEXT,
		country TEXT,
		state TEXT,
		zip TEXT,
		profile TEXT,
		confirmed_facts TEXT,
		ai_insights TEXT,
		insight_validation TEXT,
		scores TEXT,
		do_not_contact BOOLEAN,
		organization_id TEXT,
		tags TEXT,
		status TEXT,
		missing_fields TEXT,
		contact_information TEXT,
		industry TEXT,
		contact_channel TEXT,
		lifecycle_stage TEXT,
		context_level TEXT,
		outreach_decision TEXT,
		scenario TEXT,
		message_draft TEXT,
		last_outcome TEXT,
		next_step TEXT,
		meeting TEXT,
		business_stage TEXT,
		is_deleted BOOLEAN DEFAULT false,
		created_at DATETIME,
		updated_at DATETIME
	);`
	require.NoError(t, db.Exec(createContactSQL).Error)

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
