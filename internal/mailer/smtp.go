package mailer

import (
	"fmt"
	"net/smtp"
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

	auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)

	// Headers
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	for _, recipient := range to {
		// Use a display name + the sender email address
		fromHeader := fmt.Sprintf("System Design Daily <%s>", m.Sender)

		msg := []byte(fmt.Sprintf("To: %s\r\n"+
			"From: %s\r\n"+
			"Subject: %s\r\n"+
			"%s"+
			"%s", recipient, fromHeader, subject, mime, bodyHTML))

		err := smtp.SendMail(addr, auth, m.Sender, []string{recipient}, msg)
		if err != nil {
			fmt.Printf("Failed to send email to %s: %v\n", recipient, err)
		}
	}

	return nil
}
