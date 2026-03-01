package company

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog"
)

// CompanyExtractInfo contains extracted company information
type CompanyExtractInfo struct {
	CompanyDescription      string   `json:"company_description"`
	CompanyTargetingPersona []string `json:"company_targeting_persona"`
	ValueOffering           string   `json:"value_offering"`
}

// CompanyFetcher extracts company information from URLs using OpenAI
type CompanyFetcher struct {
	client *openai.Client
	model  string
	logger *zerolog.Logger
}

// NewCompanyFetcher creates a new company fetcher instance
func NewCompanyFetcher(client *openai.Client, model string, logger *zerolog.Logger) *CompanyFetcher {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &CompanyFetcher{
		client: client,
		model:  model,
		logger: logger,
	}
}

// ExtractCompanyInfo extracts company information from a URL
func (cf *CompanyFetcher) ExtractCompanyInfo(ctx context.Context, companyURL string) (*CompanyExtractInfo, error) {
	if companyURL == "" {
		return nil, fmt.Errorf("company URL cannot be empty")
	}

	// Validate URL
	parsedURL, err := url.Parse(companyURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid URL provided: %s", companyURL)
	}

	cf.logger.Info().
		Str("company_url", companyURL).
		Msg("Extracting company information")

	systemMessage := cf.buildSystemMessage()
	userMessage := fmt.Sprintf("Company URL: %s", companyURL)

	// Call OpenAI API
	completion, err := cf.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemMessage),
			openai.UserMessage(userMessage),
		},
		Temperature: openai.Float(0.0), // Deterministic for extraction
	})

	if err != nil {
		cf.logger.Error().Err(err).Msg("Failed to call OpenAI API")
		return &CompanyExtractInfo{
			CompanyDescription:      "",
			CompanyTargetingPersona: []string{},
			ValueOffering:           "",
		}, nil
	}

	if len(completion.Choices) == 0 {
		cf.logger.Error().Msg("No response from OpenAI")
		return &CompanyExtractInfo{
			CompanyDescription:      "",
			CompanyTargetingPersona: []string{},
			ValueOffering:           "",
		}, nil
	}

	// Parse JSON response
	responseContent := completion.Choices[0].Message.Content
	var result CompanyExtractInfo

	err = json.Unmarshal([]byte(responseContent), &result)
	if err != nil {
		cf.logger.Error().
			Err(err).
			Str("response", responseContent).
			Msg("Failed to parse JSON from OpenAI response")
		return &CompanyExtractInfo{
			CompanyDescription:      "",
			CompanyTargetingPersona: []string{},
			ValueOffering:           "",
		}, nil
	}

	cf.logger.Info().
		Str("company_description", result.CompanyDescription).
		Int("num_personas", len(result.CompanyTargetingPersona)).
		Str("value_offering", result.ValueOffering).
		Msg("Successfully extracted company information")

	return &result, nil
}

// buildSystemMessage constructs the system prompt with few-shot examples
func (cf *CompanyFetcher) buildSystemMessage() string {
	basePrompt := `You are a company website crawler. Extract specific, focused targeting information from SaaS/services company websites.
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
`

	fewShots := `
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
}
`

	return strings.TrimSpace(basePrompt) + fewShots
}
