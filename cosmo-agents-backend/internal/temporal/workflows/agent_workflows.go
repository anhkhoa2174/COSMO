package workflows

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/temporal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Task queue name for agent workflows
const AgentTaskQueue = "cosmo-agent-tasks"

// Retry policy for activities
var defaultRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    time.Minute,
	MaximumAttempts:    3,
}

// Activity options with default settings
var defaultActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         defaultRetryPolicy,
}

// ============ Full Contact Analysis Workflow ============

type FullAnalysisInput struct {
	ContactID string    `json:"contact_id"`
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
	SessionID string    `json:"session_id"`
}

type FullAnalysisOutput struct {
	ContactID   string                          `json:"contact_id"`
	Contact     activities.ContactInfo          `json:"contact"`
	AIInsights  map[string]interface{}          `json:"ai_insights"`
	FitScores   []activities.SegmentScore       `json:"fit_scores"`
	Email       *activities.GenerateEmailOutput `json:"suggested_email,omitempty"`
	CompletedAt time.Time                       `json:"completed_at"`
}

// FullContactAnalysisWorkflow runs a complete analysis on a contact:
// 1. Get contact info
// 2. Enrich with AI insights (if needed)
// 3. Calculate segment scores
// 4. Generate suggested outreach email
func FullContactAnalysisWorkflow(ctx workflow.Context, input FullAnalysisInput) (*FullAnalysisOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting FullContactAnalysis workflow", "contact_id", input.ContactID)

	ctx = workflow.WithActivityOptions(ctx, defaultActivityOptions)
	var a *activities.AgentActivities

	output := &FullAnalysisOutput{
		ContactID:   input.ContactID,
		CompletedAt: workflow.Now(ctx),
	}

	// Step 1: Get contact info
	var contactResult activities.GetContactOutput
	err := workflow.ExecuteActivity(ctx, a.GetContact, activities.GetContactInput{
		ContactID: input.ContactID,
	}).Get(ctx, &contactResult)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	output.Contact = contactResult.Contact
	output.AIInsights = contactResult.AIInsights
	output.FitScores = contactResult.FitScores

	// Step 2: Enrich contact if no AI insights
	if len(contactResult.AIInsights) == 0 {
		var enrichResult activities.EnrichContactOutput
		err = workflow.ExecuteActivity(ctx, a.EnrichContact, activities.EnrichContactInput{
			ContactID:    input.ContactID,
			ForceRefresh: false,
		}).Get(ctx, &enrichResult)
		if err != nil {
			logger.Warn("Failed to enrich contact", "error", err)
			// Continue anyway - enrichment is optional
		} else {
			output.AIInsights = enrichResult.AIInsights
		}
	}

	// Step 3: Calculate segment scores if not present
	if len(contactResult.FitScores) == 0 {
		var scoresResult activities.CalculateScoresOutput
		err = workflow.ExecuteActivity(ctx, a.CalculateSegmentScores, activities.CalculateScoresInput{
			ContactID: input.ContactID,
		}).Get(ctx, &scoresResult)
		if err != nil {
			logger.Warn("Failed to calculate scores", "error", err)
		} else {
			output.FitScores = scoresResult.Scores
		}
	}

	// Step 4: Generate suggested email
	var emailResult activities.GenerateEmailOutput
	err = workflow.ExecuteActivity(ctx, a.GenerateEmail, activities.GenerateEmailInput{
		ContactID: input.ContactID,
		EmailType: "cold_outreach",
	}).Get(ctx, &emailResult)
	if err != nil {
		logger.Warn("Failed to generate email", "error", err)
	} else {
		output.Email = &emailResult
	}

	output.CompletedAt = workflow.Now(ctx)
	logger.Info("FullContactAnalysis workflow completed", "contact_id", input.ContactID)

	return output, nil
}

// ============ Batch Enrichment Workflow ============

type BatchEnrichmentInput struct {
	ContactIDs   []string  `json:"contact_ids"`
	ForceRefresh bool      `json:"force_refresh"`
	UserID       uuid.UUID `json:"user_id"`
	OrgID        uuid.UUID `json:"org_id"`
}

type BatchEnrichmentOutput struct {
	TotalContacts    int       `json:"total_contacts"`
	SuccessCount     int       `json:"success_count"`
	FailedCount      int       `json:"failed_count"`
	FailedContactIDs []string  `json:"failed_contact_ids"`
	CompletedAt      time.Time `json:"completed_at"`
}

// BatchEnrichmentWorkflow enriches multiple contacts in parallel
func BatchEnrichmentWorkflow(ctx workflow.Context, input BatchEnrichmentInput) (*BatchEnrichmentOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting BatchEnrichment workflow", "count", len(input.ContactIDs))

	ctx = workflow.WithActivityOptions(ctx, defaultActivityOptions)
	var a *activities.AgentActivities

	output := &BatchEnrichmentOutput{
		TotalContacts:    len(input.ContactIDs),
		FailedContactIDs: []string{},
	}

	// Process contacts in parallel using child workflows or activities
	var futures []workflow.Future
	for _, contactID := range input.ContactIDs {
		future := workflow.ExecuteActivity(ctx, a.EnrichContact, activities.EnrichContactInput{
			ContactID:    contactID,
			ForceRefresh: input.ForceRefresh,
		})
		futures = append(futures, future)
	}

	// Collect results
	for i, future := range futures {
		var result activities.EnrichContactOutput
		err := future.Get(ctx, &result)
		if err != nil {
			output.FailedCount++
			output.FailedContactIDs = append(output.FailedContactIDs, input.ContactIDs[i])
			logger.Warn("Failed to enrich contact", "contact_id", input.ContactIDs[i], "error", err)
		} else {
			output.SuccessCount++
		}
	}

	output.CompletedAt = workflow.Now(ctx)
	logger.Info("BatchEnrichment workflow completed",
		"success", output.SuccessCount,
		"failed", output.FailedCount)

	return output, nil
}

// ============ Segment Analysis Workflow ============

type SegmentAnalysisInput struct {
	SegmentID string    `json:"segment_id"`
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
}

type SegmentAnalysisOutput struct {
	SegmentID        string    `json:"segment_id"`
	TotalContacts    int       `json:"total_contacts"`
	EnrichedContacts int       `json:"enriched_contacts"`
	AverageFitScore  float64   `json:"average_fit_score"`
	HighFitCount     int       `json:"high_fit_count"`   // >80%
	MediumFitCount   int       `json:"medium_fit_count"` // 50-80%
	LowFitCount      int       `json:"low_fit_count"`    // <50%
	Recommendations  []string  `json:"recommendations"`
	CompletedAt      time.Time `json:"completed_at"`
}

// SegmentAnalysisWorkflow analyzes a segment's health and performance
func SegmentAnalysisWorkflow(ctx workflow.Context, input SegmentAnalysisInput) (*SegmentAnalysisOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting SegmentAnalysis workflow", "segment_id", input.SegmentID)

	ctx = workflow.WithActivityOptions(ctx, defaultActivityOptions)
	var a *activities.AgentActivities

	output := &SegmentAnalysisOutput{
		SegmentID: input.SegmentID,
	}

	// Step 1: Analyze segment
	var analysisResult activities.AnalyzeSegmentOutput
	err := workflow.ExecuteActivity(ctx, a.AnalyzeSegment, activities.AnalyzeSegmentInput{
		SegmentID: input.SegmentID,
	}).Get(ctx, &analysisResult)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze segment: %w", err)
	}

	output.TotalContacts = analysisResult.TotalContacts
	output.AverageFitScore = analysisResult.AverageFitScore
	output.Recommendations = analysisResult.Recommendations

	output.CompletedAt = workflow.Now(ctx)
	logger.Info("SegmentAnalysis workflow completed", "segment_id", input.SegmentID)

	return output, nil
}

// ============ Daily Analytics Workflow ============

type DailyAnalyticsInput struct {
	Date  string    `json:"date"` // YYYY-MM-DD
	OrgID uuid.UUID `json:"org_id"`
}

type DailyAnalyticsOutput struct {
	Date             string    `json:"date"`
	NewContacts      int       `json:"new_contacts"`
	EnrichedContacts int       `json:"enriched_contacts"`
	EmailsSent       int       `json:"emails_sent"`
	TopSegments      []string  `json:"top_segments"`
	CompletedAt      time.Time `json:"completed_at"`
}

// DailyAnalyticsWorkflow generates daily analytics report
func DailyAnalyticsWorkflow(ctx workflow.Context, input DailyAnalyticsInput) (*DailyAnalyticsOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DailyAnalytics workflow", "date", input.Date)

	ctx = workflow.WithActivityOptions(ctx, defaultActivityOptions)
	var a *activities.AgentActivities

	output := &DailyAnalyticsOutput{
		Date: input.Date,
	}

	// Count new contacts for the day
	var countResult activities.CountContactsOutput
	startDate := input.Date + "T00:00:00Z"
	endDate := input.Date + "T23:59:59Z"

	err := workflow.ExecuteActivity(ctx, a.CountContactsCreated, activities.CountContactsInput{
		StartDate: startDate,
		EndDate:   endDate,
	}).Get(ctx, &countResult)
	if err != nil {
		logger.Warn("Failed to count contacts", "error", err)
	} else {
		output.NewContacts = countResult.Count
	}

	output.CompletedAt = workflow.Now(ctx)
	logger.Info("DailyAnalytics workflow completed", "date", input.Date)

	return output, nil
}
