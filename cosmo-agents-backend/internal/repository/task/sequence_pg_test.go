package task

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// The worker stops a contact's sequence once they have answered: an incoming
// interaction after one of this campaign's emails was sent to them.
func TestContactsRepliedInCampaign(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.InteractionLog{}))
	repo := NewTaskRepository(db)
	ctx := context.Background()

	campaign, other := uuid.New(), uuid.New()
	sent := time.Now().Add(-48 * time.Hour)
	repliedAfter, repliedBefore, silent, notYetSent, otherCampaign := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()

	task := func(contact, camp uuid.UUID, sentAt *time.Time) {
		status := domain.TaskStatusDone
		if sentAt == nil {
			status = domain.TaskStatusFailed
		}
		trig := sent.Add(-time.Minute)
		require.NoError(t, db.Create(&domain.Task{ContactID: contact, CampaignID: camp, TemplateID: uuid.New(),
			Status: status, TriggeredAt: &trig, DoneAt: sentAt}).Error)
	}
	reply := func(contact uuid.UUID, at time.Time, direction string) {
		require.NoError(t, db.Create(&domain.InteractionLog{UserID: uuid.New(), ContactID: contact,
			Channel: "email", Direction: direction, Timestamp: at}).Error)
	}

	task(repliedAfter, campaign, &sent)
	reply(repliedAfter, sent.Add(time.Hour), "incoming")
	// A reply from before this campaign wrote to them is about something else.
	task(repliedBefore, campaign, &sent)
	reply(repliedBefore, sent.Add(-time.Hour), "incoming")
	// Our own follow-up is outgoing, not an answer.
	task(silent, campaign, &sent)
	reply(silent, sent.Add(time.Hour), "outgoing")
	// Queued but the send failed: a first email that never went out must not
	// be cancelled by a later email from the contact.
	task(notYetSent, campaign, nil)
	reply(notYetSent, sent.Add(time.Hour), "incoming")
	// Replied, but to a different campaign.
	task(otherCampaign, other, &sent)
	reply(otherCampaign, sent.Add(time.Hour), "incoming")

	got, err := repo.ContactsRepliedInCampaign(ctx, campaign)
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]bool{repliedAfter: true}, got)
}

// Activating a paused campaign recreates its tasks. That reset every sent
// email to pending, so the whole sequence went out again.
func TestCreateTasksKeepsSentTasks(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_task_triple ON tasks(contact_id, campaign_id, template_id)`).Error)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	campaign, contact := uuid.New(), uuid.New()
	sentT, cancelledT, failedT := uuid.New(), uuid.New(), uuid.New()
	for tpl, status := range map[uuid.UUID]domain.TaskStatus{
		sentT: domain.TaskStatusDone, cancelledT: domain.TaskStatusCancelled, failedT: domain.TaskStatusFailed,
	} {
		require.NoError(t, db.Create(&domain.Task{ContactID: contact, CampaignID: campaign, TemplateID: tpl, Status: status}).Error)
	}

	soon := time.Now().Add(time.Hour)
	for _, tpl := range []uuid.UUID{sentT, cancelledT, failedT} {
		_, err := repo.CreateTasks(ctx, []domain.TaskAttributes{{ContactID: contact, CampaignID: campaign,
			TemplateID: tpl, Status: domain.TaskStatusPending, ScheduleAt: &soon}})
		require.NoError(t, err)
	}

	status := func(tpl uuid.UUID) domain.TaskStatus {
		var tk domain.Task
		require.NoError(t, db.Where("template_id = ?", tpl).First(&tk).Error)
		return tk.Status
	}
	assert.Equal(t, domain.TaskStatusDone, status(sentT), "a sent email is not sent again")
	assert.Equal(t, domain.TaskStatusCancelled, status(cancelledT), "a stopped sequence stays stopped")
	assert.Equal(t, domain.TaskStatusPending, status(failedT), "a failed email is retried")
}
