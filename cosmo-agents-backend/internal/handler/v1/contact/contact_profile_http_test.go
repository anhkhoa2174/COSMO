package contact

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// The enrichment prompt grounds its insights in profile.interests,
// profile.research_findings and profile.custom_fields. A contact created with
// those under "profile" used to lose all three.
func TestCreateContactKeepsNestedProfile(t *testing.T) {
	tn := newTenancy(t)
	status, _ := tn.do(t, http.MethodPost, "/v1/contacts", tn.adminA, map[string]interface{}{
		"name": "Nguyễn Minh Khang", "email": "khang@vietpay.example", "company": "VietPay",
		"job_title": "CTO", "industry": "Fintech",
		"profile": map[string]interface{}{
			"interests":           []string{"payments infrastructure"},
			"company_description": "Series A fintech in Ho Chi Minh City",
			"research_findings":   []map[string]string{{"fact": "hiring 3 sales reps", "source": "LinkedIn"}},
			"custom_fields":       map[string]string{"sales_team_size": "4"},
		},
	})
	require.Contains(t, []int{http.StatusOK, http.StatusCreated}, status)

	var c domain.Contact
	require.NoError(t, tn.db.Where("user_id = ? AND name = ?", tn.adminA.ID, "Nguyễn Minh Khang").First(&c).Error)
	var profile map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Profile, &profile))
	assert.Equal(t, []interface{}{"payments infrastructure"}, profile["interests"])
	assert.Equal(t, "Series A fintech in Ho Chi Minh City", profile["company_description"])
	assert.Len(t, profile["research_findings"], 1)
	assert.Equal(t, "khang@vietpay.example", profile["email"])
	cf, _ := profile["custom_fields"].(map[string]interface{})
	require.Contains(t, cf, "sales_team_size")
	assert.Equal(t, "4", cf["sales_team_size"].(map[string]interface{})["value"])
}
