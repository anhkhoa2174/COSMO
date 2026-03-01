package contact

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// TestContactSource tests contact source constants and validation
func TestContactSource(t *testing.T) {
	t.Run("All source constants", func(t *testing.T) {
		assert.Equal(t, ContactSource("cosmo-agents"), ContactSourceCosmoAgents)
		assert.Equal(t, ContactSource("google-ads"), ContactSourceGoogleAds)
		assert.Equal(t, ContactSource("csv"), ContactSourceCSV)
		assert.Equal(t, ContactSource("hubspot"), ContactSourceHubspot)
		assert.Equal(t, ContactSource("Apollo"), ContactSourceApollo)
		assert.Equal(t, ContactSource("facebook-ads"), ContactSourceFacebookAds)
		assert.Equal(t, ContactSource("tiktok-ads"), ContactSourceTiktokAds)
	})

	t.Run("Source string values", func(t *testing.T) {
		sources := []ContactSource{
			ContactSourceCosmoAgents,
			ContactSourceGoogleAds,
			ContactSourceCSV,
			ContactSourceHubspot,
			ContactSourceApollo,
			ContactSourceFacebookAds,
			ContactSourceTiktokAds,
		}

		for _, source := range sources {
			assert.NotEmpty(t, string(source))
			assert.NotEqual(t, "", string(source))
		}
	})
}

// TestContactConstants tests domain constants
func TestContactConstants(t *testing.T) {
	t.Run("NOT_AVAILABLE constant", func(t *testing.T) {
		assert.Equal(t, "N/A", NOT_AVAILABLE)
		assert.NotEmpty(t, NOT_AVAILABLE)
	})
}

// TestContactStructure tests the Contact struct and its fields
func TestContactStructure(t *testing.T) {
	t.Run("Contact struct initialization", func(t *testing.T) {
		userID := uuid.New()
		contactID := uuid.New()
		orgID := uuid.New()
		hubspotID := "hubspot_123"

		contact := &Contact{
			Base:            base.Base{ID: contactID},
			TimestampMixin:  base.TimestampMixin{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			SoftDeleteMixin: base.SoftDeleteMixin{IsDeleted: false},
			UserID:          userID,
			SourceID:        "source_123",
			HubspotID:       &hubspotID,
			Source:          string(ContactSourceCSV),
			Name:            "John Doe",
			Company:         "Acme Corp",
			JobTitle:        "Software Engineer",
			Address:         "123 Main St",
			City:            "San Francisco",
			Country:         "USA",
			State:           "CA",
			Zip:             "94105",
			Profile:         base.JSONB(`{"email":"john.doe@example.com","phone":"+1234567890"}`),
			DoNotContact:    false,
			OrganizationID:  &orgID,
		}

		// Verify all fields are set correctly
		assert.Equal(t, contactID, contact.ID)
		assert.Equal(t, userID, contact.UserID)
		assert.Equal(t, "source_123", contact.SourceID)
		assert.Equal(t, &hubspotID, contact.HubspotID)
		assert.Equal(t, string(ContactSourceCSV), contact.Source)
		assert.Equal(t, "John Doe", contact.Name)
		assert.Equal(t, "Acme Corp", contact.Company)
		assert.Equal(t, "Software Engineer", contact.JobTitle)
		assert.Equal(t, "123 Main St", contact.Address)
		assert.Equal(t, "San Francisco", contact.City)
		assert.Equal(t, "USA", contact.Country)
		assert.Equal(t, "CA", contact.State)
		assert.Equal(t, "94105", contact.Zip)
		assert.False(t, contact.DoNotContact)
		assert.Equal(t, &orgID, contact.OrganizationID)

		var profile map[string]interface{}
		require.NoError(t, contact.Profile.Unmarshal(&profile))
		assert.Equal(t, "john.doe@example.com", profile["email"])
		assert.Equal(t, "+1234567890", profile["phone"])
	})

	t.Run("Contact with zero values", func(t *testing.T) {
		contact := &Contact{}

		// Verify zero values
		assert.Equal(t, uuid.Nil, contact.ID)
		assert.Equal(t, uuid.Nil, contact.UserID)
		assert.Equal(t, "", contact.SourceID)
		assert.Nil(t, contact.HubspotID)
		assert.Equal(t, "", contact.Source)
		assert.Equal(t, "", contact.Name)
		assert.Equal(t, "", contact.Company)
		assert.Equal(t, "", contact.JobTitle)
		assert.Equal(t, "", contact.Address)
		assert.Equal(t, "", contact.City)
		assert.Equal(t, "", contact.Country)
		assert.Equal(t, "", contact.State)
		assert.Equal(t, "", contact.Zip)
		assert.False(t, contact.DoNotContact)
		assert.Nil(t, contact.OrganizationID)
	})
}

// TestContactTableName tests the TableName method
func TestContactTableName(t *testing.T) {
	t.Run("Contact table name", func(t *testing.T) {
		contact := &Contact{}
		assert.Equal(t, "contacts", contact.TableName())
	})
}

// TestContactBeforeCreate tests the BeforeCreate hook
func TestContactBeforeCreate(t *testing.T) {
	t.Run("BeforeCreate sets default SourceID", func(t *testing.T) {
		contact := &Contact{
			UserID: uuid.New(),
			Source: string(ContactSourceGoogleAds),
		}

		err := contact.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, contact.ID)
		assert.Equal(t, contact.ID.String(), contact.SourceID)
		assert.Equal(t, string(ContactSourceGoogleAds), contact.Source)
	})

	t.Run("BeforeCreate preserves existing SourceID", func(t *testing.T) {
		existingSourceID := "existing_source_123"
		contact := &Contact{
			UserID:   uuid.New(),
			SourceID: existingSourceID,
			Source:   string(ContactSourceCSV),
		}

		err := contact.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.Equal(t, existingSourceID, contact.SourceID)
	})

	t.Run("BeforeCreate sets default Source", func(t *testing.T) {
		contact := &Contact{
			UserID: uuid.New(),
		}

		err := contact.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.Equal(t, string(ContactSourceCosmoAgents), contact.Source)
	})

	t.Run("BeforeCreate preserves existing Source", func(t *testing.T) {
		contact := &Contact{
			UserID:   uuid.New(),
			Source:   string(ContactSourceApollo),
			SourceID: "test_source",
		}

		err := contact.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.Equal(t, string(ContactSourceApollo), contact.Source)
	})

	t.Run("BeforeCreate with nil transaction", func(t *testing.T) {
		contact := &Contact{
			UserID: uuid.New(),
		}

		err := contact.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, contact.ID)
	})
}

// TestContactJSONSerialization tests JSON marshaling and unmarshaling
func TestContactJSONSerialization(t *testing.T) {
	t.Run("Complete contact serialization", func(t *testing.T) {
		userID := uuid.New()
		contactID := uuid.New()
		orgID := uuid.New()
		hubspotID := "hubspot_123"

		profileData := base.JSONB(`{"email":"john.doe@example.com","phone":"+1234567890","skills":["Go","Python"],"experience":5}`)
		tagsData := base.JSONB(`{"tag1": "value1", "tag2": "value2"}`)

		contact := &Contact{
			Base:            base.Base{ID: contactID},
			TimestampMixin:  base.TimestampMixin{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			SoftDeleteMixin: base.SoftDeleteMixin{IsDeleted: false},
			UserID:          userID,
			SourceID:        "source_123",
			HubspotID:       &hubspotID,
			Source:          string(ContactSourceCSV),
			Name:            "John Doe",
			Company:         "Acme Corp",
			JobTitle:        "Software Engineer",
			Address:         "123 Main St",
			City:            "San Francisco",
			Country:         "USA",
			State:           "CA",
			Zip:             "94105",
			Profile:         profileData,
			DoNotContact:    false,
			OrganizationID:  &orgID,
			Tags:            tagsData,
		}

		// Serialize to JSON
		jsonData, err := json.Marshal(contact)
		require.NoError(t, err)

		// Deserialize back
		var deserialized Contact
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err)

		// Verify key fields
		assert.Equal(t, contactID, deserialized.ID)
		assert.Equal(t, userID, deserialized.UserID)
		assert.Equal(t, "source_123", deserialized.SourceID)
		assert.Equal(t, &hubspotID, deserialized.HubspotID)
		assert.Equal(t, string(ContactSourceCSV), deserialized.Source)
		assert.Equal(t, "John Doe", deserialized.Name)
		assert.Equal(t, "Acme Corp", deserialized.Company)
		assert.Equal(t, "Software Engineer", deserialized.JobTitle)
		assert.Equal(t, "123 Main St", deserialized.Address)
		assert.Equal(t, "San Francisco", deserialized.City)
		assert.Equal(t, "USA", deserialized.Country)
		assert.Equal(t, "CA", deserialized.State)
		assert.Equal(t, "94105", deserialized.Zip)
		assert.False(t, deserialized.DoNotContact)
		assert.Equal(t, &orgID, deserialized.OrganizationID)

		var profile map[string]interface{}
		require.NoError(t, deserialized.Profile.Unmarshal(&profile))
		assert.Equal(t, "john.doe@example.com", profile["email"])
		assert.Equal(t, "+1234567890", profile["phone"])
	})

	t.Run("Contact with minimal data serialization", func(t *testing.T) {
		userID := uuid.New()
		contact := &Contact{
			UserID:  userID,
			Profile: base.JSONB(`{"email":"test@example.com"}`),
		}

		// Set ID through BeforeCreate
		err := contact.BeforeCreate(nil)
		require.NoError(t, err)

		// Serialize to JSON
		jsonData, err := json.Marshal(contact)
		require.NoError(t, err)

		// Deserialize back
		var deserialized Contact
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, contact.ID, deserialized.ID)
		assert.Equal(t, userID, deserialized.UserID)
		assert.Equal(t, string(ContactSourceCosmoAgents), deserialized.Source)

		var profile map[string]interface{}
		require.NoError(t, deserialized.Profile.Unmarshal(&profile))
		assert.Equal(t, "test@example.com", profile["email"])
	})

	t.Run("Contact with nil optional fields", func(t *testing.T) {
		userID := uuid.New()
		contact := &Contact{
			UserID:  userID,
			Profile: base.JSONB(`{"email":"test@example.com"}`),
		}

		// Set ID through BeforeCreate
		err := contact.BeforeCreate(nil)
		require.NoError(t, err)

		jsonData, err := json.Marshal(contact)
		require.NoError(t, err)

		var deserialized Contact
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err)

		assert.Nil(t, deserialized.HubspotID)
		assert.Nil(t, deserialized.OrganizationID)
	})
}

// TestContactProfileAndTags tests JSONB fields
func TestContactProfileAndTags(t *testing.T) {
	t.Run("Profile JSONB operations", func(t *testing.T) {
		contact := &Contact{}
		profileData := `{"linkedin": "https://linkedin.com/in/johndoe", "website": "https://johndoe.com"}`

		err := contact.Profile.Scan(profileData)
		assert.NoError(t, err)
		assert.Equal(t, base.JSONB(profileData), contact.Profile)

		// Test marshal into struct
		type Profile struct {
			LinkedIn string `json:"linkedin"`
			Website  string `json:"website"`
		}

		var profile Profile
		err = contact.Profile.Unmarshal(&profile)
		assert.NoError(t, err)
		assert.Equal(t, "https://linkedin.com/in/johndoe", profile.LinkedIn)
		assert.Equal(t, "https://johndoe.com", profile.Website)
	})

	t.Run("Tags JSONB operations", func(t *testing.T) {
		contact := &Contact{}
		tagsData := `{"source": "referral", "priority": "high", "category": "enterprise"}`

		err := contact.Tags.Scan(tagsData)
		assert.NoError(t, err)
		assert.Equal(t, base.JSONB(tagsData), contact.Tags)

		// Test marshal into map
		var tags map[string]interface{}
		err = contact.Tags.Unmarshal(&tags)
		assert.NoError(t, err)
		assert.Equal(t, "referral", tags["source"])
		assert.Equal(t, "high", tags["priority"])
		assert.Equal(t, "enterprise", tags["category"])
	})

	t.Run("Empty JSONB fields", func(t *testing.T) {
		contact := &Contact{}

		// Default should be empty JSONB
		var emptyProfile map[string]interface{}
		err := contact.Profile.Unmarshal(&emptyProfile)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(emptyProfile))

		var emptyTags map[string]interface{}
		err = contact.Tags.Unmarshal(&emptyTags)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(emptyTags))
	})
}

// TestContactValidation tests contact validation logic
func TestContactValidation(t *testing.T) {
	t.Run("Valid contact data", func(t *testing.T) {
		userID := uuid.New()
		contact := &Contact{
			UserID:   userID,
			Profile:  base.JSONB(`{"email":"valid@example.com","phone":"+1234567890"}`),
			SourceID: "test_source",
			Source:   string(ContactSourceCSV),
		}

		// This should not cause any issues
		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, contact.ID)
	})

	t.Run("Contact with default values", func(t *testing.T) {
		userID := uuid.New()
		contact := &Contact{
			UserID: userID,
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)

		// Check default field values (these will be set by database defaults)
		// In Go, they start as empty strings unless explicitly set
		assert.Equal(t, "", contact.Name)
		assert.Equal(t, "", contact.Company)
		assert.Equal(t, "", contact.JobTitle)
	})

	t.Run("DoNotContact flag", func(t *testing.T) {
		contact := &Contact{
			DoNotContact: true,
		}

		assert.True(t, contact.DoNotContact)

		contact.DoNotContact = false
		assert.False(t, contact.DoNotContact)
	})
}

// TestContactSources tests different contact sources
func TestContactSources(t *testing.T) {
	t.Run("All contact sources", func(t *testing.T) {
		sources := []ContactSource{
			ContactSourceCosmoAgents,
			ContactSourceGoogleAds,
			ContactSourceCSV,
			ContactSourceHubspot,
			ContactSourceApollo,
			ContactSourceFacebookAds,
			ContactSourceTiktokAds,
		}

		userID := uuid.New()

		for _, source := range sources {
			contact := &Contact{
				UserID: userID,
				Source: string(source),
			}

			err := contact.BeforeCreate(nil)
			assert.NoError(t, err)
			assert.Equal(t, string(source), contact.Source)
		}
	})

	t.Run("Custom source", func(t *testing.T) {
		customSource := "custom-source"
		contact := &Contact{
			UserID: uuid.New(),
			Source: customSource,
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, customSource, contact.Source)
	})
}

// TestContactOrganization tests organization relationship
func TestContactOrganization(t *testing.T) {
	t.Run("Contact with organization", func(t *testing.T) {
		orgID := uuid.New()
		contact := &Contact{
			UserID:         uuid.New(),
			OrganizationID: &orgID,
		}

		assert.NotNil(t, contact.OrganizationID)
		assert.Equal(t, orgID, *contact.OrganizationID)
	})

	t.Run("Contact without organization", func(t *testing.T) {
		contact := &Contact{
			UserID: uuid.New(),
		}

		assert.Nil(t, contact.OrganizationID)
	})

	t.Run("Organization ID update", func(t *testing.T) {
		contact := &Contact{}
		orgID1 := uuid.New()
		orgID2 := uuid.New()

		contact.OrganizationID = &orgID1
		assert.Equal(t, orgID1, *contact.OrganizationID)

		contact.OrganizationID = &orgID2
		assert.Equal(t, orgID2, *contact.OrganizationID)

		contact.OrganizationID = nil
		assert.Nil(t, contact.OrganizationID)
	})
}

// TestListContactStructure tests the ListContact struct
func TestListContactStructure(t *testing.T) {
	t.Run("ListContact struct initialization", func(t *testing.T) {
		userID := uuid.New()
		listID := uuid.New()
		orgID := uuid.New()
		hubspotID := "hubspot_list_123"

		listContact := &ListContact{
			Base:            base.Base{ID: listID},
			TimestampMixin:  base.TimestampMixin{CreatedAt: time.Now(), UpdatedAt: time.Now()},
			SoftDeleteMixin: base.SoftDeleteMixin{IsDeleted: false},
			UserID:          userID,
			Name:            "Marketing Leads",
			Source:          string(ContactSourceHubspot),
			SourceID:        "list_source_123",
			HubspotID:       &hubspotID,
			OrganizationID:  &orgID,
		}

		assert.Equal(t, listID, listContact.ID)
		assert.Equal(t, userID, listContact.UserID)
		assert.Equal(t, "Marketing Leads", listContact.Name)
		assert.Equal(t, string(ContactSourceHubspot), listContact.Source)
		assert.Equal(t, "list_source_123", listContact.SourceID)
		assert.Equal(t, &hubspotID, listContact.HubspotID)
		assert.Equal(t, &orgID, listContact.OrganizationID)
	})

	t.Run("ListContact with minimal data", func(t *testing.T) {
		userID := uuid.New()
		listContact := &ListContact{
			UserID: userID,
			Name:   "Test List",
		}

		assert.Equal(t, userID, listContact.UserID)
		assert.Equal(t, "Test List", listContact.Name)
		assert.Equal(t, "", listContact.Source)
		assert.Equal(t, "", listContact.SourceID)
	})
}

// TestListContactTableName tests the TableName method for ListContact
func TestListContactTableName(t *testing.T) {
	t.Run("ListContact table name", func(t *testing.T) {
		listContact := &ListContact{}
		assert.Equal(t, "list_contacts", listContact.TableName())
	})
}

// TestListContactBeforeCreate tests the BeforeCreate hook for ListContact
func TestListContactBeforeCreate(t *testing.T) {
	t.Run("BeforeCreate sets default SourceID", func(t *testing.T) {
		listContact := &ListContact{
			UserID: uuid.New(),
			Name:   "Test List",
			Source: string(ContactSourceCSV),
		}

		err := listContact.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, listContact.ID)
		assert.Equal(t, listContact.ID.String(), listContact.SourceID)
		assert.Equal(t, string(ContactSourceCSV), listContact.Source)
	})

	t.Run("BeforeCreate preserves existing SourceID", func(t *testing.T) {
		existingSourceID := "existing_list_source"
		listContact := &ListContact{
			UserID:   uuid.New(),
			Name:     "Test List",
			SourceID: existingSourceID,
			Source:   string(ContactSourceGoogleAds),
		}

		err := listContact.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.Equal(t, existingSourceID, listContact.SourceID)
		assert.Equal(t, string(ContactSourceGoogleAds), listContact.Source)
	})

	t.Run("BeforeCreate sets default Source", func(t *testing.T) {
		listContact := &ListContact{
			UserID: uuid.New(),
			Name:   "Test List",
		}

		err := listContact.BeforeCreate(nil)

		assert.NoError(t, err)
		assert.Equal(t, string(ContactSourceCosmoAgents), listContact.Source)
	})
}

// TestListContactAssociation tests the many-to-many association table
func TestListContactAssociation(t *testing.T) {
	t.Run("ListContactAssociation struct", func(t *testing.T) {
		contactID := uuid.New()
		listContactID := uuid.New()

		association := &ListContactAssociation{
			Base:          base.Base{ID: uuid.New()},
			ListContactID: listContactID,
			ContactID:     contactID,
		}

		assert.Equal(t, listContactID, association.ListContactID)
		assert.Equal(t, contactID, association.ContactID)
	})

	t.Run("ListContactAssociation table name", func(t *testing.T) {
		association := &ListContactAssociation{}
		assert.Equal(t, "list_contact_association", association.TableName())
	})
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("Empty contact fields", func(t *testing.T) {
		contact := &Contact{
			UserID: uuid.New(),
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)

		// All string fields should be empty (will get database defaults)
		assert.Empty(t, contact.Name)
	})

	t.Run("Invalid UUID values", func(t *testing.T) {
		contact := &Contact{
			UserID: uuid.Nil, // Invalid
		}

		// BeforeCreate should still work even with nil UserID for testing
		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, contact.ID)
	})

	t.Run("Large text fields", func(t *testing.T) {
		contact := &Contact{
			UserID:  uuid.New(),
			Name:    string(make([]byte, 1000)), // Large name
			Profile: base.JSONB(`{"email":"test@example.com"}`),
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Len(t, contact.Name, 1000)
	})

	t.Run("Special characters in fields", func(t *testing.T) {
		contact := &Contact{
			UserID:  uuid.New(),
			Name:    "Jóhn-Çarlos_D'Ángelo",
			Profile: base.JSONB(`{"email":"test+special@example.com"}`),
			Company: "Acme & Co. (International)",
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "Jóhn-Çarlos_D'Ángelo", contact.Name)
		assert.Equal(t, "Acme & Co. (International)", contact.Company)

		var profile map[string]interface{}
		require.NoError(t, contact.Profile.Unmarshal(&profile))
		assert.Equal(t, "test+special@example.com", profile["email"])
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	t.Run("Large number of contacts", func(t *testing.T) {
		userID := uuid.New()
		contacts := make([]*Contact, 1000)

		for i := 0; i < 1000; i++ {
			contact := &Contact{
				UserID:  userID,
				Profile: base.JSONB(`{"email":"test@example.com"}`),
			}
			err := contact.BeforeCreate(nil)
			if err != nil {
				t.Fatalf("BeforeCreate failed for contact %d: %v", i, err)
			}
			contacts[i] = contact
		}

		assert.Len(t, contacts, 1000)
		assert.NotEqual(t, contacts[0].ID, contacts[999].ID)
	})

	t.Run("JSON serialization performance", func(t *testing.T) {
		contact := &Contact{
			UserID:  uuid.New(),
			Name:    "John Doe",
			Profile: base.JSONB(`{"email":"john.doe@example.com","large":"data with lots of information that makes the JSON bigger"}`),
		}

		err := contact.BeforeCreate(nil)
		require.NoError(t, err)

		// Test marshaling/unmarshaling performance
		start := time.Now()
		for i := 0; i < 100; i++ {
			_, err := json.Marshal(contact)
			if err != nil {
				t.Fatalf("JSON Marshal failed: %v", err)
			}
		}
		duration := time.Since(start)

		// Should complete in reasonable time (adjust threshold as needed)
		assert.Less(t, duration, 100*time.Millisecond, "100 JSON marshals should complete quickly")
	})
}

// TestContactScenarios tests real-world contact scenarios
func TestContactScenarios(t *testing.T) {
	t.Run("Import from CSV", func(t *testing.T) {
		userID := uuid.New()
		contact := &Contact{
			UserID:   userID,
			Source:   string(ContactSourceCSV),
			SourceID: "row_123",
			Name:     "Alice Smith",
			Profile:  base.JSONB(`{"email":"alice.smith@company.com","phone":"+1-555-0123"}`),
			Company:  "Tech Corp",
			JobTitle: "Product Manager",
			City:     "New York",
			Country:  "USA",
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, string(ContactSourceCSV), contact.Source)
		assert.Equal(t, "row_123", contact.SourceID)
	})

	t.Run("Import from HubSpot", func(t *testing.T) {
		userID := uuid.New()
		hubspotID := "hubspot_456789"
		contact := &Contact{
			UserID:    userID,
			Source:    string(ContactSourceHubspot),
			SourceID:  hubspotID,
			HubspotID: &hubspotID,
			Name:      "Bob Johnson",
			Profile:   base.JSONB(`{"email":"bob.johnson@enterprise.com"}`),
			Company:   "Enterprise Inc",
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, &hubspotID, contact.HubspotID)
	})

	t.Run("Contact with complete profile", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		tags := base.JSONB(`{
			"priority": "high",
			"source": "referral",
			"segment": "enterprise",
			"campaign": "q4-outreach"
		}`)

		contact := &Contact{
			UserID:         userID,
			Source:         string(ContactSourceApollo),
			Name:           "Complete Profile",
			Company:        "Complete Solutions",
			JobTitle:       "Chief Executive Officer",
			Address:        "100 Executive Blvd",
			City:           "Executive City",
			State:          "EC",
			Country:        "United States",
			Zip:            "10001",
			Profile:        base.JSONB(`{"email":"complete@profile.com","phone":"+1-555-COMPLETE","linkedin":"https://linkedin.com/in/complete-profile","twitter":"@completeprofile","skills":["Leadership","Strategy","Communication"],"experience":15,"education":"MBA from Top University"}`),
			Tags:           tags,
			DoNotContact:   false,
			OrganizationID: &orgID,
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)

		// Verify profile data
		var profileData map[string]interface{}
		err = contact.Profile.Unmarshal(&profileData)
		assert.NoError(t, err)
		assert.Equal(t, "https://linkedin.com/in/complete-profile", profileData["linkedin"])

		// Verify tags
		var tagsData map[string]interface{}
		err = contact.Tags.Unmarshal(&tagsData)
		assert.NoError(t, err)
		assert.Equal(t, "high", tagsData["priority"])
	})

	t.Run("Do not contact flag", func(t *testing.T) {
		contact := &Contact{
			UserID:       uuid.New(),
			DoNotContact: true,
			Profile:      base.JSONB(`{"email":"unsubscribed@example.com","unsubscribed_reason":"requested removal"}`),
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.True(t, contact.DoNotContact)
	})
}

// TestListContactScenarios tests real-world list contact scenarios
func TestListContactScenarios(t *testing.T) {
	t.Run("Marketing leads list", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		listContact := &ListContact{
			UserID:         userID,
			Name:           "Q4 Marketing Leads",
			Source:         string(ContactSourceFacebookAds),
			SourceID:       "fb_campaign_123",
			OrganizationID: &orgID,
		}

		err := listContact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "Q4 Marketing Leads", listContact.Name)
		assert.Equal(t, string(ContactSourceFacebookAds), listContact.Source)
	})

	t.Run("Customer list from CSV", func(t *testing.T) {
		userID := uuid.New()
		listContact := &ListContact{
			UserID:   userID,
			Name:     "Existing Customers",
			Source:   string(ContactSourceCSV),
			SourceID: "customer_upload_2023",
		}

		err := listContact.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "Existing Customers", listContact.Name)
		assert.Equal(t, string(ContactSourceCSV), listContact.Source)
	})
}

// TestIntegrationScenarios tests integration between contacts and lists
func TestIntegrationScenarios(t *testing.T) {
	t.Run("Contact belongs to multiple lists", func(t *testing.T) {
		userID := uuid.New()
		contact := &Contact{
			UserID:  userID,
			Profile: base.JSONB(`{"email":"contact@example.com"}`),
		}

		err := contact.BeforeCreate(nil)
		assert.NoError(t, err)

		// Create multiple lists
		list1 := &ListContact{
			UserID: userID,
			Name:   "Newsletter Subscribers",
		}

		list2 := &ListContact{
			UserID: userID,
			Name:   "Product Interest",
		}

		err = list1.BeforeCreate(nil)
		assert.NoError(t, err)

		err = list2.BeforeCreate(nil)
		assert.NoError(t, err)

		// In a real scenario, these would be connected through ListContactAssociation
		association1 := &ListContactAssociation{
			ListContactID: list1.ID,
			ContactID:     contact.ID,
		}

		association2 := &ListContactAssociation{
			ListContactID: list2.ID,
			ContactID:     contact.ID,
		}

		assert.Equal(t, list1.ID, association1.ListContactID)
		assert.Equal(t, contact.ID, association1.ContactID)
		assert.Equal(t, list2.ID, association2.ListContactID)
		assert.Equal(t, contact.ID, association2.ContactID)
	})

	t.Run("List with many contacts", func(t *testing.T) {
		userID := uuid.New()
		listContact := &ListContact{
			UserID: userID,
			Name:   "Large List",
		}

		err := listContact.BeforeCreate(nil)
		assert.NoError(t, err)

		// Create many contacts
		contacts := make([]*Contact, 100)
		associations := make([]*ListContactAssociation, 100)

		for i := 0; i < 100; i++ {
			contact := &Contact{
				UserID:  userID,
				Profile: base.JSONB(`{"email":"contact@example.com"}`),
			}

			err = contact.BeforeCreate(nil)
			assert.NoError(t, err)

			contacts[i] = contact
			associations[i] = &ListContactAssociation{
				ListContactID: listContact.ID,
				ContactID:     contact.ID,
			}
		}

		assert.Len(t, contacts, 100)
		assert.Len(t, associations, 100)
	})
}
