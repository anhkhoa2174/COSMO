package organization

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// TestOrganization_TableName tests the table name method
func TestOrganization_TableName(t *testing.T) {
	t.Run("Correct table name", func(t *testing.T) {
		org := Organization{}
		expected := "organizations"
		assert.Equal(t, expected, org.TableName())
	})
}

// TestOrganization_Structure tests the organization struct fields
func TestOrganization_Structure(t *testing.T) {
	t.Run("Organization struct has correct fields", func(t *testing.T) {
		userID := uuid.New()
		targetingPersona := []string{"Engineering Managers", "CTOs", "VPs of Engineering"}

		org := Organization{
			UserID:                  &userID,
			Name:                    "TechCorp Solutions",
			CompanyURL:              "https://techcorp.example.com",
			CompanyDescription:      "Leading provider of cloud infrastructure solutions",
			CompanyTargetingPersona: pq.StringArray(targetingPersona),
			ValueOffering:           "Scalable cloud infrastructure with 99.9% uptime guarantee",
		}

		// Test Base fields (ID might be zero if not yet persisted)
		// assert.NotEqual(t, uuid.Nil, org.ID, "ID should be set")

		// Test direct fields
		assert.Equal(t, &userID, org.UserID)
		assert.Equal(t, "TechCorp Solutions", org.Name)
		assert.Equal(t, "https://techcorp.example.com", org.CompanyURL)
		assert.Equal(t, "Leading provider of cloud infrastructure solutions", org.CompanyDescription)
		assert.Equal(t, pq.StringArray(targetingPersona), org.CompanyTargetingPersona)
		assert.Equal(t, "Scalable cloud infrastructure with 99.9% uptime guarantee", org.ValueOffering)
	})
}

// TestOrganization_Inheritance tests that Organization properly inherits from base structs
func TestOrganization_Inheritance(t *testing.T) {
	t.Run("Inherits Base fields", func(t *testing.T) {
		org := Organization{}

		// Should have ID field from Base
		_ = org.ID // Just verify field exists

		// Should be able to use BeforeCreate method from Base
		err := org.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, org.ID, "BeforeCreate should set ID")
	})

	t.Run("Inherits TimestampMixin fields", func(t *testing.T) {
		org := Organization{}

		// Should have CreatedAt and UpdatedAt fields
		assert.Equal(t, time.Time{}, org.CreatedAt, "CreatedAt should be zero time initially")
		assert.Equal(t, time.Time{}, org.UpdatedAt, "UpdatedAt should be zero time initially")
	})

	t.Run("Inherits SoftDeleteMixin fields", func(t *testing.T) {
		org := Organization{}

		// Should have IsDeleted field
		assert.False(t, org.IsDeleted, "IsDeleted should default to false")
	})
}

// TestOrganization_DefaultValues tests default values for organization fields
func TestOrganization_DefaultValues(t *testing.T) {
	t.Run("Zero values are correct", func(t *testing.T) {
		org := Organization{}

		// Pointer fields should be nil
		assert.Nil(t, org.UserID, "UserID should be nil by default")

		// String fields should be empty
		assert.Equal(t, "", org.Name, "Name should be empty by default")
		assert.Equal(t, "", org.CompanyURL, "CompanyURL should be empty by default")
		assert.Equal(t, "", org.CompanyDescription, "CompanyDescription should be empty by default")
		assert.Equal(t, "", org.ValueOffering, "ValueOffering should be empty by default")

		// Array fields should be empty
		assert.Empty(t, org.CompanyTargetingPersona, "CompanyTargetingPersona should be empty by default")
	})
}

// TestOrganization_UUIDGeneration tests UUID generation functionality
func TestOrganization_UUIDGeneration(t *testing.T) {
	t.Run("BeforeCreate generates UUID", func(t *testing.T) {
		org := Organization{}

		// Initially ID should be zero
		assert.Equal(t, uuid.Nil, org.ID)

		// After BeforeCreate, ID should be set
		err := org.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, org.ID)
	})

	t.Run("BeforeCreate preserves existing UUID", func(t *testing.T) {
		existingID := uuid.New()
		org := Organization{}
		org.ID = existingID // Set ID manually

		err := org.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, org.ID, "Existing ID should be preserved")
	})
}

// TestOrganization_CompanyTargetingPersona tests the targeting persona array
func TestOrganization_CompanyTargetingPersona(t *testing.T) {
	t.Run("Empty targeting persona", func(t *testing.T) {
		org := Organization{}
		assert.Empty(t, org.CompanyTargetingPersona)
	})

	t.Run("Single targeting persona", func(t *testing.T) {
		persona := []string{"Engineering Managers"}
		org := Organization{
			CompanyTargetingPersona: pq.StringArray(persona),
		}
		assert.Equal(t, pq.StringArray(persona), org.CompanyTargetingPersona)
	})

	t.Run("Multiple targeting personas", func(t *testing.T) {
		personas := []string{
			"Engineering Managers",
			"CTOs",
			"VPs of Engineering",
			"DevOps Managers",
		}
		org := Organization{
			CompanyTargetingPersona: pq.StringArray(personas),
		}
		assert.Equal(t, pq.StringArray(personas), org.CompanyTargetingPersona)
		assert.Len(t, org.CompanyTargetingPersona, 4)
	})

	t.Run("Targeting persona with special characters", func(t *testing.T) {
		personas := []string{
			"Sales & Marketing Directors",
			"C-Suite Executives",
			"Product Managers (Tech)",
			"IT Decision-Makers",
		}
		org := Organization{
			CompanyTargetingPersona: pq.StringArray(personas),
		}
		assert.Equal(t, pq.StringArray(personas), org.CompanyTargetingPersona)
		assert.Contains(t, org.CompanyTargetingPersona, "Sales & Marketing Directors")
		assert.Contains(t, org.CompanyTargetingPersona, "C-Suite Executives")
	})
}

// TestOrganization_UserID tests the UserID field
func TestOrganization_UserID(t *testing.T) {
	t.Run("Nil UserID", func(t *testing.T) {
		org := Organization{UserID: nil}
		assert.Nil(t, org.UserID)
	})

	t.Run("Valid UserID", func(t *testing.T) {
		userID := uuid.New()
		org := Organization{UserID: &userID}
		require.NotNil(t, org.UserID)
		assert.Equal(t, userID, *org.UserID)
	})

	t.Run("Change UserID", func(t *testing.T) {
		userID1 := uuid.New()
		userID2 := uuid.New()

		org := Organization{UserID: &userID1}
		assert.Equal(t, userID1, *org.UserID)

		org.UserID = &userID2
		assert.Equal(t, userID2, *org.UserID)
	})
}

// TestOrganization_Validation tests business validation rules
func TestOrganization_Validation(t *testing.T) {
	t.Run("Valid organization", func(t *testing.T) {
		userID := uuid.New()
		org := Organization{
			UserID:                  &userID,
			Name:                    "Valid Company Name",
			CompanyURL:              "https://valid-company.com",
			CompanyDescription:      "A valid company description",
			CompanyTargetingPersona: pq.StringArray{"Target Audience"},
			ValueOffering:           "A clear value proposition",
		}

		// This would typically be validated at the service/repository layer
		// For domain tests, we verify the structure is correct
		assert.NotEmpty(t, org.Name)
		assert.NotEmpty(t, org.CompanyURL)
		assert.NotEmpty(t, org.CompanyDescription)
		assert.NotEmpty(t, org.ValueOffering)
	})

	t.Run("Empty but valid organization", func(t *testing.T) {
		org := Organization{}

		// Even empty organization should have valid zero values
		assert.Equal(t, "", org.Name)
		assert.Equal(t, "", org.CompanyURL)
		assert.Equal(t, "", org.CompanyDescription)
		assert.Equal(t, "", org.ValueOffering)
		assert.Empty(t, org.CompanyTargetingPersona)
		assert.Nil(t, org.UserID)
	})
}

// TestOrganization_RealWorldScenarios tests realistic organization examples
func TestOrganization_RealWorldScenarios(t *testing.T) {
	t.Run("Startup company", func(t *testing.T) {
		userID := uuid.New()
		org := Organization{
			UserID:                  &userID,
			Name:                    "StartupAI Inc",
			CompanyURL:              "https://startupai.example.com",
			CompanyDescription:      "AI-powered startup building the future of work automation",
			CompanyTargetingPersona: pq.StringArray{"HR Managers", "Startup Founders", "VCs"},
			ValueOffering:           "AI-driven workflow automation for modern teams",
		}

		assert.Equal(t, "StartupAI Inc", org.Name)
		assert.Contains(t, org.CompanyDescription, "AI-powered")
		assert.Contains(t, org.CompanyTargetingPersona, "HR Managers")
		assert.Contains(t, org.ValueOffering, "AI-driven")
	})

	t.Run("Enterprise company", func(t *testing.T) {
		userID := uuid.New()
		org := Organization{
			UserID:             &userID,
			Name:               "GlobalCorp Enterprises",
			CompanyURL:         "https://globalcorp.com",
			CompanyDescription: "Fortune 500 company providing enterprise solutions for large-scale organizations",
			CompanyTargetingPersona: pq.StringArray{
				"CIOs",
				"CTOs",
				"IT Directors",
				"Procurement Managers",
				"Enterprise Architects",
			},
			ValueOffering: "Enterprise-grade cloud infrastructure with 24/7 support and SLA guarantees",
		}

		assert.Equal(t, "GlobalCorp Enterprises", org.Name)
		assert.Contains(t, org.CompanyDescription, "Fortune 500")
		assert.Len(t, org.CompanyTargetingPersona, 5)
		assert.Contains(t, org.ValueOffering, "SLA guarantees")
	})

	t.Run("Consulting firm", func(t *testing.T) {
		userID := uuid.New()
		org := Organization{
			UserID:             &userID,
			Name:               "Strategy & Innovation Consulting",
			CompanyURL:         "https://strategy-innovation.consulting",
			CompanyDescription: "Boutique consulting firm specializing in digital transformation and innovation strategy",
			CompanyTargetingPersona: pq.StringArray{
				"CEOs",
				"Managing Directors",
				"Heads of Strategy",
				"Innovation Officers",
			},
			ValueOffering: "Custom consulting solutions that drive measurable business outcomes and competitive advantage",
		}

		assert.Equal(t, "Strategy & Innovation Consulting", org.Name)
		assert.Contains(t, org.CompanyDescription, "Boutique consulting")
		assert.Contains(t, org.CompanyTargetingPersona, "Heads of Strategy")
		assert.Contains(t, org.ValueOffering, "measurable business outcomes")
	})
}

// TestOrganization_EdgeCases tests edge cases and unusual but valid scenarios
func TestOrganization_EdgeCases(t *testing.T) {
	t.Run("Very long name", func(t *testing.T) {
		longName := "This Is A Very Long Company Name That Might Push The Limits Of What We Consider Reasonable For A Company Name Field But Should Still Be Valid"
		org := Organization{Name: longName}
		assert.Equal(t, longName, org.Name)
		assert.Greater(t, len(org.Name), 100)
	})

	t.Run("Company URL with subdomains", func(t *testing.T) {
		org := Organization{
			CompanyURL: "https://api.staging.v2.company.example.co.uk/path/to/resource",
		}
		assert.Equal(t, "https://api.staging.v2.company.example.co.uk/path/to/resource", org.CompanyURL)
		assert.Contains(t, org.CompanyURL, "staging.v2")
		assert.Contains(t, org.CompanyURL, "example.co.uk")
	})

	t.Run("Special characters in description", func(t *testing.T) {
		description := "We specialize in AI/ML, IoT, Cloud Computing (SaaS, PaaS, IaaS), & other emerging technologies! 🚀"
		org := Organization{CompanyDescription: description}
		assert.Equal(t, description, org.CompanyDescription)
		assert.Contains(t, org.CompanyDescription, "AI/ML")
		assert.Contains(t, org.CompanyDescription, "🚀")
	})

	t.Run("Unicode in targeting personas", func(t *testing.T) {
		personas := []string{
			"CEO & Founder",
			"Directeur Général", // French
			"Geschäftsführer",   // German
			"代表取締役社長",           // Japanese
			"首席执行官",             // Chinese
		}
		org := Organization{
			CompanyTargetingPersona: pq.StringArray(personas),
		}
		assert.Equal(t, pq.StringArray(personas), org.CompanyTargetingPersona)
		assert.Len(t, org.CompanyTargetingPersona, 5)
		assert.Contains(t, org.CompanyTargetingPersona, "Directeur Général")
		assert.Contains(t, org.CompanyTargetingPersona, "代表取締役社長")
	})

	t.Run("Empty targeting persona array", func(t *testing.T) {
		org := Organization{
			CompanyTargetingPersona: pq.StringArray{},
		}
		assert.Empty(t, org.CompanyTargetingPersona)
		assert.Equal(t, pq.StringArray{}, org.CompanyTargetingPersona)
	})
}

// TestOrganization_CompleteExample tests a complete realistic example
func TestOrganization_CompleteExample(t *testing.T) {
	userID := uuid.New()

	org := Organization{
		Base: base.Base{
			ID: uuid.New(), // Would normally be set by BeforeCreate
		},
		UserID:             &userID,
		Name:               "CloudScale Technologies",
		CompanyURL:         "https://cloudscale.tech",
		CompanyDescription: "We provide scalable cloud infrastructure solutions for modern businesses",
		CompanyTargetingPersona: pq.StringArray{
			"CTOs",
			"DevOps Engineers",
			"Cloud Architects",
			"VPs of Engineering",
		},
		ValueOffering: "24/7 managed cloud services with automatic scaling and 99.99% uptime SLA",
	}

	// Verify all fields
	assert.NotEqual(t, uuid.Nil, org.ID)
	require.NotNil(t, org.UserID)
	assert.Equal(t, userID, *org.UserID)
	assert.Equal(t, "CloudScale Technologies", org.Name)
	assert.Equal(t, "https://cloudscale.tech", org.CompanyURL)
	assert.Equal(t, "We provide scalable cloud infrastructure solutions for modern businesses", org.CompanyDescription)
	assert.Equal(t, pq.StringArray{
		"CTOs",
		"DevOps Engineers",
		"Cloud Architects",
		"VPs of Engineering",
	}, org.CompanyTargetingPersona)
	assert.Equal(t, "24/7 managed cloud services with automatic scaling and 99.99% uptime SLA", org.ValueOffering)

	// Test table name
	assert.Equal(t, "organizations", org.TableName())
}

// TestOrganization_HelpersAndUtilities tests helper methods and utilities
func TestOrganization_HelpersAndUtilities(t *testing.T) {
	t.Run("Multiple organizations with different users", func(t *testing.T) {
		userID1 := uuid.New()
		userID2 := uuid.New()
		userID3 := uuid.New()

		orgs := []Organization{
			{UserID: &userID1, Name: "Company A"},
			{UserID: &userID2, Name: "Company B"},
			{UserID: &userID3, Name: "Company C"},
			{UserID: nil, Name: "Company D"}, // No user
		}

		assert.Len(t, orgs, 4)
		assert.Equal(t, "Company A", orgs[0].Name)
		assert.Equal(t, userID1, *orgs[0].UserID)
		assert.Equal(t, "Company D", orgs[3].Name)
		assert.Nil(t, orgs[3].UserID)
	})

	t.Run("Organization lifecycle simulation", func(t *testing.T) {
		org := Organization{}

		// Initial state
		assert.Equal(t, uuid.Nil, org.ID)
		assert.Nil(t, org.UserID)
		assert.Empty(t, org.Name)

		// Simulate creation
		userID := uuid.New()
		org.UserID = &userID
		org.Name = "New Company"
		org.CompanyURL = "https://newcompany.com"
		org.CompanyDescription = "A new company description"
		org.CompanyTargetingPersona = pq.StringArray{"Early Adopters"}
		org.ValueOffering = "Innovative solutions for modern problems"

		// Verify creation state
		assert.Equal(t, userID, *org.UserID)
		assert.Equal(t, "New Company", org.Name)
		assert.Equal(t, "https://newcompany.com", org.CompanyURL)
		assert.Equal(t, "A new company description", org.CompanyDescription)
		assert.Equal(t, pq.StringArray{"Early Adopters"}, org.CompanyTargetingPersona)
		assert.Equal(t, "Innovative solutions for modern problems", org.ValueOffering)

		// Simulate BeforeCreate
		err := org.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, org.ID)
	})
}
