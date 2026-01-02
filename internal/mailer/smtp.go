package mailer

import (
	"crypto/tls"
	"fmt"
	"github.com/go-mail/mail/v2"
)

type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
}

func NewSMTPMailer(host string, port int, user, pass, sender string) *SMTPMailer {
	return &SMTPMailer{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
		Sender:   sender,
	}
}

func (m *SMTPMailer) Send(to []string, subject, bodyHTML string) error {
	if len(to) == 0 {
		return nil
	}

	d := mail.NewDialer(m.Host, m.Port, m.Username, m.Password)
	
	// If Port 465, enforce SSL. If 587, StartTLS is automatic.
	// Render often requires this skip verify if certificates on shared IPs are weird, 
	// but usually Gmail is fine. We'll set it to be safe.
	d.TLSConfig = &tls.Config{InsecureSkipVerify: false} 

	// Send individually to hide recipients from each other
	for _, recipient := range to {
		msg := mail.NewMessage()
		// Use a display name + the sender email address
		msg.SetHeader("From", fmt.Sprintf("System Design Daily <%s>", m.Sender))
		msg.SetHeader("To", recipient)
		msg.SetHeader("Subject", subject)
		msg.SetBody("text/html", bodyHTML)

		if err := d.DialAndSend(msg); err != nil {
			fmt.Printf("Failed to send email to %s: %v\n", recipient, err)
			// Continue sending to others
		}
	}

	return nil
}
