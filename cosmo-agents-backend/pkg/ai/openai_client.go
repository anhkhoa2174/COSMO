package ai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClient wraps OpenAI API operations.
type OpenAIClient struct {
	client *openai.Client
	model  string
}

// Config holds OpenAI configuration.
type Config struct {
	APIKey string
	Model  string // e.g., "gpt-4", "gpt-3.5-turbo"
}

// NewOpenAIClient creates a new OpenAI client.
func NewOpenAIClient(cfg Config) *OpenAIClient {
	client := openai.NewClient(cfg.APIKey)

	model := cfg.Model
	if model == "" {
		model = openai.GPT4 // Default to GPT-4
	}

	return &OpenAIClient{
		client: client,
		model:  model,
	}
}

// ChatCompletionRequest represents a chat completion request.
type ChatCompletionRequest struct {
	Messages     []Message
	Temperature  float32
	MaxTokens    int
	SystemPrompt string
}

// Message represents a chat message.
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// ChatCompletion generates a chat completion.
func (c *OpenAIClient) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	// Add user messages
	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Set defaults
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1000
	}

	// Create request
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       c.model,
			Messages:    messages,
			Temperature: temperature,
			MaxTokens:   maxTokens,
		},
	)

	if err != nil {
		return "", fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// ChatCompletionWithParts generates a chat completion using multi-part content (e.g., images + text).
func (c *OpenAIClient) ChatCompletionWithParts(
	ctx context.Context,
	systemPrompt string,
	parts []openai.ChatMessagePart,
	temperature float32,
	maxTokens int,
) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:         openai.ChatMessageRoleUser,
		MultiContent: parts,
	})

	if temperature == 0 {
		temperature = 0.2
	}
	if maxTokens == 0 {
		maxTokens = 1000
	}

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       c.model,
			Messages:    messages,
			Temperature: temperature,
			MaxTokens:   maxTokens,
		},
	)
	if err != nil {
		return "", fmt.Errorf("chat completion failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}
	return resp.Choices[0].Message.Content, nil
}

// GenerateEmailContent generates personalized email content.
func (c *OpenAIClient) GenerateEmailContent(ctx context.Context, params EmailGenerationParams) (*EmailContent, error) {
	// Build prompt
	prompt := c.buildEmailPrompt(params)

	// Generate content
	content, err := c.ChatCompletion(ctx, ChatCompletionRequest{
		SystemPrompt: params.SystemPrompt,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.8, // More creative for email generation
		MaxTokens:   1500,
	})

	if err != nil {
		return nil, err
	}

	return &EmailContent{
		Subject: params.TemplateSubject, // Use template subject for now
		Body:    content,
	}, nil
}

// GenerateEmbedding generates text embeddings for semantic search.
func (c *OpenAIClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3, // Upgraded: 5x cheaper, 62% smaller, same performance
	}

	resp, err := c.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding generation failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return resp.Data[0].Embedding, nil
}

// buildEmailPrompt constructs the prompt for email generation.
func (c *OpenAIClient) buildEmailPrompt(params EmailGenerationParams) string {
	prompt := fmt.Sprintf(`Generate a personalized email based on the following information:

Contact Information:
- Name: %s
- Email: %s
- Company: %s
- Title: %s

Email Template:
Subject: %s
Content: %s

Campaign Goal: %s

Additional Context:
%s

Instructions:
1. Personalize the email using the contact's information
2. Keep the tone professional and friendly
3. Make it concise and engaging
4. Include a clear call-to-action
5. Do not include subject line in the body (it's already set)
6. Use HTML formatting if needed

Generate ONLY the email body content:`,
		params.ContactName,
		params.ContactEmail,
		params.ContactCompany,
		params.ContactTitle,
		params.TemplateSubject,
		params.TemplateContent,
		params.CampaignGoal,
		params.AdditionalContext,
	)

	return prompt
}

// EmailGenerationParams represents parameters for email generation.
type EmailGenerationParams struct {
	ContactName       string
	ContactEmail      string
	ContactCompany    string
	ContactTitle      string
	TemplateSubject   string
	TemplateContent   string
	CampaignGoal      string
	AdditionalContext string
	SystemPrompt      string
}

// EmailContent represents generated email content.
type EmailContent struct {
	Subject string
	Body    string
}
