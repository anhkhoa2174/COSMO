package segmentation

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	segdomain "github.com/rockship/cosmo-agents-go/internal/domain/segmentation"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	"github.com/rockship/cosmo-agents-go/internal/service/intelligence"
	workerdto "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

const (
	defaultBatchSize = 200
)

// RecalculateWorker recalculates segment scores on a schedule.
type RecalculateWorker struct {
	db          *gorm.DB
	intelSvc    *intelligence.Service
	contactRepo *contactrepo.ContactRepository
}

func NewRecalculateWorker(
	db *gorm.DB,
	intelSvc *intelligence.Service,
	contactRepo *contactrepo.ContactRepository,
) *RecalculateWorker {
	return &RecalculateWorker{
		db:          db,
		intelSvc:    intelSvc,
		contactRepo: contactRepo,
	}
}

// HandleRecalculateScores recalculates fit scores for contacts.
func (w *RecalculateWorker) HandleRecalculateScores(ctx context.Context, task *asynq.Task) error {
	var payload workerdto.RecalculateSegmentScoresPayload
	if len(task.Payload()) > 0 {
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			logger.Logger.Warn().Err(err).Msg("segmentation: failed to parse payload, using defaults")
		}
	}

	if payload.ContactID != nil {
		return w.recalculateContact(ctx, payload)
	}

	userIDs, err := w.listActiveSegmentUsers(ctx)
	if err != nil {
		return err
	}
	logger.Logger.Info().
		Int("users", len(userIDs)).
		Msg("segmentation: recalc started")

	for _, userID := range userIDs {
		if err := w.recalculateUser(ctx, userID); err != nil {
			logger.Logger.Error().Err(err).Str("user_id", userID.String()).Msg("segmentation: recalc failed for user")
		}
	}

	logger.Logger.Info().Msg("segmentation: recalc finished")
	return nil
}

func (w *RecalculateWorker) recalculateContact(ctx context.Context, payload workerdto.RecalculateSegmentScoresPayload) error {
	contactID := *payload.ContactID
	contactModel, err := w.contactRepo.GetByID(ctx, contactID)
	if err != nil || contactModel == nil {
		return err
	}
	userID := contactModel.UserID
	if payload.UserID != nil {
		userID = *payload.UserID
	}
	_, err = w.intelSvc.CalculateSegmentScores(ctx, userID, uuid.Nil, contactID, nil)
	if err != nil {
		logger.Logger.Error().Err(err).Str("contact_id", contactID.String()).Msg("segmentation: recalc failed for contact")
		return err
	}
	logger.Logger.Info().Str("contact_id", contactID.String()).Msg("segmentation: recalc completed for contact")
	return nil
}

func (w *RecalculateWorker) recalculateUser(ctx context.Context, userID uuid.UUID) error {
	if !w.hasActiveSegments(ctx, userID) {
		logger.Logger.Debug().Str("user_id", userID.String()).Msg("segmentation: no active segments, skipping")
		return nil
	}

	offset := 0
	totalProcessed := 0
	for {
		contacts, total, err := w.contactRepo.FindByUserIDWithPagination(ctx, userID, offset, defaultBatchSize)
		if err != nil {
			return err
		}
		if len(contacts) == 0 {
			break
		}
		for _, contact := range contacts {
			_, err := w.intelSvc.CalculateSegmentScores(ctx, userID, uuid.Nil, contact.ID, nil)
			if err != nil {
				logger.Logger.Error().
					Err(err).
					Str("user_id", userID.String()).
					Str("contact_id", contact.ID.String()).
					Msg("segmentation: recalc failed for contact")
				continue
			}
			totalProcessed++
		}
		offset += len(contacts)
		if offset >= total {
			break
		}
	}

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Int("processed", totalProcessed).
		Msg("segmentation: recalc completed for user")
	return nil
}

func (w *RecalculateWorker) listActiveSegmentUsers(ctx context.Context) ([]uuid.UUID, error) {
	var userIDs []uuid.UUID
	if err := w.db.WithContext(ctx).
		Model(&segdomain.Segmentation{}).
		Where("is_active = ?", true).
		Distinct("user_id").
		Pluck("user_id", &userIDs).Error; err != nil {
		return nil, err
	}
	return userIDs, nil
}

func (w *RecalculateWorker) hasActiveSegments(ctx context.Context, userID uuid.UUID) bool {
	var count int64
	if err := w.db.WithContext(ctx).
		Model(&segdomain.Segmentation{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}
