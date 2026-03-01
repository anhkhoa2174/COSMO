package contact

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

func emailFromProfile(t *testing.T, contact *domain.Contact) string {
	t.Helper()
	var profile map[string]interface{}
	require.NoError(t, contact.Profile.Unmarshal(&profile))
	email, _ := profile["email"].(string)
	return email
}

// sqliteContactRepo builds a sqlite-backed repo exercising real GORM code.
func sqliteContactRepo(t *testing.T) (*ContactRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Manually create contacts table without Postgres-specific indexes
	createContacts := `
	DROP TABLE IF EXISTS contacts;
	CREATE TABLE contacts (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		is_deleted BOOLEAN,
		user_id TEXT,
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
		business_stage TEXT
	);`
	require.NoError(t, db.Exec(createContacts).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_user_source_source_id ON contacts(user_id, source, source_id);`).Error)

	createListContacts := `
	DROP TABLE IF EXISTS list_contacts;
	CREATE TABLE list_contacts (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		is_deleted BOOLEAN,
		user_id TEXT,
		name TEXT,
		source TEXT,
		source_id TEXT,
		hubspot_id TEXT,
		organization_id TEXT
	);`
	require.NoError(t, db.Exec(createListContacts).Error)

	createAssoc := `
	DROP TABLE IF EXISTS list_contact_association;
	CREATE TABLE list_contact_association (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		is_deleted BOOLEAN,
		list_contact_id TEXT,
		contact_id TEXT
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_list_contact_assoc_pair ON list_contact_association(list_contact_id, contact_id);`
	require.NoError(t, db.Exec(createAssoc).Error)

	createInboundAssoc := `
	DROP TABLE IF EXISTS inbound_lead_form_list_contact_association;
	CREATE TABLE inbound_lead_form_list_contact_association (
		id TEXT PRIMARY KEY,
		created_at DATETIME,
		updated_at DATETIME,
		is_deleted BOOLEAN,
		inbound_lead_form_id TEXT,
		list_contact_id TEXT
	);`
	require.NoError(t, db.Exec(createInboundAssoc).Error)

	createInboundLeadForm := `
	DROP TABLE IF EXISTS inbound_lead_forms;
	CREATE TABLE inbound_lead_forms (
		id TEXT PRIMARY KEY
	);`
	require.NoError(t, db.Exec(createInboundLeadForm).Error)

	return NewContactRepository(db), db
}

func TestContactRepository_FindByEmail_SQLite(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c := &domain.Contact{
		Base:     domain.Base{ID: uuid.New()},
		UserID:   user,
		Profile:  base.JSONB(`{"email":"john@example.com"}`),
		Name:     "John Doe",
		Source:   string(domain.ContactSourceCosmoAgents),
		SourceID: uuid.New().String(),
	}
	require.NoError(t, db.Table("contacts").Create(map[string]interface{}{
		"id":         c.ID.String(),
		"user_id":    c.UserID.String(),
		"email":      "john@example.com",
		"name":       c.Name,
		"source":     c.Source,
		"source_id":  c.SourceID,
		"profile":    string(c.Profile),
		"is_deleted": false,
	}).Error)

	found, err := repo.FindByEmail(ctx, user, "john@example.com")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, c.ID, found.ID)
	assert.Equal(t, "john@example.com", emailFromProfile(t, found))

	none, err := repo.FindByEmail(ctx, user, "missing@example.com")
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestContactRepository_DeleteByIDs_SQLite(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c1 := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"a@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.New().String()}
	c2 := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"b@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.New().String()}
	require.NoError(t, db.Create([]*domain.Contact{c1, c2}).Error)

	deleted, err := repo.DeleteByIDs(ctx, []uuid.UUID{c1.ID}, user, nil)
	require.NoError(t, err)
	assert.Len(t, deleted, 1)
	assert.True(t, deleted[0].IsDeleted)

	var check domain.Contact
	require.NoError(t, db.First(&check, "id = ?", c1.ID).Error)
	assert.True(t, check.IsDeleted)
}

func TestContactRepository_FindIDsByListContact_SQLite(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()
	listID := uuid.New()

	c1 := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"a@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.New().String()}
	c2 := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"b@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.New().String()}
	require.NoError(t, db.Create([]*domain.Contact{c1, c2}).Error)

	assoc := []domain.ListContactAssociation{
		{ListContactID: listID, ContactID: c1.ID},
		{ListContactID: listID, ContactID: c2.ID},
	}
	require.NoError(t, db.Clauses(clause.OnConflict{DoNothing: true}).Create(&assoc).Error)

	ids, err := repo.FindIDsByListContact(ctx, listID, user, nil, 0, 10)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{c1.ID, c2.ID}, ids)
}

func TestContactRepository_Search_SQLite(t *testing.T) {
	t.Skip("sqlite does not support ILIKE; repository uses ILIKE for search")
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	records := []*domain.Contact{
		{UserID: user, Profile: base.JSONB(`{"email":"alice@example.com"}`), Name: "Alice", Company: "Acme", Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.New().String()},
		{UserID: user, Profile: base.JSONB(`{"email":"bob@example.com"}`), Name: "Bob", Company: "Beta", Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.New().String()},
	}
	require.NoError(t, db.Create(&records).Error)

	list, total, err := repo.Search(ctx, user, "acme", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, list, 1)
	assert.Equal(t, "alice@example.com", emailFromProfile(t, list[0]))

	// SearchWithFilter should also work (empty filter)
	page := baseRepo.PaginationParams{Limit: 10, Offset: 0}
	filtered, total2, err := repo.SearchWithFilter(ctx, user, nil, map[string]interface{}{}, &page)
	require.NoError(t, err)
	assert.Equal(t, 2, total2)
	assert.Len(t, filtered, 2)
}

func TestContactRepository_UpsertMany_SQLite(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	contacts := []*domain.Contact{
		{UserID: user, Source: string(domain.ContactSourceCosmoAgents), SourceID: "s1", Profile: base.JSONB(`{"email":"a@example.com"}`)},
	}
	require.NoError(t, repo.UpsertMany(ctx, contacts))

	// update same (user, source, source_id)
	contacts[0].Name = "Updated"
	require.NoError(t, repo.UpsertMany(ctx, contacts))

	var check domain.Contact
	require.NoError(t, db.First(&check, "user_id = ? AND source_id = ?", user, "s1").Error)
	assert.Equal(t, "Updated", check.Name)
}

func TestContactRepository_UpsertBySource_SQLite(t *testing.T) {
	repo, _ := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c := &domain.Contact{UserID: user, Source: string(domain.ContactSourceCosmoAgents), SourceID: "s1", Profile: base.JSONB(`{"email":"a@example.com"}`)}
	created, err := repo.UpsertBySource(ctx, c)
	require.NoError(t, err)
	assert.Equal(t, "a@example.com", emailFromProfile(t, created))

	c.Profile = base.JSONB(`{"email":"updated@example.com"}`)
	updated, err := repo.UpsertBySource(ctx, c)
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", emailFromProfile(t, updated))
}

func TestContactRepository_DeleteByIDs_WithOrg_SQLite(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()
	org := uuid.New()

	c := &domain.Contact{UserID: user, OrganizationID: &org, Profile: base.JSONB(`{"email":"org@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.New().String()}
	require.NoError(t, db.Create(c).Error)

	deleted, err := repo.DeleteByIDs(ctx, []uuid.UUID{c.ID}, user, &org)
	require.NoError(t, err)
	assert.Len(t, deleted, 1)
	assert.True(t, deleted[0].IsDeleted)
}

func TestContactRepository_FindByUserIDWithPagination_SQLite(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	for i := 0; i < 3; i++ {
		require.NoError(t, db.Create(&domain.Contact{
			UserID:   user,
			Profile:  base.JSONB(`{"email":"` + uuid.NewString() + `@example.com"}`),
			Source:   string(domain.ContactSourceCosmoAgents),
			SourceID: uuid.NewString(),
		}).Error)
	}

	list, total, err := repo.FindByUserIDWithPagination(ctx, user, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, list, 2)
}

func TestContactRepository_UpdateFields_AccessControl(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()
	org := uuid.New()

	c := &domain.Contact{
		UserID:         user,
		OrganizationID: &org,
		Profile:        base.JSONB(`{"email":"update@example.com"}`),
		Source:         string(domain.ContactSourceCosmoAgents),
		SourceID:       uuid.NewString(),
		Company:        "OldCo",
	}
	require.NoError(t, db.Create(c).Error)

	updated, err := repo.UpdateFields(ctx, c.ID, map[string]interface{}{
		"company":        "NewCo",
		"do_not_contact": true,
	}, user, org)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "NewCo", updated.Company)
	assert.True(t, updated.DoNotContact)
}

func TestContactRepository_UpdateFields_Denied(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c := &domain.Contact{
		UserID:   user,
		Profile:  base.JSONB(`{"email":"deny@example.com"}`),
		Source:   string(domain.ContactSourceCosmoAgents),
		SourceID: uuid.NewString(),
	}
	require.NoError(t, db.Create(c).Error)

	_, err := repo.UpdateFields(ctx, c.ID, map[string]interface{}{"company": "Nope"}, uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contact not found")
}

func TestContactRepository_GetDistinctFieldValues(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	records := []*domain.Contact{
		{UserID: user, Profile: base.JSONB(`{"email":"a@example.com"}`), Company: "Acme", Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.NewString()},
		{UserID: user, Profile: base.JSONB(`{"email":"b@example.com"}`), Company: "Beta", Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.NewString()},
		{UserID: user, Profile: base.JSONB(`{"email":"c@example.com"}`), Company: "Acme", Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.NewString()},
	}
	require.NoError(t, db.Create(&records).Error)

	values, total, err := repo.GetDistinctFieldValues(ctx, "company", user, nil, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.ElementsMatch(t, []interface{}{"Acme", "Beta"}, values)
}

func TestContactRepository_FindByIDs(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	empty, err := repo.FindByIDs(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, empty)

	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		c := &domain.Contact{
			UserID:   user,
			Profile:  base.JSONB(`{"email":"` + uuid.NewString() + `@example.com"}`),
			Source:   string(domain.ContactSourceCosmoAgents),
			SourceID: uuid.NewString(),
		}
		require.NoError(t, db.Create(c).Error)
		ids = append(ids, c.ID)
	}

	found, err := repo.FindByIDs(ctx, ids)
	require.NoError(t, err)
	assert.Len(t, found, 3)
}

func TestContactRepository_BatchCreateAndUpdate(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	contacts := []*domain.Contact{
		{UserID: user, Profile: base.JSONB(`{"email":"x@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: "s1", Name: "X"},
		{UserID: user, Profile: base.JSONB(`{"email":"y@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: "s2", Name: "Y"},
	}
	require.NoError(t, repo.BatchCreate(ctx, contacts))

	contacts[0].Name = "X2"
	contacts[1].Name = "Y2"
	require.NoError(t, repo.BatchUpdate(ctx, contacts))

	var check1, check2 domain.Contact
	require.NoError(t, db.First(&check1, "source_id = ?", "s1").Error)
	require.NoError(t, db.First(&check2, "source_id = ?", "s2").Error)
	assert.Equal(t, "X2", check1.Name)
	assert.Equal(t, "Y2", check2.Name)
}

func TestContactRepository_AddToList_NoDuplicate(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()
	listID := uuid.New()

	c := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"list@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.NewString()}
	require.NoError(t, db.Create(c).Error)

	require.NoError(t, repo.AddToList(ctx, c.ID, listID))
	require.NoError(t, repo.AddToList(ctx, c.ID, listID))

	var count int64
	require.NoError(t, db.Table("list_contact_association").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestContactRepository_SearchWithFilter_DefaultPagination(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&domain.Contact{
			UserID:   user,
			Profile:  base.JSONB(`{"email":"` + uuid.NewString() + `@example.com"}`),
			Source:   string(domain.ContactSourceCosmoAgents),
			SourceID: uuid.NewString(),
		}).Error)
	}

	results, total, err := repo.SearchWithFilter(ctx, user, nil, map[string]interface{}{}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)
}

func TestContactRepository_FindByIDAndSourceAndSearch(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c := &domain.Contact{
		UserID:   user,
		Profile:  base.JSONB(`{"email":"find@example.com"}`),
		Source:   string(domain.ContactSourceCosmoAgents),
		SourceID: "src-1",
	}
	require.NoError(t, db.Create(c).Error)

	found, err := repo.FindByIDAndUserID(ctx, user, c.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, c.ID, found.ID)

	foundBySource, err := repo.FindBySourceID(ctx, user, c.Source, c.SourceID)
	require.NoError(t, err)
	require.NotNil(t, foundBySource)
	assert.Equal(t, emailFromProfile(t, c), emailFromProfile(t, foundBySource))

	list, total, err := repo.Search(ctx, user, "", 0, 5)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, list, 1)
}

func TestContactRepository_GetBySourceIDs(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()

	contacts := []*domain.Contact{
		{UserID: uuid.New(), Profile: base.JSONB(`{"email":"a@example.com"}`), Source: "hubspot", SourceID: "s1"},
		{UserID: uuid.New(), Profile: base.JSONB(`{"email":"b@example.com"}`), Source: "hubspot", SourceID: "s2"},
	}
	require.NoError(t, db.Create(&contacts).Error)

	found, err := repo.GetBySourceIDs(ctx, "hubspot", []string{"s1", "s2"})
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestContactRepository_CreateAndSoftDelete(t *testing.T) {
	repo, db := sqliteContactRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c := &domain.Contact{
		UserID:   user,
		Profile:  base.JSONB(`{"email":"create@example.com"}`),
		Source:   string(domain.ContactSourceCosmoAgents),
		SourceID: uuid.NewString(),
	}
	require.NoError(t, repo.Create(ctx, c))

	require.NoError(t, repo.SoftDelete(ctx, c.ID))

	var check domain.Contact
	require.NoError(t, db.Unscoped().First(&check, "id = ?", c.ID).Error)
	assert.True(t, check.IsDeleted)
}

func TestContactRepository_IsValidContactField(t *testing.T) {
	assert.True(t, IsValidContactField("email"))
	assert.False(t, IsValidContactField("unknown_field"))
}

func TestListContactRepository_Basics(t *testing.T) {
	contactRepo, db := sqliteContactRepo(t)
	_ = contactRepo
	listRepo := NewListContactRepository(db)
	ctx := context.Background()

	user := uuid.New()
	l1 := &domain.ListContact{UserID: user, Name: "List 1", Source: "src", SourceID: "s1"}
	l2 := &domain.ListContact{UserID: user, Name: "List 2", Source: "src", SourceID: "s2"}
	require.NoError(t, db.Create([]*domain.ListContact{l1, l2}).Error)

	c1 := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"l1a@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.NewString()}
	c2 := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"l1b@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.NewString()}
	require.NoError(t, db.Create([]*domain.Contact{c1, c2}).Error)

	assoc := []domain.ListContactAssociation{
		{ListContactID: l1.ID, ContactID: c1.ID},
		{ListContactID: l1.ID, ContactID: c2.ID},
	}
	require.NoError(t, db.Create(&assoc).Error)

	counts, err := listRepo.BatchCountContacts(ctx, []uuid.UUID{l1.ID, l2.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts[l1.ID])
	assert.Equal(t, int64(0), counts[l2.ID])

	found, err := listRepo.FindByID(ctx, l1.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Len(t, found.Contacts, 2)

	page, err := listRepo.SearchOptimized(ctx, baseRepo.Filter{}, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	listsByUser, totalUser, err := listRepo.GetByUserID(ctx, user.String(), 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), totalUser)
	assert.Len(t, listsByUser, 2)

	bySource, totalSource, err := listRepo.GetBySource(ctx, "src", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), totalSource)
	assert.Len(t, bySource, 2)
}

func TestListContactRepository_InboundAndUpsert(t *testing.T) {
	_, db := sqliteContactRepo(t)
	listRepo := NewListContactRepository(db)
	ctx := context.Background()
	user := uuid.New()

	list := &domain.ListContact{UserID: user, Name: "Inbound List", Source: "api", SourceID: "api-1"}
	require.NoError(t, db.Create(list).Error)

	f1 := uuid.New()
	f2 := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO inbound_lead_forms (id) VALUES (?)", f1.String()).Error)
	require.NoError(t, db.Exec("INSERT INTO inbound_lead_forms (id) VALUES (?)", f2.String()).Error)

	require.NoError(t, listRepo.AddInboundForms(ctx, list.ID, []uuid.UUID{f1, f2}))

	inboundCounts, err := listRepo.BatchCountInboundForms(ctx, []uuid.UUID{list.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), inboundCounts[list.ID])

	forms, err := listRepo.GetInboundLeadForms(ctx, list.ID)
	require.NoError(t, err)
	assert.Len(t, forms, 2)

	upsertTarget := &domain.ListContact{UserID: user, Name: "Upserted", Source: "api", SourceID: "api-2"}
	require.NoError(t, listRepo.Upsert(ctx, upsertTarget))

	updated := &domain.ListContact{UserID: user, Name: "Upserted Updated", Source: "api", SourceID: "api-2"}
	require.NoError(t, listRepo.Upsert(ctx, updated))

	var fetched domain.ListContact
	require.NoError(t, db.First(&fetched, "source_id = ?", "api-2").Error)
	assert.NotEmpty(t, fetched.Name)

	c1 := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"assoc1@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.NewString()}
	c2 := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"assoc2@example.com"}`), Source: string(domain.ContactSourceCosmoAgents), SourceID: uuid.NewString()}
	require.NoError(t, db.Create([]*domain.Contact{c1, c2}).Error)

	require.NoError(t, listRepo.AddContacts(ctx, list.ID, []uuid.UUID{c1.ID, c2.ID}))

	counts, err := listRepo.BatchCountContacts(ctx, []uuid.UUID{list.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts[list.ID])
}
