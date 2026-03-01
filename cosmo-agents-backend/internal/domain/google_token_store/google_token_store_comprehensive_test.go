package google_token_store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleTokenStore_EdgeCaseTimestamps(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected time.Time
	}{
		{
			name: "edge of year boundary",
			input: map[string]interface{}{
				"token":         "year-boundary",
				"refresh_token": "year-refresh",
				"scopes":        []string{"email"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
			expected: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name: "start of year",
			input: map[string]interface{}{
				"token":         "start-year",
				"refresh_token": "start-refresh",
				"scopes":        []string{"email"},
				"expiry":        "2025-01-01T00:00:00Z",
			},
			expected: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "leap year",
			input: map[string]interface{}{
				"token":         "leap-year",
				"refresh_token": "leap-refresh",
				"scopes":        []string{"calendar"},
				"expiry":        "2024-02-29T12:00:00Z",
			},
			expected: time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "early time conversion",
			input: map[string]interface{}{
				"token":         "early-time",
				"refresh_token": "early-refresh",
				"scopes":        []string{"drive"},
				"expiry":        "2024-03-15T03:30:00+07:00",
			},
			expected: time.Date(2024, 3, 14, 20, 30, 0, 0, time.UTC), // Previous day in UTC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewGoogleTokenStoreFromMap(tt.input)
			require.NoError(t, err)
			assert.True(t, tt.expected.Equal(result.Expiry))
		})
	}
}

func TestGoogleTokenStore_ToMapTimezoneHandling(t *testing.T) {
	// Test specific timezone conversion edge cases
	tests := []struct {
		name     string
		input    GoogleTokenStore
		expected string
	}{
		{
			name: "UTC time with +00:00 conversion to Z",
			input: GoogleTokenStore{
				AccessToken:  "utc-test",
				RefreshToken: "utc-refresh",
				Scopes:       []string{"email"},
				Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			expected: "2024-12-31T23:59:59Z",
		},
		{
			name: "Local time conversion to UTC Z format",
			input: GoogleTokenStore{
				AccessToken:  "local-test",
				RefreshToken: "local-refresh",
				Scopes:       []string{"profile"},
				Expiry:       time.Date(2024, 6, 15, 10, 30, 0, 0, time.FixedZone("EST", -5*3600)),
			},
			expected: "2024-06-15T15:30:00Z", // 10:30-05:00 = 15:30UTC
		},
		{
			name: "Time with nanoseconds truncated",
			input: GoogleTokenStore{
				AccessToken:  "nano-test",
				RefreshToken: "nano-refresh",
				Scopes:       []string{"drive"},
				Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 999999999, time.UTC),
			},
			expected: "2024-12-31T23:59:59Z", // Nanoseconds truncated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ToMap()
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result["expiry"])
		})
	}
}

func TestGoogleTokenStore_MalformedInput(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]interface{}
		expectedError string
	}{
		{
			name: "invalid expiry - not RFC3339",
			input: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "refresh123",
				"scopes":        []string{"email"},
				"expiry":        "31-12-2024 23:59:59", // Wrong format
			},
			expectedError: "cannot parse",
		},
		{
			name: "expiry with text instead of time",
			input: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "refresh123",
				"scopes":        []string{"email"},
				"expiry":        "never expires", // Not a time
			},
			expectedError: "cannot parse",
		},
		{
			name: "missing quotes around expiry",
			input: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "refresh123",
				"scopes":        []string{"email"},
				"expiry":        1234567890, // Number instead of string
			},
			expectedError: "json: cannot unmarshal", // JSON unmarshal error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewGoogleTokenStoreFromMap(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Nil(t, result)
		})
	}
}

func TestGoogleTokenStore_ScopeHandling(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
		check func(*testing.T, *GoogleTokenStore)
	}{
		{
			name: "scopes with special characters",
			input: map[string]interface{}{
				"token":         "special-scopes",
				"refresh_token": "special-refresh",
				"scopes":        []string{"email-with-dash", "profile_with_underscore", "scope.with.dots"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
			check: func(t *testing.T, result *GoogleTokenStore) {
				require.NotNil(t, result)
				assert.Equal(t, "special-scopes", result.AccessToken)
				assert.Contains(t, result.Scopes, "email-with-dash")
				assert.Contains(t, result.Scopes, "profile_with_underscore")
				assert.Contains(t, result.Scopes, "scope.with.dots")
			},
		},
		{
			name: "scopes with URL-like patterns",
			input: map[string]interface{}{
				"token":         "url-scopes",
				"refresh_token": "url-refresh",
				"scopes":        []string{"https://www.googleapis.com/auth/gmail.readonly", "https://www.googleapis.com/auth/calendar.events"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
			check: func(t *testing.T, result *GoogleTokenStore) {
				require.NotNil(t, result)
				assert.Len(t, result.Scopes, 2)
				assert.Contains(t, result.Scopes, "https://www.googleapis.com/auth/gmail.readonly")
				assert.Contains(t, result.Scopes, "https://www.googleapis.com/auth/calendar.events")
			},
		},
		{
			name: "duplicate scopes",
			input: map[string]interface{}{
				"token":         "dup-scopes",
				"refresh_token": "dup-refresh",
				"scopes":        []string{"email", "email", "profile"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
			check: func(t *testing.T, result *GoogleTokenStore) {
				require.NotNil(t, result)
				assert.Len(t, result.Scopes, 3) // Preserves duplicates as-is
				assert.Equal(t, "email", result.Scopes[0])
				assert.Equal(t, "email", result.Scopes[1])
				assert.Equal(t, "profile", result.Scopes[2])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewGoogleTokenStoreFromMap(tt.input)
			require.NoError(t, err)
			tt.check(t, result)
		})
	}
}

func TestGoogleTokenStore_EmptyStringFields(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]interface{}
		expectedError string
	}{
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
			name: "empty refresh_token field",
			input: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "",
				"scopes":        []string{"email"},
				"expiry":        "2024-12-31T23:59:59Z",
			},
			expectedError: "missing required token fields",
		},
		{
			name: "empty expiry field",
			input: map[string]interface{}{
				"token":         "access123",
				"refresh_token": "refresh123",
				"scopes":        []string{"email"},
				"expiry":        "",
			},
			expectedError: "missing required token fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewGoogleTokenStoreFromMap(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Nil(t, result)
		})
	}
}

func TestGoogleTokenStore_TimeZoneEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		expiry   string
		expected time.Time
	}{
		{
			name:     "UTC+14 (earliest timezone)",
			expiry:   "2024-12-31T23:59:59+14:00",
			expected: time.Date(2024, 12, 31, 9, 59, 59, 0, time.UTC), // 23:59+14:00 = 09:59UTC previous day
		},
		{
			name:     "UTC-12 (latest timezone)",
			expiry:   "2024-01-01T00:00:00-12:00",
			expected: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), // 00:00-12:00 = 12:00UTC next day
		},
		{
			name:     "Fractional timezone (India)",
			expiry:   "2024-06-15T10:30:00+05:30",
			expected: time.Date(2024, 6, 15, 5, 0, 0, 0, time.UTC), // 10:30+05:30 = 05:00UTC
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]interface{}{
				"token":         "timezone-test",
				"refresh_token": "timezone-refresh",
				"scopes":        []string{"email"},
				"expiry":        tt.expiry,
			}

			result, err := NewGoogleTokenStoreFromMap(input)
			require.NoError(t, err)
			assert.True(t, tt.expected.Equal(result.Expiry))
		})
	}
}

func TestGoogleTokenStore_ToMapNilValueHandling(t *testing.T) {
	tests := []struct {
		name  string
		input GoogleTokenStore
		check func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "nil scopes array",
			input: GoogleTokenStore{
				AccessToken:  "nil-test",
				RefreshToken: "nil-refresh",
				Scopes:       nil,
				Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			check: func(t *testing.T, result map[string]interface{}) {
				assert.Nil(t, result["scopes"])
			},
		},
		{
			name: "empty scopes array",
			input: GoogleTokenStore{
				AccessToken:  "empty-test",
				RefreshToken: "empty-refresh",
				Scopes:       []string{},
				Expiry:       time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			check: func(t *testing.T, result map[string]interface{}) {
				require.NotNil(t, result["scopes"])
				scopes := result["scopes"]
				require.NotNil(t, scopes)
				// Convert to JSON to verify empty array
				jsonData, err := json.Marshal(scopes)
				require.NoError(t, err)
				assert.JSONEq(t, "[]", string(jsonData))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ToMap()
			require.NotNil(t, result)
			tt.check(t, result)
		})
	}
}

func TestGoogleTokenStore_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		scenario func(t *testing.T)
	}{
		{
			name: "Gmail API token",
			scenario: func(t *testing.T) {
				input := map[string]interface{}{
					"token":         "ya29.a0AfH6SMC...", // Real Gmail access token pattern
					"refresh_token": "1//0gxt...",        // Real refresh token pattern
					"scopes": []string{
						"https://www.googleapis.com/auth/gmail.readonly",
						"https://www.googleapis.com/auth/gmail.send",
					},
					"expiry": "2024-12-31T23:59:59Z",
				}

				result, err := NewGoogleTokenStoreFromMap(input)
				require.NoError(t, err)
				assert.Equal(t, "ya29.a0AfH6SMC...", result.AccessToken)
				assert.Equal(t, "1//0gxt...", result.RefreshToken)
				assert.Len(t, result.Scopes, 2)
			},
		},
		{
			name: "Calendar API token",
			scenario: func(t *testing.T) {
				input := map[string]interface{}{
					"token":         "ya29.a0AfH6SMB...",
					"refresh_token": "1//0gxu...",
					"scopes":        []string{"https://www.googleapis.com/auth/calendar"},
					"expiry":        "2024-06-30T15:30:00-04:00", // Eastern Time
				}

				result, err := NewGoogleTokenStoreFromMap(input)
				require.NoError(t, err)
				assert.Equal(t, "ya29.a0AfH6SMB...", result.AccessToken)
				assert.Equal(t, []string{"https://www.googleapis.com/auth/calendar"}, result.Scopes)
				// 15:30-04:00 = 19:30UTC
				expectedTime := time.Date(2024, 6, 30, 19, 30, 0, 0, time.UTC)
				assert.True(t, expectedTime.Equal(result.Expiry))
			},
		},
		{
			name: "Multi-scope token",
			scenario: func(t *testing.T) {
				input := map[string]interface{}{
					"token":         "ya29.a0AfH6SMX...",
					"refresh_token": "1//0gxv...",
					"scopes": []string{
						"https://www.googleapis.com/auth/userinfo.email",
						"https://www.googleapis.com/auth/userinfo.profile",
						"https://www.googleapis.com/auth/drive.file",
						"https://www.googleapis.com/auth/spreadsheets.readonly",
					},
					"expiry": "2024-09-15T08:00:00+09:00", // Japan Standard Time
				}

				result, err := NewGoogleTokenStoreFromMap(input)
				require.NoError(t, err)
				assert.Len(t, result.Scopes, 4)
				assert.Contains(t, result.Scopes, "https://www.googleapis.com/auth/userinfo.email")
				assert.Contains(t, result.Scopes, "https://www.googleapis.com/auth/drive.file")
				// 08:00+09:00 = 23:00UTC previous day
				expectedTime := time.Date(2024, 9, 14, 23, 0, 0, 0, time.UTC)
				assert.True(t, expectedTime.Equal(result.Expiry))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.scenario)
	}
}
