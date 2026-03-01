package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/cache"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

const companyInfoCachePrefix = "ai:company"

var (
	// ErrInvalidCompanyURL is returned when the provided company URL is invalid.
	ErrInvalidCompanyURL = errors.New("invalid company URL")
	// ErrAIClientNotConfigured indicates the OpenAI client dependency is missing.
	ErrAIClientNotConfigured = errors.New("ai client not configured")
)

// AICompanyService handles company insight extraction backed by OpenAI.
type AICompanyService struct {
	aiClient *ai.OpenAIClient
	cache    *cache.Manager

	cacheTTL time.Duration
}

// NewAICompanyService creates a new AICompanyService instance.
func NewAICompanyService(aiClient *ai.OpenAIClient, cacheManager *cache.Manager) *AICompanyService {
	if cacheManager == nil {
		cacheManager = cache.GetGlobalManager()
	}

	return &AICompanyService{
		aiClient: aiClient,
		cache:    cacheManager,
		cacheTTL: 24 * time.Hour,
	}
}

// ExtractCompanyInfo returns AI-generated company insights for the provided URL.
// The method first attempts to read from the cache, falling back to OpenAI generation.
func (s *AICompanyService) ExtractCompanyInfo(ctx context.Context, companyURL string) (*domain.CompanyInfo, error) {
	companyURL = strings.TrimSpace(companyURL)
	if companyURL == "" {
		return nil, ErrInvalidCompanyURL
	}

	parsed, err := url.Parse(companyURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, ErrInvalidCompanyURL
	}

	cacheKey := fmt.Sprintf("%s:%s", companyInfoCachePrefix, companyURL)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		switch value := cached.(type) {
		case *domain.CompanyInfo:
			return value, nil
		case domain.CompanyInfo:
			return &value, nil
		}
	}

	info, err := s.generateCompanyInfo(ctx, companyURL)
	if err != nil {
		return nil, err
	}

	if info == nil {
		info = &domain.CompanyInfo{}
	}

	if err := s.cache.Set(ctx, cacheKey, info, s.cacheTTL); err != nil {
		logger.FromContext(ctx).Warn().Err(err).Str("cache_key", cacheKey).Msg("failed to cache company info")
	}

	return info, nil
}

func (s *AICompanyService) generateCompanyInfo(ctx context.Context, companyURL string) (*domain.CompanyInfo, error) {
	if s.aiClient == nil {
		return nil, ErrAIClientNotConfigured
	}

	systemPrompt := buildCompanyInfoSystemPrompt()
	responseText, err := s.aiClient.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: systemPrompt,
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("Company URL: %s", companyURL),
			},
		},
		Temperature: 0,
		MaxTokens:   800,
	})
	if err != nil {
		logger.FromContext(ctx).Error().Err(err).Str("company_url", companyURL).Msg("failed to generate company info via OpenAI")
		// Return an empty payload to match Python fallback behaviour.
		return &domain.CompanyInfo{}, nil
	}

	var payload struct {
		CompanyDescription      string        `json:"company_description"`
		CompanyTargetingPersona []interface{} `json:"company_targeting_persona"`
		ValueOffering           string        `json:"value_offering"`
	}

	if err := json.Unmarshal([]byte(responseText), &payload); err != nil {
		logger.FromContext(ctx).Error().Err(err).Str("response", responseText).Msg("failed to parse company info response")
		return &domain.CompanyInfo{}, nil
	}

	persona := make([]string, 0, len(payload.CompanyTargetingPersona))
	for _, item := range payload.CompanyTargetingPersona {
		switch v := item.(type) {
		case string:
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				persona = append(persona, trimmed)
			}
		}
	}

	info := &domain.CompanyInfo{
		CompanyDescription:      strings.TrimSpace(payload.CompanyDescription),
		CompanyTargetingPersona: persona,
		ValueOffering:           strings.TrimSpace(payload.ValueOffering),
	}

	return info, nil
}

func buildCompanyInfoSystemPrompt() string {
	return `You are a company website crawler. Extract specific, focused targeting information from SaaS/services company websites.
Your extracted information must be specific and concise in these areas:
1. Company Description:
- Extract specific business problem they solve
- Max 30 words
- Must be specific (e.g., 'automates financial reporting', not 'helps finance teams')
- Include type of automation/solution
2. Targeting Persona:
Both decision makers and end users:
- Decision makers who buy the solution
- End users who use it daily
- Convert generic roles to specific titles
- Examples: 'Accounts Payable Manager' and 'AP Specialist', not 'finance team'
3. Value Offering:
- Extract specific value/benefit
- Max 15 words
- Must include measurable impact if available
- Focus on key business outcomes
Rules:
- If information isn't on website, use empty value
- Don't make up or infer information
- Convert vague descriptions to specific ones
- Only extract, don't create
Format as JSON:
{
  "company_description": "string",
  "company_targeting_persona": ["job title", "job title", ...],
  "value_offering": "string"
}
Use these few-shot examples:
Input URL: www.docusign.com
{
 "company_description": "Digital document signing and contract management platform that automates approval workflows.",
 "company_targeting_persona": [
   "Legal Operations Manager",
   "Contract Manager",
   "Procurement Manager",
   "HR Manager",
   "Sales Operations Manager",
   "Legal Assistant",
   "Contract Administrator",
   "Sales Representative",
   "HR Coordinator"
 ],
 "value_offering": "Reduce contract signing time from days to minutes with 82% less paperwork."
}

Input URL: www.freshdesk.com
{
 "company_description": "Customer support platform that automates ticket routing, responses, and support workflows.",
 "company_targeting_persona": [
   "Customer Service Manager",
   "Support Team Lead",
   "Help Desk Manager",
   "Customer Experience Manager",
   "Support Agent",
   "Help Desk Technician",
   "Customer Service Representative"
 ],
 "value_offering": "Resolve customer tickets 35% faster with automated workflows and instant responses."
}

Input URL: www.monday.com
{
 "company_description": "Project management platform that streamlines task tracking, team collaboration, and workflow automation.",
 "company_targeting_persona": [
   "Project Manager",
   "Team Lead",
   "Operations Manager",
   "Product Manager",
   "Project Coordinator",
   "Team Member",
   "Task Owner",
   "Project Assistant"
 ],
 "value_offering": "Save 20+ hours weekly per team with centralized project tracking and automation."
}

Input URL: www.unknown-startup.com
{
 "company_description": "",
 "company_targeting_persona": [],
 "value_offering": ""
}`
}
