package mail

import (
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"path/filepath"

	"gopkg.in/gomail.v2"
)

// Attachment structures memory-buffered attachments for secure email delivery.
type Attachment struct {
	Name    string
	Content []byte
}

// Mailer houses SMTP server credentials.
type Mailer struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	FromName string
}

// NewMailer instantiates a Mailer configuration.
func NewMailer(host string, port int, user, password, from, fromName string) *Mailer {
	return &Mailer{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		From:     from,
		FromName: fromName,
	}
}

// Send sends an email with HTML body and optional byte-array attachments.
func (m *Mailer) Send(to string, subject string, bodyHTML string, attachments ...Attachment) error {
	msg := gomail.NewMessage()

	// Formulate clean Sender Address
	if m.FromName != "" {
		msg.SetAddressHeader("From", m.From, m.FromName)
	} else {
		msg.SetHeader("From", m.From)
	}

	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", bodyHTML)

	// Inject attachments directly from buffer to prevent disk usage
	for _, att := range attachments {
		fileName := att.Name
		content := att.Content

		msg.Attach(fileName, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(content)
			return err
		}), gomail.SetHeader(map[string][]string{
			"Content-Type": {mime.TypeByExtension(filepath.Ext(fileName))},
		}))
	}

	d := gomail.NewDialer(m.Host, m.Port, m.User, m.Password)
	
	// Support self-hosted environments which may use self-signed certificates
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(msg); err != nil {
		return fmt.Errorf("smtp connection failed: %w", err)
	}

	return nil
}
