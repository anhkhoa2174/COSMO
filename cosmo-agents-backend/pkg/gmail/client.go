package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client wraps Gmail API operations.
type Client struct {
	service    *gmail.Service
	httpClient *http.Client
}

// Close releases idle transport resources.
func (c *Client) Close() {
	if c == nil || c.httpClient == nil {
		return
	}
	if transport, ok := c.httpClient.Transport.(*oauth2.Transport); ok {
		if base, ok := transport.Base.(*http.Transport); ok {
			base.CloseIdleConnections()
		}
	}
}

// WatchResult represents the response metadata after registering Gmail push notifications.
type WatchResult struct {
	HistoryID  string
	Expiration int64
}

// NewClient creates a new Gmail API client with OAuth2 token.
func NewClient(ctx context.Context, accessToken string) (*Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
	})
	httpClient := oauth2.NewClient(ctx, tokenSource)

	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	return &Client{service: service, httpClient: httpClient}, nil
}

// GetProfile retrieves the Gmail user's profile.
func (c *Client) GetProfile(ctx context.Context) (*Profile, error) {
	profile, err := c.service.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return &Profile{
		EmailAddress:  profile.EmailAddress,
		MessagesTotal: profile.MessagesTotal,
		ThreadsTotal:  profile.ThreadsTotal,
		HistoryID:     profile.HistoryId,
	}, nil
}

// SendMessage sends an email message.
func (c *Client) SendMessage(ctx context.Context, req *SendMessageRequest) (*Message, error) {
	// Construct RFC 2822 email
	email := c.constructEmail(req)

	// Encode to base64
	encodedEmail := base64.URLEncoding.EncodeToString([]byte(email))

	msg := &gmail.Message{
		Raw: encodedEmail,
	}

	sent, err := c.service.Users.Messages.Send("me", msg).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &Message{
		ID:       sent.Id,
		ThreadID: sent.ThreadId,
		LabelIDs: sent.LabelIds,
	}, nil
}

// GetMessage retrieves a specific message.
func (c *Client) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	msg, err := c.service.Users.Messages.
		Get("me", messageID).
		Format("full").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return c.convertMessage(msg), nil
}

// ListMessages lists messages matching the query.
func (c *Client) ListMessages(ctx context.Context, query string, maxResults int64) ([]*Message, error) {
	call := c.service.Users.Messages.List("me")

	if query != "" {
		call = call.Q(query)
	}

	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := make([]*Message, len(resp.Messages))
	for i, msg := range resp.Messages {
		messages[i] = &Message{
			ID:       msg.Id,
			ThreadID: msg.ThreadId,
		}
	}

	return messages, nil
}

// GetThread retrieves a specific thread.
func (c *Client) GetThread(ctx context.Context, threadID string) (*Thread, error) {
	thread, err := c.service.Users.Threads.Get("me", threadID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	messages := make([]*Message, len(thread.Messages))
	for i, msg := range thread.Messages {
		messages[i] = c.convertMessage(msg)
	}

	return &Thread{
		ID:       thread.Id,
		Snippet:  thread.Snippet,
		Messages: messages,
	}, nil
}

// ModifyLabels modifies labels on a message.
func (c *Client) ModifyLabels(ctx context.Context, messageID string, addLabels, removeLabels []string) error {
	req := &gmail.ModifyMessageRequest{
		AddLabelIds:    addLabels,
		RemoveLabelIds: removeLabels,
	}

	_, err := c.service.Users.Messages.Modify("me", messageID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to modify labels: %w", err)
	}

	return nil
}

// Watch registers this Gmail account for push notifications via Google Pub/Sub.
func (c *Client) Watch(ctx context.Context, topicName string, labelIDs []string) (*WatchResult, error) {
	if topicName == "" {
		return nil, fmt.Errorf("topic name is required")
	}

	req := &gmail.WatchRequest{
		TopicName: topicName,
	}
	if len(labelIDs) > 0 {
		req.LabelIds = labelIDs
		req.LabelFilterBehavior = "include"
	}

	resp, err := c.service.Users.Watch("me", req).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to start Gmail watch: %w", err)
	}

	return &WatchResult{
		HistoryID:  fmt.Sprintf("%d", resp.HistoryId),
		Expiration: resp.Expiration,
	}, nil
}

// StopWatch removes the push notification watch configuration for this Gmail account.
func (c *Client) StopWatch(ctx context.Context) error {
	if err := c.service.Users.Stop("me").Context(ctx).Do(); err != nil {
		return fmt.Errorf("failed to stop Gmail watch: %w", err)
	}
	return nil
}

// extractBody traverses message parts to find body of a given mime type.
func extractBody(part *gmail.MessagePart, mime string) string {
	if part == nil {
		return ""
	}

	if strings.EqualFold(part.MimeType, mime) && part.Body != nil && part.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err != nil {
			return ""
		}
		return string(decoded)
	}

	for _, p := range part.Parts {
		if body := extractBody(p, mime); body != "" {
			return body
		}
	}

	return ""
}

// constructEmail builds an RFC 2822 email string.
func (c *Client) constructEmail(req *SendMessageRequest) string {
	var email strings.Builder

	email.WriteString(fmt.Sprintf("From: %s\r\n", req.From))
	email.WriteString(fmt.Sprintf("To: %s\r\n", req.To))

	if req.Cc != "" {
		email.WriteString(fmt.Sprintf("Cc: %s\r\n", req.Cc))
	}

	if req.Bcc != "" {
		email.WriteString(fmt.Sprintf("Bcc: %s\r\n", req.Bcc))
	}

	email.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))

	if req.InReplyTo != "" {
		email.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", req.InReplyTo))
		email.WriteString(fmt.Sprintf("References: %s\r\n", req.InReplyTo))
	}

	email.WriteString("MIME-Version: 1.0\r\n")

	if req.IsHTML {
		email.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	} else {
		email.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	}

	email.WriteString("\r\n")
	if req.IsHTML {
		normalizedBody := strings.ReplaceAll(req.Body, "\r\n", "\n")
		normalizedBody = strings.ReplaceAll(normalizedBody, "\n", "<br/>")
		email.WriteString(normalizedBody)
	} else {
		normalizedBody := strings.ReplaceAll(req.Body, "\r\n", "\n")
		normalizedBody = strings.ReplaceAll(normalizedBody, "\n", "\r\n")
		email.WriteString(normalizedBody)
	}

	return email.String()
}

// convertMessage converts Gmail API message to our Message type.
func (c *Client) convertMessage(msg *gmail.Message) *Message {
	message := &Message{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		LabelIDs: msg.LabelIds,
		Snippet:  msg.Snippet,
	}

	if msg.Payload != nil {
		message.Headers = make(map[string]string)
		for _, header := range msg.Payload.Headers {
			message.Headers[header.Name] = header.Value
		}

		// Prefer text/plain body, fallback to HTML if needed.
		if body := extractBody(msg.Payload, "text/plain"); body != "" {
			message.Body = body
		} else if body := extractBody(msg.Payload, "text/html"); body != "" {
			message.Body = body
		}
	}

	return message
}

// Profile represents Gmail user profile.
type Profile struct {
	EmailAddress  string
	MessagesTotal int64
	ThreadsTotal  int64
	HistoryID     uint64
}

// Message represents an email message.
type Message struct {
	ID       string
	ThreadID string
	LabelIDs []string
	Snippet  string
	Headers  map[string]string
	Body     string
}

// Thread represents an email thread.
type Thread struct {
	ID       string
	Snippet  string
	Messages []*Message
}

// SendMessageRequest represents a request to send an email.
type SendMessageRequest struct {
	From      string
	To        string
	Cc        string
	Bcc       string
	Subject   string
	Body      string
	IsHTML    bool
	InReplyTo string // For threading
}

// HistoryMessage is a lightweight representation of a message change from Gmail history.
type HistoryMessage struct {
	ID       string
	ThreadID string
}

// HistoryResult captures a page of Gmail history changes.
type HistoryResult struct {
	Messages      []HistoryMessage
	NextPageToken string
	NewestHistory string
}

// ListHistory retrieves history changes starting from the provided history ID.
func (c *Client) ListHistory(ctx context.Context, startHistoryID string, pageToken string) (*HistoryResult, error) {
	if startHistoryID == "" {
		return nil, fmt.Errorf("startHistoryID is required")
	}

	startID, err := strconv.ParseUint(startHistoryID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid startHistoryID: %w", err)
	}

	call := c.service.Users.History.List("me").
		HistoryTypes("messageAdded").
		StartHistoryId(startID)
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list history: %w", err)
	}

	var messages []HistoryMessage
	var newest string

	for i := range resp.History {
		h := resp.History[i]
		if h.Id > 0 {
			newest = fmt.Sprintf("%d", h.Id)
		}
		for _, m := range h.MessagesAdded {
			if m.Message == nil {
				continue
			}
			messages = append(messages, HistoryMessage{
				ID:       m.Message.Id,
				ThreadID: m.Message.ThreadId,
			})
		}
	}

	result := &HistoryResult{
		Messages:      messages,
		NextPageToken: resp.NextPageToken,
		NewestHistory: newest,
	}
	return result, nil
}
