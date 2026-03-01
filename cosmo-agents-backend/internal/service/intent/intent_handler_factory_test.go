package intent

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func TestIntentHandlerFactory_PriorityHandlers(t *testing.T) {
	factory := NewIntentHandlerFactory(nil, nil, nil, nil, nil, nil, nil, nil, &zerolog.Logger{})
	campaign := &domain.Campaign{Base: domain.Base{ID: uuid.New()}, UserID: uuid.New()}

	handler, err := factory.Build(campaign, domain.IntentDoNotContact)
	require.NoError(t, err)
	assert.IsType(t, &DoNotContactHandler{}, handler)

	handler, err = factory.Build(campaign, domain.IntentOutOfOffice)
	require.NoError(t, err)
	assert.IsType(t, &OutOfOfficeHandler{}, handler)

	handler, err = factory.Build(campaign, domain.IntentUnknown)
	require.NoError(t, err)
	assert.IsType(t, &AssignToPersonHandler{}, handler)
}

func TestIntentHandlerFactory_ConfiguredHandlers(t *testing.T) {
	factory := NewIntentHandlerFactory(nil, nil, nil, nil, nil, nil, nil, nil, &zerolog.Logger{})
	campaign := &domain.Campaign{Base: domain.Base{ID: uuid.New()}, UserID: uuid.New()}

	// AI handler
	campaign.SetIntentAssignee(domain.CampaignMember{IntentType: domain.IntentInterested, Who: domain.HandlerAI})
	handler, err := factory.Build(campaign, domain.IntentInterested)
	require.NoError(t, err)
	assert.IsType(t, &AIReplyHandler{}, handler)

	// Draft handler
	campaign.SetIntentAssignee(domain.CampaignMember{IntentType: domain.IntentNotInterested, Who: domain.HandlerDraft})
	handler, err = factory.Build(campaign, domain.IntentNotInterested)
	require.NoError(t, err)
	assert.IsType(t, &DraftReplyHandler{}, handler)

	// Human handler with payload
	userID := uuid.New()
	campaign.SetIntentAssignee(domain.CampaignMember{
		IntentType: domain.IntentRequestForPricing,
		Who:        domain.HandlerHuman,
		Payload:    map[string]interface{}{"user_id": userID.String()},
	})
	handler, err = factory.Build(campaign, domain.IntentRequestForPricing)
	require.NoError(t, err)
	assert.IsType(t, &AssignToPersonHandler{}, handler)

	// Missing config returns nil
	handler, err = factory.Build(campaign, domain.IntentRequestForInfo)
	require.NoError(t, err)
	assert.Nil(t, handler)
}

func TestIntentHandlerFactory_InvalidPayload(t *testing.T) {
	factory := NewIntentHandlerFactory(nil, nil, nil, nil, nil, nil, nil, nil, &zerolog.Logger{})
	campaign := &domain.Campaign{Base: domain.Base{ID: uuid.New()}, UserID: uuid.New()}
	campaign.SetIntentAssignee(domain.CampaignMember{
		IntentType: domain.IntentRequestForPricing,
		Who:        domain.HandlerHuman,
		Payload:    map[string]interface{}{"user": "bad"},
	})

	handler, err := factory.Build(campaign, domain.IntentRequestForPricing)
	require.NoError(t, err)
	assert.Nil(t, handler)
}
