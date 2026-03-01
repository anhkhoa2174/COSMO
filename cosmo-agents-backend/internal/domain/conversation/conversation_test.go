package conversation

import (
	"testing"

	"github.com/google/uuid"
)

func TestConversationBeforeCreateDefaults(t *testing.T) {
	conversation := Conversation{}

	if err := conversation.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if conversation.ID == uuid.Nil {
		t.Fatal("expected conversation ID to be set")
	}
	if conversation.Status != ConversationStatusUnread {
		t.Fatalf("expected default status %q, got %q", ConversationStatusUnread, conversation.Status)
	}
	if conversation.Labels == nil || len(conversation.Labels) != 0 {
		t.Fatalf("expected labels to default to empty array, got %v", conversation.Labels)
	}
	if conversation.Intents == nil || len(conversation.Intents) != 0 {
		t.Fatalf("expected intents to default to empty array, got %v", conversation.Intents)
	}
	if string(conversation.CMetadata) != "{}" {
		t.Fatalf("expected cmetadata '{}', got %s", string(conversation.CMetadata))
	}
}

func TestConversationReadUnread(t *testing.T) {
	conversation := Conversation{}

	conversation.Read()
	if conversation.Status != ConversationStatusRead {
		t.Fatalf("expected status %q after Read, got %q", ConversationStatusRead, conversation.Status)
	}

	conversation.Unread()
	if conversation.Status != ConversationStatusUnread {
		t.Fatalf("expected status %q after Unread, got %q", ConversationStatusUnread, conversation.Status)
	}
}
