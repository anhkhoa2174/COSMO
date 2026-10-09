package contact

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactRepository_PipelineSummary_Postgres(t *testing.T) {
	repo, db := pgContactRepo(t)
	ctx := context.Background()
	require.NoError(t, db.Exec(`CREATE TABLE interaction_logs (contact_id UUID, "timestamp" TIMESTAMPTZ)`).Error)
	user, other := uuid.New(), uuid.New()
	now := time.Now()
	old := now.AddDate(0, 0, -10)
	recent := now.AddDate(0, 0, -1)

	add := func(owner uuid.UUID, stage string, followups int, last *time.Time, deleted bool) {
		id := uuid.New()
		row := map[string]interface{}{
			"id": id.String(), "user_id": owner.String(),
			"name": "c", "source": "test", "source_id": uuid.New().String(),
			"outreach_stage": stage, "followup_count": followups,
			"is_deleted": deleted,
		}
		require.NoError(t, db.Table("contacts").Create(row).Error)
		if last != nil {
			// An older interaction too, so the latest one is what counts.
			require.NoError(t, db.Exec(`INSERT INTO interaction_logs VALUES (?, ?), (?, ?)`,
				id, last.AddDate(0, 0, -30), id, *last).Error)
		}
	}

	add(user, "COLD", 0, nil, false)      // never contacted: not stalled
	// COLD but the column says 3: imported or test rows carry stray counts.
	// Never written to, it has had no follow-ups.
	add(user, "COLD", 3, nil, false)
	add(user, "", 0, nil, false)          // no stage recorded counts as COLD
	add(user, "NO_REPLY", 1, &old, false) // silent for 10 days: stalled
	add(user, "NO_REPLY", 2, &recent, false)
	add(user, "REPLIED", 1, &old, false) // quiet but replied: waiting on us, not stalled
	add(user, "DROPPED", 5, &old, false) // retired: not stalled
	add(user, "REPLIED", 0, nil, true)   // deleted: ignored
	add(other, "REPLIED", 0, nil, false) // someone else's: ignored

	got, err := repo.PipelineSummary(ctx, user, now.AddDate(0, 0, -5))
	require.NoError(t, err)

	assert.Equal(t, int64(7), got.Total)
	assert.Equal(t, map[string]int64{"COLD": 3, "NO_REPLY": 2, "REPLIED": 1, "DROPPED": 1}, got.ByStage)
	assert.Equal(t, [4]int64{3, 2, 1, 1}, got.FollowupDepth)
	assert.Equal(t, int64(1), got.Stalled)
}
