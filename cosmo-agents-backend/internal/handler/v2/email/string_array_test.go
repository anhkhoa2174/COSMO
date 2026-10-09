package email

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestStringArrayToSlice(t *testing.T) {
	assert.Equal(t, []string{"Interested"}, stringArrayToSlice(pq.StringArray{"Interested"}))
	assert.Equal(t, []string{"a"}, stringArrayToSlice([]string{"a"}))
	assert.Equal(t, []string{}, stringArrayToSlice(nil))
	assert.Equal(t, []string{}, stringArrayToSlice(pq.StringArray(nil)))
}
