package relationship

import (
	"context"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/agents"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

const defaultBatchSize = 200

// RecalculateWorker recomputes relationship scores for contacts on a schedule.
type RecalculateWorker struct {
	contactRepo *contactrepo.ContactRepository
	agent       *agents.RelationshipScorerAgent
}

func NewRecalculateWorker(contactRepo *contactrepo.ContactRepository, agent *agents.RelationshipScorerAgent) *RecalculateWorker {
	return &RecalculateWorker{
		contactRepo: contactRepo,
		agent:       agent,
	}
}

func (w *RecalculateWorker) HandleRecalculateScores(ctx context.Context, _ *asynq.Task) error {
	userIDs, err := w.listUsers(ctx)
	if err != nil {
		return err
	}
	logger.Logger.Info().Int("users", len(userIDs)).Msg("relationship: recalc started")

	for _, userID := range userIDs {
		offset := 0
		for {
			contacts, total, err := w.contactRepo.FindByUserIDWithPagination(ctx, userID, offset, defaultBatchSize)
			if err != nil {
				logger.Logger.Error().Err(err).Str("user_id", userID.String()).Msg("relationship: list contacts failed")
				break
			}
			if len(contacts) == 0 {
				break
			}
			for _, contact := range contacts {
				orgID := uuid.Nil
				if contact.OrganizationID != nil {
					orgID = *contact.OrganizationID
				}
				if _, err := w.agent.Run(ctx, userID, orgID, contact.ID); err != nil {
					logger.Logger.Error().Err(err).Str("contact_id", contact.ID.String()).Msg("relationship: recalc failed")
				}
			}
			offset += len(contacts)
			if offset >= total {
				break
			}
		}
	}

	logger.Logger.Info().Msg("relationship: recalc finished")
	return nil
}

func (w *RecalculateWorker) listUsers(ctx context.Context) ([]uuid.UUID, error) {
	type row struct {
		UserID uuid.UUID `gorm:"column:user_id"`
	}
	var rows []row
	if err := w.contactRepo.GetDB().WithContext(ctx).
		Table("contacts").
		Select("DISTINCT user_id").
		Where("is_deleted = ?", false).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.UserID)
	}
	return out, nil
}
