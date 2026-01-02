package mailer

import (
	"crypto/tls"
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

		var err error
		if m.Port == 465 {
			err = m.sendMailTLS(addr, auth, recipient, msg)
		} else {
			err = smtp.SendMail(addr, auth, m.Sender, []string{recipient}, msg)
		}

		if err != nil {
			fmt.Printf("Failed to send email to %s: %v\n", recipient, err)
		}
	}

	return nil
}

func (m *SMTPMailer) sendMailTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         m.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		return err
	}
	// We handle Close() via defer conn.Close(), forcing Quit might be redundant or fail if connection closed,
	// but standard practice is to try Quit.
	defer client.Quit()

	if err = client.Auth(auth); err != nil {
		return err
	}
	if err = client.Mail(m.Sender); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return nil
}