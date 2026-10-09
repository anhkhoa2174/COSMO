package organization

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

// An organisation that never saved outreach settings has NULL in the column.
// Scanning that into []byte failed, and every AI reply for its members was
// drafted without the organisation's guidance.
func TestOutreachSettingsForUser_NullSettings(t *testing.T) {
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Organization{}, &domain.Role{}))
	repo := NewOrganizationRepository(db)

	user := uuid.New()
	org := &domain.Organization{Name: "acme", CompanyURL: "https://acme.test"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Exec(`UPDATE organizations SET outreach_settings = NULL WHERE id = ?`, org.ID).Error)
	require.NoError(t, db.Create(&domain.Role{UserID: user, OrganizationID: org.ID, Name: domain.RoleNameMember, Status: domain.RoleStatusActive}).Error)

	raw, err := repo.OutreachSettingsForUser(context.Background(), user)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(raw))
}
