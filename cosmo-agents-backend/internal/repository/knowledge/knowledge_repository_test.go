package knowledge

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

func newTestRepo(t *testing.T) (*KnowledgeRepository, *gorm.DB) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	db = db.Session(&gorm.Session{AllowGlobalUpdate: true})
	require.NoError(t, db.AutoMigrate(&domain.Knowledge{}))
	return NewKnowledgeRepository(db), db
}

func makeKnowledge(userID uuid.UUID, embedding string) *domain.Knowledge {
	return &domain.Knowledge{
		UserID:       userID,
		SourceType:   domain.KnowledgeSourceUpload,
		SummaryPair:  pq.StringArray{"compressed", "raw"},
		EmbeddingGID: &embedding,
		CMetadata:    base.JSONB([]byte(`{"foo":"bar"}`)),
	}
}

func TestKnowledgeRepository_FindAndGet(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	userID := uuid.New()

	k1 := makeKnowledge(userID, "gid-1")
	k2 := makeKnowledge(userID, "gid-2")
	k3 := makeKnowledge(uuid.New(), "gid-3")
	require.NoError(t, db.Create([]*domain.Knowledge{k1, k2, k3}).Error)

	found, err := repo.FindByIDs(ctx, []uuid.UUID{k1.ID, k2.ID})
	require.NoError(t, err)
	assert.Len(t, found, 2)
	idSet := map[uuid.UUID]bool{k1.ID: false, k2.ID: false}
	for _, k := range found {
		idSet[k.ID] = true
	}
	for _, seen := range idSet {
		assert.True(t, seen)
	}

	got, err := repo.GetByEmbeddingGID(ctx, "gid-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "gid-1", *got.EmbeddingGID)

	missing, err := repo.GetByEmbeddingGID(ctx, "not-there")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestKnowledgeRepository_UpdateAndDelete(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	userID := uuid.New()

	k := makeKnowledge(userID, "gid-update")
	require.NoError(t, db.Create(k).Error)

	updatePayload := map[string]interface{}{
		"source_type": domain.KnowledgeSourceWebsite,
		"cmetadata":   base.JSONB([]byte(`{"updated":true}`)),
	}
	require.NoError(t, repo.UpdateByEmbeddingGID(ctx, "gid-update", updatePayload))

	updated, err := repo.GetByEmbeddingGID(ctx, "gid-update")
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, domain.KnowledgeSourceWebsite, updated.SourceType)
	assert.JSONEq(t, `{"updated":true}`, string(updated.CMetadata))

	require.NoError(t, repo.DeleteByEmbeddingGID(ctx, "gid-update"))
	afterDelete, err := repo.GetByEmbeddingGID(ctx, "gid-update")
	require.NoError(t, err)
	assert.Nil(t, afterDelete)
}

func TestKnowledgeRepository_UpsertCreateAndUpdate(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()
	userID := uuid.New()

	toCreate := makeKnowledge(userID, "gid-upsert")
	created, err := repo.UpsertByEmbeddingGID(ctx, toCreate)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.False(t, created.IsDeleted)

	newDataset := "dataset-1"
	updateCandidate := makeKnowledge(userID, "gid-upsert")
	updateCandidate.SourceType = domain.KnowledgeSourceWebsite
	updateCandidate.CMetadata = base.JSONB([]byte(`{"next":true}`))
	updateCandidate.CozeDatasetID = &newDataset

	updated, err := repo.UpsertByEmbeddingGID(ctx, updateCandidate)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, domain.KnowledgeSourceWebsite, updated.SourceType)
	assert.Equal(t, newDataset, *updated.CozeDatasetID)
	assert.JSONEq(t, `{"next":true}`, string(updated.CMetadata))

	fetched, err := repo.GetByEmbeddingGID(ctx, "gid-upsert")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, domain.KnowledgeSourceWebsite, fetched.SourceType)
}

func TestKnowledgeRepository_UpsertResurrectsSoftDeleted(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	userID := uuid.New()

	softDeleted := makeKnowledge(userID, "gid-soft")
	require.NoError(t, db.Create(softDeleted).Error)
	require.NoError(t, db.Model(&domain.Knowledge{}).Where("id = ?", softDeleted.ID).Update("is_deleted", true).Error)

	updateCandidate := makeKnowledge(userID, "gid-soft")
	updateCandidate.SourceType = domain.KnowledgeSourceWebsite
	resurrected, err := repo.UpsertByEmbeddingGID(ctx, updateCandidate)
	require.NoError(t, err)
	require.NotNil(t, resurrected)
	assert.False(t, resurrected.IsDeleted)
	assert.Equal(t, domain.KnowledgeSourceWebsite, resurrected.SourceType)

	fromDB, err := repo.GetByEmbeddingGID(ctx, "gid-soft")
	require.NoError(t, err)
	require.NotNil(t, fromDB)
	assert.False(t, fromDB.IsDeleted)
}

func TestKnowledgeRepository_UpsertOwnershipConflict(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	owner := uuid.New()
	conflictUser := uuid.New()
	existing := makeKnowledge(owner, "gid-conflict")
	require.NoError(t, db.Create(existing).Error)

	conflict := makeKnowledge(conflictUser, "gid-conflict")
	_, err := repo.UpsertByEmbeddingGID(ctx, conflict)
	require.Error(t, err)
	assert.Equal(t, ErrKnowledgeOwnershipConflict, err)
}

func TestKnowledgeRepository_GetByEmbeddingGIDUnscoped(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	k := makeKnowledge(uuid.New(), "gid-unscoped")
	require.NoError(t, db.Create(k).Error)
	require.NoError(t, db.Model(&domain.Knowledge{}).Where("id = ?", k.ID).Update("is_deleted", true).Error)

	unscoped, err := repo.GetByEmbeddingGIDUnscoped(ctx, "gid-unscoped")
	require.NoError(t, err)
	require.NotNil(t, unscoped)
	assert.True(t, unscoped.IsDeleted)
}

func TestKnowledgeRepository_GetByUserIDAndCollection(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	// Add collection column for this in-memory database to match repository queries.
	err := db.Exec("ALTER TABLE knowledges ADD COLUMN collection text").Error
	require.NoError(t, err)

	userID := uuid.New()
	otherUser := uuid.New()

	first := makeKnowledge(userID, "gid-user-1")
	second := makeKnowledge(userID, "gid-user-2")
	third := makeKnowledge(otherUser, "gid-user-3")

	require.NoError(t, db.Create([]*domain.Knowledge{first, second, third}).Error)
	require.NoError(t, db.Model(&domain.Knowledge{}).Where("id = ?", second.ID).Update("updated_at", time.Now().Add(time.Minute)).Error)
	require.NoError(t, db.Model(&domain.Knowledge{}).Where("id = ?", third.ID).Update("collection", "c-other").Error)
	require.NoError(t, db.Model(&domain.Knowledge{}).Where("id = ?", first.ID).Update("collection", "c-one").Error)
	require.NoError(t, db.Model(&domain.Knowledge{}).Where("id = ?", second.ID).Update("collection", "c-one").Error)

	list, total, err := repo.GetByUserID(ctx, userID.String(), 5, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	assert.Equal(t, second.ID, list[0].ID, "should order by updated_at desc")

	collectionList, collectionTotal, err := repo.GetByCollection(ctx, "c-one", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), collectionTotal)
	assert.Len(t, collectionList, 2)

	// Confirm deleted entries are excluded
	require.NoError(t, db.Model(&domain.Knowledge{}).Where("id = ?", first.ID).Update("is_deleted", true).Error)
	collectionList, collectionTotal, err = repo.GetByCollection(ctx, "c-one", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), collectionTotal)
	assert.Len(t, collectionList, 1)
}

func TestKnowledgeRepository_UpsertHandlesNilInput(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	got, err := repo.UpsertByEmbeddingGID(ctx, nil)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, strings.Contains(err.Error(), "embedding_gid"))
}
