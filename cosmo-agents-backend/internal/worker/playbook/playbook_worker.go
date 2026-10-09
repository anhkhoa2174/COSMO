package playbook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rockship/cosmo-agents-go/internal/contactinfo"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	playbookDomain "github.com/rockship/cosmo-agents-go/internal/domain/playbook"
	agentrepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	playbookRepo "github.com/rockship/cosmo-agents-go/internal/repository/playbook"
	segrepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	playbookService "github.com/rockship/cosmo-agents-go/internal/service/playbook"
	workerpayloads "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

// extractContactEmail extracts email from contact, checking ContactInformation first, then profile JSONB.
func extractContactEmail(contactInformation string, profile base.JSONB) string {
	return contactinfo.Resolve(contactInformation, profile)
}

type Worker struct {
	playbookRepo   *playbookRepo.Repository
	automationRepo *playbookRepo.AutomationRuleRepository
	enrollmentRepo *playbookRepo.EnrollmentRepository
	approvalRepo   *playbookRepo.ApprovalRequestRepository
	contactRepo    *contactrepo.ContactRepository
	segRepo        *segrepo.SegmentationRepository
	scoreRepo      *segrepo.ScoreRepository
	agentRepo      *agentrepo.AgentRepository
	workerClient   *queueworker.Client
	openAI         *ai.OpenAIClient
}

type executionLog struct {
	StagesCompleted []stageLog `json:"stages_completed"`
	Events          []eventLog `json:"events"`
}

type stageLog struct {
	StageID    string    `json:"stage_id"`
	StageOrder int       `json:"stage_order"`
	Status     string    `json:"status"`
	ExecutedAt time.Time `json:"executed_at"`
}

type eventLog struct {
	Type    string    `json:"type"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

// New creates a playbook automation worker.
func New(
	playbookRepo *playbookRepo.Repository,
	automationRepo *playbookRepo.AutomationRuleRepository,
	enrollmentRepo *playbookRepo.EnrollmentRepository,
	approvalRepo *playbookRepo.ApprovalRequestRepository,
	contactRepo *contactrepo.ContactRepository,
	segRepo *segrepo.SegmentationRepository,
	scoreRepo *segrepo.ScoreRepository,
	agentRepo *agentrepo.AgentRepository,
	workerClient *queueworker.Client,
	openAI *ai.OpenAIClient,
) *Worker {
	return &Worker{
		playbookRepo:   playbookRepo,
		automationRepo: automationRepo,
		enrollmentRepo: enrollmentRepo,
		approvalRepo:   approvalRepo,
		contactRepo:    contactRepo,
		segRepo:        segRepo,
		scoreRepo:      scoreRepo,
		agentRepo:      agentRepo,
		workerClient:   workerClient,
		openAI:         openAI,
	}
}

// HandleEvaluateRules auto-enrolls contacts that match active automation rules.
func (w *Worker) HandleEvaluateRules(ctx context.Context, task *asynq.Task) error {
	var payload workerpayloads.EvaluateRulesPayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return err
	}

	logger.Logger.Info().Msg("playbook: evaluate rules started")

	var rules []*playbookDomain.AutomationRule
	if payload.RuleID != nil {
		rule, err := w.automationRepo.GetByID(ctx, *payload.RuleID)
		if err != nil {
			return err
		}
		if rule == nil {
			return nil
		}
		rules = []*playbookDomain.AutomationRule{rule}
	} else {
		activeRules, err := w.automationRepo.ListActive(ctx)
		if err != nil {
			return err
		}
		rules = activeRules
	}

	logger.Logger.Info().Int("rules", len(rules)).Msg("playbook: evaluate rules loaded")

	for _, rule := range rules {
		criteria := v1schema.EnrollmentCriteria{
			FitScoreThreshold:    0,
			RequireHumanApproval: false,
		}
		if len(rule.EnrollmentCriteria) > 0 {
			_ = json.Unmarshal(rule.EnrollmentCriteria, &criteria)
		}
		if criteria.FitScoreThreshold == 0 {
			criteria.FitScoreThreshold = 50
		}

		scores, err := w.scoreRepo.ListBySegmentThreshold(ctx, rule.SegmentID, criteria.FitScoreThreshold)
		if err != nil {
			logger.Logger.Error().Err(err).Str("segment_id", rule.SegmentID.String()).Msg("playbook: failed to list scores")
			continue
		}

		logger.Logger.Info().
			Str("rule_id", rule.AutomationRuleID.String()).
			Str("segment_id", rule.SegmentID.String()).
			Int("eligible_contacts", len(scores)).
			Msg("playbook: evaluate rule")

		for _, score := range scores {
			existing, err := w.enrollmentRepo.GetByContactAndPlaybook(ctx, score.ContactID, rule.PlaybookID)
			if err != nil {
				logger.Logger.Error().Err(err).Str("contact_id", score.ContactID.String()).Msg("playbook: failed to check enrollment")
				continue
			}
			if existing != nil {
				continue
			}

			contact, err := w.contactRepo.GetByID(ctx, score.ContactID)
			if err != nil || contact == nil {
				continue
			}
			if criteria.EngagementScoreThreshold != nil {
				engagement := extractEngagementScore(contact.Profile)
				if engagement < *criteria.EngagementScoreThreshold {
					continue
				}
			}

			if criteria.RequireHumanApproval {
				latest, err := w.approvalRepo.GetLatestByContactRule(ctx, score.ContactID, rule.PlaybookID, rule.AutomationRuleID)
				if err != nil {
					logger.Logger.Error().Err(err).Msg("playbook: failed to check previous approval")
					continue
				}
				if !shouldRequestApproval(latest, score.FitScore) {
					continue
				}

				req := &playbookDomain.EnrollmentApprovalRequest{
					ContactID:        score.ContactID,
					PlaybookID:       rule.PlaybookID,
					AutomationRuleID: rule.AutomationRuleID,
					Reason:           fmt.Sprintf("Auto-enroll via rule %s", rule.Name),
					FitScore:         score.FitScore,
					EngagementScore:  extractEngagementScore(contact.Profile),
					Status:           "pending",
				}
				if err := w.approvalRepo.Create(ctx, req); err != nil {
					logger.Logger.Error().Err(err).Msg("playbook: failed to create approval request")
				}
				continue
			}

			if err := w.createEnrollment(ctx, score.ContactID, rule.PlaybookID, &rule.AutomationRuleID); err != nil {
				logger.Logger.Error().Err(err).Msg("playbook: failed to create enrollment")
			} else {
				logger.Logger.Info().
					Str("contact_id", score.ContactID.String()).
					Str("playbook_id", rule.PlaybookID.String()).
					Msg("playbook: enrollment created")
			}
		}
	}

	logger.Logger.Info().Msg("playbook: evaluate rules finished")
	return nil
}

// HandleProcessEnrollments executes playbook stages for active enrollments.
func (w *Worker) HandleProcessEnrollments(ctx context.Context, task *asynq.Task) error {
	var payload workerpayloads.ProcessEnrollmentsPayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return err
	}

	logger.Logger.Info().Msg("playbook: process enrollments started")

	var enrollments []*playbookDomain.ContactEnrollment
	if payload.EnrollmentID != nil {
		enrollment, err := w.enrollmentRepo.GetByID(ctx, *payload.EnrollmentID)
		if err != nil || enrollment == nil {
			return err
		}
		enrollments = []*playbookDomain.ContactEnrollment{enrollment}
	} else {
		activeEnrollments, err := w.enrollmentRepo.ListByStatus(ctx, "active")
		if err != nil {
			return err
		}
		enrollments = activeEnrollments
	}

	logger.Logger.Info().Int("enrollments", len(enrollments)).Msg("playbook: process enrollments loaded")

	for _, enrollment := range enrollments {
		if enrollment.EnrollmentStatus != "active" {
			continue
		}

		pb, err := w.playbookRepo.GetByID(ctx, enrollment.PlaybookID)
		if err != nil || pb == nil {
			continue
		}

		var config v1schema.PlaybookConfig
		if err := json.Unmarshal(pb.Config, &config); err != nil || len(config.Stages) == 0 {
			continue
		}

		stage := findStage(config.Stages, enrollment.CurrentStageID, enrollment.CurrentStageOrder)
		if stage == nil {
			continue
		}

		log := parseExecutionLog(enrollment.ExecutionLog)
		if stageCompleted(log, stage.ID) {
			if err := w.advanceEnrollment(ctx, enrollment, config, *stage, log, stage.SuccessCriteria.OnTimeout); err != nil {
				logger.Logger.Error().Err(err).Msg("playbook: failed to advance enrollment")
			}
			continue
		}

		if !stageReady(*stage, enrollment, log) {
			continue
		}

		if err := w.executeStage(ctx, enrollment, *stage, &log); err != nil {
			logger.Logger.Error().Err(err).Msg("playbook: failed to execute stage")
			continue
		}

		if err := w.saveExecutionLog(ctx, enrollment.EnrollmentID, log); err != nil {
			logger.Logger.Error().Err(err).Msg("playbook: failed to save execution log")
		}

		if err := w.advanceEnrollment(ctx, enrollment, config, *stage, log, stage.SuccessCriteria.OnTimeout); err != nil {
			logger.Logger.Error().Err(err).Msg("playbook: failed to advance enrollment")
		}
	}

	logger.Logger.Info().Msg("playbook: process enrollments finished")
	return nil
}

func (w *Worker) createEnrollment(ctx context.Context, contactID, playbookID uuid.UUID, ruleID *uuid.UUID) error {
	pb, err := w.playbookRepo.GetByID(ctx, playbookID)
	if err != nil || pb == nil {
		return fmt.Errorf("playbook not found")
	}

	var config v1schema.PlaybookConfig
	if err := json.Unmarshal(pb.Config, &config); err != nil {
		return fmt.Errorf("invalid playbook config")
	}
	if len(config.Stages) == 0 {
		return fmt.Errorf("playbook has no stages")
	}

	firstStage := config.Stages[0]
	logBytes, _ := json.Marshal(executionLog{
		StagesCompleted: []stageLog{},
		Events:          []eventLog{},
	})

	now := time.Now()
	enrollment := &playbookDomain.ContactEnrollment{
		ContactID:         contactID,
		PlaybookID:        playbookID,
		AutomationRuleID:  ruleID,
		EnrollmentStatus:  "active",
		CurrentStageOrder: firstStage.Order,
		CurrentStageID:    firstStage.ID,
		EnrolledAt:        &now,
		ExecutionLog:      base.JSONB(logBytes),
	}

	return w.enrollmentRepo.Create(ctx, enrollment)
}

func (w *Worker) executeStage(ctx context.Context, enrollment *playbookDomain.ContactEnrollment, stage v1schema.PlaybookStage, log *executionLog) error {
	switch stage.Type {
	case v1schema.StageTypeEmail:
		return w.executeEmailStage(ctx, enrollment, stage, log)
	case v1schema.StageTypeLinkedIn:
		log.Events = append(log.Events, eventLog{
			Type:    "linkedin",
			Message: "LinkedIn stage scheduled",
			At:      time.Now(),
		})
	case v1schema.StageTypeCall:
		log.Events = append(log.Events, eventLog{
			Type:    "call",
			Message: "Call task scheduled",
			At:      time.Now(),
		})
	case v1schema.StageTypeWait:
		// Wait stage just advances after wait duration.
	}

	log.StagesCompleted = append(log.StagesCompleted, stageLog{
		StageID:    stage.ID,
		StageOrder: stage.Order,
		Status:     "completed",
		ExecutedAt: time.Now(),
	})
	return nil
}

func (w *Worker) executeEmailStage(ctx context.Context, enrollment *playbookDomain.ContactEnrollment, stage v1schema.PlaybookStage, log *executionLog) error {
	contact, err := w.contactRepo.GetByID(ctx, enrollment.ContactID)
	if err != nil || contact == nil {
		return fmt.Errorf("contact not found")
	}

	// Tôn trọng opt-out: contact đã DO_NOT_CONTACT thì không gửi gì nữa
	if contact.DoNotContact {
		log.Events = append(log.Events, eventLog{
			Type:    "email",
			Message: "Contact is marked do-not-contact; skipping send",
			At:      time.Now(),
		})
		return nil
	}

	agentID, err := w.findDefaultAgentID(ctx, contact.UserID)
	if err != nil {
		log.Events = append(log.Events, eventLog{
			Type:    "email",
			Message: "No active agent found for user",
			At:      time.Now(),
		})
		return nil
	}

	subject, body := buildStageContent(stage, contact.Name, contact.Company, contact.JobTitle)
	if stage.ContentConfig != nil && strings.TrimSpace(stage.ContentConfig.AIGenerationPrompt) != "" {
		generated, err := w.generateContent(ctx, enrollment.PlaybookID, enrollment.ContactID, stage.ID, stage.ContentConfig.AIGenerationPrompt)
		if err == nil {
			subject = generated.Subject
			body = generated.Body
		}
	}

	contactEmail := extractContactEmail(contact.ContactInformation, contact.Profile)
	payload := workerpayloads.SendEmailPayload{
		AgentID:   agentID,
		ContactID: contact.ID,
		To:        contactEmail,
		Subject:   subject,
		Body:      body,
		IsHTML:    false,
	}

	if w.workerClient != nil {
		if _, err := w.workerClient.EnqueueTask(ctx, queueworker.TypeSendEmail, payload); err != nil {
			return err
		}
	}

	log.Events = append(log.Events, eventLog{
		Type:    "email",
		Message: fmt.Sprintf("Email queued to %s", contactEmail),
		At:      time.Now(),
	})
	return nil
}

func (w *Worker) generateContent(ctx context.Context, playbookID, contactID uuid.UUID, stageID, prompt string) (*v1schema.GenerateContentResponse, error) {
	service := playbookService.NewService(
		w.playbookRepo,
		w.automationRepo,
		w.enrollmentRepo,
		w.approvalRepo,
		w.contactRepo,
		w.segRepo,
		w.openAI,
	)

	return service.GenerateContent(ctx, playbookID, contactID, stageID, prompt)
}

func (w *Worker) findDefaultAgentID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	agents, err := w.agentRepo.GetByUserID(ctx, userID.String())
	if err != nil {
		return uuid.Nil, err
	}
	for _, agent := range agents {
		if agent.Status == "active" {
			return agent.ID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("no active agent found")
}

func (w *Worker) saveExecutionLog(ctx context.Context, enrollmentID uuid.UUID, log executionLog) error {
	logBytes, err := json.Marshal(log)
	if err != nil {
		return err
	}
	return w.enrollmentRepo.UpdateExecutionLog(ctx, enrollmentID, logBytes)
}

func (w *Worker) advanceEnrollment(
	ctx context.Context,
	enrollment *playbookDomain.ContactEnrollment,
	config v1schema.PlaybookConfig,
	stage v1schema.PlaybookStage,
	log executionLog,
	action v1schema.SuccessAction,
) error {
	if action == "" {
		action = v1schema.SuccessActionAdvance
	}
	switch action {
	case v1schema.SuccessActionPause:
		return w.enrollmentRepo.UpdateStatus(ctx, enrollment.EnrollmentID, "paused")
	case v1schema.SuccessActionComplete:
		if err := w.enrollmentRepo.UpdateStatus(ctx, enrollment.EnrollmentID, "completed"); err != nil {
			return err
		}
		return w.enrollmentRepo.UpdateCompletedAt(ctx, enrollment.EnrollmentID)
	}

	nextStage := nextStageByOrder(config.Stages, stage.Order)
	if nextStage == nil {
		if err := w.enrollmentRepo.UpdateStatus(ctx, enrollment.EnrollmentID, "completed"); err != nil {
			return err
		}
		return w.enrollmentRepo.UpdateCompletedAt(ctx, enrollment.EnrollmentID)
	}

	return w.enrollmentRepo.UpdateStage(ctx, enrollment.EnrollmentID, nextStage.Order, nextStage.ID)
}

func parseExecutionLog(raw base.JSONB) executionLog {
	if len(raw) == 0 {
		return executionLog{StagesCompleted: []stageLog{}, Events: []eventLog{}}
	}
	var log executionLog
	if err := json.Unmarshal(raw, &log); err != nil {
		return executionLog{StagesCompleted: []stageLog{}, Events: []eventLog{}}
	}
	if log.StagesCompleted == nil {
		log.StagesCompleted = []stageLog{}
	}
	if log.Events == nil {
		log.Events = []eventLog{}
	}
	return log
}

func stageCompleted(log executionLog, stageID string) bool {
	for _, entry := range log.StagesCompleted {
		if entry.StageID == stageID {
			return true
		}
	}
	return false
}

func stageReady(stage v1schema.PlaybookStage, enrollment *playbookDomain.ContactEnrollment, log executionLog) bool {
	waitDays := stage.TriggerConditions.WaitDuration
	if waitDays <= 0 {
		return true
	}
	ref := time.Now()
	if len(log.StagesCompleted) > 0 {
		ref = log.StagesCompleted[len(log.StagesCompleted)-1].ExecutedAt
	} else if enrollment.EnrolledAt != nil {
		ref = *enrollment.EnrolledAt
	} else {
		ref = enrollment.CreatedAt
	}
	return time.Since(ref) >= time.Duration(waitDays)*24*time.Hour
}

func findStage(stages []v1schema.PlaybookStage, stageID string, stageOrder int) *v1schema.PlaybookStage {
	for _, stage := range stages {
		if stageID != "" && stage.ID == stageID {
			stageCopy := stage
			return &stageCopy
		}
	}
	for _, stage := range stages {
		if stage.Order == stageOrder {
			stageCopy := stage
			return &stageCopy
		}
	}
	return nil
}

// nextStageByOrder returns the stage with the lowest order above currentOrder.
// Orders need not be consecutive: looking only for currentOrder+1 ended the
// playbook early at the first gap (orders 1, 3 never reached stage 3).
func nextStageByOrder(stages []v1schema.PlaybookStage, currentOrder int) *v1schema.PlaybookStage {
	var next *v1schema.PlaybookStage
	for _, stage := range stages {
		if stage.Order > currentOrder && (next == nil || stage.Order < next.Order) {
			stageCopy := stage
			next = &stageCopy
		}
	}
	return next
}

func buildStageContent(stage v1schema.PlaybookStage, name, company, jobTitle string) (string, string) {
	// Extract first name from full name for greeting
	firstName := name
	if parts := strings.Fields(name); len(parts) > 0 {
		firstName = parts[0]
	}

	subject := fmt.Sprintf("Quick intro, %s", firstName)
	body := ""
	if stage.ContentConfig != nil {
		body = stage.ContentConfig.Template
	}
	if strings.TrimSpace(body) == "" {
		body = "Hi {{name}},\n\nWanted to reach out briefly.\n\nBest,\n"
	}

	replacer := strings.NewReplacer(
		"{{name}}", name,
		"{{first_name}}", firstName,
		"{{company}}", company,
		"{{job_title}}", jobTitle,
	)
	return subject, replacer.Replace(body)
}

func extractEngagementScore(profileRaw base.JSONB) int {
	if len(profileRaw) == 0 {
		return 0
	}
	var profile map[string]interface{}
	if err := json.Unmarshal(profileRaw, &profile); err != nil {
		return 0
	}
	scores, ok := profile["scores"].(map[string]interface{})
	if !ok {
		return 0
	}
	val, ok := scores["engagement"]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

// shouldRequestApproval decides whether a rule may ask for approval to enroll
// a contact, given the contact's latest request under that rule.
//
// A pending request is still waiting for a person, so asking again would only
// duplicate it. A rejected one is reconsidered only once the contact's fit
// score has risen above the score it was rejected at: the rule re-runs every
// few minutes, and without this a contact a person turned down would be put
// back in front of them on the next run with nothing about it changed.
func shouldRequestApproval(latest *playbookDomain.EnrollmentApprovalRequest, fitScore int) bool {
	if latest == nil {
		return true
	}
	switch latest.Status {
	case "pending":
		return false
	case "rejected":
		return fitScore > latest.FitScore
	default:
		return true
	}
}
