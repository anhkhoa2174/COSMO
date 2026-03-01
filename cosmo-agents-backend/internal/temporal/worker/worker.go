package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/rockship/cosmo-agents-go/internal/repository/contact"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	"github.com/rockship/cosmo-agents-go/internal/service/intelligence"
	"github.com/rockship/cosmo-agents-go/internal/temporal/activities"
	"github.com/rockship/cosmo-agents-go/internal/temporal/workflows"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Config for Temporal worker
type Config struct {
	HostPort  string // e.g., "localhost:7233"
	Namespace string // e.g., "default"
}

// Dependencies holds services required for Temporal activities
type Dependencies struct {
	ContactRepo      *contact.ContactRepository
	IntelligenceSvc  *intelligence.Service
	SegmentationRepo *segRepo.SegmentationRepository
	ScoreRepo        *segRepo.ScoreRepository
	OpenAI           *ai.OpenAIClient
}

// TemporalWorker wraps the Temporal worker
type TemporalWorker struct {
	client client.Client
	worker worker.Worker
	config Config
}

// NewTemporalWorker creates a new Temporal worker with dependencies
func NewTemporalWorker(cfg Config, deps *Dependencies) (*TemporalWorker, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "default"
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	// Create worker
	w := worker.New(c, workflows.AgentTaskQueue, worker.Options{
		// Worker options
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register workflows
	w.RegisterWorkflow(workflows.FullContactAnalysisWorkflow)
	w.RegisterWorkflow(workflows.BatchEnrichmentWorkflow)
	w.RegisterWorkflow(workflows.SegmentAnalysisWorkflow)
	w.RegisterWorkflow(workflows.DailyAnalyticsWorkflow)

	// Register activities with injected dependencies
	var agentActivities *activities.AgentActivities
	if deps != nil {
		agentActivities = activities.NewAgentActivities(
			deps.ContactRepo,
			deps.IntelligenceSvc,
			deps.SegmentationRepo,
			deps.ScoreRepo,
			deps.OpenAI,
		)
	} else {
		// Fallback with nil dependencies (for testing)
		agentActivities = activities.NewAgentActivities(nil, nil, nil, nil, nil)
	}
	w.RegisterActivity(agentActivities.SearchContacts)
	w.RegisterActivity(agentActivities.GetContact)
	w.RegisterActivity(agentActivities.EnrichContact)
	w.RegisterActivity(agentActivities.CalculateSegmentScores)
	w.RegisterActivity(agentActivities.CountContactsCreated)
	w.RegisterActivity(agentActivities.AnalyzeSegment)
	w.RegisterActivity(agentActivities.GenerateEmail)
	w.RegisterActivity(agentActivities.SaveConversation)
	w.RegisterActivity(agentActivities.LoadContext)

	return &TemporalWorker{
		client: c,
		worker: w,
		config: cfg,
	}, nil
}

// NewTemporalClient creates a Temporal client only (for API server to trigger workflows)
func NewTemporalClient(cfg Config) (client.Client, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "default"
	}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	return c, nil
}

// Start starts the Temporal worker (blocking)
func (tw *TemporalWorker) Start() error {
	log.Printf("Starting Temporal worker on task queue: %s", workflows.AgentTaskQueue)
	return tw.worker.Run(worker.InterruptCh())
}

// Stop stops the Temporal worker
func (tw *TemporalWorker) Stop() {
	tw.worker.Stop()
	tw.client.Close()
}

// GetClient returns the Temporal client for starting workflows
func (tw *TemporalWorker) GetClient() client.Client {
	return tw.client
}

// ============ Helper functions to start workflows ============

// StartFullAnalysis starts a full contact analysis workflow
func (tw *TemporalWorker) StartFullAnalysis(ctx context.Context, input workflows.FullAnalysisInput) (client.WorkflowRun, error) {
	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("full-analysis-%s-%s", input.ContactID, input.SessionID),
		TaskQueue: workflows.AgentTaskQueue,
	}

	return tw.client.ExecuteWorkflow(ctx, options, workflows.FullContactAnalysisWorkflow, input)
}

// StartBatchEnrichment starts a batch enrichment workflow
func (tw *TemporalWorker) StartBatchEnrichment(ctx context.Context, input workflows.BatchEnrichmentInput) (client.WorkflowRun, error) {
	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("batch-enrichment-%s-%d", input.OrgID.String(), len(input.ContactIDs)),
		TaskQueue: workflows.AgentTaskQueue,
	}

	return tw.client.ExecuteWorkflow(ctx, options, workflows.BatchEnrichmentWorkflow, input)
}

// StartSegmentAnalysis starts a segment analysis workflow
func (tw *TemporalWorker) StartSegmentAnalysis(ctx context.Context, input workflows.SegmentAnalysisInput) (client.WorkflowRun, error) {
	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("segment-analysis-%s", input.SegmentID),
		TaskQueue: workflows.AgentTaskQueue,
	}

	return tw.client.ExecuteWorkflow(ctx, options, workflows.SegmentAnalysisWorkflow, input)
}

// StartDailyAnalytics starts a daily analytics workflow
func (tw *TemporalWorker) StartDailyAnalytics(ctx context.Context, input workflows.DailyAnalyticsInput) (client.WorkflowRun, error) {
	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("daily-analytics-%s-%s", input.OrgID.String(), input.Date),
		TaskQueue: workflows.AgentTaskQueue,
	}

	return tw.client.ExecuteWorkflow(ctx, options, workflows.DailyAnalyticsWorkflow, input)
}
