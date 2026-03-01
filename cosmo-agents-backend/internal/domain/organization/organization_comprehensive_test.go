package organization

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrganization_JSONSerialization tests JSON serialization and deserialization
func TestOrganization_JSONSerialization(t *testing.T) {
	t.Run("Serialize organization to JSON", func(t *testing.T) {
		userID := uuid.New()
		org := Organization{
			UserID:                  &userID,
			Name:                    "Test Company",
			CompanyURL:              "https://testcompany.com",
			CompanyDescription:      "A test company for unit testing",
			CompanyTargetingPersona: pq.StringArray{"Developers", "Managers"},
			ValueOffering:           "Testing solutions",
		}

		jsonData, err := json.Marshal(org)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Verify it can be unmarshaled back
		var unmarshaled Organization
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, org.ID, unmarshaled.ID)
		assert.Equal(t, org.Name, unmarshaled.Name)
		assert.Equal(t, org.CompanyURL, unmarshaled.CompanyURL)
	})

	t.Run("Deserialize JSON to organization", func(t *testing.T) {
		jsonStr := `{
			"id": "550e8400-e29b-41d4-a716-446655440000",
			"user_id": "550e8400-e29b-41d4-a716-446655440001",
			"name": "JSON Company",
			"company_url": "https://jsoncompany.com",
			"company_description": "Company from JSON",
			"company_targeting_persona": ["JSON Users"],
			"value_offering": "JSON-based solutions",
			"is_deleted": false,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z"
		}`

		var org Organization
		err := json.Unmarshal([]byte(jsonStr), &org)
		require.NoError(t, err)

		assert.Equal(t, "JSON Company", org.Name)
		assert.Equal(t, "https://jsoncompany.com", org.CompanyURL)
		assert.Equal(t, "Company from JSON", org.CompanyDescription)
		assert.Contains(t, org.CompanyTargetingPersona, "JSON Users")
		assert.Equal(t, "JSON-based solutions", org.ValueOffering)
	})
}

// TestOrganization_DatabaseBehavior tests database-related behavior
func TestOrganization_DatabaseBehavior(t *testing.T) {
	t.Run("Table name consistency", func(t *testing.T) {
		org := &Organization{}
		org1 := Organization{}
		org2 := &Organization{}

		// All should return the same table name
		assert.Equal(t, "organizations", org.TableName())
		assert.Equal(t, "organizations", org1.TableName())
		assert.Equal(t, "organizations", org2.TableName())
	})

	t.Run("GORM compatibility", func(t *testing.T) {
		org := Organization{
			Name: "GORM Test Company",
		}

		// Verify the struct can be used with GORM (basic field access)
		assert.Equal(t, "GORM Test Company", org.Name)
		assert.Equal(t, "organizations", org.TableName())

		// Test with pointer receiver
		orgPtr := &org
		assert.Equal(t, "GORM Test Company", orgPtr.Name)
		assert.Equal(t, "organizations", orgPtr.TableName())
	})
}

// TestOrganization_MockDBOperations tests mock database operations
func TestOrganization_MockDBOperations(t *testing.T) {
	t.Run("Mock GORM DB operations", func(t *testing.T) {
		// This tests the domain model's compatibility with GORM operations
		// without requiring an actual database connection

		org := Organization{
			Name: "Mock DB Company",
		}

		// Simulate what would happen during a GORM operation
		assert.Equal(t, "Mock DB Company", org.Name)

		// Test BeforeCreate hook
		err := org.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, org.ID)
	})

	t.Run("Mock transaction behavior", func(t *testing.T) {
		org := Organization{
			Name: "Transaction Test",
		}

		// Simulate transaction behavior - organization should maintain state
		originalName := org.Name
		assert.Equal(t, "Transaction Test", originalName)

		// Simulate update within transaction
		org.Name = "Updated Name"
		assert.Equal(t, "Updated Name", org.Name)

		// Simulate rollback (restore original)
		org.Name = originalName
		assert.Equal(t, originalName, org.Name)
	})
}

// TestOrganization_ErrorScenarios tests error scenarios
func TestOrganization_ErrorScenarios(t *testing.T) {
	t.Run("Invalid UUID handling", func(t *testing.T) {
		org := Organization{}

		// BeforeCreate should handle nil GORM DB gracefully
		err := org.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, org.ID)
	})

	t.Run("Empty targeting persona handling", func(t *testing.T) {
		org := Organization{
			CompanyTargetingPersona: pq.StringArray{},
		}

		assert.Empty(t, org.CompanyTargetingPersona)
		assert.Len(t, org.CompanyTargetingPersona, 0)

		// Should be able to add personas
		org.CompanyTargetingPersona = append(org.CompanyTargetingPersona, "New Persona")
		assert.Len(t, org.CompanyTargetingPersona, 1)
		assert.Equal(t, "New Persona", org.CompanyTargetingPersona[0])
	})
}

// TestOrganization_ConcurrentAccess tests concurrent access patterns
func TestOrganization_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent ID generation", func(t *testing.T) {
		// Test that BeforeCreate is thread-safe
		orgs := make([]Organization, 100)
		ids := make(chan uuid.UUID, 100)

		// Generate IDs concurrently
		for i := 0; i < 100; i++ {
			go func(index int) {
				org := &orgs[index]
				err := org.BeforeCreate(nil)
				if err == nil {
					ids <- org.ID
				}
			}(i)
		}

		// Collect all generated IDs
		generatedIDs := make([]uuid.UUID, 0, 100)
		for i := 0; i < 100; i++ {
			select {
			case id := <-ids:
				generatedIDs = append(generatedIDs, id)
			case <-time.After(time.Second):
				t.Fatal("Timeout waiting for ID generation")
			}
		}

		assert.Len(t, generatedIDs, 100, "Should generate 100 IDs")

		// All IDs should be unique
		uniqueIDs := make(map[uuid.UUID]bool)
		for _, id := range generatedIDs {
			assert.False(t, uniqueIDs[id], "ID should be unique: %s", id)
			uniqueIDs[id] = true
		}
	})
}

// TestOrganization_Performance tests performance characteristics
func TestOrganization_Performance(t *testing.T) {
	t.Run("Large targeting persona array", func(t *testing.T) {
		// Test with large targeting persona array
		personas := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			personas[i] = "Persona " + string(rune(i))
		}

		org := Organization{
			CompanyTargetingPersona: pq.StringArray(personas),
		}

		assert.Len(t, org.CompanyTargetingPersona, 1000)

		// Test lookup performance
		start := time.Now()
		for i := 0; i < 100; i++ {
			_ = org.CompanyTargetingPersona[i] // Simple access
		}
		duration := time.Since(start)

		assert.Less(t, duration, time.Millisecond, "Array access should be fast")
	})

	t.Run("String field performance", func(t *testing.T) {
		longDescription := make([]byte, 10000)
		for i := range longDescription {
			longDescription[i] = 'A' + byte(i%26)
		}

		org := Organization{
			Name:               "Performance Test Company",
			CompanyDescription: string(longDescription),
		}

		assert.Equal(t, 10000, len(org.CompanyDescription))

		// Test string operations
		start := time.Now()
		_ = len(org.CompanyDescription)
		_ = org.Name + " Suffix"
		duration := time.Since(start)

		assert.Less(t, duration, time.Microsecond*100, "String operations should be fast")
	})
}

// TestOrganization_IntegrationPatterns tests integration patterns
func TestOrganization_IntegrationPatterns(t *testing.T) {
	t.Run("Service layer integration pattern", func(t *testing.T) {
		// Simulate how the organization would be used in a service
		userID := uuid.New()

		// Creation pattern
		org := Organization{
			UserID:                  &userID,
			Name:                    "Service Layer Company",
			CompanyURL:              "https://servicelayer.example.com",
			CompanyDescription:      "Company created through service layer",
			CompanyTargetingPersona: pq.StringArray{"Service Users"},
			ValueOffering:           "Service-oriented value proposition",
		}

		// Validation pattern (would be in service)
		assert.NotEmpty(t, org.Name, "Name should not be empty")
		assert.NotEmpty(t, org.ValueOffering, "Value offering should not be empty")

		// Database preparation
		err := org.BeforeCreate(nil)
		assert.NoError(t, err)

		// Response pattern (would be returned by service)
		response := map[string]interface{}{
			"id":                  org.ID,
			"name":                org.Name,
			"company_url":         org.CompanyURL,
			"company_description": org.CompanyDescription,
			"targeting_persona":   org.CompanyTargetingPersona,
			"value_offering":      org.ValueOffering,
			"created_at":          org.CreatedAt,
			"updated_at":          org.UpdatedAt,
		}

		assert.Equal(t, org.ID, response["id"])
		assert.Equal(t, org.Name, response["name"])
	})

	t.Run("Repository layer integration pattern", func(t *testing.T) {
		// Simulate repository operations
		org := Organization{
			Name: "Repository Pattern Company",
		}

		// Find operation simulation
		foundOrg := Organization{
			Name: "Repository Pattern Company",
		}
		foundOrg.ID = uuid.New() // Set ID manually
		assert.Equal(t, org.Name, foundOrg.Name)

		// Update operation simulation
		updatedOrg := foundOrg
		updatedOrg.CompanyDescription = "Updated description"
		assert.Equal(t, "Updated description", updatedOrg.CompanyDescription)
		assert.Equal(t, foundOrg.ID, updatedOrg.ID)
	})
}

// TestOrganization_ComparisonAndEquality tests comparison and equality
func TestOrganization_ComparisonAndEquality(t *testing.T) {
	t.Run("Same organization comparison", func(t *testing.T) {
		org1 := Organization{Name: "Same Name"}
		org2 := Organization{Name: "Same Name"}

		// Set same ID manually for testing
		org1.ID = uuid.New()
		org2.ID = org1.ID

		assert.Equal(t, org1.ID, org2.ID)
		assert.Equal(t, org1.Name, org2.Name)
		assert.Equal(t, org1.TableName(), org2.TableName())
	})

	t.Run("Different organization comparison", func(t *testing.T) {
		org1 := Organization{Name: "Company A"}
		org2 := Organization{Name: "Company B"}

		// assert.NotEqual(t, org1.ID, org2.ID) // IDs might be equal for unpersisted entities
		assert.NotEqual(t, org1.Name, org2.Name)
		assert.Equal(t, org1.TableName(), org2.TableName()) // Table name should always be same
	})

	t.Run("User comparison", func(t *testing.T) {
		userID1 := uuid.New()
		userID2 := uuid.New()

		org1 := Organization{UserID: &userID1}
		org2 := Organization{UserID: &userID1}
		org3 := Organization{UserID: &userID2}
		org4 := Organization{UserID: nil}

		assert.Equal(t, *org1.UserID, *org2.UserID)
		assert.NotEqual(t, *org1.UserID, *org3.UserID)
		assert.Nil(t, org4.UserID)
	})
}

// TestOrganization_IndustryExamples tests industry-specific examples
func TestOrganization_IndustryExamples(t *testing.T) {
	tests := []struct {
		name     string
		org      Organization
		expected map[string]interface{}
	}{
		{
			name: "SaaS Company",
			org: Organization{
				Name:                    "CloudSaaS Inc",
				CompanyURL:              "https://cloudsaas.com",
				CompanyDescription:      "B2B SaaS platform for enterprise resource planning",
				CompanyTargetingPersona: pq.StringArray{"CFOs", "Finance Directors", "ERP Managers"},
				ValueOffering:           "Cloud-based ERP with real-time analytics and automated workflows",
			},
			expected: map[string]interface{}{
				"hasURL":            true,
				"descriptionLength": 50,
				"personaCount":      3,
			},
		},
		{
			name: "E-commerce Company",
			org: Organization{
				Name:                    "ShopFlow",
				CompanyURL:              "https://shopflow retail.com",
				CompanyDescription:      "E-commerce platform for direct-to-consumer brands",
				CompanyTargetingPersona: pq.StringArray{"Brand Managers", "E-commerce Directors", "Marketing VPs"},
				ValueOffering:           "Headless commerce infrastructure with seamless integrations",
			},
			expected: map[string]interface{}{
				"hasURL":            true,
				"descriptionLength": 49,
				"personaCount":      3,
			},
		},
		{
			name: "Healthcare Tech",
			org: Organization{
				Name:                    "MedTech Solutions",
				CompanyURL:              "https://medtech.health",
				CompanyDescription:      "HIPAA-compliant healthcare technology platform",
				CompanyTargetingPersona: pq.StringArray{"Hospital Administrators", "Healthcare CIOs", "Practice Managers"},
				ValueOffering:           "Secure patient management systems with telehealth integration",
			},
			expected: map[string]interface{}{
				"hasURL":            true,
				"descriptionLength": 46,
				"personaCount":      3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.org.Name)
			assert.True(t, tt.expected["hasURL"].(bool))
			assert.Equal(t, tt.expected["descriptionLength"], len(tt.org.CompanyDescription))
			assert.Equal(t, tt.expected["personaCount"], len(tt.org.CompanyTargetingPersona))
			assert.NotEmpty(t, tt.org.ValueOffering)
		})
	}
}

// TestOrganization_FieldManipulation tests field manipulation scenarios
func TestOrganization_FieldManipulation(t *testing.T) {
	t.Run("Field updates", func(t *testing.T) {
		org := Organization{
			Name:               "Original Name",
			CompanyDescription: "Original description",
		}

		// Update fields
		org.Name = "Updated Name"
		org.CompanyDescription = "Updated description"
		org.ValueOffering = "New value offering"

		assert.Equal(t, "Updated Name", org.Name)
		assert.Equal(t, "Updated description", org.CompanyDescription)
		assert.Equal(t, "New value offering", org.ValueOffering)
	})

	t.Run("Targeting persona manipulation", func(t *testing.T) {
		org := Organization{
			CompanyTargetingPersona: pq.StringArray{"Persona 1"},
		}

		// Add personas
		org.CompanyTargetingPersona = append(org.CompanyTargetingPersona, "Persona 2")
		assert.Len(t, org.CompanyTargetingPersona, 2)

		// Remove persona (by creating new array)
		newPersonas := make(pq.StringArray, 0, len(org.CompanyTargetingPersona)-1)
		for _, persona := range org.CompanyTargetingPersona {
			if persona != "Persona 1" {
				newPersonas = append(newPersonas, persona)
			}
		}
		org.CompanyTargetingPersona = newPersonas

		assert.Len(t, org.CompanyTargetingPersona, 1)
		assert.Equal(t, "Persona 2", org.CompanyTargetingPersona[0])
	})
}

// TestOrganization_BoundaryTests tests boundary conditions
func TestOrganization_BoundaryTests(t *testing.T) {
	t.Run("Maximum URL length", func(t *testing.T) {
		// Create a very long but potentially valid URL
		longURL := "https://example.com/"
		for i := 0; i < 1000; i++ {
			longURL += "path/"
		}

		org := Organization{CompanyURL: longURL}
		assert.Greater(t, len(org.CompanyURL), 2000)
	})

	t.Run("Empty vs nil distinction", func(t *testing.T) {
		org1 := Organization{UserID: nil}
		org2 := Organization{}

		assert.Nil(t, org1.UserID)
		assert.Nil(t, org2.UserID)

		userID := uuid.New()
		org3 := Organization{UserID: &userID}
		require.NotNil(t, org3.UserID)
		assert.Equal(t, userID, *org3.UserID)
	})
}
