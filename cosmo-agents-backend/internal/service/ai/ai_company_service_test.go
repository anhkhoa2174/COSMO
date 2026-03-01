package ai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/cache"
)

func TestAICompanyService_ExtractCompanyInfo(t *testing.T) {
	backend := cache.NewMemoryBackend(1 << 20)
	manager := cache.NewManager(backend, nil)
	service := NewAICompanyService(nil, manager)
	ctx := context.Background()

	_, err := service.ExtractCompanyInfo(ctx, " ")
	assert.ErrorIs(t, err, ErrInvalidCompanyURL)

	companyURL := "https://example.com"
	cacheKey := "ai:company:" + companyURL
	info := &domain.CompanyInfo{CompanyDescription: "cached", ValueOffering: "value", CompanyTargetingPersona: []string{"persona"}}
	require.NoError(t, manager.Set(ctx, cacheKey, info, time.Hour))

	fromCache, err := service.ExtractCompanyInfo(ctx, companyURL)
	require.NoError(t, err)
	assert.Equal(t, "cached", fromCache.CompanyDescription)

	// Cache miss with missing client returns configuration error
	serviceNoClient := NewAICompanyService(nil, manager)
	_, err = serviceNoClient.ExtractCompanyInfo(ctx, "https://newsite.com")
	assert.ErrorIs(t, err, ErrAIClientNotConfigured)
}

func TestAICompanyService_GenerateCompanyInfoSuccess(t *testing.T) {
	response := `{"choices":[{"message":{"content":"{\"company_description\":\"desc\",\"company_targeting_persona\":[\"p1\",\"\"],\"value_offering\":\"offer\"}"}}]}`
	client := newPatchedAIClient(t, response)
	service := NewAICompanyService(client, cache.NewManager(cache.NewMemoryBackend(1<<20), nil))

	info, err := service.ExtractCompanyInfo(context.Background(), "https://acme.com")
	require.NoError(t, err)
	assert.Equal(t, "desc", info.CompanyDescription)
	assert.Equal(t, []string{"p1"}, info.CompanyTargetingPersona)
	assert.Equal(t, "offer", info.ValueOffering)
}
