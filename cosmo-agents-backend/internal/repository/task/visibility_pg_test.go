package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// GET /v1/task used FindAll with no owner at all: any user listed every
// tenant's tasks, and the dashboard's "Pending tasks" counted them all.
func TestFindAllVisibleTo(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.Campaign{}, &domain.Role{}))
	repo := NewTaskRepository(db)
	ctx := context.Background()

	me, teammate, stranger := uuid.New(), uuid.New(), uuid.New()
	org, otherOrg := uuid.New(), uuid.New()
	for _, r := range []domain.Role{
		{UserID: me, OrganizationID: org, Name: domain.RoleNameMember, Status: domain.RoleStatusActive},
		{UserID: teammate, OrganizationID: org, Name: domain.RoleNameMember, Status: domain.RoleStatusActive},
	} {
		require.NoError(t, db.Create(&r).Error)
	}
	campaign := func(owner uuid.UUID, o uuid.UUID) uuid.UUID {
		c := &domain.Campaign{UserID: owner, OrganizationID: &o, Name: "c", Playbook: "p", Status: domain.CampaignStatusActive}
		require.NoError(t, db.Create(c).Error)
		return c.ID
	}
	mine, teams, theirs := campaign(me, org), campaign(teammate, org), campaign(stranger, otherOrg)
	for _, c := range []uuid.UUID{mine, teams, theirs} {
		require.NoError(t, db.Create(&domain.Task{ContactID: uuid.New(), CampaignID: c, TemplateID: uuid.New(), Status: domain.TaskStatusPending}).Error)
	}

	campaigns := func(scope TaskScope) []uuid.UUID {
		res, err := repo.FindAllVisibleTo(ctx, me, scope, baseRepo.Filter{"status": domain.TaskStatusPending}, nil)
		require.NoError(t, err)
		ids := []uuid.UUID{}
		for _, tk := range res.List {
			ids = append(ids, tk.CampaignID)
		}
		return ids
	}
	assert.ElementsMatch(t, []uuid.UUID{mine, teams}, campaigns(TaskScope{}), "own and organisation tasks, never a stranger's")
	assert.ElementsMatch(t, []uuid.UUID{mine}, campaigns(TaskScope{OwnOnly: true}), "My work: own campaigns only")

	ok, err := repo.CampaignVisibleTo(ctx, theirs, me)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = repo.CampaignVisibleTo(ctx, teams, me)
	require.NoError(t, err)
	assert.True(t, ok)
}
