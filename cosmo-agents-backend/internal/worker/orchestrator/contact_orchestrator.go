package orchestrator

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/agents"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	workerdto "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

const orchestratorBatchSize = 200

// ContactOrchestrator chains agent tasks for contact events.
type ContactOrchestrator struct {
	contactRepo       *contactrepo.ContactRepository
	enrichmentAgent   *agents.ContactEnrichmentAgent
	segmentAgent      *agents.SegmentCalculatorAgent
	relationshipAgent *agents.RelationshipScorerAgent
	workerClient      *worker.Client
}

func NewContactOrchestrator(
	contactRepo *contactrepo.ContactRepository,
	enrichmentAgent *agents.ContactEnrichmentAgent,
	segmentAgent *agents.SegmentCalculatorAgent,
	relationshipAgent *agents.RelationshipScorerAgent,
	workerClient *worker.Client,
) *ContactOrchestrator {
	return &ContactOrchestrator{
		contactRepo:       contactRepo,
		enrichmentAgent:   enrichmentAgent,
		segmentAgent:      segmentAgent,
		relationshipAgent: relationshipAgent,
		workerClient:      workerClient,
	}
}

// HandleContactOrchestration chains: enrich -> score -> enroll (via playbook eval).
func (o *ContactOrchestrator) HandleContactOrchestration(ctx context.Context, task *asynq.Task) error {
	var payload workerdto.ContactOrchestrationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		logger.Logger.Error().Err(err).Msg("orchestrator: invalid payload")
		return err
	}

	contactModel, err := o.contactRepo.GetByID(ctx, payload.ContactID)
	if err != nil || contactModel == nil {
		return err
	}

	shouldEnrich := payload.ForceRefresh || isEmptyJSONB(contactModel.AIInsights)
	if shouldEnrich && o.enrichmentAgent != nil {
		if _, err := o.enrichmentAgent.Run(ctx, payload.UserID, payload.OrgID, payload.ContactID, payload.ForceRefresh); err != nil {
			logger.Logger.Warn().Err(err).Str("contact_id", payload.ContactID.String()).Msg("orchestrator: enrich failed")
		}
	}

	if o.segmentAgent != nil {
		if _, err := o.segmentAgent.Run(ctx, payload.UserID, payload.OrgID, payload.ContactID, nil); err != nil {
			logger.Logger.Warn().Err(err).Str("contact_id", payload.ContactID.String()).Msg("orchestrator: score failed")
		}
	}

	if o.relationshipAgent != nil {
		switch payload.Event {
		case "interaction_logged", "contact_updated", "contact_created":
			if _, err := o.relationshipAgent.Run(ctx, payload.UserID, payload.OrgID, payload.ContactID); err != nil {
				logger.Logger.Warn().Err(err).Str("contact_id", payload.ContactID.String()).Msg("orchestrator: relationship score failed")
			}
		}
	}

	if o.workerClient != nil {
		_, _ = o.workerClient.EnqueueTask(ctx, worker.TypePlaybookEvaluateRules, map[string]interface{}{})
	}

	logger.Logger.Info().
		Str("contact_id", payload.ContactID.String()).
		Str("event", payload.Event).
		Msg("orchestrator: contact chain completed")

	return nil
}

// HandleRelationshipNightly recalculates relationship scores for all contacts.
func (o *ContactOrchestrator) HandleRelationshipNightly(ctx context.Context, task *asynq.Task) error {
	if o.relationshipAgent == nil {
		return nil
	}

	offset := 0
	for {
		contacts, total, err := o.contactRepo.FindAllWithPagination(ctx, offset, orchestratorBatchSize)
		if err != nil {
			return err
		}
		if len(contacts) == 0 {
			break
		}
		for _, contact := range contacts {
			if contact == nil {
				continue
			}
			orgID := uuid.Nil
			if contact.OrganizationID != nil {
				orgID = *contact.OrganizationID
			}
			_, _ = o.relationshipAgent.Run(ctx, contact.UserID, orgID, contact.ID)
		}
		offset += len(contacts)
		if offset >= total {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	logger.Logger.Info().Msg("orchestrator: relationship nightly completed")
	return nil
}

func isEmptyJSONB(jb domain.JSONB) bool {
	if len(jb) == 0 {
		return true
	}
	var m map[string]interface{}
	if err := jb.Unmarshal(&m); err != nil {
		return true
	}
	return len(m) == 0
}
