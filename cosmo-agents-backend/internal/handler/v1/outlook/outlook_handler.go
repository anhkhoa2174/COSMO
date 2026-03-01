package outlook

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	outlookService "github.com/rockship/cosmo-agents-go/internal/service/outlook"
)

// OutlookHandler manages OAuth and Graph API proxy endpoints for Outlook.
type OutlookHandler struct {
	service *outlookService.OutlookService
}

// NewOutlookHandler constructs a handler instance.
func NewOutlookHandler(service *outlookService.OutlookService) *OutlookHandler {
	return &OutlookHandler{service: service}
}

// Authorize handles GET /v1/outlook/authorize.
func (h *OutlookHandler) Authorize(c fiber.Ctx) error {
	if h.service == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable, "Outlook integration not configured", ""))
	}
	url, err := h.service.AuthorizationURL(c.Query("redirect_url", ""))
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to build authorization URL", err.Error(),
		))
	}
	return c.JSON(schema.SuccessResponse(v1schema.OutlookAuthResponse{URL: url}))
}

// Callback handles GET /v1/outlook/oauth2callback.
func (h *OutlookHandler) Callback(c fiber.Ctx) error {
	if h.service == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable, "Outlook integration not configured", ""))
	}
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Missing authorization code", "",
		))
	}

	redirectURI := c.Query("redirect_uri", "")
	tokens, err := h.service.ExchangeCode(c.Context(), code, redirectURI)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to exchange authorization code", err.Error(),
		))
	}

	resp := v1schema.OutlookTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
	}
	return c.JSON(schema.SuccessResponse(resp))
}

// RefreshToken handles GET /v1/outlook/refresh_token.
func (h *OutlookHandler) RefreshToken(c fiber.Ctx) error {
	if h.service == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable, "Outlook integration not configured", ""))
	}
	refreshToken := strings.TrimSpace(c.Query("refresh_token"))
	if refreshToken == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "refresh_token is required", "",
		))
	}

	tokens, err := h.service.RefreshToken(c.Context(), refreshToken)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to refresh token", err.Error(),
		))
	}

	resp := v1schema.OutlookTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
	}
	return c.JSON(schema.SuccessResponse(resp))
}

// GetContacts handles GET /v1/outlook/contacts.
func (h *OutlookHandler) GetContacts(c fiber.Ctx) error {
	if h.service == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable, "Outlook integration not configured", ""))
	}
	accessToken := strings.TrimSpace(c.Query("access_token"))
	if accessToken == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "access_token is required", "",
		))
	}

	data, err := h.service.GetContacts(c.Context(), accessToken)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to fetch contacts", err.Error(),
		))
	}
	return c.JSON(schema.SuccessResponse(data))
}

// GetEmails handles GET /v1/outlook/emails.
func (h *OutlookHandler) GetEmails(c fiber.Ctx) error {
	if h.service == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable, "Outlook integration not configured", ""))
	}
	accessToken := strings.TrimSpace(c.Query("access_token"))
	if accessToken == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "access_token is required", "",
		))
	}

	data, err := h.service.GetMessages(c.Context(), accessToken)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to fetch emails", err.Error(),
		))
	}
	return c.JSON(schema.SuccessResponse(data))
}

// SendMail handles POST /v1/outlook/send_email.
func (h *OutlookHandler) SendMail(c fiber.Ctx) error {
	if h.service == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable, "Outlook integration not configured", ""))
	}
	payload, accessToken, err := buildOutlookSendPayload(c, true)
	if err != nil {
		return err
	}

	resp, err := h.service.SendMail(c.Context(), accessToken, payload)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to send email", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(resp))
}

// ReplyMail handles POST /v1/outlook/reply_email.
func (h *OutlookHandler) ReplyMail(c fiber.Ctx) error {
	if h.service == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable, "Outlook integration not configured", ""))
	}
	payload, accessToken, err := buildOutlookSendPayload(c, false)
	if err != nil {
		return err
	}

	messageID := strings.TrimSpace(c.FormValue("message_id"))
	if messageID == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "message_id is required", "",
		))
	}

	resp, err := h.service.ReplyMail(c.Context(), accessToken, messageID, payload)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to reply email", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(resp))
}

// ForwardMail handles POST /v1/outlook/forward_email.
func (h *OutlookHandler) ForwardMail(c fiber.Ctx) error {
	if h.service == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable, "Outlook integration not configured", ""))
	}
	payload, accessToken, err := buildOutlookSendPayload(c, false)
	if err != nil {
		return err
	}

	messageID := strings.TrimSpace(c.FormValue("message_id"))
	if messageID == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "message_id is required", "",
		))
	}

	resp, err := h.service.ForwardMail(c.Context(), accessToken, messageID, payload)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to forward email", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(resp))
}

func buildOutlookSendPayload(c fiber.Ctx, requireBody bool) (map[string]interface{}, string, error) {
	accessToken := strings.TrimSpace(c.Query("access_token"))
	if accessToken == "" {
		return nil, "", c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "access_token is required", "",
		))
	}

	subject := c.FormValue("subject")
	content := c.FormValue("content")
	contentType := c.FormValue("content_type")
	if requireBody && (subject == "" || content == "" || contentType == "") {
		return nil, "", c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "subject, content, and content_type are required", "",
		))
	}

	sendMode := strings.ToLower(strings.TrimSpace(c.FormValue("send_mode")))
	if sendMode == "" {
		sendMode = "to"
	}

	form, err := c.MultipartForm()
	if err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return nil, "", c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Invalid multipart form", err.Error(),
		))
	}

	recipients := extractRecipients(form, c.FormValue("recipients"))
	if len(recipients) == 0 {
		return nil, "", c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "recipients are required", "",
		))
	}

	message := map[string]interface{}{}
	if requireBody {
		message["subject"] = subject
		message["body"] = map[string]string{
			"contentType": contentType,
			"content":     content,
		}
	}

	assignRecipients(message, sendMode, recipients)

	attachments, err := extractAttachments(form)
	if err != nil {
		return nil, "", c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Failed to read attachments", err.Error(),
		))
	}
	if len(attachments) > 0 {
		message["attachments"] = attachments
	}

	payload := map[string]interface{}{
		"message":         message,
		"saveToSentItems": "true",
	}

	if !requireBody {
		payload["comment"] = c.FormValue("comment")
	}

	return payload, accessToken, nil
}

func extractRecipients(form *multipart.Form, fallback string) []string {
	var recipients []string
	if form != nil {
		if values, ok := form.Value["recipients"]; ok {
			recipients = append(recipients, values...)
		}
	}

	if len(recipients) == 0 && fallback != "" {
		for _, part := range strings.Split(fallback, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				recipients = append(recipients, trimmed)
			}
		}
	}
	return recipients
}

func assignRecipients(message map[string]interface{}, mode string, recipients []string) {
	items := make([]map[string]map[string]string, len(recipients))
	for i, email := range recipients {
		items[i] = map[string]map[string]string{
			"emailAddress": {
				"address": email,
			},
		}
	}

	switch mode {
	case "cc":
		message["ccRecipients"] = items
	case "bcc":
		message["bccRecipients"] = items
	default:
		message["toRecipients"] = items
	}
}

func extractAttachments(form *multipart.Form) ([]map[string]interface{}, error) {
	if form == nil {
		return nil, nil
	}
	files := form.File["attachments"]
	if len(files) == 0 {
		return nil, nil
	}

	attachments := make([]map[string]interface{}, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, outlookService.BuildAttachment(header.Filename, header.Header.Get("Content-Type"), data))
	}
	return attachments, nil
}
