package outlook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	graphAPIBaseURL = "https://graph.microsoft.com/v1.0"
)

// Client is the Outlook API client
type Client struct {
	accessToken string
	httpClient  *http.Client
}

// Message represents an Outlook email message
type Message struct {
	ID                string      `json:"id,omitempty"`
	Subject           string      `json:"subject"`
	Body              MessageBody `json:"body"`
	ToRecipients      []Recipient `json:"toRecipients"`
	CcRecipients      []Recipient `json:"ccRecipients,omitempty"`
	BccRecipients     []Recipient `json:"bccRecipients,omitempty"`
	From              *Recipient  `json:"from,omitempty"`
	ReplyTo           []Recipient `json:"replyTo,omitempty"`
	Sender            *Recipient  `json:"sender,omitempty"`
	ReceivedDateTime  string      `json:"receivedDateTime,omitempty"`
	SentDateTime      string      `json:"sentDateTime,omitempty"`
	HasAttachments    bool        `json:"hasAttachments,omitempty"`
	InternetMessageID string      `json:"internetMessageId,omitempty"`
	ConversationID    string      `json:"conversationId,omitempty"`
	IsRead            bool        `json:"isRead,omitempty"`
	IsDraft           bool        `json:"isDraft,omitempty"`
}

// MessageBody represents the body of an email
type MessageBody struct {
	ContentType string `json:"contentType"` // "text" or "html"
	Content     string `json:"content"`
}

// Recipient represents an email recipient
type Recipient struct {
	EmailAddress EmailAddress `json:"emailAddress"`
}

// EmailAddress represents an email address
type EmailAddress struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

// Contact represents an Outlook contact
type Contact struct {
	ID            string `json:"id,omitempty"`
	DisplayName   string `json:"displayName"`
	GivenName     string `json:"givenName,omitempty"`
	Surname       string `json:"surname,omitempty"`
	EmailAddress  string `json:"emailAddress,omitempty"`
	CompanyName   string `json:"companyName,omitempty"`
	JobTitle      string `json:"jobTitle,omitempty"`
	MobilePhone   string `json:"mobilePhone,omitempty"`
	BusinessPhone string `json:"businessPhone,omitempty"`
}

// MessagesResponse represents a list of messages
type MessagesResponse struct {
	Value    []Message `json:"value"`
	NextLink string    `json:"@odata.nextLink,omitempty"`
}

// ContactsResponse represents a list of contacts
type ContactsResponse struct {
	Value    []Contact `json:"value"`
	NextLink string    `json:"@odata.nextLink,omitempty"`
}

// SendMessageRequest represents a send mail request
type SendMessageRequest struct {
	Message         Message `json:"message"`
	SaveToSentItems bool    `json:"saveToSentItems"`
}

// ReplyRequest represents a reply request
type ReplyRequest struct {
	Comment string  `json:"comment,omitempty"`
	Message Message `json:"message,omitempty"`
}

// ForwardRequest represents a forward request
type ForwardRequest struct {
	Comment      string      `json:"comment,omitempty"`
	ToRecipients []Recipient `json:"toRecipients"`
}

// NewClient creates a new Outlook API client
func NewClient(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	url := graphAPIBaseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetMessages retrieves messages from the user's mailbox
func (c *Client) GetMessages(ctx context.Context) (*MessagesResponse, error) {
	respBody, err := c.doRequest(ctx, "GET", "/me/messages", nil)
	if err != nil {
		return nil, err
	}

	var messages MessagesResponse
	if err := json.Unmarshal(respBody, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	return &messages, nil
}

// GetMessage retrieves a specific message by ID
func (c *Client) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	respBody, err := c.doRequest(ctx, "GET", fmt.Sprintf("/me/messages/%s", messageID), nil)
	if err != nil {
		return nil, err
	}

	var message Message
	if err := json.Unmarshal(respBody, &message); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	return &message, nil
}

// SendMessage sends an email message
func (c *Client) SendMessage(ctx context.Context, message Message) error {
	req := SendMessageRequest{
		Message:         message,
		SaveToSentItems: true,
	}

	_, err := c.doRequest(ctx, "POST", "/me/sendMail", req)
	return err
}

// ReplyToMessage replies to a message
func (c *Client) ReplyToMessage(ctx context.Context, messageID string, comment string) error {
	req := ReplyRequest{
		Comment: comment,
	}

	_, err := c.doRequest(ctx, "POST", fmt.Sprintf("/me/messages/%s/reply", messageID), req)
	return err
}

// ReplyAllToMessage replies to all recipients of a message
func (c *Client) ReplyAllToMessage(ctx context.Context, messageID string, comment string) error {
	req := ReplyRequest{
		Comment: comment,
	}

	_, err := c.doRequest(ctx, "POST", fmt.Sprintf("/me/messages/%s/replyAll", messageID), req)
	return err
}

// ForwardMessage forwards a message
func (c *Client) ForwardMessage(ctx context.Context, messageID string, toRecipients []Recipient, comment string) error {
	req := ForwardRequest{
		Comment:      comment,
		ToRecipients: toRecipients,
	}

	_, err := c.doRequest(ctx, "POST", fmt.Sprintf("/me/messages/%s/forward", messageID), req)
	return err
}

// GetContacts retrieves contacts from the user's contact list
func (c *Client) GetContacts(ctx context.Context) (*ContactsResponse, error) {
	respBody, err := c.doRequest(ctx, "GET", "/me/contacts", nil)
	if err != nil {
		return nil, err
	}

	var contacts ContactsResponse
	if err := json.Unmarshal(respBody, &contacts); err != nil {
		return nil, fmt.Errorf("failed to parse contacts: %w", err)
	}

	return &contacts, nil
}

// CreateContact creates a new contact
func (c *Client) CreateContact(ctx context.Context, contact Contact) (*Contact, error) {
	respBody, err := c.doRequest(ctx, "POST", "/me/contacts", contact)
	if err != nil {
		return nil, err
	}

	var createdContact Contact
	if err := json.Unmarshal(respBody, &createdContact); err != nil {
		return nil, fmt.Errorf("failed to parse created contact: %w", err)
	}

	return &createdContact, nil
}

// DeleteMessage deletes a message
func (c *Client) DeleteMessage(ctx context.Context, messageID string) error {
	_, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/me/messages/%s", messageID), nil)
	return err
}

// MarkAsRead marks a message as read
func (c *Client) MarkAsRead(ctx context.Context, messageID string) error {
	update := map[string]bool{"isRead": true}
	_, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/me/messages/%s", messageID), update)
	return err
}

// MarkAsUnread marks a message as unread
func (c *Client) MarkAsUnread(ctx context.Context, messageID string) error {
	update := map[string]bool{"isRead": false}
	_, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/me/messages/%s", messageID), update)
	return err
}
