package personal_api_key

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/user"
)

// TestPersonalApiKey_TableName tests the table name method
func TestPersonalApiKey_TableName(t *testing.T) {
	t.Run("Correct table name", func(t *testing.T) {
		apiKey := PersonalApiKey{}
		expected := "personal_api_keys"
		assert.Equal(t, expected, apiKey.TableName())
	})
}

// TestPersonalApiKey_Structure tests the personal API key struct fields
func TestPersonalApiKey_Structure(t *testing.T) {
	t.Run("PersonalApiKey struct has correct fields", func(t *testing.T) {
		userID := uuid.New()
		expiresAt := time.Now().Add(24 * time.Hour)
		lastUsedAt := time.Now().Add(-1 * time.Hour)

		apiKey := PersonalApiKey{
			UserID:     userID,
			Name:       "Test API Key",
			HashedKey:  "hashed_key_value",
			Prefix:     "pk_test_",
			ExpiresAt:  expiresAt,
			LastUsedAt: &lastUsedAt,
		}

		assert.Equal(t, userID, apiKey.UserID)
		assert.Equal(t, "Test API Key", apiKey.Name)
		assert.Equal(t, "hashed_key_value", apiKey.HashedKey)
		assert.Equal(t, "pk_test_", apiKey.Prefix)
		assert.Equal(t, expiresAt, apiKey.ExpiresAt)
		require.NotNil(t, apiKey.LastUsedAt)
		assert.Equal(t, lastUsedAt, *apiKey.LastUsedAt)
	})

	t.Run("PersonalApiKey with nil LastUsedAt", func(t *testing.T) {
		apiKey := PersonalApiKey{
			UserID:     uuid.New(),
			Name:       "API Key",
			HashedKey:  "hashed",
			Prefix:     "pk_",
			ExpiresAt:  time.Now(),
			LastUsedAt: nil,
		}

		assert.Nil(t, apiKey.LastUsedAt)
	})
}

// TestPersonalApiKey_Relationships tests the User relationship
func TestPersonalApiKey_Relationships(t *testing.T) {
	t.Run("User relationship", func(t *testing.T) {
		userID := uuid.New()
		testUser := &user.User{
			Base:  base.Base{ID: userID},
			Email: "test@example.com",
		}

		apiKey := PersonalApiKey{
			UserID: userID,
			User:   testUser,
		}

		require.NotNil(t, apiKey.User)
		assert.Equal(t, userID, apiKey.User.ID)
		assert.Equal(t, "test@example.com", apiKey.User.Email)
	})

	t.Run("Nil user relationship", func(t *testing.T) {
		apiKey := PersonalApiKey{
			UserID: uuid.New(),
		}

		assert.Nil(t, apiKey.User)
	})
}

// TestPersonalApiKey_DefaultValues tests default values for API key fields
func TestPersonalApiKey_DefaultValues(t *testing.T) {
	t.Run("Zero values are correct", func(t *testing.T) {
		apiKey := PersonalApiKey{}

		// UUID fields should be zero
		assert.Equal(t, uuid.Nil, apiKey.UserID)

		// String fields should be empty
		assert.Empty(t, apiKey.Name)
		assert.Empty(t, apiKey.HashedKey)
		assert.Empty(t, apiKey.Prefix)

		// Time fields should be zero
		assert.True(t, apiKey.ExpiresAt.IsZero())
		assert.Nil(t, apiKey.LastUsedAt)

		// Relationship should be nil
		assert.Nil(t, apiKey.User)
	})
}

// TestPersonalApiKey_Inheritance tests that PersonalApiKey properly inherits from base structs
func TestPersonalApiKey_Inheritance(t *testing.T) {
	t.Run("Inherits Base fields", func(t *testing.T) {
		apiKey := PersonalApiKey{}

		// Should have ID field from Base
		_ = apiKey.ID // Just verify field exists

		// Should be able to use BeforeCreate method from Base
		err := apiKey.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, apiKey.ID, "BeforeCreate should set ID")
	})

	t.Run("Inherits TimestampMixin fields", func(t *testing.T) {
		apiKey := PersonalApiKey{}

		// Should have CreatedAt and UpdatedAt fields
		assert.Equal(t, time.Time{}, apiKey.CreatedAt, "CreatedAt should be zero time initially")
		assert.Equal(t, time.Time{}, apiKey.UpdatedAt, "UpdatedAt should be zero time initially")
	})
}

// TestCreatePersonalAPIKeyRequest tests the request struct
func TestCreatePersonalAPIKeyRequest(t *testing.T) {
	t.Run("Request with all fields", func(t *testing.T) {
		expiresAt := FlexibleTime{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		}

		req := CreatePersonalAPIKeyRequest{
			Name:      "Production API Key",
			ExpiresAt: &expiresAt,
		}

		assert.Equal(t, "Production API Key", req.Name)
		require.NotNil(t, req.ExpiresAt)
		assert.True(t, req.ExpiresAt.Valid)
		assert.False(t, req.ExpiresAt.Time.IsZero())
	})

	t.Run("Request without expiration", func(t *testing.T) {
		req := CreatePersonalAPIKeyRequest{
			Name:      "Test Key",
			ExpiresAt: nil,
		}

		assert.Equal(t, "Test Key", req.Name)
		assert.Nil(t, req.ExpiresAt)
	})

	t.Run("Request with empty fields", func(t *testing.T) {
		req := CreatePersonalAPIKeyRequest{}

		assert.Empty(t, req.Name)
		assert.Nil(t, req.ExpiresAt)
	})
}

// TestFlexibleTime_UnmarshalJSON tests the flexible time unmarshaling
func TestFlexibleTime_UnmarshalJSON(t *testing.T) {
	t.Run("RFC3339 string format", func(t *testing.T) {
		testCases := []struct {
			name     string
			json     string
			expected time.Time
			valid    bool
		}{
			{
				name:     "Valid RFC3339",
				json:     `"2024-12-31T23:59:59Z"`,
				expected: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				valid:    true,
			},
			{
				name:     "Valid RFC3339 with microseconds",
				json:     `"2024-12-31T23:59:59.123456Z"`,
				expected: time.Date(2024, 12, 31, 23, 59, 59, 123456000, time.UTC),
				valid:    true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var ft FlexibleTime
				err := ft.UnmarshalJSON([]byte(tc.json))
				require.NoError(t, err)
				assert.Equal(t, tc.valid, ft.Valid)
				assert.Equal(t, tc.expected, ft.Time)
			})
		}
	})

	t.Run("Unix timestamp format", func(t *testing.T) {
		testCases := []struct {
			name     string
			json     string
			expected time.Time
			valid    bool
		}{
			{
				name:     "Unix timestamp (seconds)",
				json:     "1704067200", // 2024-01-01 00:00:00 UTC
				expected: time.Unix(1704067200, 0),
				valid:    true,
			},
			{
				name:     "Unix timestamp (milliseconds)",
				json:     "1704067200000", // 2024-01-01 00:00:00 UTC (ms)
				expected: time.UnixMilli(1704067200000),
				valid:    true,
			},
			{
				name:     "Unix timestamp with decimal",
				json:     "1704067200.123",
				expected: time.Unix(1704067200, 0), // Truncates to seconds
				valid:    true,
			},
			{
				name:     "Negative timestamp (1969)",
				json:     `-86400`, // One day before Unix epoch
				expected: time.Unix(-86400, 0),
				valid:    true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var ft FlexibleTime
				err := ft.UnmarshalJSON([]byte(tc.json))
				require.NoError(t, err)
				assert.Equal(t, tc.valid, ft.Valid)
				assert.Equal(t, tc.expected, ft.Time)
			})
		}
	})

	t.Run("Null and empty values", func(t *testing.T) {
		testCases := []struct {
			name string
			json string
		}{
			{"null JSON", `null`},
			{"empty JSON", `""`},
			{"empty string", `""`},
			{"zero value", `0`},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var ft FlexibleTime
				err := ft.UnmarshalJSON([]byte(tc.json))
				require.NoError(t, err)
				assert.False(t, ft.Valid)
				assert.True(t, ft.Time.IsZero())
			})
		}
	})

	t.Run("Invalid formats", func(t *testing.T) {
		testCases := []struct {
			name string
			json string
		}{
			{"invalid date string", `"invalid-date"`},
			{"invalid JSON structure", `{"time": "now"}`},
			{"boolean", `true`},
			{"array", `[]`},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var ft FlexibleTime
				err := ft.UnmarshalJSON([]byte(tc.json))
				assert.Error(t, err)
			})
		}
	})

	t.Run("Whitespace handling", func(t *testing.T) {
		testCases := []struct {
			name     string
			json     string
			expected time.Time
			valid    bool
		}{
			{
				name:     "RFC3339 with whitespace",
				json:     `  "2024-12-31T23:59:59Z"  `,
				expected: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				valid:    true,
			},
			{
				name:     "Number with whitespace",
				json:     `  1704067200  `,
				expected: time.Unix(1704067200, 0),
				valid:    true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var ft FlexibleTime
				err := ft.UnmarshalJSON([]byte(tc.json))
				require.NoError(t, err)
				assert.Equal(t, tc.valid, ft.Valid)
				assert.Equal(t, tc.expected, ft.Time)
			})
		}
	})
}

// TestPersonalApiKey_RealWorldScenarios tests realistic API key scenarios
func TestPersonalApiKey_RealWorldScenarios(t *testing.T) {
	t.Run("Production API key", func(t *testing.T) {
		userID := uuid.New()
		expiresAt := time.Now().Add(365 * 24 * time.Hour) // 1 year
		lastUsedAt := time.Now().Add(-5 * time.Minute)

		apiKey := PersonalApiKey{
			UserID:     userID,
			Name:       "Production API Key",
			HashedKey:  "hash_5f4dcc3b5aa765d61d8327deb882cf99",
			Prefix:     "pk_prod_1234567890abcdef",
			ExpiresAt:  expiresAt,
			LastUsedAt: &lastUsedAt,
		}

		assert.Equal(t, "Production API Key", apiKey.Name)
		assert.Contains(t, apiKey.Prefix, "pk_prod_")
		assert.True(t, apiKey.ExpiresAt.After(time.Now()))
		require.NotNil(t, apiKey.LastUsedAt)
		assert.True(t, apiKey.LastUsedAt.Before(time.Now()))
	})

	t.Run("Development API key", func(t *testing.T) {
		userID := uuid.New()
		expiresAt := time.Now().Add(7 * 24 * time.Hour) // 1 week

		apiKey := PersonalApiKey{
			UserID:     userID,
			Name:       "Development API Key",
			HashedKey:  "hash_a5b9c3d4e2f1a6b8c0d7e3f5a9b2c4d6",
			Prefix:     "pk_dev_dev123456789",
			ExpiresAt:  expiresAt,
			LastUsedAt: nil, // Never used
		}

		assert.Equal(t, "Development API Key", apiKey.Name)
		assert.Contains(t, apiKey.Prefix, "pk_dev_")
		assert.True(t, apiKey.ExpiresAt.After(time.Now()))
		assert.True(t, apiKey.ExpiresAt.Before(time.Now().Add(8*24*time.Hour)))
		assert.Nil(t, apiKey.LastUsedAt)
	})

	t.Run("Temporary API key", func(t *testing.T) {
		userID := uuid.New()
		expiresAt := time.Now().Add(1 * time.Hour) // 1 hour

		apiKey := PersonalApiKey{
			UserID:     userID,
			Name:       "Temporary Access Key",
			HashedKey:  "hash_098f6bcd4621d373cade4e832627b4f6",
			Prefix:     "pk_temp_abc123",
			ExpiresAt:  expiresAt,
			LastUsedAt: nil,
		}

		assert.Equal(t, "Temporary Access Key", apiKey.Name)
		assert.Contains(t, apiKey.Prefix, "pk_temp_")
		assert.True(t, apiKey.ExpiresAt.After(time.Now()))
		assert.True(t, apiKey.ExpiresAt.Before(time.Now().Add(2*time.Hour)))
	})
}

// TestPersonalApiKey_EdgeCases tests edge cases and unusual but valid scenarios
func TestPersonalApiKey_EdgeCases(t *testing.T) {
	t.Run("Very long name", func(t *testing.T) {
		longName := "This Is A Very Long API Key Name That Might Push The Limits Of What We Consider Reasonable But Should Still Be Valid"
		apiKey := PersonalApiKey{Name: longName}
		assert.Equal(t, longName, apiKey.Name)
		assert.Greater(t, len(apiKey.Name), 80)
	})

	t.Run("Special characters in name", func(t *testing.T) {
		name := "API Key (Production) - 2024 [Confidential]"
		apiKey := PersonalApiKey{Name: name}
		assert.Equal(t, name, apiKey.Name)
		assert.Contains(t, apiKey.Name, "(")
		assert.Contains(t, apiKey.Name, ")")
		assert.Contains(t, apiKey.Name, "[")
		assert.Contains(t, apiKey.Name, "]")
	})

	t.Run("Unicode in name", func(t *testing.T) {
		names := []string{
			"Clé API",   // French
			"APIキー",     // Japanese
			"API密钥",     // Chinese
			"مفتاح API", // Arabic
			"API Ключ",  // Russian
		}

		for _, name := range names {
			apiKey := PersonalApiKey{Name: name}
			assert.Equal(t, name, apiKey.Name)
		}
	})

	t.Run("Hashed key with various patterns", func(t *testing.T) {
		hashedKeys := []string{
			"hash_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
			"$2b$12$hash_with_bcrypt_format_and_salt",
			"sha256:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8",
			"pbkdf2_sha256$260000$abc123$def456", // Django-style
		}

		for _, hashedKey := range hashedKeys {
			apiKey := PersonalApiKey{HashedKey: hashedKey}
			assert.Equal(t, hashedKey, apiKey.HashedKey)
		}
	})

	t.Run("Prefix formats", func(t *testing.T) {
		prefixes := []string{
			"pk_prod_",
			"pk_dev_",
			"pk_test_",
			"pk_temp_",
			"pk_live_",
			"pk_beta_",
			"sk_prod_", // Secret key prefix
			"pk_123abc",
		}

		for _, prefix := range prefixes {
			apiKey := PersonalApiKey{Prefix: prefix}
			assert.Equal(t, prefix, apiKey.Prefix)
			assert.True(t, len(prefix) >= 4) // Should have reasonable length
		}
	})

	t.Run("Edge case expiration times", func(t *testing.T) {
		testCases := []struct {
			name      string
			expiresAt time.Time
		}{
			{
				name:      "Already expired",
				expiresAt: time.Now().Add(-1 * time.Hour),
			},
			{
				name:      "Expires exactly now",
				expiresAt: time.Now(),
			},
			{
				name:      "Expires in 1 second",
				expiresAt: time.Now().Add(1 * time.Second),
			},
			{
				name:      "Expires in 10 years",
				expiresAt: time.Now().Add(10 * 365 * 24 * time.Hour),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				apiKey := PersonalApiKey{ExpiresAt: tc.expiresAt}
				assert.Equal(t, tc.expiresAt, apiKey.ExpiresAt)
			})
		}
	})
}

// TestPersonalApiKey_FieldManipulation tests field manipulation scenarios
func TestPersonalApiKey_FieldManipulation(t *testing.T) {
	t.Run("Update LastUsedAt", func(t *testing.T) {
		apiKey := PersonalApiKey{}
		assert.Nil(t, apiKey.LastUsedAt)

		// Set LastUsedAt
		now := time.Now()
		apiKey.LastUsedAt = &now
		require.NotNil(t, apiKey.LastUsedAt)
		assert.Equal(t, now, *apiKey.LastUsedAt)

		// Update LastUsedAt
		later := now.Add(1 * time.Hour)
		apiKey.LastUsedAt = &later
		assert.Equal(t, later, *apiKey.LastUsedAt)
	})

	t.Run("Change expiration", func(t *testing.T) {
		apiKey := PersonalApiKey{
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		originalExpiry := apiKey.ExpiresAt
		newExpiry := time.Now().Add(24 * time.Hour)

		apiKey.ExpiresAt = newExpiry
		assert.Equal(t, newExpiry, apiKey.ExpiresAt)
		assert.True(t, apiKey.ExpiresAt.After(originalExpiry))
	})
}

// TestPersonalApiKey_ValidationScenarios tests validation logic
func TestPersonalApiKey_ValidationScenarios(t *testing.T) {
	t.Run("Valid API key configuration", func(t *testing.T) {
		userID := uuid.New()
		apiKey := PersonalApiKey{
			UserID:     userID,
			Name:       "Valid API Key",
			HashedKey:  "hash_with_sufficient_length_123456789",
			Prefix:     "pk_valid_123",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
			LastUsedAt: nil,
		}

		// These would typically be validated at the service/repository layer
		assert.NotEqual(t, uuid.Nil, apiKey.UserID)
		assert.NotEmpty(t, apiKey.Name)
		assert.NotEmpty(t, apiKey.HashedKey)
		assert.NotEmpty(t, apiKey.Prefix)
		assert.False(t, apiKey.ExpiresAt.IsZero())
		assert.True(t, apiKey.ExpiresAt.After(time.Now()))
	})

	t.Run("API key with required fields", func(t *testing.T) {
		apiKey := PersonalApiKey{
			UserID:    uuid.New(),
			Name:      "Required Fields Key",
			HashedKey: "required_hash",
			Prefix:    "pk_req_",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		// All required fields are present
		assert.NotEqual(t, uuid.Nil, apiKey.UserID)
		assert.NotEmpty(t, apiKey.Name)
		assert.NotEmpty(t, apiKey.HashedKey)
		assert.NotEmpty(t, apiKey.Prefix)
		assert.False(t, apiKey.ExpiresAt.IsZero())
	})
}

// TestPersonalApiKey_JSONSerialization tests JSON serialization
func TestPersonalApiKey_JSONSerialization(t *testing.T) {
	t.Run("Serialize personal API key", func(t *testing.T) {
		userID := uuid.New()
		lastUsedAt := time.Now().Add(-1 * time.Hour)
		apiKey := PersonalApiKey{
			Base:       base.Base{ID: uuid.New()},
			UserID:     userID,
			Name:       "JSON Test Key",
			Prefix:     "pk_json_123",
			ExpiresAt:  time.Now().Add(24 * time.Hour),
			LastUsedAt: &lastUsedAt,
		}

		jsonData, err := json.Marshal(apiKey)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Verify it contains expected fields
		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, "JSON Test Key")
		assert.Contains(t, jsonStr, "pk_json_123")
		assert.Contains(t, jsonStr, userID.String())
	})

	t.Run("Serialize CreatePersonalAPIKeyRequest", func(t *testing.T) {
		expiresAt := FlexibleTime{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		}

		req := CreatePersonalAPIKeyRequest{
			Name:      "Request Test",
			ExpiresAt: &expiresAt,
		}

		jsonData, err := json.Marshal(req)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Verify it contains expected fields
		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, "Request Test")
	})
}

// TestPersonalApiKey_ComparisonAndEquality tests comparison and equality
func TestPersonalApiKey_ComparisonAndEquality(t *testing.T) {
	t.Run("Same API key comparison", func(t *testing.T) {
		userID := uuid.New()
		apiKey1 := PersonalApiKey{
			Base:      base.Base{ID: uuid.New()},
			UserID:    userID,
			Name:      "Same Name",
			HashedKey: "same_hash",
			Prefix:    "pk_same_123",
		}
		apiKey2 := PersonalApiKey{
			UserID:    userID,
			Name:      "Same Name",
			HashedKey: "same_hash",
			Prefix:    "pk_same_123",
		}

		assert.Equal(t, apiKey1.UserID, apiKey2.UserID)
		assert.Equal(t, apiKey1.Name, apiKey2.Name)
		assert.Equal(t, apiKey1.HashedKey, apiKey2.HashedKey)
		assert.Equal(t, apiKey1.Prefix, apiKey2.Prefix)
		assert.Equal(t, apiKey1.TableName(), apiKey2.TableName())
	})

	t.Run("Different API key comparison", func(t *testing.T) {
		userID1 := uuid.New()
		userID2 := uuid.New()

		apiKey1 := PersonalApiKey{
			UserID:    userID1,
			Name:      "Key A",
			HashedKey: "hash_a",
			Prefix:    "pk_a_123",
		}
		apiKey2 := PersonalApiKey{
			UserID:    userID2,
			Name:      "Key B",
			HashedKey: "hash_b",
			Prefix:    "pk_b_456",
		}

		assert.NotEqual(t, apiKey1.UserID, apiKey2.UserID)
		assert.NotEqual(t, apiKey1.Name, apiKey2.Name)
		assert.NotEqual(t, apiKey1.HashedKey, apiKey2.HashedKey)
		assert.NotEqual(t, apiKey1.Prefix, apiKey2.Prefix)
		assert.Equal(t, apiKey1.TableName(), apiKey2.TableName()) // Table name should always be same
	})
}
