package facebook_token

import (
	"testing"

	"github.com/google/uuid"
)

func TestFacebookTokenFields(t *testing.T) {
	token := FacebookToken{
		UserID:      uuid.New(),
		FBUserID:    "user",
		TokenType:   TokenTypeFBUser,
		AccessToken: "token",
	}

	if token.TableName() != "facebook_tokens" {
		t.Fatalf("unexpected table")
	}

	if token.FBUserID != "user" {
		t.Fatalf("expected fb_user")
	}
}
