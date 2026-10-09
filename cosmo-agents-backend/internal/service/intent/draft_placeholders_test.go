package intent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTidyDraftPlaceholders(t *testing.T) {
	in := "Thanks!\n\nBest,  \n[Your Name]  \n[Your Position]  \nCOSMO Team\n\nBook here: [Book a 20-minute demo](https://x)"
	got := tidyDraftPlaceholders(in, "Trần Anh Khoa")
	assert.Equal(t, "Thanks!\n\nBest,\nTrần Anh Khoa\nCOSMO Team\n\nBook here: [Book a 20-minute demo](https://x)", got)

	t.Run("no name: the placeholder goes, nothing is invented", func(t *testing.T) {
		assert.Equal(t, "Best,\nAcme", tidyDraftPlaceholders("Best,\n[Your Name]\nAcme", ""))
	})
	t.Run("no brackets: untouched", func(t *testing.T) {
		assert.Equal(t, "Hi there", tidyDraftPlaceholders("Hi there", "Khoa"))
	})
}
