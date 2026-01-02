package mailer

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GmailMailer struct {
	Service   *gmail.Service
	Sender    string
	PublicURL string
}

func NewGmailMailer(ctx context.Context, senderEmail, publicURL string, credentialsJSON []byte) (*GmailMailer, error) {
	// 1. Parse the credentials (looks for "web" or "installed" app in JSON)
	config, err := google.ConfigFromJSON(credentialsJSON, gmail.GmailSendScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	// 2. Retrieve the token from environment variable or file
	// For server-side (Render), passing the token JSON via ENV is safest/easiest
	// so we don't have to do an interactive login flow on the server.
	client := getClient(ctx, config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	return &GmailMailer{
		Service:   srv,
		Sender:    senderEmail,
		PublicURL: publicURL,
	},
	nil
}

func (m *GmailMailer) Send(to []string, subject, bodyHTML string) error {
	if len(to) == 0 {
		return nil
	}

	for _, recipient := range to {
		var message gmail.Message

		unsubscribeHTML := fmt.Sprintf(
			`<br><br><hr><p style="font-size: 12px; color: #666; text-align: center;">
			<a href="%s/unsubscribe?email=%s">Unsubscribe</a> from these emails.</p>`,
			m.PublicURL, recipient,
		)

		fullBody := bodyHTML + unsubscribeHTML

		// Construct the email message (MIME)
		emailContent := fmt.Sprintf("From: %s\r\n" +
			"To: %s\r\n" +
			"Subject: %s\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			"%s", m.Sender, recipient, subject, fullBody)

		// Gmail API requires base64url encoding
		message.Raw = base64.URLEncoding.EncodeToString([]byte(emailContent))

		_, err := m.Service.Users.Messages.Send("me", &message).Do()
		if err != nil {
			log.Printf("Failed to send email to %s via API: %v", recipient, err)
		} else {
			log.Printf("Email sent successfully to %s via API", recipient)
		}
	}
	return nil
}
