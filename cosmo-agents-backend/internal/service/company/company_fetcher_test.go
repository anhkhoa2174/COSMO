package company

import (
	"context"
	"testing"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOpenAIClient is a mock implementation of OpenAI client
type MockOpenAIClient struct {
	mock.Mock
}

func TestNewCompanyFetcher(t *testing.T) {
	// Test with default model
	logger := zerolog.New(zerolog.NewTestWriter(t))
	client := &openai.Client{} // In real tests, this would be a mock

	fetcher := NewCompanyFetcher(client, "", &logger)

	assert.NotNil(t, fetcher)
	assert.Equal(t, client, fetcher.client)
	assert.Equal(t, "gpt-4o-mini", fetcher.model)
	assert.Equal(t, &logger, fetcher.logger)
}

func TestNewCompanyFetcher_CustomModel(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	client := &openai.Client{} // In real tests, this would be a mock
	customModel := "gpt-4"

	fetcher := NewCompanyFetcher(client, customModel, &logger)

	assert.NotNil(t, fetcher)
	assert.Equal(t, client, fetcher.client)
	assert.Equal(t, customModel, fetcher.model)
	assert.Equal(t, &logger, fetcher.logger)
}

func TestCompanyExtractInfo_Validation(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	client := &openai.Client{}
	fetcher := NewCompanyFetcher(client, "", &logger)

	ctx := context.Background()

	// Test empty URL
	t.Run("EmptyURL", func(t *testing.T) {
		result, err := fetcher.ExtractCompanyInfo(ctx, "")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "company URL cannot be empty")
	})

	// Test invalid URL
	t.Run("InvalidURL", func(t *testing.T) {
		result, err := fetcher.ExtractCompanyInfo(ctx, "not-a-url")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid URL provided")
	})

	// Test URL without scheme
	t.Run("URLWithoutScheme", func(t *testing.T) {
		result, err := fetcher.ExtractCompanyInfo(ctx, "example.com")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid URL provided")
	})

	// Test URL without host
	t.Run("URLWithoutHost", func(t *testing.T) {
		result, err := fetcher.ExtractCompanyInfo(ctx, "http://")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid URL provided")
	})

	// Test valid URLs
	t.Run("ValidURLs", func(t *testing.T) {
		validURLs := []string{
			"https://example.com",
			"http://example.com",
			"https://www.example.com/path",
			"https://subdomain.example.com:8080/path?query=1",
		}

		for _, url := range validURLs {
			result, err := fetcher.ExtractCompanyInfo(ctx, url)

			// Should not fail on validation (might fail on API call, but that's different)
			// It may return an error (like base URL not set) or return nil error with default values
			// The important thing is it should not fail on URL validation
			if err != nil {
				assert.NotContains(t, err.Error(), "invalid URL provided")
			}
			assert.NotNil(t, result)
		}
	})
}

func TestCompanyExtractInfo_DefaultValues(t *testing.T) {
	// For testing purposes, we'll skip the API call test
	// since we can't easily mock the OpenAI client without complex setup
	t.Skip("Skipping OpenAI API integration test - requires proper mocking setup")
}

func TestBuildSystemMessage(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	client := &openai.Client{}
	fetcher := NewCompanyFetcher(client, "", &logger)

	systemMessage := fetcher.buildSystemMessage()

	// Check that system message contains key components
	assert.Contains(t, systemMessage, "company_description")
	assert.Contains(t, systemMessage, "company_targeting_persona")
	assert.Contains(t, systemMessage, "value_offering")
	assert.Contains(t, systemMessage, "Max 30 words")
	assert.Contains(t, systemMessage, "Max 15 words")
	assert.Contains(t, systemMessage, "docusign.com")
	assert.Contains(t, systemMessage, "freshdesk.com")
	assert.Contains(t, systemMessage, "monday.com")
	assert.Contains(t, systemMessage, "unknown-startup.com")
	assert.Contains(t, systemMessage, "JSON")
}

func TestCompanyExtractInfo_Structure(t *testing.T) {
	// Test the structure of CompanyExtractInfo
	info := &CompanyExtractInfo{
		CompanyDescription:      "Test description",
		CompanyTargetingPersona: []string{"Manager", "Developer"},
		ValueOffering:           "Test value",
	}

	assert.Equal(t, "Test description", info.CompanyDescription)
	assert.Equal(t, []string{"Manager", "Developer"}, info.CompanyTargetingPersona)
	assert.Equal(t, "Test value", info.ValueOffering)
}

func TestCompanyExtractInfo_EmptyStructure(t *testing.T) {
	// Test empty CompanyExtractInfo structure
	info := &CompanyExtractInfo{}

	assert.Equal(t, "", info.CompanyDescription)
	assert.Equal(t, []string(nil), info.CompanyTargetingPersona) // Compare with nil instead of empty slice
	assert.Equal(t, "", info.ValueOffering)
}

func TestCompanyFetchContext_Cancellation(t *testing.T) {
	// Skip context cancellation test as it requires actual API call
	t.Skip("Skipping context cancellation test - requires OpenAI client setup")
}

// Integration test examples (would need real API key or more sophisticated mocking)
func TestCompanyExtractInfo_IntegrationExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would be an integration test that actually calls the OpenAI API
	// In a real scenario, you would:
	// 1. Set up a real OpenAI client with API key from environment
	// 2. Make actual API calls
	// 3. Verify the structure and content of responses

	t.Run("ExampleUsage", func(t *testing.T) {
		// Example of how you would use the CompanyFetcher in production
		logger := zerolog.New(zerolog.NewTestWriter(t))

		// In a real test: client := openai.NewClient(option.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
		client := &openai.Client{} // Mock for this example

		fetcher := NewCompanyFetcher(client, "", &logger)

		// Test the structure and validation logic without making actual API calls
		assert.NotNil(t, fetcher)
		assert.NotNil(t, fetcher.buildSystemMessage())

		// Verify URL validation works
		_, err := fetcher.ExtractCompanyInfo(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}
