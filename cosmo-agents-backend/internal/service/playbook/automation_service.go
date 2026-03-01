package playbook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/playbook"
	playbookRepo "github.com/rockship/cosmo-agents-go/internal/repository/playbook"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// AutomationService handles automation rule business logic
type AutomationService struct {
	automationRepo   *playbookRepo.AutomationRuleRepository
	playbookRepo     *playbookRepo.Repository
	segmentationRepo *segRepo.SegmentationRepository
	enrollmentRepo   *playbookRepo.EnrollmentRepository
}

// NewAutomationService creates a new automation service
func NewAutomationService(
	automationRepo *playbookRepo.AutomationRuleRepository,
	playbookRepo *playbookRepo.Repository,
	segmentationRepo *segRepo.SegmentationRepository,
	enrollmentRepo *playbookRepo.EnrollmentRepository,
) *AutomationService {
	return &AutomationService{
		automationRepo:   automationRepo,
		playbookRepo:     playbookRepo,
		segmentationRepo: segmentationRepo,
		enrollmentRepo:   enrollmentRepo,
	}
}

// CreateAutomationRule creates a new automation rule
func (s *AutomationService) CreateAutomationRule(ctx context.Context, req *v1schema.AutomationRuleRequest) (*v1schema.AutomationRuleRead, error) {
	// Validate segment exists
	segment, err := s.segmentationRepo.GetByID(ctx, req.SegmentID)
	if err != nil || segment == nil {
		return nil, fmt.Errorf("segment not found")
	}

	// Validate playbook exists
	pb, err := s.playbookRepo.GetByID(ctx, req.PlaybookID)
	if err != nil || pb == nil {
		return nil, fmt.Errorf("playbook not found")
	}

	// Marshal enrollment criteria to JSONB
	criteriaBytes, err := json.Marshal(req.EnrollmentCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enrollment criteria: %w", err)
	}

	// Create domain model
	rule := &playbook.AutomationRule{
		Name:               req.Name,
		SegmentID:          req.SegmentID,
		PlaybookID:         req.PlaybookID,
		EnrollmentCriteria: base.JSONB(criteriaBytes),
		IsActive:           req.IsActive,
	}

	// Insert into database
	if err := s.automationRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create automation rule: %w", err)
	}

	// Convert to response
	return s.toAutomationRuleRead(ctx, rule, segment.Name, pb.Name)
}

// ListAutomationRules lists all automation rules
func (s *AutomationService) ListAutomationRules(ctx context.Context) ([]*v1schema.AutomationRuleRead, error) {
	rules, err := s.automationRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list automation rules: %w", err)
	}

	result := make([]*v1schema.AutomationRuleRead, 0, len(rules))
	for _, rule := range rules {
		// Fetch segment name
		segment, err := s.segmentationRepo.GetByID(ctx, rule.SegmentID)
		if err != nil || segment == nil {
			continue
		}

		// Fetch playbook name
		pb, err := s.playbookRepo.GetByID(ctx, rule.PlaybookID)
		if err != nil || pb == nil {
			continue
		}

		ruleRead, err := s.toAutomationRuleRead(ctx, rule, segment.Name, pb.Name)
		if err != nil {
			continue
		}

		result = append(result, ruleRead)
	}

	return result, nil
}

// GetAutomationRule gets an automation rule by ID
func (s *AutomationService) GetAutomationRule(ctx context.Context, ruleID uuid.UUID) (*v1schema.AutomationRuleRead, error) {
	rule, err := s.automationRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get automation rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("automation rule not found")
	}

	// Fetch segment name
	segment, err := s.segmentationRepo.GetByID(ctx, rule.SegmentID)
	if err != nil || segment == nil {
		return nil, fmt.Errorf("segment not found")
	}

	// Fetch playbook name
	pb, err := s.playbookRepo.GetByID(ctx, rule.PlaybookID)
	if err != nil || pb == nil {
		return nil, fmt.Errorf("playbook not found")
	}

	return s.toAutomationRuleRead(ctx, rule, segment.Name, pb.Name)
}

// ToggleAutomationRule toggles an automation rule active status
func (s *AutomationService) ToggleAutomationRule(ctx context.Context, ruleID uuid.UUID, isActive bool) error {
	return s.automationRepo.UpdateStatus(ctx, ruleID, isActive)
}

// DeleteAutomationRule deletes an automation rule
func (s *AutomationService) DeleteAutomationRule(ctx context.Context, ruleID uuid.UUID) error {
	// Check if there are active enrollments
	// For now, just delete - in production, you might want to pause enrollments first
	return s.automationRepo.Delete(ctx, ruleID)
}

// Helper function to convert domain model to API response
func (s *AutomationService) toAutomationRuleRead(ctx context.Context, rule *playbook.AutomationRule, segmentName, playbookName string) (*v1schema.AutomationRuleRead, error) {
	var criteria v1schema.EnrollmentCriteria
	if err := json.Unmarshal(rule.EnrollmentCriteria, &criteria); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enrollment criteria: %w", err)
	}

	// Get stats - count enrollments by status
	// For now, return zeros - this would require additional repository methods
	stats := v1schema.AutomationStats{
		ContactsEnrolled:        0,
		ContactsPendingApproval: 0,
		ContactsInProgress:      0,
		ContactsCompleted:       0,
	}

	return &v1schema.AutomationRuleRead{
		AutomationRuleID:   rule.AutomationRuleID,
		Name:               rule.Name,
		SegmentID:          rule.SegmentID,
		SegmentName:        segmentName,
		PlaybookID:         rule.PlaybookID,
		PlaybookName:       playbookName,
		EnrollmentCriteria: criteria,
		IsActive:           rule.IsActive,
		Stats:              stats,
		CreatedAt:          rule.CreatedAt,
		UpdatedAt:          rule.UpdatedAt,
	}, nil
}
