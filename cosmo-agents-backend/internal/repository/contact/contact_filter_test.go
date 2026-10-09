package contact

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

func TestNormalizeContactFilter_StringToILike(t *testing.T) {
	// address stands in for a free-text column (job_title, company, city,
	// country and state get fuzzy handling of their own). The test used email, which
	// migration 000043 removed from contacts; filtering on it produced SQL
	// against a column that no longer exists.
	in := map[string]interface{}{
		"address": "abc",
	}
	out := NormalizeContactFilter(in)
	val, ok := out["address"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "%abc%", val[string(baseRepo.OpILike)])
}

func TestNormalizeContactFilter_SliceToIn(t *testing.T) {
	in := map[string]interface{}{
		"tags": []string{"a", "b"},
	}
	out := NormalizeContactFilter(in)
	val, ok := out["tags"].(map[string]interface{})
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"a", "b"}, val[string(baseRepo.OpIn)])
}

func TestNormalizeContactFilter_LogicalPassThrough(t *testing.T) {
	in := map[string]interface{}{
		string(baseRepo.OpAnd): []interface{}{"a", "b"},
		"company":              "",
	}
	out := NormalizeContactFilter(in)
	assert.Contains(t, out, string(baseRepo.OpAnd))
	assert.NotContains(t, out, "company") // empty strings are dropped
}

func TestNormalizeContactFilter_MapPassthrough(t *testing.T) {
	sub := map[string]interface{}{string(baseRepo.OpEqual): "x"}
	in := map[string]interface{}{
		"organization_id": uuid.New(),
		"meta":            sub,
	}
	out := NormalizeContactFilter(in)
	assert.Equal(t, sub, out["meta"])
	assert.Equal(t, in["organization_id"], out["organization_id"])
}
