package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"gorm.io/gorm"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/handler/v1/agent"
	v1ai "github.com/rockship/cosmo-agents-go/internal/handler/v1/ai"
	v1auth "github.com/rockship/cosmo-agents-go/internal/handler/v1/auth"
	v1campaign "github.com/rockship/cosmo-agents-go/internal/handler/v1/campaign"
	v1contact "github.com/rockship/cosmo-agents-go/internal/handler/v1/contact"
	v1conversation "github.com/rockship/cosmo-agents-go/internal/handler/v1/conversation"
	v1customfield "github.com/rockship/cosmo-agents-go/internal/handler/v1/custom-field"
	v1email "github.com/rockship/cosmo-agents-go/internal/handler/v1/email"
	v1feedback "github.com/rockship/cosmo-agents-go/internal/handler/v1/feedback"
	v1file "github.com/rockship/cosmo-agents-go/internal/handler/v1/file"
	v1gmail "github.com/rockship/cosmo-agents-go/internal/handler/v1/gmail"
	v1googleads "github.com/rockship/cosmo-agents-go/internal/handler/v1/google-ads"
	v1hubspot "github.com/rockship/cosmo-agents-go/internal/handler/v1/hubspot"
	v1inboundleadform "github.com/rockship/cosmo-agents-go/internal/handler/v1/inbound_lead_form"
	v1intelligence "github.com/rockship/cosmo-agents-go/internal/handler/v1/intelligence"
	v1interaction "github.com/rockship/cosmo-agents-go/internal/handler/v1/interaction"
	v1knowledge "github.com/rockship/cosmo-agents-go/internal/handler/v1/knowledge"
	v1lab "github.com/rockship/cosmo-agents-go/internal/handler/v1/lab"
	v1listcontact "github.com/rockship/cosmo-agents-go/internal/handler/v1/list-contact"
	v1mcp "github.com/rockship/cosmo-agents-go/internal/handler/v1/mcp"
	v1meta "github.com/rockship/cosmo-agents-go/internal/handler/v1/meta"
	v1organization "github.com/rockship/cosmo-agents-go/internal/handler/v1/organization"
	v1outlook "github.com/rockship/cosmo-agents-go/internal/handler/v1/outlook"
	v1outreach "github.com/rockship/cosmo-agents-go/internal/handler/v1/outreach"
	v1playbook "github.com/rockship/cosmo-agents-go/internal/handler/v1/playbook"
	v1pubsub "github.com/rockship/cosmo-agents-go/internal/handler/v1/pubsub"
	v1saleRep "github.com/rockship/cosmo-agents-go/internal/handler/v1/sale_rep"
	v1segmentation "github.com/rockship/cosmo-agents-go/internal/handler/v1/segmentation"
	v1task "github.com/rockship/cosmo-agents-go/internal/handler/v1/task"
	v1template "github.com/rockship/cosmo-agents-go/internal/handler/v1/template"
	v1temporal "github.com/rockship/cosmo-agents-go/internal/handler/v1/temporal"
	v1user "github.com/rockship/cosmo-agents-go/internal/handler/v1/user"
	v1workFlow "github.com/rockship/cosmo-agents-go/internal/handler/v1/workflow"
	v2auth "github.com/rockship/cosmo-agents-go/internal/handler/v2/auth"
	v2campaign "github.com/rockship/cosmo-agents-go/internal/handler/v2/campaign"
	v2contact "github.com/rockship/cosmo-agents-go/internal/handler/v2/contact"
	v2conversation "github.com/rockship/cosmo-agents-go/internal/handler/v2/conversation"
	v2email "github.com/rockship/cosmo-agents-go/internal/handler/v2/email"
	v2gmail "github.com/rockship/cosmo-agents-go/internal/handler/v2/gmail"
	v2hubspot "github.com/rockship/cosmo-agents-go/internal/handler/v2/hubspot"
	v2knowledge "github.com/rockship/cosmo-agents-go/internal/handler/v2/knowledge"
	v2listcontact "github.com/rockship/cosmo-agents-go/internal/handler/v2/list-contact"
	v2organization "github.com/rockship/cosmo-agents-go/internal/handler/v2/organization"
	v2template "github.com/rockship/cosmo-agents-go/internal/handler/v2/template"
	v2user "github.com/rockship/cosmo-agents-go/internal/handler/v2/user"
	v2worker "github.com/rockship/cosmo-agents-go/internal/handler/v2/worker"
	v3handler "github.com/rockship/cosmo-agents-go/internal/handler/v3"
	v3campaign "github.com/rockship/cosmo-agents-go/internal/handler/v3/campaign"
	v3contact "github.com/rockship/cosmo-agents-go/internal/handler/v3/contact"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
	draftTemplateRepo "github.com/rockship/cosmo-agents-go/internal/repository/draft_template"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	facebookTokenRepo "github.com/rockship/cosmo-agents-go/internal/repository/facebook_token"
	feedbackRepo "github.com/rockship/cosmo-agents-go/internal/repository/feedback"
	fileRepo "github.com/rockship/cosmo-agents-go/internal/repository/file"
	gmailRepo "github.com/rockship/cosmo-agents-go/internal/repository/gmail"
	inboundLeadFormRepo "github.com/rockship/cosmo-agents-go/internal/repository/inbound_lead_form"
	integrationRepo "github.com/rockship/cosmo-agents-go/internal/repository/integration"
	interactionRepo "github.com/rockship/cosmo-agents-go/internal/repository/interaction"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	notificationRepo "github.com/rockship/cosmo-agents-go/internal/repository/notification"
	operationRepo "github.com/rockship/cosmo-agents-go/internal/repository/operation"
	organizationRepo "github.com/rockship/cosmo-agents-go/internal/repository/organization"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	personalApiKeyRepo "github.com/rockship/cosmo-agents-go/internal/repository/personal_api_key"
	playbookRepo "github.com/rockship/cosmo-agents-go/internal/repository/playbook"
	pubsubRepo "github.com/rockship/cosmo-agents-go/internal/repository/pubsub"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	saleRepRepo "github.com/rockship/cosmo-agents-go/internal/repository/sale_rep"
	segmentationRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	templateRepo "github.com/rockship/cosmo-agents-go/internal/repository/template"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	workflowRepo "github.com/rockship/cosmo-agents-go/internal/repository/workflow"
	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
	gmailService "github.com/rockship/cosmo-agents-go/internal/service/gmail"
	googleService "github.com/rockship/cosmo-agents-go/internal/service/google"
	hubspotService "github.com/rockship/cosmo-agents-go/internal/service/hubspot"
	inboundLeadFromService "github.com/rockship/cosmo-agents-go/internal/service/inbound_lead_form"
	intelService "github.com/rockship/cosmo-agents-go/internal/service/intelligence"
	intentService "github.com/rockship/cosmo-agents-go/internal/service/intent"
	knowledgeSerive "github.com/rockship/cosmo-agents-go/internal/service/knowledge"
	orgService "github.com/rockship/cosmo-agents-go/internal/service/organization"
	outlookService "github.com/rockship/cosmo-agents-go/internal/service/outlook"
	outreachService "github.com/rockship/cosmo-agents-go/internal/service/outreach"
	playbookService "github.com/rockship/cosmo-agents-go/internal/service/playbook"
	s3Service "github.com/rockship/cosmo-agents-go/internal/service/s3"
	scraperService "github.com/rockship/cosmo-agents-go/internal/service/scraper"
	"github.com/rockship/cosmo-agents-go/internal/skills"
	"github.com/rockship/cosmo-agents-go/internal/temporal/worker"
	aiUsecase "github.com/rockship/cosmo-agents-go/internal/usecase/ai"
	googleAdsUsecase "github.com/rockship/cosmo-agents-go/internal/usecase/google_ads"
	pubsubUsecase "github.com/rockship/cosmo-agents-go/internal/usecase/pubsub"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
	"github.com/rockship/cosmo-agents-go/pkg/cache"
	"github.com/rockship/cosmo-agents-go/pkg/config"
	"github.com/rockship/cosmo-agents-go/pkg/hubspot"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
	pkgworker "github.com/rockship/cosmo-agents-go/pkg/worker"
	temporalClient "go.temporal.io/sdk/client"
)

// Dependencies holds all application dependencies
type Dependencies struct {
	// Database
	DB *gorm.DB

	// Repositories
	Repos *Repositories

	// Services
	Services *Services

	// Use Cases
	UseCases *UseCases

	// Handlers
	V1Handlers *V1Handlers
	V2Handlers *V2Handlers
	V3Handlers *V3Handlers

	// Clients
	OAuth2Client     *googleoauth.Client
	HubspotClient    *hubspot.Client
	WorkerClient     *pkgworker.Client
	OpenAIClient     *ai.OpenAIClient
	RedisClient      *redis.Client
	IntentClassifier *intentService.IntentClassifier
	TemporalClient   temporalClient.Client

	// Session
	Session core.Session
}

// Repositories holds all repository instances
type Repositories struct {
	User                *userRepo.UserRepository
	Organization        *organizationRepo.OrganizationRepository
	Contact             *contactRepo.ContactRepository
	Campaign            *campaignRepo.CampaignRepository
	Agent               *agentRepo.AgentRepository
	Task                *taskRepo.TaskRepository
	Template            *templateRepo.TemplateRepository
	Email               *emailRepo.Repository
	Conversation        *conversationRepo.ConversationRepository
	Role                *roleRepo.RoleRepository
	CustomField         *customFieldRepo.CustomFieldRepository
	ListContact         *contactRepo.ListContactRepository
	SaleRep             *saleRepRepo.SaleRepRepository
	Workflow            *workflowRepo.WorkflowRepository
	Knowledge           *knowledgeRepo.KnowledgeRepository
	InboundLeadForm     *inboundLeadFormRepo.InboundLeadFormRepository
	LeadFormIntegration *inboundLeadFormRepo.LeadFormIntegrationRepository
	FacebookToken       *facebookTokenRepo.FacebookTokenRepository
	File                *fileRepo.FileRepository
	Interaction         *interactionRepo.Repository
	Feedback            *feedbackRepo.Repository
	PersonalApiKey      *personalApiKeyRepo.PersonalApiKeyRepository
	DraftTemplate       *draftTemplateRepo.DraftTemplateRepository
	Operation           *operationRepo.OperationRepository
	Integration         *integrationRepo.IntegrationRepository
	Notification        *notificationRepo.NotificationRepository
	PubSub              pubsubRepo.PubSubRepository
	Segmentation        *segmentationRepo.SegmentationRepository
	SegmentScore        *segmentationRepo.ScoreRepository
	Playbook            *playbookRepo.Repository
	AutomationRule      *playbookRepo.AutomationRuleRepository
	Enrollment          *playbookRepo.EnrollmentRepository
	ApprovalRequest     *playbookRepo.ApprovalRequestRepository
	InteractionLog      *outreachRepo.InteractionLogRepository
	OutreachState       *outreachRepo.OutreachStateRepository
	Meeting             *outreachRepo.MeetingRepository
	OutreachFeedback    *outreachRepo.FeedbackRepository
}

// Services holds all service instances
type Services struct {
	AICompany           *aiService.AICompanyService
	AIEmail             *aiService.AIEmailService
	Auth                *googleService.GoogleAuthService
	Gmail               *gmailService.GmailService // Gmail service
	HubspotAPI          *hubspotService.HubspotAPI
	HubspotIntegration  *hubspotService.HubspotIntegrationService
	Outlook             *outlookService.OutlookService
	InboundLeadForm     *inboundLeadFromService.InboundLeadFormService
	LeadFormIntegration *inboundLeadFromService.LeadFormIntegrationService
	GoogleAds           *googleService.GoogleAdsService
	Intelligence        *intelService.Service
	Scraper             *scraperService.Service
	S3                  *s3Service.S3Service
	Knowledge           *knowledgeSerive.KnowledgeService
	Organization        *orgService.Service
	Playbook            *playbookService.Service
	AutomationRule      *playbookService.AutomationService
	Enrollment          *playbookService.EnrollmentService
	Outreach            *outreachService.Service
}

// UseCases holds all use case instances
type UseCases struct {
	GoogleAds *googleAdsUsecase.GoogleAdsUseCase
	PubSub    *pubsubUsecase.PubSubUseCase
}

// V1Handlers holds all V1 handler instances
type V1Handlers struct {
	Auth                *v1auth.AuthHandler
	User                *v1user.UserHandler
	Organization        *v1organization.OrganizationHandler
	Contact             *v1contact.Handler
	Campaign            *v1campaign.Handler
	Agent               *agent.AgentHandler
	Task                *v1task.TaskHandler
	Template            *v1template.TemplateHandler
	Email               *v1email.EmailHandler
	Conversation        *v1conversation.ConversationHandler
	Interaction         *v1interaction.Handler
	Segmentation        *v1segmentation.Handler
	Feedback            *v1feedback.Handler
	Intelligence        *v1intelligence.Handler
	Gmail               *v1gmail.GmailHandler
	TaskEnqueue         *v1task.TaskEnqueueHandler
	SaleRep             *v1saleRep.SaleRepHandler
	CustomField         *v1customfield.Handler
	ListContact         *v1listcontact.Handler
	Workflow            *v1workFlow.WorkflowHandler
	Knowledge           *v1knowledge.Handler
	AICompany           *v1ai.AIHandler
	AIEmail             *v1ai.AIEmailHandler
	Hubspot             *v1hubspot.HubspotHandler
	Outlook             *v1outlook.OutlookHandler
	InboundLeadForm     *v1inboundleadform.InboundLeadFormHandler
	LeadFormIntegration *v1inboundleadform.LeadFormIntegrationHandler
	GoogleAds           *v1googleads.Handler
	PubSub              *v1pubsub.Handler
	Lab                 *v1lab.LabHandler
	File                *v1file.FileHandler
	Meta                *v1meta.MetaHandler
	MCP                 *v1mcp.MCPHandler
	Playbook            *v1playbook.Handler
	AutomationRule      *v1playbook.AutomationHandler
	Enrollment          *v1playbook.EnrollmentHandler
	Temporal            *v1temporal.Handler
	Outreach            *v1outreach.Handler
}

// V2Handlers holds all V2 handler instances
type V2Handlers struct {
	User          *v2user.UserHandler
	Auth          *v2auth.AuthHandler
	DraftTemplate *v2template.DraftTemplateHandler
	Organization  *v2organization.OrganizationHandler
	ListContact   *v2listcontact.Handler
	Knowledge     *v2knowledge.Handler
	Template      *v2template.TemplateHandler
	Hubspot       *v2hubspot.HubspotHandler
	Gmail         *v2gmail.GmailHandler
	Email         *v2email.EmailHandler
	Contact       *v2contact.Handler
	Conversation  *v2conversation.ConversationHandler
	Campaign      *v2campaign.Handler
}

// V3Handlers holds all V3 handler instances
type V3Handlers struct {
	Operation *v3handler.OperationHandler
	Contact   *v3contact.Handler
	Campaign  *v3campaign.Handler
}

// InitDependencies initializes all application dependencies
func InitDependencies(db *gorm.DB, redisClient *redis.Client, cfg *config.Config, jwtManager *auth.JWTManager) *Dependencies {
	deps := &Dependencies{
		DB:          db,
		RedisClient: redisClient,
	}

	// Initialize repositories
	deps.Repos = initRepositories(db, cfg)

	// Initialize PubSub repository (requires RedisClient)
	deps.Repos.PubSub = pubsubRepo.NewPubSubRepository(redisClient)

	// Initialize clients
	deps.OAuth2Client = initOAuth2Client(cfg)
	deps.HubspotClient = initHubspotClient(cfg)
	deps.WorkerClient = initWorkerClient(cfg)
	deps.OpenAIClient, deps.IntentClassifier = initAIClients(cfg)
	deps.TemporalClient = initTemporalClient(cfg)

	// Initialize services
	deps.Services = initServices(deps, cfg, jwtManager)

	// Initialize use cases
	deps.UseCases = initUseCases(deps, cfg)

	// Initialize session
	deps.Session = core.NewGormSession(db, nil)

	// Initialize handlers
	deps.V1Handlers = initV1Handlers(deps, cfg, jwtManager)
	deps.V2Handlers = initV2Handlers(deps, cfg, jwtManager)
	deps.V3Handlers = initV3Handlers(deps, cfg)

	return deps
}

// getSqlxDB creates sqlx.DB from gorm.DB for playbook repositories
func getSqlxDB(gormDB *gorm.DB) (*sqlx.DB, error) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(sqlDB, "postgres"), nil
}

// initRepositories initializes all repositories
func initRepositories(db *gorm.DB, cfg *config.Config) *Repositories {
	apiKeySecret := cfg.Auth.PersonalAPIKeySecret
	if apiKeySecret == "" {
		apiKeySecret = cfg.Auth.JWTSecret
	}

	// Create sqlx.DB for playbook repositories
	sqlxDB, err := getSqlxDB(db)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to create sqlx.DB")
	}

	return &Repositories{
		User:                userRepo.NewUserRepository(db),
		Organization:        organizationRepo.NewOrganizationRepository(db),
		Contact:             contactRepo.NewContactRepository(db),
		Campaign:            campaignRepo.NewCampaignRepository(db),
		Agent:               agentRepo.NewAgentRepository(db),
		Task:                taskRepo.NewTaskRepository(db),
		Template:            templateRepo.NewTemplateRepository(db),
		Email:               emailRepo.NewEmailRepository(db),
		Conversation:        conversationRepo.NewConversationRepository(db),
		Role:                roleRepo.NewRoleRepository(db),
		CustomField:         customFieldRepo.NewCustomFieldRepository(db),
		ListContact:         contactRepo.NewListContactRepository(db),
		SaleRep:             saleRepRepo.NewSaleRepRepository(db),
		Workflow:            workflowRepo.NewWorkflowRepository(db),
		Knowledge:           knowledgeRepo.NewKnowledgeRepository(db),
		InboundLeadForm:     inboundLeadFormRepo.NewInboundLeadFormRepository(db),
		LeadFormIntegration: inboundLeadFormRepo.NewLeadFormIntegrationRepository(db),
		FacebookToken:       facebookTokenRepo.NewFacebookTokenRepository(db),
		File:                fileRepo.NewFileRepository(db),
		Interaction:         interactionRepo.NewRepository(db),
		PersonalApiKey:      personalApiKeyRepo.NewPersonalApiKeyRepository(db, apiKeySecret),
		DraftTemplate:       draftTemplateRepo.NewDraftTemplateRepository(db),
		Operation:           operationRepo.NewOperationRepository(db),
		Integration:         integrationRepo.NewIntegrationRepository(db),
		Notification:        notificationRepo.NewNotificationRepository(db),
		Segmentation:        segmentationRepo.NewSegmentationRepository(db),
		SegmentScore:        segmentationRepo.NewScoreRepository(db),
		Feedback:            feedbackRepo.NewRepository(db),
		Playbook:            playbookRepo.NewRepository(sqlxDB),
		AutomationRule:      playbookRepo.NewAutomationRuleRepository(sqlxDB),
		Enrollment:          playbookRepo.NewEnrollmentRepository(sqlxDB),
		ApprovalRequest:     playbookRepo.NewApprovalRequestRepository(sqlxDB),
		InteractionLog:      outreachRepo.NewInteractionLogRepository(db),
		OutreachState:       outreachRepo.NewOutreachStateRepository(db),
		Meeting:             outreachRepo.NewMeetingRepository(db),
		OutreachFeedback:    outreachRepo.NewFeedbackRepository(db),
		// PubSub will be initialized in InitDependencies with RedisClient
	}
}

// initOAuth2Client initializes Google OAuth2 client
func initOAuth2Client(cfg *config.Config) *googleoauth.Client {
	return googleoauth.NewClient(googleoauth.Config{
		ClientID:     cfg.OAuth.GoogleClientID,
		ClientSecret: cfg.OAuth.GoogleClientSecret,
		RedirectURI:  cfg.OAuth.GoogleRedirectURI,
	})
}

// initHubspotClient initializes HubSpot client
func initHubspotClient(cfg *config.Config) *hubspot.Client {
	if cfg.Hubspot.ClientID != "" && cfg.Hubspot.ClientSecret != "" {
		scopes := strings.Split(cfg.Hubspot.AuthScopes, " ")
		client := hubspot.NewClient(
			cfg.Hubspot.ClientID,
			cfg.Hubspot.ClientSecret,
			cfg.Hubspot.RedirectURI,
			scopes,
		)
		logger.Logger.Info().Msg("HubSpot client initialized")
		return client
	}
	logger.Logger.Warn().Msg("HubSpot not configured - HubSpot endpoints will be unavailable")
	return nil
}

// initWorkerClient initializes Asynq worker client
func initWorkerClient(cfg *config.Config) *pkgworker.Client {
	addr := "127.0.0.1:6379"
	password := cfg.Redis.Password
	db := 0

	if cfg.Redis.URL != "" {
		if parsed, err := url.Parse(cfg.Redis.URL); err != nil {
			logger.Logger.Warn().Err(err).Msg("failed to parse redis url for worker client; falling back to defaults")
		} else {
			host := parsed.Hostname()
			port := parsed.Port()
			if port == "" {
				port = "6379"
			}
			if host != "" {
				addr = fmt.Sprintf("%s:%s", host, port)
			}
			if parsed.User != nil {
				if pass, ok := parsed.User.Password(); ok && pass != "" {
					password = pass
				}
			}
			if len(parsed.Path) > 1 {
				if dbVal, err := strconv.Atoi(strings.TrimPrefix(parsed.Path, "/")); err == nil {
					db = dbVal
				}
			}
		}
	}

	workerCfg := pkgworker.Config{
		RedisAddr:     addr,
		RedisPassword: password,
		RedisDB:       db,
		Concurrency:   10,
	}
	return pkgworker.NewClient(workerCfg)
}

// initAIClients initializes OpenAI client and intent classifier
func initAIClients(cfg *config.Config) (*ai.OpenAIClient, *intentService.IntentClassifier) {
	if cfg.AI.OpenAIAPIKey == "" {
		logger.Logger.Warn().Msg("OPENAI_API_KEY not configured - AI endpoints will return errors")
		return nil, nil
	}

	openAIClient := ai.NewOpenAIClient(ai.Config{
		APIKey: cfg.AI.OpenAIAPIKey,
		Model:  cfg.AI.OpenAIModel,
	})

	classifierClient := openai.NewClient(
		option.WithAPIKey(cfg.AI.OpenAIAPIKey),
	)
	intentClassifier := intentService.NewIntentClassifier(&classifierClient, cfg.AI.OpenAIModel, &logger.Logger)

	return openAIClient, intentClassifier
}

// initTemporalClient initializes Temporal client for triggering workflows
func initTemporalClient(cfg *config.Config) temporalClient.Client {
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}
	temporalNamespace := os.Getenv("TEMPORAL_NAMESPACE")
	if temporalNamespace == "" {
		temporalNamespace = "default"
	}

	client, err := worker.NewTemporalClient(worker.Config{
		HostPort:  temporalHost,
		Namespace: temporalNamespace,
	})
	if err != nil {
		logger.Logger.Warn().Err(err).Msg("Failed to connect to Temporal - workflow endpoints will be unavailable")
		return nil
	}

	logger.Logger.Info().Str("host", temporalHost).Str("namespace", temporalNamespace).Msg("Temporal client initialized")
	return client
}

// initServices initializes all services
func initServices(deps *Dependencies, cfg *config.Config, jwtManager *auth.JWTManager) *Services {
	globalCacheManager := cache.GetGlobalManager()

	// Initialize Gmail service following clean architecture pattern

	// Create Gmail service worker adapter to match interface
	gmailWorkerAdapter := &GmailWorkerAdapter{Client: deps.WorkerClient}

	// Get Pubsub topic from environment or use default
	pubsubTopic := os.Getenv("GMAIL_PUBSUB_TOPIC")
	if pubsubTopic == "" {
		pubsubTopic = "projects/your-project/topics/gmail-notifications"
	}

	// Create a new repository instance for Gmail service using the new repository type
	agentRepo := agentRepo.NewAgentRepository(deps.DB)
	gmailAccountRepo := gmailRepo.NewGmailAccountRepository(deps.DB)

	// Create Gmail service config
	gmailConfig := gmailService.GmailConfig{
		OAuthClient:      deps.OAuth2Client,
		AgentRepo:        agentRepo,
		GmailAccountRepo: gmailAccountRepo,
		RedisClient:      deps.RedisClient,
		JWTSecret:        cfg.Auth.JWTSecret,
		WorkerClient:     gmailWorkerAdapter,
		PubsubTopic:      pubsubTopic,
	}

	// Initialize Vector Search infrastructure
	redisVectorStore := vectorstore.NewRedisVectorStore(deps.RedisClient)
	ctx := context.Background()
	if err := redisVectorStore.InitializeIndexes(ctx); err != nil {
		logger.Logger.Warn().Err(err).Msg("Failed to initialize vector search indexes - vector search will be unavailable")
	}
	vectorSearchSkill := skills.NewVectorSearchSkill(redisVectorStore, deps.OpenAIClient)

	services := &Services{
		AICompany:           aiService.NewAICompanyService(deps.OpenAIClient, globalCacheManager),
		AIEmail:             aiService.NewAIEmailService(deps.Repos.Email, deps.Repos.User, deps.Repos.Knowledge, deps.IntentClassifier, deps.OpenAIClient),
		Intelligence:        intelService.NewService(deps.Repos.Contact, deps.Repos.Interaction, deps.Repos.Segmentation, deps.Repos.SegmentScore, deps.OpenAIClient, vectorSearchSkill, redisVectorStore),
		Scraper:             scraperService.NewService(deps.OpenAIClient),
		Auth:                googleService.NewGoogleAuthService(deps.Repos.User, deps.Repos.Role, jwtManager, deps.OAuth2Client),
		Gmail:               gmailService.NewGmailService(gmailConfig),
		HubspotAPI:          hubspotService.NewHubspotAPI(cfg.Hubspot),
		Outlook:             outlookService.NewOutlookService(cfg.Outlook),
		InboundLeadForm:     inboundLeadFromService.NewInboundLeadFormService(deps.Repos.InboundLeadForm, deps.Repos.CustomField, deps.Repos.Contact, deps.Repos.ListContact, deps.Repos.Organization),
		LeadFormIntegration: inboundLeadFromService.NewLeadFormIntegrationService(deps.Repos.Campaign, deps.Repos.LeadFormIntegration, deps.Repos.InboundLeadForm, deps.Repos.FacebookToken),
		Organization:        orgService.NewService(deps.Repos.Organization, deps.Repos.User, deps.Repos.Role, deps.DB),
	}

	services.HubspotIntegration = hubspotService.NewHubspotIntegrationService(services.HubspotAPI, deps.Repos.User, deps.WorkerClient)

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", cfg.App.Host, cfg.App.Port)
	}
	services.GoogleAds = googleService.NewGoogleAdsService(
		deps.Repos.InboundLeadForm,
		deps.Repos.Contact,
		deps.Repos.ListContact,
		deps.Repos.Campaign,
		deps.WorkerClient,
		baseURL,
	)

	// Initialize S3 service (optional)
	logger.Logger.Debug().
		Str("bucket", cfg.S3.BucketName).
		Str("region", cfg.S3.Region).
		Str("endpoint", cfg.S3.EndpointURL).
		Bool("has_access_key", cfg.S3.AccessKeyID != "").
		Msg("S3 Configuration")

	if cfg.S3.BucketName != "" {
		s3Svc, err := s3Service.NewS3Service(s3Service.S3Config{
			Region:          cfg.S3.Region,
			AccessKeyID:     cfg.S3.AccessKeyID,
			SecretAccessKey: cfg.S3.SecretAccessKey,
			EndpointURL:     cfg.S3.EndpointURL,
		})
		if err != nil {
			logger.Logger.Warn().Err(err).Msg("Failed to initialize S3 service")
		} else {
			services.S3 = s3Svc
			logger.Logger.Info().Str("bucket", cfg.S3.BucketName).Msg("S3 service initialized")
		}
	}

	services.Knowledge = knowledgeSerive.NewKnowledgeService(
		deps.Repos.Knowledge,
		services.S3,
		deps.WorkerClient,
		cfg.S3.BucketName,
	)

	// Initialize playbook services
	services.Playbook = playbookService.NewService(
		deps.Repos.Playbook,
		deps.Repos.AutomationRule,
		deps.Repos.Enrollment,
		deps.Repos.ApprovalRequest,
		deps.Repos.Contact,
		deps.Repos.Segmentation,
		deps.OpenAIClient,
	)

	services.AutomationRule = playbookService.NewAutomationService(
		deps.Repos.AutomationRule,
		deps.Repos.Playbook,
		deps.Repos.Segmentation,
		deps.Repos.Enrollment,
	)

	services.Enrollment = playbookService.NewEnrollmentService(
		deps.Repos.Enrollment,
		deps.Repos.ApprovalRequest,
		deps.Repos.Playbook,
		deps.Repos.Contact,
	)

	// Initialize outreach service with AI support
	services.Outreach = outreachService.NewServiceWithAI(
		deps.Repos.InteractionLog,
		deps.Repos.OutreachState,
		deps.Repos.Meeting,
		deps.Repos.OutreachFeedback,
		deps.Repos.Contact,
		deps.OpenAIClient,
		nil, // Use default config
	)

	return services
}

// initUseCases initializes all use cases
func initUseCases(deps *Dependencies, cfg *config.Config) *UseCases {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", cfg.App.Host, cfg.App.Port)
	}

	return &UseCases{
		GoogleAds: googleAdsUsecase.NewGoogleAdsUseCase(
			deps.Repos.InboundLeadForm,
			deps.Repos.Contact,
			deps.Repos.ListContact,
			deps.Repos.Campaign,
			deps.WorkerClient,
			baseURL,
		),
		PubSub: pubsubUsecase.NewPubSubUseCase(deps.Repos.PubSub),
	}
}

// mustInitAgentHandler initializes AgentHandler and panics on error
func mustInitAgentHandler(
	agentRepo *agentRepo.AgentRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	userRepo *userRepo.UserRepository,
	emailRepo *emailRepo.Repository,
	roleRepo *roleRepo.RoleRepository,
	workerClient *pkgworker.Client,
) *agent.AgentHandler {
	handler, err := agent.NewAgentHandler(agentRepo, conversationRepo, userRepo, emailRepo, roleRepo, workerClient)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to initialize AgentHandler")
	}
	return handler
}

// initV1Handlers initializes all V1 handlers
func initV1Handlers(deps *Dependencies, cfg *config.Config, jwtManager *auth.JWTManager) *V1Handlers {
	// Initialize WebSocket manager for PubSub
	wsManager := v1pubsub.NewWebSocketManager(deps.RedisClient)

	handlers := &V1Handlers{
		Auth:         v1auth.NewAuthHandler(deps.Repos.User, jwtManager, deps.Services.Auth, deps.DB),
		User:         v1user.NewUserHandler(deps.Repos.User, deps.Repos.PersonalApiKey, deps.Repos.Organization, deps.DB),
		Organization: v1organization.NewOrganizationHandler(deps.Session, deps.Repos.Organization, deps.Repos.User, deps.Repos.Role, deps.Services.Organization),
		Campaign: v1campaign.NewHandler(
			deps.Repos.Campaign,
			deps.Repos.Role,
			deps.Repos.Notification,
			deps.Repos.ListContact,
			deps.Repos.Template,
			deps.Repos.DraftTemplate,
			deps.Repos.Agent,
			deps.Repos.Conversation,
			deps.Services.AIEmail,
			deps.WorkerClient,
		),
		Contact: v1contact.New(
			deps.Repos.Contact,
			deps.Repos.User,
			deps.Repos.Role,
			deps.Repos.Organization,
			deps.Repos.CustomField,
			deps.Repos.ListContact,
			deps.Repos.Operation,
			deps.Repos.Feedback,
			deps.Services.Intelligence,
			deps.Services.Scraper,
			deps.WorkerClient,
		),
		Agent:        mustInitAgentHandler(deps.Repos.Agent, deps.Repos.Conversation, deps.Repos.User, deps.Repos.Email, deps.Repos.Role, deps.WorkerClient),
		Task:         v1task.NewTaskHandler(deps.Repos.Task),
		Template:     v1template.NewTemplateHandler(deps.Repos.Template),
		Email:        v1email.NewEmailHandler(deps.Repos.Task, deps.Repos.Role),
		Conversation: v1conversation.NewConversationHandler(deps.Repos.Conversation, deps.Repos.Email, deps.Repos.User),
		Interaction:  v1interaction.New(deps.Repos.Interaction, deps.Repos.User, deps.Repos.Role, deps.WorkerClient),
		Segmentation: v1segmentation.NewHandler(deps.Repos.Segmentation, deps.Repos.SegmentScore, deps.Repos.Contact, deps.Repos.User, deps.Repos.Role),
		Intelligence: v1intelligence.New(
			deps.Services.Intelligence,
			deps.Repos.User,
			deps.Repos.Role,
			deps.Repos.Contact,
			deps.Repos.Interaction,
			deps.Repos.Campaign,
			deps.Repos.Conversation,
			deps.Repos.Email,
		),
		Gmail:               v1gmail.NewGmailHandler(deps.OAuth2Client, deps.Repos.Agent, deps.Repos.User, cfg.OAuth.GooglePubSubGmailTopic),
		TaskEnqueue:         v1task.NewTaskEnqueueHandler(deps.WorkerClient, deps.Repos.Campaign, deps.Repos.Agent),
		SaleRep:             v1saleRep.NewSaleRepHandler(deps.Repos.SaleRep, deps.Repos.User, deps.Services.S3, cfg.S3.BucketName),
		CustomField:         v1customfield.NewHandler(deps.Repos.CustomField, deps.Repos.User, deps.Repos.Role),
		ListContact:         v1listcontact.NewHandler(deps.Repos.ListContact, deps.Repos.User),
		Workflow:            v1workFlow.NewWorkflowHandler(deps.Repos.Workflow, deps.Repos.User),
		Knowledge:           v1knowledge.NewHandler(deps.Repos.Knowledge, deps.Repos.User, deps.Services.Knowledge),
		AICompany:           v1ai.NewAIHandler(deps.Services.AICompany),
		AIEmail:             v1ai.NewAIEmailHandler(aiUsecase.NewAIEmailUsecaseAdapter(deps.Services.AIEmail)),
		Hubspot:             v1hubspot.NewHubspotHandler(deps.Services.HubspotIntegration),
		Outlook:             v1outlook.NewOutlookHandler(deps.Services.Outlook),
		InboundLeadForm:     v1inboundleadform.NewInboundLeadFormHandler(deps.Services.InboundLeadForm),
		LeadFormIntegration: v1inboundleadform.NewLeadFormIntegrationHandler(deps.Services.LeadFormIntegration),
		GoogleAds:           v1googleads.NewHandler(deps.UseCases.GoogleAds),
		PubSub:              v1pubsub.NewHandler(deps.UseCases.PubSub, wsManager),
		Lab:                 v1lab.NewLabHandler(deps.Repos.Email, deps.Repos.User, deps.Repos.Organization, cfg.AI.OpenAIAPIKey),
		Meta:                v1meta.NewMetaHandler(deps.Repos.FacebookToken, cfg.Facebook.ClientID, cfg.Facebook.ClientSecret, "pages_manage_ads,leads_retrieval", cfg.Facebook.VerifyToken),
		Feedback:            v1feedback.New(deps.Repos.Feedback, deps.Repos.User, deps.Repos.Role),
	}

	// File handler (only if S3 is configured)
	if deps.Services.S3 != nil {
		handlers.File = v1file.NewFileHandler(deps.Repos.File, deps.Services.S3, cfg.S3.BucketName)
	}

	// MCP handler
	handlers.MCP = v1mcp.NewMCPHandler(
		deps.Repos.Campaign,
		deps.Repos.Contact,
		deps.Repos.ListContact,
		deps.Repos.Role,
		deps.Repos.Notification,
		deps.Repos.Agent,
		deps.Services.AIEmail,
	)

	// Playbook handlers
	handlers.Playbook = v1playbook.NewHandler(deps.Services.Playbook)
	handlers.AutomationRule = v1playbook.NewAutomationHandler(deps.Services.AutomationRule)
	handlers.Enrollment = v1playbook.NewEnrollmentHandler(deps.Services.Enrollment)

	// Outreach handler
	handlers.Outreach = v1outreach.NewHandler(deps.Services.Outreach, deps.Repos.User, deps.Repos.Role)

	// Temporal workflow handler (only if Temporal is configured)
	if deps.TemporalClient != nil {
		handlers.Temporal = v1temporal.New(deps.TemporalClient, deps.Repos.User, deps.Repos.Role)
	}

	return handlers
}

// initV2Handlers initializes all V2 handlers
func initV2Handlers(deps *Dependencies, cfg *config.Config, jwtManager *auth.JWTManager) *V2Handlers {
	handlers := &V2Handlers{
		User:          v2user.NewUserHandler(deps.Repos.User),
		Auth:          v2auth.NewAuthHandler(deps.Repos.User, deps.Services.Auth),
		DraftTemplate: v2template.NewDraftTemplateHandler(deps.Repos.DraftTemplate),
		Organization:  v2organization.NewOrganizationHandler(deps.Session, deps.Repos.Organization, deps.Repos.User, deps.Repos.Role, deps.WorkerClient),
		ListContact:   v2listcontact.NewHandler(deps.Repos.ListContact, deps.Repos.User),
		Knowledge:     v2knowledge.NewHandler(deps.Repos.Knowledge, deps.Repos.User, deps.Services.Knowledge),
		Template:      v2template.NewTemplateHandler(deps.Repos.Template, deps.Repos.Knowledge),
		Hubspot:       v2hubspot.NewHubspotHandler(deps.Session, deps.Repos.Integration, deps.Repos.User, deps.HubspotClient),
		Gmail:         v2gmail.NewGmailHandler(deps.OAuth2Client, deps.Repos.Agent, deps.Repos.User, cfg.OAuth.GooglePubSubGmailTopic, cfg.Auth.JWTSecret, v2worker.NewWorkerClientAdapter(deps.WorkerClient), deps.RedisClient),
		Email:         v2email.NewEmailHandler(deps.Repos.Email, deps.Repos.Integration, deps.Repos.Agent),
		Contact:       v2contact.New(deps.Repos.Contact, deps.Repos.User, deps.Repos.Role, deps.Repos.ListContact, deps.Repos.CustomField, deps.Repos.InboundLeadForm, deps.Repos.Operation, deps.Repos.Integration, deps.WorkerClient),
		Conversation:  v2conversation.NewConversationHandler(deps.Repos.Conversation, deps.Repos.Email, deps.Repos.Campaign),
	}

	// V2 Campaign handler - always initialize, OpenAI methods will check for nil client
	var openAIClient *openai.Client
	if deps.OpenAIClient != nil {
		cli := openai.NewClient(option.WithAPIKey(cfg.AI.OpenAIAPIKey))
		openAIClient = &cli
	}

	handlers.Campaign = v2campaign.NewHandler(
		deps.Repos.Campaign,
		deps.Repos.Template,
		deps.Repos.DraftTemplate,
		deps.Repos.User,
		deps.Repos.Role,
		deps.Repos.Organization,
		deps.Repos.Knowledge,
		deps.Repos.Conversation,
		openAIClient,
		deps.IntentClassifier,
	)

	return handlers
}

// initV3Handlers initializes all V3 handlers
func initV3Handlers(deps *Dependencies, cfg *config.Config) *V3Handlers {
	handlers := &V3Handlers{
		Operation: v3handler.NewOperationHandler(deps.Repos.Operation),
		Contact:   v3contact.New(deps.Repos.Contact, deps.Repos.User, deps.Repos.Role, deps.Repos.ListContact, deps.Repos.CustomField, deps.Repos.Operation),
	}

	// V3 Campaign handler (requires OpenAI)
	if deps.OpenAIClient != nil {
		openaiCli := openai.NewClient(option.WithAPIKey(cfg.AI.OpenAIAPIKey))
		handlers.Campaign = v3campaign.NewHandler(
			deps.Repos.Campaign,
			deps.Repos.Template,
			deps.Repos.User,
			deps.Repos.Organization,
			deps.Repos.Knowledge,
			&openaiCli,
		)
	}

	return handlers
}

// GmailWorkerAdapter adapts the existing worker.Client to match GmailService.WorkerClient interface
type GmailWorkerAdapter struct {
	Client *pkgworker.Client
}

// EnqueueTask implements GmailService.WorkerClient interface
func (a *GmailWorkerAdapter) EnqueueTask(ctx context.Context, taskType string, payload interface{}) error {
	_, err := a.Client.EnqueueTask(ctx, taskType, payload)
	return err
}

// Close closes all dependency connections
func (d *Dependencies) Close() {
	if d.WorkerClient != nil {
		d.WorkerClient.Close()
	}
}
