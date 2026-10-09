package contact

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Everything the caller sends under "profile" must survive into the stored
// profile; it used to be dropped wholesale when the "profile" key was deleted.
func TestMergeProfileRequest(t *testing.T) {
	profile := map[string]interface{}{
		"email":   "khang@vietpay.example",
		"company": "VietPay",
		"city":    "",
		"custom_fields": map[string]interface{}{
			"from_extra": map[string]interface{}{"value": "x", "updated_at": "2026-10-01T00:00:00Z"},
		},
	}
	mergeProfileRequest(profile, map[string]interface{}{
		"interests":           []interface{}{"payments", "AI"},
		"company_description": "Series A fintech",
		"research_findings":   []interface{}{map[string]interface{}{"fact": "hiring 3 reps"}},
		"email":               "other@example.com", // standard column already set: ignored
		"city":                "Ho Chi Minh City",  // empty standard field: filled
		"custom_fields": map[string]interface{}{
			"company_size": "45", // plain value: wrapped
			"tech_stack":   map[string]interface{}{"value": "Go", "updated_at": "2026-09-01T00:00:00Z"},
			"id":           "ignored",
		},
		"ai_insights":     map[string]interface{}{"forged": true}, // reserved: ignored
		"organization_id": "11111111-1111-1111-1111-111111111111",
		"profile":         map[string]interface{}{"nested": true},
	})

	assert.Equal(t, []interface{}{"payments", "AI"}, profile["interests"])
	assert.Equal(t, "Series A fintech", profile["company_description"])
	assert.Len(t, profile["research_findings"], 1)
	assert.Equal(t, "khang@vietpay.example", profile["email"], "a set standard field is not overridden")
	assert.Equal(t, "Ho Chi Minh City", profile["city"], "an empty standard field is filled")
	assert.NotContains(t, profile, "ai_insights")
	assert.NotContains(t, profile, "organization_id")
	assert.NotContains(t, profile, "profile")

	cf := profile["custom_fields"].(map[string]interface{})
	assert.Contains(t, cf, "from_extra", "custom fields from extra fields are kept")
	assert.Equal(t, "45", cf["company_size"].(map[string]interface{})["value"], "plain values are wrapped")
	assert.NotEmpty(t, cf["company_size"].(map[string]interface{})["updated_at"])
	assert.Equal(t, "Go", cf["tech_stack"].(map[string]interface{})["value"], "already-wrapped values are kept as-is")
	assert.NotContains(t, cf, "id")
}
