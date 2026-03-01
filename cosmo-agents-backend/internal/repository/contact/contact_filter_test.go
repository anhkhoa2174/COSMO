package contact

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

func TestNormalizeContactFilter_StringToILike(t *testing.T) {
	in := map[string]interface{}{
		"email": "abc",
	}
	out := normalizeContactFilter(in)
	val, ok := out["email"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "%abc%", val[string(baseRepo.OpILike)])
}

func TestNormalizeContactFilter_SliceToIn(t *testing.T) {
	in := map[string]interface{}{
		"tags": []string{"a", "b"},
	}
	out := normalizeContactFilter(in)
	val, ok := out["tags"].(map[string]interface{})
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"a", "b"}, val[string(baseRepo.OpIn)])
}

func TestNormalizeContactFilter_LogicalPassThrough(t *testing.T) {
	in := map[string]interface{}{
		string(baseRepo.OpAnd): []interface{}{"a", "b"},
		"company":              "",
	}
	out := normalizeContactFilter(in)
	assert.Contains(t, out, string(baseRepo.OpAnd))
	assert.NotContains(t, out, "company") // empty strings are dropped
}

func TestNormalizeContactFilter_MapPassthrough(t *testing.T) {
	sub := map[string]interface{}{string(baseRepo.OpEqual): "x"}
	in := map[string]interface{}{
		"organization_id": uuid.New(),
		"meta":            sub,
	}
	out := normalizeContactFilter(in)
	assert.Equal(t, sub, out["meta"])
	assert.Equal(t, in["organization_id"], out["organization_id"])
}
