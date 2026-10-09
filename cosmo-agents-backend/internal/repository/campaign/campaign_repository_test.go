package campaign

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// newTestRepo creates a sqlite in-memory repo for testing.
func newTestRepo(t *testing.T) (*CampaignRepository, *gorm.DB) {
	t.Helper()

	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&domain.Campaign{}))

	return NewCampaignRepository(db), db
}

func TestFindByUserIDFiltersByUserAndNotDeleted(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	c1 := &domain.Campaign{Name: "A1", Playbook: "pb1", UserID: userA}
	c2 := &domain.Campaign{Name: "A2", Playbook: "pb2", UserID: userA}
	c3 := &domain.Campaign{Name: "B1", Playbook: "pb3", UserID: userB}
	require.NoError(t, db.Create([]*domain.Campaign{c1, c2, c3}).Error)
	// mark c2 deleted
	require.NoError(t, db.Model(c2).Update("is_deleted", true).Error)

	page := baseRepo.PaginationParams{Offset: 0, Limit: 10}
	res, err := repo.FindByUserID(ctx, userA, &page)
	require.NoError(t, err)
	require.Equal(t, int64(1), res.Total)
	require.Len(t, res.List, 1)
	require.Equal(t, c1.ID, res.List[0].ID)
}

func TestFindByStatusFiltersStatus(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	user := uuid.New()

	active := &domain.Campaign{Name: "active", Playbook: "p1", UserID: user, Status: domain.CampaignStatusActive}
	draft := &domain.Campaign{Name: "draft", Playbook: "p2", UserID: user, Status: domain.CampaignStatusDraft}
	require.NoError(t, db.Create([]*domain.Campaign{active, draft}).Error)

	page := baseRepo.PaginationParams{Offset: 0, Limit: 10}
	res, err := repo.FindByStatus(ctx, user, domain.CampaignStatusActive, &page)
	require.NoError(t, err)
	require.Equal(t, int64(1), res.Total)
	require.Equal(t, active.ID, res.List[0].ID)
}

func TestFindWithRelationsReturnsCampaign(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c := &domain.Campaign{Name: "has-rel", Playbook: "pb", UserID: user}
	require.NoError(t, db.Create(c).Error)

	found, err := repo.FindWithRelations(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, c.ID, found.ID)

	missing, err := repo.FindWithRelations(ctx, uuid.New())
	require.NoError(t, err)
	require.Nil(t, missing)
}

func TestUpdateStatus(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c := &domain.Campaign{Name: "update-status", Playbook: "pb", UserID: user, Status: domain.CampaignStatusDraft}
	require.NoError(t, db.Create(c).Error)

	require.NoError(t, repo.UpdateStatus(ctx, c.ID, domain.CampaignStatusActive))

	var refreshed domain.Campaign
	require.NoError(t, db.First(&refreshed, "id = ?", c.ID).Error)
	require.Equal(t, domain.CampaignStatusActive, refreshed.Status)
}

func TestUpdateAttributes(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	user := uuid.New()

	c := &domain.Campaign{Name: "before", Playbook: "pb", UserID: user}
	require.NoError(t, db.Create(c).Error)

	attrs := map[string]interface{}{
		"name":     "after",
		"playbook": "new-pb",
	}
	require.NoError(t, repo.UpdateAttributes(ctx, c.ID, attrs))

	var refreshed domain.Campaign
	require.NoError(t, db.First(&refreshed, "id = ?", c.ID).Error)
	require.Equal(t, "after", refreshed.Name)
	require.Equal(t, "new-pb", refreshed.Playbook)
}

func TestUpdateAttributes_NoAttrs(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	userID := uuid.New()

	c := &domain.Campaign{Name: "keep", Playbook: "p1", UserID: userID}
	require.NoError(t, db.Create(c).Error)

	require.NoError(t, repo.UpdateAttributes(ctx, c.ID, map[string]interface{}{}))

	var check domain.Campaign
	require.NoError(t, db.First(&check, "id = ?", c.ID).Error)
	require.Equal(t, "keep", check.Name)
	require.Equal(t, "p1", check.Playbook)
}

type stubResult struct {
	columns []string
	rows    [][]driver.Value
}

type stubDriver struct {
	results []stubResult
}

func (d *stubDriver) Open(name string) (driver.Conn, error) {
	results := make([]stubResult, len(d.results))
	copy(results, d.results)
	return &stubConn{results: results}, nil
}

type stubConn struct {
	results []stubResult
	idx     int
}

func (c *stubConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (c *stubConn) Close() error              { return nil }
func (c *stubConn) Begin() (driver.Tx, error) { return nil, errors.New("tx not supported") }

func (c *stubConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.idx >= len(c.results) {
		return nil, errors.New("no more results")
	}
	res := c.results[c.idx]
	c.idx++
	return &stubRows{columns: res.columns, rows: res.rows}, nil
}

func (c *stubConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

type stubRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *stubRows) Columns() []string { return r.columns }
func (r *stubRows) Close() error      { return nil }
func (r *stubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

func newStubStatsRepo(t *testing.T, results []stubResult) *CampaignRepository {
	t.Helper()
	driverName := "campaign_stub_" + uuid.NewString()
	sql.Register(driverName, &stubDriver{results: results})

	sqlDB, err := sql.Open(driverName, "")
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return NewCampaignRepository(gdb)
}

func TestCampaignRepository_FindWithStats_Stub(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orgID := uuid.New()

	results := []stubResult{
		{columns: []string{"count"}, rows: [][]driver.Value{{int64(1)}}},
		{
			columns: []string{
				"id", "name", "user_id", "is_deleted", "status",
				"creator", "sent", "reply", "reply_rate", "interested",
				"interest_rate", "agent_name", "agent_cmetadata",
			},
			rows: [][]driver.Value{{
				uuid.NewString(), "Camp", userID.String(), false, string(domain.CampaignStatusActive),
				"creator", int64(10), int64(2), float64(0.2), int64(1),
				float64(0.5), "Agent", "{}",
			}},
		},
	}

	repo := newStubStatsRepo(t, results)

	rows, total, err := repo.FindWithStats(ctx, userID, &orgID, baseRepo.Filter{}, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	require.Equal(t, "Camp", rows[0].Name)
	require.Equal(t, "creator", rows[0].Creator.String)
}
