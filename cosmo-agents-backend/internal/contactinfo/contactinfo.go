package contactinfo

import (
	"encoding/json"
	"strings"
)

// FromProfile digs an email address out of a contact's profile JSONB.
//
// The value can sit in three places. `profile.email` is the intended home, but
// the create endpoint has no contact_information member on its request struct,
// so a value posted under that name is swallowed by ExtraFields and stored at
// profile.custom_fields.contact_information instead. Contacts created that way
// have an empty column and a null profile.email, and the send worker used to
// hand Gmail an empty recipient.
func FromProfile(profile []byte) string {
	if len(profile) == 0 {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(profile, &data); err != nil {
		return ""
	}

	if email, ok := data["email"].(string); ok && isEmail(email) {
		return email
	}

	custom, ok := data["custom_fields"].(map[string]interface{})
	if !ok {
		return ""
	}

	for _, key := range []string{"email", "contact_information"} {
		switch field := custom[key].(type) {
		case string:
			if isEmail(field) {
				return field
			}
		case map[string]interface{}:
			// AI-written custom fields are {value, source, updated_at}.
			if value, ok := field["value"].(string); ok && isEmail(value) {
				return value
			}
		}
	}

	return ""
}

// Resolve returns the contact's email, preferring the dedicated column.
func Resolve(contactInformation string, profile []byte) string {
	if isEmail(contactInformation) {
		return contactInformation
	}
	return FromProfile(profile)
}

func isEmail(value string) bool {
	trimmed := strings.TrimSpace(value)
	if !strings.Contains(trimmed, "@") {
		return false
	}
	// Placeholders written by the importers when no real address is known.
	if strings.Contains(trimmed, "@linkedin.placeholder") ||
		strings.Contains(trimmed, "@na.local") ||
		strings.HasPrefix(trimmed, "unknown-") {
		return false
	}
	return true
}
