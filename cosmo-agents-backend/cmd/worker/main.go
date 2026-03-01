package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/agents"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentrepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	integrationRepo "github.com/rockship/cosmo-agents-go/internal/repository/integration"
	interactionRepo "github.com/rockship/cosmo-agents-go/internal/repository/interaction"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	operationRepo "github.com/rockship/cosmo-agents-go/internal/repository/operation"
	organizationRepo "github.com/rockship/cosmo-agents-go/internal/repository/organization"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	playbookRepo "github.com/rockship/cosmo-agents-go/internal/repository/playbook"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	taskrepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	templateRepo "github.com/rockship/cosmo-agents-go/internal/repository/template"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	hubspotService "github.com/rockship/cosmo-agents-go/internal/service/hubspot"
	intelligenceService "github.com/rockship/cosmo-agents-go/internal/service/intelligence"
	mailService "github.com/rockship/cosmo-agents-go/internal/service/mail"
	summaryService "github.com/rockship/cosmo-agents-go/internal/service/summary"
	aiworker "github.com/rockship/cosmo-agents-go/internal/worker/ai"
	campaignworker "github.com/rockship/cosmo-agents-go/internal/worker/campaign"
	contactworker "github.com/rockship/cosmo-agents-go/internal/worker/contact"
	emailworker "github.com/rockship/cosmo-agents-go/internal/worker/email"
	gmailnotificationworker "github.com/rockship/cosmo-agents-go/internal/worker/gmailnotification"
	knowledgeworker "github.com/rockship/cosmo-agents-go/internal/worker/knowledge"
	mailwriterworker "github.com/rockship/cosmo-agents-go/internal/worker/mailwriter"
	orchestratorworker "github.com/rockship/cosmo-agents-go/internal/worker/orchestrator"
	outreachworker "github.com/rockship/cosmo-agents-go/internal/worker/outreach"
	playbookworker "github.com/rockship/cosmo-agents-go/internal/worker/playbook"
	relationshipworker "github.com/rockship/cosmo-agents-go/internal/worker/relationship"
	segmentationworker "github.com/rockship/cosmo-agents-go/internal/worker/segmentation"
	summarizerworker "github.com/rockship/cosmo-agents-go/internal/worker/summarizer"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/config"
	"github.com/rockship/cosmo-agents-go/pkg/database"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
	redisutil "github.com/rockship/cosmo-agents-go/pkg/redis"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// Placeholder implementations to satisfy email worker interfaces.
type googleAuthPlaceholder struct{}

func (g *googleAuthPlaceholder) Refresh(ctx context.Context, credentials map[string]interface{}, scopes []string) (map[string]interface{}, error) {
	logger.Logger.Warn().Msg("Using placeholder GoogleAuthService - no token refresh performed")
	return credentials, nil
}

type gmailPlaceholder struct{}

func (g *gmailPlaceholder) SendEmail(ctx context.Context, sender string, to []string, subject string, body map[string]string) (*emailworker.GmailMessage, error) {
	logger.Logger.Warn().
		Str("sender", sender).
		Strs("to", to).
		Str("subject", subject).
		Msg("Using placeholder GmailService - email not sent")
	return &emailworker.GmailMessage{
		ID:       "placeholder-id",
		ThreadID: "placeholder-thread-id",
	}, nil
}

type taskMonitorPlaceholder struct{}

func (t *taskMonitorPlaceholder) RespondToTask(ctx context.Context, taskIDs []string) error {
	logger.Logger.Warn().Strs("task_ids", taskIDs).Msg("Using placeholder TaskMonitor - RespondToTask noop")
	return nil
}

func (t *taskMonitorPlaceholder) ResolveTask(ctx context.Context, taskIDs []string) error {
	logger.Logger.Warn().Strs("task_ids", taskIDs).Msg("Using placeholder TaskMonitor - ResolveTask noop")
	return nil
}

// listContactRepoAdapter bridges the worker interface with the ListContactRepository.
type listContactRepoAdapter struct {
	*contactrepo.ListContactRepository
}

func (a *listContactRepoAdapter) AddContactsToList(ctx context.Context, listID uuid.UUID, contacts []*domain.Contact) error {
	ids := make([]uuid.UUID, 0, len(contacts))
	for _, c := range contacts {
		ids = append(ids, c.ID)
	}
	return a.ListContactRepository.AddContacts(ctx, listID, ids)
}

// integrationRepoAdapter adds the missing FindByHubID for the worker.
type integrationRepoAdapter struct {
	*integrationRepo.IntegrationRepository
}

func (r *integrationRepoAdapter) FindByHubID(ctx context.Context, hubID string) (*domain.Integration, error) {
	var integration domain.Integration
	err := r.GetDB().WithContext(ctx).
		Where("source_id = ?", hubID).
		First(&integration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &integration, nil
}

func main() {
	// Load .env file
	_ = godotenv.Load()

	// Load configuration
	cfg := config.MustLoad()

	// Initialize logger
	logger.Init(logger.Config{
		Level:       cfg.Logger.Level,
		Environment: cfg.App.Environment,
		Output:      os.Stdout,
	})

	logger.Info("Starting Cosmo Agents Worker Server")
	logger.Logger.Info().
		Str("environment", cfg.App.Environment).
		Msg("Configuration loaded")

	// Connect to PostgreSQL
	db, err := database.NewPostgresDB(database.PostgresConfig{
		DSN:         cfg.Database.PostgresURI,
		MaxOpenConn: cfg.Database.MaxOpenConn,
		MaxIdleConn: cfg.Database.MaxIdleConn,
		MaxLifetime: 1 * time.Hour,
	})
	if err != nil {
		logger.Fatal("Failed to connect to database")
	}
	defer database.Close(db)

	// Parse Redis URL using shared utility
	redisCfg := redisutil.ParseRedisURL(cfg.Redis.URL, cfg.Redis.Password)

	// Initialize session
	session := core.NewGormSession(db, nil)

	// Create worker server
	workerCfg := worker.Config{
		RedisAddr:     redisCfg.Addr,
		RedisPassword: redisCfg.Password,
		RedisDB:       redisCfg.DB,
		Concurrency:   10, // 10 concurrent workers
	}

	workerServer := worker.NewServer(workerCfg)

	// Create worker client for enqueueing sub-tasks
	workerClient := worker.NewClient(workerCfg)
	defer workerClient.Close()

	// Initialize OAuth2 client
	oauth2Client := googleoauth.NewClient(googleoauth.Config{
		ClientID:     cfg.OAuth.GoogleClientID,
		ClientSecret: cfg.OAuth.GoogleClientSecret,
		RedirectURI:  cfg.OAuth.GoogleRedirectURI,
	})

	// Initialize OpenAI client
	openaiClient := ai.NewOpenAIClient(ai.Config{
		APIKey: cfg.AI.OpenAIAPIKey,
		Model:  cfg.AI.OpenAIModel,
	})

	// Register task handlers
	registerHandlers(workerServer, db, session, workerClient, oauth2Client, openaiClient, cfg)

	// Start playbook automation schedulers
	playbookCancel := startPlaybookSchedulers(workerClient)
	defer playbookCancel()

	// Start worker server
	if err := workerServer.Start(); err != nil {
		logger.Fatal(fmt.Sprintf("Failed to start worker server: %v", err))
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker server...")
	playbookCancel()
	workerServer.Shutdown()
	logger.Info("Worker server stopped")
}

func getSqlxDB(gormDB *gorm.DB) (*sqlx.DB, error) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(sqlDB, "postgres"), nil
}

func startPlaybookSchedulers(workerClient *worker.Client) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	runTicker := func(interval time.Duration, taskType string) {
		ticker := time.NewTicker(interval)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_, _ = workerClient.EnqueueTask(ctx, taskType, map[string]interface{}{}, asynq.Unique(30*time.Second))
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	runTicker(5*time.Minute, worker.TypePlaybookEvaluateRules)
	runTicker(1*time.Minute, worker.TypePlaybookProcessEnrollments)
	runTicker(2*time.Hour, worker.TypeRecalculateSegmentScores)
	runTicker(24*time.Hour, worker.TypeRecalculateRelationshipScores)
	runTicker(1*time.Minute, worker.TypeRecalculateOutreachNextStep) // Run every 1 minute for testing (normally every hour)
	return cancel
}

// registerHandlers registers all task handlers with the worker server.
func registerHandlers(
	srv *worker.Server,
	db *gorm.DB,
	session core.Session,
	workerClient *worker.Client,
	oauth2Client *googleoauth.Client,
	openaiClient *ai.OpenAIClient,
	cfg *config.Config,
) {
	sqlxDB, err := getSqlxDB(db)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to create sqlx.DB for playbook worker")
	}

	// Initialize repositories
	agentRepoWithMetrics := agentrepo.NewAgentRepository(db)
	emailRepository := emailRepo.NewEmailRepository(db)
	taskRepository := taskrepo.NewTaskRepository(db)
	contactRepository := contactrepo.NewContactRepository(db)
	listContactRepo := &listContactRepoAdapter{ListContactRepository: contactrepo.NewListContactRepository(db)}
	conversationRepoInstance := conversationRepo.NewConversationRepository(db)
	interactionRepoInstance := interactionRepo.NewRepository(db)
	campaignRepository := campaignRepo.NewCampaignRepository(db)
	templateRepoInstance := templateRepo.NewTemplateRepository(db)
	knowledgeRepository := knowledgeRepo.NewKnowledgeRepository(db)
	userRepoInstance := userRepo.NewUserRepository(db)
	orgRepo := organizationRepo.NewOrganizationRepository(db)
	operationRepository := operationRepo.NewOperationRepository(db)
	integrationRepository := &integrationRepoAdapter{IntegrationRepository: integrationRepo.NewIntegrationRepository(db)}
	segmentationRepository := segRepo.NewSegmentationRepository(db)
	segmentationScoreRepo := segRepo.NewScoreRepository(db)

	// Outreach repositories
	outreachInteractionRepo := outreachRepo.NewInteractionLogRepository(db)
	outreachStateRepo := outreachRepo.NewOutreachStateRepository(db)
	outreachMeetingRepo := outreachRepo.NewMeetingRepository(db)
	outreachFeedbackRepo := outreachRepo.NewFeedbackRepository(db)

	playbookRepository := playbookRepo.NewRepository(sqlxDB)
	automationRuleRepo := playbookRepo.NewAutomationRuleRepository(sqlxDB)
	enrollmentRepo := playbookRepo.NewEnrollmentRepository(sqlxDB)
	approvalRepo := playbookRepo.NewApprovalRequestRepository(sqlxDB)

	// Initialize services
	hubspotAPI := hubspotService.NewHubspotAPI(cfg.Hubspot)
	intelService := intelligenceService.NewService(
		contactRepository,
		interactionRepoInstance,
		segmentationRepository,
		segmentationScoreRepo,
		openaiClient,
		nil,
		nil,
	)

	// Initialize workers
	agentEmailWorker := emailworker.NewEmailWorker(
		db,
		&googleAuthPlaceholder{},
		&gmailPlaceholder{},
		&taskMonitorPlaceholder{},
		agentRepoWithMetrics,
		campaignRepository,
		taskRepository,
		emailRepository,
		conversationRepoInstance,
	)

	generalEmailWorker := emailworker.NewGeneralEmailWorker(
		db,
		oauth2Client,
		agentRepoWithMetrics,
		emailRepository,
		taskRepository,
		contactRepository,
		conversationRepoInstance,
		orgRepo,
	)

	campaignWorker := campaignworker.New(
		db,
		workerClient,
		campaignRepository,
		contactRepository,
		templateRepoInstance,
		taskRepository,
		agentRepoWithMetrics,
	)

	aiWorker := aiworker.New(
		db,
		workerClient,
		openaiClient,
		campaignRepository,
		contactRepository,
		templateRepoInstance,
		taskRepository,
		knowledgeRepository,
	)

	// Initialize OpenAI client for new services (uses openai-go library)
	openaiServiceClient := openai.NewClient(option.WithAPIKey(cfg.AI.OpenAIAPIKey))

	// Initialize AI services for new workers
	summarizer := summaryService.NewSummarizer(&openaiServiceClient, cfg.AI.OpenAIModel, &logger.Logger)

	// For MailWriter, we'll use default parameters - can be customized per campaign
	mailWriter := mailService.NewMailWriter(
		&openaiServiceClient,
		cfg.AI.OpenAIModel,
		mailService.EmailParameters{
			Sender: mailService.SenderInfo{
				CompanyDescription:      "Default company description",
				CompanyTargetingPersona: "Default targeting persona",
			},
			CampaignType: "outreach",
			Tone:         "Zero marketing jargon, write the way you speak (conversational) Not overly formal",
		},
		&logger.Logger,
	)

	// Initialize new workers
	summarizerWorker := summarizerworker.New(
		emailRepository,
		conversationRepoInstance,
		summarizer,
		&logger.Logger,
	)

	mailWriterWorker := mailwriterworker.New(
		campaignRepository,
		contactRepository,
		conversationRepoInstance,
		emailRepository,
		templateRepoInstance,
		userRepoInstance,
		orgRepo,
		mailWriter,
		&logger.Logger,
	)

	gmailNotificationWorker := gmailnotificationworker.New(
		agentRepoWithMetrics,
		conversationRepoInstance,
		emailRepository,
		contactRepository,
		taskRepository,
		oauth2Client,
		workerClient,
	)

	contactWorker := contactworker.New(
		db,
		session,
		nil, // Don't see where Enrich function is used, please ignore it, will remove in refactor phase
		hubspotAPI,
		contactRepository,
		listContactRepo,
		operationRepository,
		userRepoInstance,
		integrationRepository,
	)

	playbookWorker := playbookworker.New(
		playbookRepository,
		automationRuleRepo,
		enrollmentRepo,
		approvalRepo,
		contactRepository,
		segmentationRepository,
		segmentationScoreRepo,
		agentRepoWithMetrics,
		workerClient,
		openaiClient,
	)

	segmentationWorker := segmentationworker.NewRecalculateWorker(
		db,
		intelService,
		contactRepository,
	)

	enrichmentAgent := agents.NewContactEnrichmentAgent(intelService)
	segmentAgent := agents.NewSegmentCalculatorAgent(intelService)
	relationshipAgent := agents.NewRelationshipScorerAgent(contactRepository, interactionRepoInstance)
	relationshipWorker := relationshipworker.NewRecalculateWorker(contactRepository, relationshipAgent)
	outreachRecalculateWorker := outreachworker.NewRecalculateWorker(
		db,
		contactRepository,
		outreachInteractionRepo,
		outreachMeetingRepo,
		outreachStateRepo,
		outreachFeedbackRepo,
		nil, // Use default config
	)
	orchestratorWorker := orchestratorworker.NewContactOrchestrator(
		contactRepository,
		enrichmentAgent,
		segmentAgent,
		relationshipAgent,
		workerClient,
	)

	// Register email task handlers
	srv.HandleFunc(emailworker.TypeAgentSendEmail, agentEmailWorker.HandleAgentSendEmail)
	srv.HandleFunc(emailworker.TypeAgentDailyReset, agentEmailWorker.HandleAgentDailyReset)
	srv.HandleFunc(emailworker.TypeSendEmail, generalEmailWorker.HandleSendEmail)
	srv.HandleFunc(emailworker.TypeSyncGmailHistory, generalEmailWorker.HandleSyncGmailHistory)
	srv.HandleFunc(emailworker.TypeSendInviteMemberEmail, generalEmailWorker.SendInviteMemberEmail)
	// NOTE: TypeProcessIncoming handler was removed in refactor.
	// Register Gmail notification handler.
	srv.HandleFunc(worker.TypeGmailNotification, gmailNotificationWorker.ProcessTask)

	// Register campaign task handlers
	srv.HandleFunc(worker.TypeExecuteCampaign, campaignWorker.HandleExecuteCampaign)
	srv.HandleFunc(worker.TypeScheduleTasks, campaignWorker.HandleScheduleTasks)

	// Register AI task handlers
	srv.HandleFunc(worker.TypeGenerateEmail, aiWorker.HandleGenerateEmail)
	srv.HandleFunc(worker.TypeGenerateEmbedding, aiWorker.HandleGenerateEmbedding)

	// Register summarizer task handlers
	srv.HandleFunc(summarizerworker.TypeSummarizeEmail, summarizerWorker.ProcessTask)
	srv.HandleFunc(summarizerworker.TypeSummarizeConversation, summarizerWorker.ProcessTask)

	// Register mail writer task handlers
	srv.HandleFunc(mailwriterworker.TypeGenerateCampaignEmail, mailWriterWorker.ProcessTask)
	srv.HandleFunc(mailwriterworker.TypeGenerateReplyEmail, mailWriterWorker.ProcessTask)
	srv.HandleFunc(mailwriterworker.TypeEmailIndexing, mailWriterWorker.ProcessTask)

	// Register contact task handlers
	srv.HandleFunc(contactworker.TypeContactImportHubspot, contactWorker.HandleContactImportHubspot)

	// Register playbook automation handlers
	srv.HandleFunc(worker.TypePlaybookEvaluateRules, playbookWorker.HandleEvaluateRules)
	srv.HandleFunc(worker.TypePlaybookProcessEnrollments, playbookWorker.HandleProcessEnrollments)
	srv.HandleFunc(worker.TypeRecalculateSegmentScores, segmentationWorker.HandleRecalculateScores)
	srv.HandleFunc(worker.TypeRecalculateRelationshipScores, relationshipWorker.HandleRecalculateScores)
	srv.HandleFunc(worker.TypeRecalculateOutreachNextStep, outreachRecalculateWorker.HandleRecalculateNextStep)
	srv.HandleFunc(worker.TypeOrchestrateContact, orchestratorWorker.HandleContactOrchestration)

	// Initialize knowledge worker
	knowledgeWorker := knowledgeworker.New(
		db,
		nil, // VectorStore - not implemented yet
		summarizer,
	)

	// Register knowledge task handlers
	srv.HandleFunc(knowledgeworker.TypeKnowledgeIndexing, knowledgeWorker.HandleKnowledgeIndexing)
	srv.HandleFunc(knowledgeworker.TypeKnowledgesPruning, knowledgeWorker.HandleKnowledgesPruning)
	srv.HandleFunc(knowledgeworker.TypeKnowledgesSummarize, knowledgeWorker.HandleKnowledgesSummarize)

	logger.Logger.Info().Msg("All task handlers registered")
}
