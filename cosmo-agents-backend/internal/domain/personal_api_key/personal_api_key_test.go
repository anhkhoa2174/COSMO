package personal_api_key

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPersonalApiKeyFields(t *testing.T) {
	expires := time.Now().Add(24 * time.Hour)
	key := PersonalApiKey{
		UserID:    uuid.New(),
		Name:      "CLI",
		HashedKey: "hashed-value",
		Prefix:    "pk_123",
		ExpiresAt: expires,
	}

	if key.TableName() != "personal_api_keys" {
		t.Fatalf("unexpected table name")
	}

	if key.ExpiresAt.IsZero() {
		t.Fatalf("expected expires_at set")
	}
}
