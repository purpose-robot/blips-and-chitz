package mailer

import (
	"bytes"
	"context"
	"fmt"
	htmlTemplate "html/template"
	textTemplate "text/template"
	"time"

	"github.com/purpose-robot/blips-and-chitz/assets"
	"github.com/wneessen/go-mail"
)

type EmailGateway struct {
	from   string
	client *mail.Client
}

func NewEmailGateway(host string, port int, from, username, password string) (*EmailGateway, error) {
	client, err := mail.NewClient(
		host,
		mail.WithPort(port),
		mail.WithTimeout(10*time.Second),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
	)
	if err != nil {
		return nil, err
	}

	return &EmailGateway{from: from, client: client}, nil
}

func (m *EmailGateway) Send(ctx context.Context, recipient string, data any, patterns ...string) error {
	for i := range patterns {
		patterns[i] = "emails/" + patterns[i]
	}

	msg := mail.NewMsg()

	if err := msg.From(m.from); err != nil {
		return fmt.Errorf("ext.email.send: setting sender: %v", err)
	}

	if err := msg.To(recipient); err != nil {
		return fmt.Errorf("ext.email.send: setting recipient: %v", err)
	}

	t, err := textTemplate.New("").ParseFS(assets.EmailFS, patterns...)
	if err != nil {
		return fmt.Errorf("ext.email.send: parsing text templates %v: %v", patterns, err)
	}

	subject := new(bytes.Buffer)
	if err := t.ExecuteTemplate(subject, "subject", data); err != nil {
		return fmt.Errorf("ext.email.send: executing subject template: %v", err)
	}

	msg.Subject(subject.String())

	content := new(bytes.Buffer)
	if err := t.ExecuteTemplate(content, "content", data); err != nil {
		return fmt.Errorf("ext.email.send: executing content template: %v", err)
	}

	msg.SetBodyString(mail.TypeTextPlain, content.String())

	if t.Lookup("htmlContent") != nil {
		t, err := htmlTemplate.New("").ParseFS(assets.EmailFS, patterns...)
		if err != nil {
			return fmt.Errorf("ext.email.send: parsing html templates %v: %v", patterns, err)
		}

		htmlContent := new(bytes.Buffer)
		if err := t.ExecuteTemplate(htmlContent, "htmlContent", data); err != nil {
			return fmt.Errorf("ext.email.send: executing html content template: %v", err)
		}

		msg.AddAlternativeString(mail.TypeTextHTML, htmlContent.String())
	}

	err = m.client.DialAndSendWithContext(ctx, msg)
	if err != nil {
		return fmt.Errorf("ext.email.send: sending mail to recipient %s: %v", recipient, err)
	}

	return nil
}
