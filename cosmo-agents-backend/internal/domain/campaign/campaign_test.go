package campaign

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCampaignStatus tests campaign status constants and behaviors
func TestCampaignStatus(t *testing.T) {
	t.Run("All status constants", func(t *testing.T) {
		assert.Equal(t, CampaignStatus("active"), CampaignStatusActive)
		assert.Equal(t, CampaignStatus("ended"), CampaignStatusEnded)
		assert.Equal(t, CampaignStatus("paused"), CampaignStatusPaused)
		assert.Equal(t, CampaignStatus("draft"), CampaignStatusDraft)
		assert.Equal(t, CampaignStatus("scheduled"), CampaignStatusScheduled)
	})

	t.Run("JSON serialization of status", func(t *testing.T) {
		status := CampaignStatusActive
		data, err := json.Marshal(status)
		assert.NoError(t, err)
		assert.Equal(t, `"active"`, string(data))

		var unmarshaled CampaignStatus
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, CampaignStatusActive, unmarshaled)
	})
}

// TestIntentType tests intent type constants and validation
func TestIntentType(t *testing.T) {
	t.Run("All intent constants", func(t *testing.T) {
		assert.Equal(t, IntentType("Interested"), IntentInterested)
		assert.Equal(t, IntentType("Not interested"), IntentNotInterested)
		assert.Equal(t, IntentType("Referral"), IntentReferral)
		assert.Equal(t, IntentType("Request for pricing"), IntentRequestForPricing)
		assert.Equal(t, IntentType("Request for information"), IntentRequestForInfo)
		assert.Equal(t, IntentType("Nurture"), IntentNurture)
		assert.Equal(t, IntentType("Do not contact"), IntentDoNotContact)
		assert.Equal(t, IntentType("Out of office"), IntentOutOfOffice)
		assert.Equal(t, IntentType("Unknown intent"), IntentUnknown)
	})

	t.Run("IsValid method", func(t *testing.T) {
		validIntents := []IntentType{
			IntentInterested,
			IntentNotInterested,
			IntentReferral,
			IntentRequestForPricing,
			IntentRequestForInfo,
			IntentNurture,
			IntentDoNotContact,
			IntentOutOfOffice,
			IntentUnknown,
		}

		for _, intent := range validIntents {
			t.Run(string(intent), func(t *testing.T) {
				assert.True(t, intent.IsValid(), "%s should be valid", intent)
			})
		}

		// Test invalid intent
		invalidIntent := IntentType("Invalid Intent")
		assert.False(t, invalidIntent.IsValid(), "Invalid intent should not be valid")
	})

	t.Run("AllIntents function", func(t *testing.T) {
		allIntents := AllIntents()
		assert.Len(t, allIntents, 9, "Should have 9 valid intent types")

		// Verify all valid intents are included
		expectedIntents := []IntentType{
			IntentInterested,
			IntentNotInterested,
			IntentReferral,
			IntentRequestForPricing,
			IntentRequestForInfo,
			IntentNurture,
			IntentDoNotContact,
			IntentOutOfOffice,
			IntentUnknown,
		}

		for _, expected := range expectedIntents {
			assert.Contains(t, allIntents, expected, "Should contain %s", expected)
		}
	})

	t.Run("Intent JSON serialization", func(t *testing.T) {
		intent := IntentInterested
		data, err := json.Marshal(intent)
		assert.NoError(t, err)
		assert.Equal(t, `"Interested"`, string(data))

		var unmarshaled IntentType
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, IntentInterested, unmarshaled)
	})
}

// TestHandler tests handler constants
func TestHandler(t *testing.T) {
	t.Run("All handler constants", func(t *testing.T) {
		assert.Equal(t, Handler("Let AI reply"), HandlerAI)
		assert.Equal(t, Handler("Assign to a person"), HandlerHuman)
		assert.Equal(t, Handler("Draft an email"), HandlerDraft)
	})

	t.Run("Handler JSON serialization", func(t *testing.T) {
		handler := HandlerAI
		data, err := json.Marshal(handler)
		assert.NoError(t, err)
		assert.Equal(t, `"Let AI reply"`, string(data))
	})
}

// TestCampaignMember tests campaign member structure
func TestCampaignMember(t *testing.T) {
	t.Run("CampaignMember creation", func(t *testing.T) {
		member := CampaignMember{
			Who:        HandlerAI,
			IntentType: IntentInterested,
			Payload:    map[string]interface{}{"key": "value"},
		}

		assert.Equal(t, HandlerAI, member.Who)
		assert.Equal(t, IntentInterested, member.IntentType)
		assert.NotNil(t, member.Payload)
	})

	t.Run("CampaignMember JSON serialization", func(t *testing.T) {
		member := CampaignMember{
			Who:        HandlerHuman,
			IntentType: IntentRequestForPricing,
			Payload:    struct{ Name string }{Name: "John Doe"},
		}

		data, err := json.Marshal(member)
		require.NoError(t, err)

		var unmarshaled CampaignMember
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, HandlerHuman, unmarshaled.Who)
		assert.Equal(t, IntentRequestForPricing, unmarshaled.IntentType)
	})

	t.Run("CampaignMember with nil payload", func(t *testing.T) {
		member := CampaignMember{
			Who:        HandlerDraft,
			IntentType: IntentNotInterested,
			Payload:    nil,
		}

		data, err := json.Marshal(member)
		require.NoError(t, err)

		var unmarshaled CampaignMember
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, HandlerDraft, unmarshaled.Who)
		assert.Equal(t, IntentNotInterested, unmarshaled.IntentType)
	})
}

// TestEmailSequenceItem tests email sequence structure
func TestEmailSequenceItem(t *testing.T) {
	t.Run("EmailSequenceItem creation", func(t *testing.T) {
		item := EmailSequenceItem{
			Type:      "outreach",
			Content:   "Hello {{name}},",
			Subject:   "Introduction",
			ToEmail:   "client@example.com",
			FromEmail: "sender@company.com",
		}

		assert.Equal(t, "outreach", item.Type)
		assert.Equal(t, "Hello {{name}},", item.Content)
		assert.Equal(t, "Introduction", item.Subject)
		assert.Equal(t, "client@example.com", item.ToEmail)
		assert.Equal(t, "sender@company.com", item.FromEmail)
	})

	t.Run("EmailSequenceItem JSON serialization", func(t *testing.T) {
		item := EmailSequenceItem{
			Type:      "follow-up",
			Content:   "Following up on our previous conversation",
			Subject:   "Re: Introduction",
			ToEmail:   "recipient@example.com",
			FromEmail: "sender@example.com",
		}

		data, err := json.Marshal(item)
		require.NoError(t, err)

		var unmarshaled EmailSequenceItem
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, "follow-up", unmarshaled.Type)
		assert.Equal(t, "Following up on our previous conversation", unmarshaled.Content)
	})
}

// TestCampaignMetadata tests campaign metadata structure and methods
func TestCampaignMetadata(t *testing.T) {
	t.Run("CampaignMetadata creation", func(t *testing.T) {
		metadata := CampaignMetadata{
			Config: []CampaignMember{
				{Who: HandlerAI, IntentType: IntentInterested},
				{Who: HandlerHuman, IntentType: IntentRequestForPricing},
			},
			Sequence: []EmailSequenceItem{
				{Type: "outreach", Subject: "Hello"},
			},
			Client: map[string]interface{}{
				"name": "Test Client",
				"size": "Enterprise",
			},
		}

		assert.Len(t, metadata.Config, 2)
		assert.Len(t, metadata.Sequence, 1)
		assert.Equal(t, "Test Client", metadata.Client["name"])
	})

	t.Run("CampaignMetadata Scan from bytes", func(t *testing.T) {
		testData := map[string]interface{}{
			"config": []map[string]interface{}{
				{"who": "Let AI reply", "intent_type": "Interested"},
				{"who": "Assign to a person", "intent_type": "Request for pricing"},
			},
			"sequence": []map[string]interface{}{
				{"type": "outreach", "subject": "Test"},
			},
			"client": map[string]interface{}{
				"name": "Test Client",
			},
		}

		jsonData, err := json.Marshal(testData)
		require.NoError(t, err)

		var metadata CampaignMetadata
		err = metadata.Scan(jsonData)
		assert.NoError(t, err)

		assert.Len(t, metadata.Config, 2)
		assert.Len(t, metadata.Sequence, 1)
		assert.Equal(t, "Test Client", metadata.Client["name"])
	})

	t.Run("CampaignMetadata Scan from nil", func(t *testing.T) {
		var metadata CampaignMetadata
		err := metadata.Scan(nil)
		assert.NoError(t, err)
		assert.Len(t, metadata.Config, 0)
	})

	t.Run("CampaignMetadata Scan with invalid type", func(t *testing.T) {
		var metadata CampaignMetadata
		err := metadata.Scan("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type assertion to []byte failed")
	})

	t.Run("CampaignMetadata Value method", func(t *testing.T) {
		metadata := CampaignMetadata{
			Config: []CampaignMember{
				{Who: HandlerAI, IntentType: IntentInterested},
			},
			Client: map[string]interface{}{
				"test": "value",
			},
		}

		value, err := metadata.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)

		// Verify it's valid JSON
		var unmarshaled map[string]interface{}
		err = json.Unmarshal(value.([]byte), &unmarshaled)
		assert.NoError(t, err)
		assert.Contains(t, unmarshaled, "config")
		assert.Contains(t, unmarshaled, "client")
	})

	t.Run("CampaignMetadata empty Value", func(t *testing.T) {
		metadata := CampaignMetadata{
			Config:   []CampaignMember{},
			Sequence: []EmailSequenceItem{},
			Client:   map[string]interface{}{},
		}

		value, err := metadata.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)
	})
}

// TestSaleRepNodeState tests sale rep node state
func TestSaleRepNodeState(t *testing.T) {
	t.Run("SaleRepNodeState creation", func(t *testing.T) {
		state := SaleRepNodeState{
			RRobinCount: 5,
		}

		assert.Equal(t, 5, state.RRobinCount)
	})

	t.Run("SaleRepNodeState JSON", func(t *testing.T) {
		state := SaleRepNodeState{RRobinCount: 3}
		data, err := json.Marshal(state)
		assert.NoError(t, err)

		var unmarshaled SaleRepNodeState
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, 3, unmarshaled.RRobinCount)
	})
}

// TestSaleRepNodeStates tests sale rep node states map
func TestSaleRepNodeStates(t *testing.T) {
	t.Run("SaleRepNodeStates creation", func(t *testing.T) {
		states := make(SaleRepNodeStates)
		states[IntentInterested] = SaleRepNodeState{RRobinCount: 1}
		states[IntentNotInterested] = SaleRepNodeState{RRobinCount: 2}

		assert.Equal(t, 1, states[IntentInterested].RRobinCount)
		assert.Equal(t, 2, states[IntentNotInterested].RRobinCount)
	})

	t.Run("SaleRepNodeStates Scan from bytes", func(t *testing.T) {
		testData := map[string]SaleRepNodeState{
			"Interested":          {RRobinCount: 3},
			"Not interested":      {RRobinCount: 5},
			"Request for pricing": {RRobinCount: 7},
		}

		jsonData, err := json.Marshal(testData)
		require.NoError(t, err)

		var states SaleRepNodeStates
		err = states.Scan(jsonData)
		assert.NoError(t, err)

		assert.Equal(t, 3, states[IntentInterested].RRobinCount)
		assert.Equal(t, 5, states[IntentNotInterested].RRobinCount)
		assert.Equal(t, 7, states[IntentRequestForPricing].RRobinCount)
	})

	t.Run("SaleRepNodeStates Scan from nil", func(t *testing.T) {
		var states SaleRepNodeStates
		err := states.Scan(nil)
		assert.NoError(t, err)
		assert.Nil(t, states)
	})

	t.Run("SaleRepNodeStates Value method", func(t *testing.T) {
		states := make(SaleRepNodeStates)
		states[IntentInterested] = SaleRepNodeState{RRobinCount: 10}

		value, err := states.Value()
		assert.NoError(t, err)
		assert.NotNil(t, value)

		// Verify it's valid JSON
		var unmarshaled map[string]interface{}
		err = json.Unmarshal(value.([]byte), &unmarshaled)
		assert.NoError(t, err)
	})

	t.Run("SaleRepNodeStates nil Value", func(t *testing.T) {
		var states SaleRepNodeStates
		value, err := states.Value()
		assert.NoError(t, err)
		assert.Nil(t, value)
	})
}

// TestCampaign tests the main Campaign struct
func TestCampaign(t *testing.T) {
	t.Run("Campaign creation", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		agentID := uuid.New()
		now := time.Now()

		campaign := Campaign{
			Name:           "Test Campaign",
			Playbook:       "test-playbook",
			UserID:         userID,
			OrganizationID: &orgID,
			Status:         CampaignStatusActive,
			AgentID:        &agentID,
			Schedule:       &now,
			CMetadata: CampaignMetadata{
				Config: []CampaignMember{
					{Who: HandlerAI, IntentType: IntentInterested},
				},
			},
		}

		assert.Equal(t, "Test Campaign", campaign.Name)
		assert.Equal(t, "test-playbook", campaign.Playbook)
		assert.Equal(t, userID, campaign.UserID)
		assert.Equal(t, &orgID, campaign.OrganizationID)
		assert.Equal(t, CampaignStatusActive, campaign.Status)
		assert.Equal(t, &agentID, campaign.AgentID)
		assert.Equal(t, &now, campaign.Schedule)
		assert.Len(t, campaign.CMetadata.Config, 1)
	})

	t.Run("Campaign TableName", func(t *testing.T) {
		campaign := Campaign{}
		assert.Equal(t, "campaigns", campaign.TableName())
	})

	t.Run("Campaign BeforeCreate with existing name", func(t *testing.T) {
		campaign := &Campaign{
			Name:     "Existing Name",
			Playbook: "test-playbook",
		}

		err := campaign.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "Existing Name", campaign.Name) // Should not change
	})

	t.Run("Campaign BeforeCreate with playbook but no name", func(t *testing.T) {
		testCases := []struct {
			playbook string
			expected string
		}{
			{"test-playbook", "Test Playbook"},
			{"my-playbook_name", "My Playbook Name"},
			{"ANOTHER-PLAYBOOK", "ANOTHER PLAYBOOK"},
			{"simple", "Simple"},
			{"", ""}, // Empty playbook
		}

		for _, tc := range testCases {
			t.Run(tc.playbook, func(t *testing.T) {
				campaign := &Campaign{
					Playbook: tc.playbook,
				}

				err := campaign.BeforeCreate(nil)
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, campaign.Name)
			})
		}
	})

	t.Run("Campaign BeforeCreate initializes CMetadata", func(t *testing.T) {
		campaign := &Campaign{
			Name:     "Test",
			Playbook: "test",
		}

		err := campaign.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotNil(t, campaign.CMetadata.Config)
		assert.NotNil(t, campaign.CMetadata.Sequence)
		assert.NotNil(t, campaign.CMetadata.Client)
	})

	t.Run("Campaign BeforeCreate with nil Base", func(t *testing.T) {
		campaign := &Campaign{}
		campaign.Base.ID = uuid.Nil

		err := campaign.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, campaign.ID)
	})

	t.Run("GetIntentAssignee", func(t *testing.T) {
		config := []CampaignMember{
			{Who: HandlerAI, IntentType: IntentInterested},
			{Who: HandlerHuman, IntentType: IntentRequestForPricing},
		}
		campaign := &Campaign{
			CMetadata: CampaignMetadata{Config: config},
		}

		// Test existing intent
		assignee := campaign.GetIntentAssignee(IntentInterested)
		assert.NotNil(t, assignee)
		assert.Equal(t, HandlerAI, assignee.Who)
		assert.Equal(t, IntentInterested, assignee.IntentType)

		// Test non-existing intent
		assignee = campaign.GetIntentAssignee(IntentNotInterested)
		assert.Nil(t, assignee)
	})

	t.Run("SetIntentAssignee new intent", func(t *testing.T) {
		campaign := &Campaign{
			CMetadata: CampaignMetadata{Config: []CampaignMember{}},
		}

		newAssignee := CampaignMember{
			Who:        HandlerDraft,
			IntentType: IntentNotInterested,
			Payload:    map[string]string{"test": "value"},
		}

		campaign.SetIntentAssignee(newAssignee)
		assert.Len(t, campaign.CMetadata.Config, 1)
		assert.Equal(t, HandlerDraft, campaign.CMetadata.Config[0].Who)
		assert.Equal(t, IntentNotInterested, campaign.CMetadata.Config[0].IntentType)
	})

	t.Run("SetIntentAssignee update existing", func(t *testing.T) {
		existingConfig := []CampaignMember{
			{Who: HandlerAI, IntentType: IntentInterested},
		}
		campaign := &Campaign{
			CMetadata: CampaignMetadata{Config: existingConfig},
		}

		updatedAssignee := CampaignMember{
			Who:        HandlerHuman,
			IntentType: IntentInterested,
			Payload:    map[string]interface{}{"updated": true},
		}

		campaign.SetIntentAssignee(updatedAssignee)
		assert.Len(t, campaign.CMetadata.Config, 1) // Should still be 1, not 2
		assert.Equal(t, HandlerHuman, campaign.CMetadata.Config[0].Who)
		assert.Equal(t, true, campaign.CMetadata.Config[0].Payload.(map[string]interface{})["updated"])
	})

	t.Run("GetAndIncrSaleRepCounter", func(t *testing.T) {
		campaign := &Campaign{}

		// First call should initialize and return 0
		count := campaign.GetAndIncrSaleRepCounter(IntentInterested)
		assert.Equal(t, 0, count)
		assert.NotNil(t, campaign.SaleRepNodeStates)
		assert.Equal(t, 1, campaign.SaleRepNodeStates[IntentInterested].RRobinCount)

		// Second call should return 1
		count = campaign.GetAndIncrSaleRepCounter(IntentInterested)
		assert.Equal(t, 1, count)
		assert.Equal(t, 2, campaign.SaleRepNodeStates[IntentInterested].RRobinCount)

		// Different intent should start from 0
		count = campaign.GetAndIncrSaleRepCounter(IntentNotInterested)
		assert.Equal(t, 0, count)
		assert.Equal(t, 1, campaign.SaleRepNodeStates[IntentNotInterested].RRobinCount)
	})

	t.Run("Campaign JSON serialization", func(t *testing.T) {
		userID := uuid.New()
		campaign := Campaign{
			Name:   "Test Campaign",
			UserID: userID,
			Status: CampaignStatusActive,
			CMetadata: CampaignMetadata{
				Config: []CampaignMember{
					{Who: HandlerAI, IntentType: IntentInterested},
				},
			},
		}

		data, err := json.Marshal(campaign)
		require.NoError(t, err)

		var unmarshaled Campaign
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, "Test Campaign", unmarshaled.Name)
		assert.Equal(t, userID, unmarshaled.UserID)
		assert.Equal(t, CampaignStatusActive, unmarshaled.Status)
		assert.Len(t, unmarshaled.CMetadata.Config, 1)
	})
}

// TestCampaignEdgeCases tests edge cases and error conditions
func TestCampaignEdgeCases(t *testing.T) {
	t.Run("Campaign with all optional fields nil", func(t *testing.T) {
		userID := uuid.New()
		campaign := Campaign{
			UserID: userID,
		}

		assert.Nil(t, campaign.OrganizationID)
		assert.Nil(t, campaign.ListContactID)
		assert.Nil(t, campaign.Schedule)
		assert.Nil(t, campaign.AgentID)
	})

	t.Run("Campaign with empty CMetadata", func(t *testing.T) {
		campaign := &Campaign{
			CMetadata: CampaignMetadata{},
		}

		assignee := campaign.GetIntentAssignee(IntentInterested)
		assert.Nil(t, assignee)

		campaign.SetIntentAssignee(CampaignMember{
			Who:        HandlerAI,
			IntentType: IntentInterested,
		})
		assert.Len(t, campaign.CMetadata.Config, 1)
	})

	t.Run("Campaign with complex payloads", func(t *testing.T) {
		aiPayload := map[string]interface{}{
			"model":       "gpt-4",
			"temperature": 0.7,
			"max_tokens":  1000,
		}

		humanPayload := map[string]interface{}{
			"user_id":   uuid.New().String(),
			"team_name": "Sales Team A",
			"priority":  1,
		}

		config := []CampaignMember{
			{Who: HandlerAI, IntentType: IntentInterested, Payload: aiPayload},
			{Who: HandlerHuman, IntentType: IntentRequestForPricing, Payload: humanPayload},
		}

		campaign := &Campaign{
			CMetadata: CampaignMetadata{Config: config},
		}

		// Test retrieving AI config
		aiAssignee := campaign.GetIntentAssignee(IntentInterested)
		assert.NotNil(t, aiAssignee)
		assert.Equal(t, HandlerAI, aiAssignee.Who)

		aiPayloadMap := aiAssignee.Payload.(map[string]interface{})
		assert.Equal(t, "gpt-4", aiPayloadMap["model"])
		assert.Equal(t, 0.7, aiPayloadMap["temperature"])

		// Test retrieving human config
		humanAssignee := campaign.GetIntentAssignee(IntentRequestForPricing)
		assert.NotNil(t, humanAssignee)
		assert.Equal(t, HandlerHuman, humanAssignee.Who)

		humanPayloadMap := humanAssignee.Payload.(map[string]interface{})
		assert.NotNil(t, humanPayloadMap["user_id"])
		assert.Equal(t, "Sales Team A", humanPayloadMap["team_name"])
	})

	t.Run("Campaign with multiple sequence items", func(t *testing.T) {
		sequence := []EmailSequenceItem{
			{Type: "outreach", Subject: "Introduction"},
			{Type: "follow-up", Subject: "Checking in"},
			{Type: "closing", Subject: "Last call"},
		}

		campaign := &Campaign{
			CMetadata: CampaignMetadata{
				Sequence: sequence,
			},
		}

		assert.Len(t, campaign.CMetadata.Sequence, 3)
		assert.Equal(t, "Introduction", campaign.CMetadata.Sequence[0].Subject)
		assert.Equal(t, "follow-up", campaign.CMetadata.Sequence[1].Type)
		assert.Equal(t, "Last call", campaign.CMetadata.Sequence[2].Subject)
	})

	t.Run("Campaign with round-robin states", func(t *testing.T) {
		states := SaleRepNodeStates{
			IntentInterested:        {RRobinCount: 5},
			IntentNotInterested:     {RRobinCount: 3},
			IntentRequestForPricing: {RRobinCount: 7},
		}

		campaign := &Campaign{
			SaleRepNodeStates: states,
		}

		// Verify existing states
		assert.Equal(t, 5, campaign.SaleRepNodeStates[IntentInterested].RRobinCount)
		assert.Equal(t, 3, campaign.SaleRepNodeStates[IntentNotInterested].RRobinCount)
		assert.Equal(t, 7, campaign.SaleRepNodeStates[IntentRequestForPricing].RRobinCount)

		// Test incrementing existing state
		count := campaign.GetAndIncrSaleRepCounter(IntentInterested)
		assert.Equal(t, 5, count) // Returns current value
		assert.Equal(t, 6, campaign.SaleRepNodeStates[IntentInterested].RRobinCount)

		// Test adding new state
		count = campaign.GetAndIncrSaleRepCounter(IntentOutOfOffice)
		assert.Equal(t, 0, count)
		assert.Equal(t, 1, campaign.SaleRepNodeStates[IntentOutOfOffice].RRobinCount)
	})
}

// TestCampaignIntegration tests integration scenarios
func TestCampaignIntegration(t *testing.T) {
	t.Run("Complete campaign lifecycle", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		agentID := uuid.New()

		// Create campaign
		campaign := &Campaign{
			Playbook:       "enterprise-outreach",
			UserID:         userID,
			OrganizationID: &orgID,
			Status:         CampaignStatusDraft,
			AgentID:        &agentID,
		}

		// BeforeCreate should set name and initialize metadata
		err := campaign.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "Enterprise Outreach", campaign.Name)
		assert.NotNil(t, campaign.ID)

		// Add configurations
		campaign.SetIntentAssignee(CampaignMember{
			Who:        HandlerAI,
			IntentType: IntentInterested,
			Payload:    map[string]string{"model": "gpt-4"},
		})

		campaign.SetIntentAssignee(CampaignMember{
			Who:        HandlerHuman,
			IntentType: IntentRequestForPricing,
			Payload:    map[string]string{"team": "Enterprise Sales"},
		})

		// Verify configurations
		aiConfig := campaign.GetIntentAssignee(IntentInterested)
		assert.NotNil(t, aiConfig)
		assert.Equal(t, HandlerAI, aiConfig.Who)

		humanConfig := campaign.GetIntentAssignee(IntentRequestForPricing)
		assert.NotNil(t, humanConfig)
		assert.Equal(t, HandlerHuman, humanConfig.Who)

		// Test round-robin assignment
		for i := 0; i < 3; i++ {
			count := campaign.GetAndIncrSaleRepCounter(IntentInterested)
			assert.Equal(t, i, count)
		}

		// Update status
		campaign.Status = CampaignStatusActive
		assert.Equal(t, CampaignStatusActive, campaign.Status)

		// Serialize and deserialize
		data, err := json.Marshal(campaign)
		require.NoError(t, err)

		var restored Campaign
		err = json.Unmarshal(data, &restored)
		assert.NoError(t, err)

		assert.Equal(t, campaign.Name, restored.Name)
		assert.Equal(t, campaign.Playbook, restored.Playbook)
		assert.Equal(t, campaign.UserID, restored.UserID)
		assert.Equal(t, campaign.Status, restored.Status)
		assert.Len(t, restored.CMetadata.Config, 2)
	})

	t.Run("Campaign with all handlers", func(t *testing.T) {
		campaign := &Campaign{
			Name: "Multi-Handler Campaign",
			CMetadata: CampaignMetadata{
				Config: []CampaignMember{
					{Who: HandlerAI, IntentType: IntentInterested},
					{Who: HandlerHuman, IntentType: IntentRequestForPricing},
					{Who: HandlerDraft, IntentType: IntentNurture},
					{Who: HandlerAI, IntentType: IntentReferral},
					{Who: HandlerHuman, IntentType: IntentRequestForInfo},
				},
			},
		}

		// Test all intent configurations
		testCases := []struct {
			intent      IntentType
			expectedWho Handler
		}{
			{IntentInterested, HandlerAI},
			{IntentRequestForPricing, HandlerHuman},
			{IntentNurture, HandlerDraft},
			{IntentReferral, HandlerAI},
			{IntentRequestForInfo, HandlerHuman},
		}

		for _, tc := range testCases {
			assignee := campaign.GetIntentAssignee(tc.intent)
			assert.NotNil(t, assignee, "Should have config for %s", tc.intent)
			assert.Equal(t, tc.expectedWho, assignee.Who, "Wrong handler for %s", tc.intent)
		}
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	t.Run("Large campaign config", func(t *testing.T) {
		// Create campaign with many intent configurations
		var config []CampaignMember
		for i := 0; i < 1000; i++ {
			config = append(config, CampaignMember{
				Who:        HandlerAI,
				IntentType: IntentInterested,
				Payload:    map[string]int{"index": i},
			})
		}

		campaign := &Campaign{
			CMetadata: CampaignMetadata{Config: config},
		}

		// Test retrieval performance
		start := time.Now()
		for i := 0; i < 100; i++ {
			assignee := campaign.GetIntentAssignee(IntentInterested)
			assert.NotNil(t, assignee)
		}
		duration := time.Since(start)

		assert.Less(t, duration, 100*time.Millisecond, "100 lookups should complete quickly")
	})

	t.Run("Many round-robin increments", func(t *testing.T) {
		campaign := &Campaign{}

		start := time.Now()
		for i := 0; i < 10000; i++ {
			campaign.GetAndIncrSaleRepCounter(IntentInterested)
		}
		duration := time.Since(start)

		assert.Less(t, duration, 100*time.Millisecond, "10000 increments should complete quickly")
		assert.Equal(t, 10000, campaign.SaleRepNodeStates[IntentInterested].RRobinCount)
	})
}
