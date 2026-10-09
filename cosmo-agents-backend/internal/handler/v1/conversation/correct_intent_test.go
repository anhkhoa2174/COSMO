package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func TestNormalizeIntent(t *testing.T) {
	t.Run("accepts the stored display string", func(t *testing.T) {
		got, ok := normalizeIntent("Do not contact")
		assert.True(t, ok)
		assert.Equal(t, domain.IntentDoNotContact, got)
	})

	t.Run("accepts the SCREAMING_SNAKE form the UI works in", func(t *testing.T) {
		got, ok := normalizeIntent("REQUEST_FOR_PRICING")
		assert.True(t, ok)
		assert.Equal(t, domain.IntentRequestForPricing, got)
	})

	t.Run("tolerates casing, hyphens and stray whitespace", func(t *testing.T) {
		for _, raw := range []string{"out of office", " Out-Of-Office ", "OUT_OF_OFFICE"} {
			got, ok := normalizeIntent(raw)
			assert.True(t, ok, raw)
			assert.Equal(t, domain.IntentOutOfOffice, got, raw)
		}
	})

	t.Run("every canonical label round-trips through its own display string", func(t *testing.T) {
		all := []domain.IntentType{
			domain.IntentInterested,
			domain.IntentNotInterested,
			domain.IntentReferral,
			domain.IntentRequestForPricing,
			domain.IntentRequestForInfo,
			domain.IntentNurture,
			domain.IntentDoNotContact,
			domain.IntentOutOfOffice,
			domain.IntentUnknown,
		}
		for _, intent := range all {
			got, ok := normalizeIntent(string(intent))
			assert.True(t, ok, string(intent))
			assert.Equal(t, intent, got, string(intent))
		}
	})

	t.Run("rejects anything outside the taxonomy", func(t *testing.T) {
		for _, raw := range []string{"", "maybe", "Very interested", "DROP TABLE"} {
			_, ok := normalizeIntent(raw)
			assert.False(t, ok, raw)
		}
	})
}
