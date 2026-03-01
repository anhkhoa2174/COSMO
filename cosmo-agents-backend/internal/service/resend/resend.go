package resend

import (
	"fmt"
	"log"
	"os"

	"github.com/resend/resend-go/v3"
)

type EmailContent struct {
	PlainText string
	HTML      string
}

type ResendService struct {
	client *resend.Client
}

func NewResendService() *ResendService {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Fatal("RESEND_API_KEY not set in environment variables")
	}
	client := resend.NewClient(apiKey)
	return &ResendService{client: client}
}

func (r *ResendService) SendEmail(
	sender string,
	to []string,
	subject string,
	messageText EmailContent,
	cc []string,
	bcc []string,
) error {
	htmlBody := messageText.HTML
	if htmlBody == "" {
		htmlBody = messageText.PlainText
	}

	params := &resend.SendEmailRequest{
		From:    sender,
		To:      to,
		Cc:      cc,
		Bcc:     bcc,
		Subject: subject,
		Html:    htmlBody,
	}

	resp, err := r.client.Emails.Send(params)
	if err != nil {
		log.Printf("Error sending email: %v", err)
		return fmt.Errorf("error sending email: %w", err)
	}

	log.Printf("Email sent successfully. ID: %s", resp.Id)
	return nil
}
