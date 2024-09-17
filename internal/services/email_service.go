package services

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"

	"github.com/DIMO-Network/accounts-api/internal/config"
)

type EmailService interface {
	SendConfirmationEmail(ctx context.Context, emailTemplate *template.Template, userEmail, confCode string) error
}

type emailSvc struct {
	url           string
	emailFrom     string
	emailUsername string
	emailPassword string
	emailHost     string
	emailPort     string
	emailTemplate *template.Template
}

func NewEmailService(settings *config.Settings) EmailService {
	return &emailSvc{
		url:           settings.IdentityAPIURL,
		emailFrom:     settings.EmailFrom,
		emailUsername: settings.EmailUsername,
		emailPassword: settings.EmailPassword,
		emailHost:     settings.EmailHost,
		emailPort:     settings.EmailPort,
	}
}

func (e *emailSvc) SendConfirmationEmail(ctx context.Context, emailTemplate *template.Template, userEmail, confCode string) error {
	auth := smtp.PlainAuth("", e.emailUsername, e.emailPassword, e.emailHost)
	addr := fmt.Sprintf("%s:%s", e.emailHost, e.emailPort)

	var partsBuffer bytes.Buffer
	w := multipart.NewWriter(&partsBuffer)
	defer w.Close() //nolint

	p, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return err
	}

	pw := quotedprintable.NewWriter(p)
	if _, err := pw.Write([]byte("Hi,\r\n\r\nYour email verification code is: " + confCode + "\r\n")); err != nil {
		return err
	}
	pw.Close()

	h, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return err
	}

	hw := quotedprintable.NewWriter(h)
	if err := e.emailTemplate.Execute(hw, struct{ Key string }{confCode}); err != nil {
		return err
	}
	hw.Close()

	var buffer bytes.Buffer
	buffer.WriteString("From: DIMO <" + e.emailFrom + ">\r\n" +
		"To: " + userEmail + "\r\n" +
		"Subject: [DIMO] Verification Code\r\n" +
		"Content-Type: multipart/alternative; boundary=\"" + w.Boundary() + "\"\r\n" +
		"\r\n")
	if _, err := partsBuffer.WriteTo(&buffer); err != nil {
		return err
	}

	if err := smtp.SendMail(addr, auth, e.emailFrom, []string{userEmail}, buffer.Bytes()); err != nil {
		return err
	}

	return nil
}
