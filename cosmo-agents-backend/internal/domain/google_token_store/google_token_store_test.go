package google_token_store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleTokenStore_NewGoogleTokenStoreFromMap(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]interface{}
		expected      *GoogleTokenStore
		expectedError string
	}{
		{
			name: "valid token data with Z suffix",
			input: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "refresh456",
				"scopes":        []string{"email", "profile"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
			expected: &GoogleTokenStore{
				AccessToken:  "access123",
				RefreshToken: "refresh456",
				Scopes:       []string{"email", "profile"},
				Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
		},
		{
			name: "valid token data with timezone offset",
			input: map[string]interface{}{
				"token":         "access789",
				"refresh_token": "refresh012",
				"scopes":        []string{"https://www.googleapis.com/auth/calendar"},
				"expiry":        "2024-06-15T10:30:00+07:00",
			},
			expected: &GoogleTokenStore{
				AccessToken:  "access789",
				RefreshToken: "refresh012",
				Scopes:       []string{"https://www.googleapis.com/auth/calendar"},
				Expiry:       time.Date(2024, 6, 15, 3, 30, 0, 0, time.UTC), // 10:30+07:00 = 03:30UTC
			},
		},
		{
			name: "valid token data with empty scopes",
			input: map[string]interface{}{
				"token":         "access-scopes",
				"refresh_token": "refresh-scopes",
				"scopes":        []string{},
				"expiry":        "2024-03-01T12:00:00Z",
			},
			expected: &GoogleTokenStore{
				AccessToken:  "access-scopes",
				RefreshToken: "refresh-scopes",
				Scopes:       []string{},
				Expiry:       time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "valid token data with nil scopes",
			input: map[string]interface{}{
				"token":         "access-nil",
				"refresh_token": "refresh-nil",
				"expiry":        "2024-08-20T15:45:30Z",
			},
			expected: &GoogleTokenStore{
				AccessToken:  "access-nil",
				RefreshToken: "refresh-nil",
				Scopes:       nil,
				Expiry:       time.Date(2024, 8, 20, 15, 45, 30, 0, time.UTC),
			},
		},
		{
			name: "missing token field",
			input: map[string]interface{}{
				"refresh_token": "refresh123",
				"scopes":        []string{"email"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
			expectedError: "missing required token fields",
		},
		{
			name: "missing refresh_token field",
			input: map[string]interface{}{
				"token":  "access123",
				"scopes": []string{"email"},
				"expiry": "2024-12-31T23:59:59Z",
			},
			expectedError: "missing required token fields",
		},
		{
			name: "missing expiry field",
			input: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "refresh123",
				"scopes":        []string{"email"},
			},
			expectedError: "missing required token fields",
		},
		{
			name: "empty token field",
			input: map[string]interface{}{
				"token":         "",
				"refresh_token": "refresh123",
				"scopes":        []string{"email"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
			expectedError: "missing required token fields",
		},
		{
			name: "invalid JSON structure",
			input: map[string]interface{}{
				"token": func() {}, // invalid JSON type
			},
			expectedError: "json: unsupported type",
		},
		{
			name: "invalid expiry format",
			input: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "refresh123",
				"scopes":        []string{"email"},
				"expiry":        "not-a-date",
			},
			expectedError: "cannot parse",
		},
		{
			name: "expiry with microseconds",
			input: map[string]interface{}{
				"token":         "access-micro",
				"refresh_token": "refresh-micro",
				"scopes":        []string{"drive"},
				"expiry":        "2024-12-31T23:59:59.123456Z",
			},
			expected: &GoogleTokenStore{
				AccessToken:  "access-micro",
				RefreshToken: "refresh-micro",
				Scopes:       []string{"drive"},
				Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 123456000, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewGoogleTokenStoreFromMap(tt.input)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.AccessToken, result.AccessToken)
				assert.Equal(t, tt.expected.RefreshToken, result.RefreshToken)
				assert.Equal(t, tt.expected.Scopes, result.Scopes)
				assert.True(t, tt.expected.Expiry.Equal(result.Expiry),
					"Expected expiry %v, got %v", tt.expected.Expiry, result.Expiry)
			}
		})
	}
}

func TestGoogleTokenStore_ToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    GoogleTokenStore
		expected map[string]interface{}
	}{
		{
			name: "basic token conversion",
			input: GoogleTokenStore{
				AccessToken:  "access123",
				RefreshToken: "refresh456",
				Scopes:       []string{"email", "profile"},
				Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			expected: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "refresh456",
				"scopes":        []interface{}{"email", "profile"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
		},
		{
			name: "token with non-UTC time",
			input: GoogleTokenStore{
				AccessToken:  "local-time",
				RefreshToken: "refresh-local",
				Scopes:       []string{"calendar"},
				Expiry:       time.Date(2024, 6, 15, 10, 30, 0, 0, time.FixedZone("UTC+7", 7*3600)),
			},
			expected: map[string]interface{}{
				"token":         "local-time",
				"refresh_token": "refresh-local",
				"scopes":        []interface{}{"calendar"},
				"expiry":        "2024-06-15T03:30:00Z", // Converted to UTC
			},
		},
		{
			name: "token with nanoseconds",
			input: GoogleTokenStore{
				AccessToken:  "nano-access",
				RefreshToken: "nano-refresh",
				Scopes:       []string{"drive"},
				Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 500000000, time.UTC),
			},
			expected: map[string]interface{}{
				"token":         "nano-access",
				"refresh_token": "nano-refresh",
				"scopes":        []interface{}{"drive"},
				"expiry":        "2024-12-31T23:59:59Z", // Nanoseconds truncated
			},
		},
		{
			name: "empty scopes",
			input: GoogleTokenStore{
				AccessToken:  "empty-scopes",
				RefreshToken: "refresh-empty",
				Scopes:       []string{},
				Expiry:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: map[string]interface{}{
				"token":         "empty-scopes",
				"refresh_token": "refresh-empty",
				"scopes":        []interface{}{},
				"expiry":        "2024-01-01T00:00:00Z",
			},
		},
		{
			name: "nil scopes",
			input: GoogleTokenStore{
				AccessToken:  "nil-scopes",
				RefreshToken: "refresh-nil",
				Scopes:       nil,
				Expiry:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: map[string]interface{}{
				"token":         "nil-scopes",
				"refresh_token": "refresh-nil",
				"scopes":        nil,
				"expiry":        "2024-01-01T00:00:00Z",
			},
		},
		{
			name: "single scope",
			input: GoogleTokenStore{
				AccessToken:  "single-scope",
				RefreshToken: "refresh-single",
				Scopes:       []string{"https://www.googleapis.com/auth/gmail.readonly"},
				Expiry:       time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
			expected: map[string]interface{}{
				"token":         "single-scope",
				"refresh_token": "refresh-single",
				"scopes":        []interface{}{"https://www.googleapis.com/auth/gmail.readonly"},
				"expiry":        "2024-03-15T12:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ToMap()

			require.NotNil(t, result)
			assert.Equal(t, tt.expected["token"], result["token"])
			assert.Equal(t, tt.expected["refresh_token"], result["refresh_token"])
			assert.Equal(t, tt.expected["expiry"], result["expiry"])

			// Handle scopes comparison with nil check
			if tt.expected["scopes"] == nil {
				assert.Nil(t, result["scopes"])
			} else {
				require.NotNil(t, result["scopes"])
				expectedScopes := tt.expected["scopes"].([]interface{})
				actualScopes := result["scopes"]

				// Convert to JSON and back for proper comparison
				expectedJSON, err := json.Marshal(expectedScopes)
				require.NoError(t, err)
				actualJSON, err := json.Marshal(actualScopes)
				require.NoError(t, err)
				assert.JSONEq(t, string(expectedJSON), string(actualJSON))
			}
		})
	}
}

func TestGoogleTokenStore_RoundTripConversion(t *testing.T) {
	original := GoogleTokenStore{
		AccessToken:  "round-trip-access",
		RefreshToken: "round-trip-refresh",
		Scopes:       []string{"email", "profile", "https://www.googleapis.com/auth/calendar"},
		Expiry:       time.Date(2024, 12, 25, 15, 30, 45, 123456789, time.UTC),
	}

	// Convert to map
	tokenMap := original.ToMap()
	require.NotNil(t, tokenMap)

	// Convert back from map
	reconstructed, err := NewGoogleTokenStoreFromMap(tokenMap)
	require.NoError(t, err)
	require.NotNil(t, reconstructed)

	// Verify all fields match (except nanoseconds are truncated)
	assert.Equal(t, original.AccessToken, reconstructed.AccessToken)
	assert.Equal(t, original.RefreshToken, reconstructed.RefreshToken)
	assert.Equal(t, original.Scopes, reconstructed.Scopes)
	assert.True(t, original.Expiry.Truncate(time.Second).Equal(reconstructed.Expiry),
		"Expected expiry %v, got %v", original.Expiry.Truncate(time.Second), reconstructed.Expiry)
}

func TestGoogleTokenStore_JSONSerialization(t *testing.T) {
	// Test that the structure can be properly JSON serialized
	tokenStore := GoogleTokenStore{
		AccessToken:  "json-test-access",
		RefreshToken: "json-test-refresh",
		Scopes:       []string{"email", "profile"},
		Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	// Convert to map and then to JSON
	tokenMap := tokenStore.ToMap()
	jsonData, err := json.Marshal(tokenMap)
	require.NoError(t, err)

	// Parse back from JSON
	var parsedMap map[string]interface{}
	err = json.Unmarshal(jsonData, &parsedMap)
	require.NoError(t, err)

	// Reconstruct from parsed map
	reconstructed, err := NewGoogleTokenStoreFromMap(parsedMap)
	require.NoError(t, err)

	assert.Equal(t, tokenStore.AccessToken, reconstructed.AccessToken)
	assert.Equal(t, tokenStore.RefreshToken, reconstructed.RefreshToken)
	assert.Equal(t, tokenStore.Scopes, reconstructed.Scopes)
	assert.True(t, tokenStore.Expiry.Equal(reconstructed.Expiry))
}

func TestGoogleTokenStore_ExpiryTimeFormats(t *testing.T) {
	tests := []struct {
		name        string
		inputExpiry string
		expectTime  time.Time
	}{
		{
			name:        "RFC3339 with Z suffix",
			inputExpiry: "2024-12-31T23:59:59Z",
			expectTime:  time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:        "RFC3339 with timezone offset",
			inputExpiry: "2024-06-15T10:30:00+07:00",
			expectTime:  time.Date(2024, 6, 15, 3, 30, 0, 0, time.UTC), // 10:30+07:00 = 03:30UTC
		},
		{
			name:        "RFC3339 with negative timezone offset",
			inputExpiry: "2024-01-01T12:00:00-05:00",
			expectTime:  time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC), // 12:00-05:00 = 17:00UTC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]interface{}{
				"token":         "test-access",
				"refresh_token": "test-refresh",
				"scopes":        []string{"email"},
				"expiry":        tt.inputExpiry,
			}

			result, err := NewGoogleTokenStoreFromMap(input)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.True(t, tt.expectTime.Equal(result.Expiry),
				"Expected expiry %v, got %v", tt.expectTime, result.Expiry)
		})
	}
}

func TestGoogleTokenStore_EdgeCases(t *testing.T) {
	t.Run("empty input map", func(t *testing.T) {
		result, err := NewGoogleTokenStoreFromMap(map[string]interface{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required token fields")
		assert.Nil(t, result)
	})

	t.Run("nil input map", func(t *testing.T) {
		result, err := NewGoogleTokenStoreFromMap(nil)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("additional fields in input", func(t *testing.T) {
		input := map[string]interface{}{
			"token":         "access123",
			"refresh_token": "refresh123",
			"scopes":        []string{"email"},
			"expiry":        "2024-12-31T23:59:59Z",
			"extra_field":   "should_be_ignored",
			"another_field": 42,
		}

		result, err := NewGoogleTokenStoreFromMap(input)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "access123", result.AccessToken)
		assert.Equal(t, "refresh123", result.RefreshToken)
		assert.Equal(t, []string{"email"}, result.Scopes)
	})

	t.Run("scopes with special characters", func(t *testing.T) {
		input := map[string]interface{}{
			"token":         "special-access",
			"refresh_token": "special-refresh",
			"scopes":        []string{"https://www.googleapis.com/auth/gmail.readonly", "https://www.googleapis.com/auth/calendar.events"},
			"expiry":        "2024-12-31T23:59:59Z",
		}

		result, err := NewGoogleTokenStoreFromMap(input)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, []string{"https://www.googleapis.com/auth/gmail.readonly", "https://www.googleapis.com/auth/calendar.events"}, result.Scopes)
	})
}

func TestGoogleTokenStore_Performance(t *testing.T) {
	// Test performance with large scope arrays
	largeScopes := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		largeScopes[i] = "https://www.googleapis.com/auth/scope" + string(rune(i))
	}

	input := map[string]interface{}{
		"token":         "perf-access",
		"refresh_token": "perf-refresh",
		"scopes":        largeScopes,
		"expiry":        "2024-12-31T23:59:59Z",
	}

	start := time.Now()
	result, err := NewGoogleTokenStoreFromMap(input)
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, len(largeScopes), len(result.Scopes))
	assert.Less(t, duration, 100*time.Millisecond, "Operation should complete quickly")
}
