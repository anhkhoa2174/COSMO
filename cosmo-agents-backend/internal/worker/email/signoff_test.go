package email

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// Generated templates close with "Best," and the signature opens with "Best
// regards,", so prospects received both lines back to back.
func TestDropDuplicateSignOff(t *testing.T) {
	sig := "Best regards,\n\nTrần Anh Khoa\n\ntiger tribe"
	tests := []struct{ name, template, want string }{
		{"closing before the tag is dropped", "Hello,\n\nLet's catch up!\n\nBest,\n{agent_signature}", "Hello,\n\nLet's catch up!\n\n{agent_signature}"},
		{"closing with blank lines between is dropped", "Thanks for your time.\nBest regards,\n\n{agent_signature}", "Thanks for your time.\n\n{agent_signature}"},
		{"vietnamese closing is dropped", "Em cảm ơn anh.\nTrân trọng,\n{agent_signature}", "Em cảm ơn anh.\n{agent_signature}"},
		{"a normal sentence is kept", "Looking forward to hearing from you!\n{agent_signature}", "Looking forward to hearing from you!\n{agent_signature}"},
		{"no tag: untouched", "Best,\nKhoa", "Best,\nKhoa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, dropDuplicateSignOff(tt.template, sig))
		})
	}
	t.Run("signature without a closing keeps the template's", func(t *testing.T) {
		in := "Best,\n{agent_signature}"
		assert.Equal(t, in, dropDuplicateSignOff(in, "Trần Anh Khoa\ntiger tribe"))
	})
}

func TestApplyTemplateDataRendersSingleSignOffAndOrganisation(t *testing.T) {
	org := uuid.New()
	agent := &domain.Agent{Name: "Trần Anh Khoa", OrganizationID: &org, Signature: "Best regards,\n\n{sender_name}\n\n{organization_name}"}
	contact := &domain.Contact{Name: "Nguyễn Minh Khang", Company: "VietPay"}
	data := buildTemplateData(agent, contact, "tiger tribe")

	subject := applyTemplateData("We Miss You at {organization_name}!", data)
	assert.Equal(t, "We Miss You at tiger tribe!", subject, "the organisation name used to be hard-coded empty on send")

	body := applyTemplateData("Hi {contact_first_name},\n\nLet's catch up!\n\nBest,\n{agent_signature}", data)
	assert.Equal(t, "Hi Nguyễn,\n\nLet's catch up!\n\n\nBest regards,\n\nTrần Anh Khoa\n\ntiger tribe", body)
	assert.Equal(t, 1, countOccurrences(body, "Best"), "one closing, not two")
}

func countOccurrences(s, sub string) int {
	n := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			n++
		}
	}
	return n
}
