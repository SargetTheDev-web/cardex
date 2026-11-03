// internal/mail/notification_sender.go

package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type NotificationEmailData struct {
	RecipientName string
	Subject       string
	Heading       string
	Message       string
	Details       []NotificationEmailDetail
	CurrentYear   int
}

type NotificationEmailDetail struct {
	Label string
	Value string
}

func SendNotificationEmail(to string, data NotificationEmailData) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	login := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM_EMAIL")
	fromName := os.Getenv("SMTP_FROM_NAME")

	if smtpHost == "" || smtpPort == "" || login == "" ||
		password == "" || from == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	if strings.ContainsAny(data.Subject, "\r\n") {
		return fmt.Errorf("invalid email subject")
	}
	if strings.ContainsAny(fromName, "\r\n") {
		return fmt.Errorf("invalid sender name")
	}
	if strings.TrimSpace(data.Subject) == "" ||
		strings.TrimSpace(data.Heading) == "" ||
		strings.TrimSpace(data.Message) == "" {
		return fmt.Errorf("notification subject, heading, and message are required")
	}

	recipient, err := mail.ParseAddress(to)
	if err != nil || recipient.Address != to {
		return fmt.Errorf("invalid recipient email address")
	}

	sender, err := mail.ParseAddress(from)
	if err != nil || sender.Address != from {
		return fmt.Errorf("invalid sender email address")
	}

	data.CurrentYear = time.Now().Year()

	const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Heading}} - CARDex Smart Library</title>
</head>
<body style="
    margin: 0;
    padding: 0;
    background-color: #f4f6f9;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    -webkit-font-smoothing: antialiased;
">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0"
       style="background-color: #f4f6f9; padding: 40px 0;">
    <tr>
        <td align="center">
            <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0"
                   style="max-width: 560px; background-color: #ffffff; border-radius: 10px; overflow: hidden;">

                <tr>
                    <td style="background-color: #5750F1; padding: 32px 30px; text-align: center;">
                        <h1 style="color: #ffffff; margin: 0; font-size: 23px; font-weight: 700;">
                            CARDex Smart Library
                        </h1>
                        <p style="color: #e4e3ff; margin: 7px 0 0 0; font-size: 13px;">
                            Centralized Academic Resource Database &amp; Exchange
                        </p>
                    </td>
                </tr>

                <tr>
                    <td style="padding: 40px 32px; color: #333333; font-size: 15px; line-height: 1.6;">
                        <h2 style="color: #1c2434; font-size: 20px; margin: 0 0 18px 0;">
                            {{.Heading}}
                        </h2>

                        {{if .RecipientName}}
                        <p style="margin: 0 0 18px 0;">Hello {{.RecipientName}},</p>
                        {{end}}

                        <p style="margin: 0 0 22px 0; white-space: pre-line;">{{.Message}}</p>

                        {{if .Details}}
                        <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0"
                               style="background-color: #f8f9ff; border: 1px solid #dedcff; border-radius: 8px;">
                            <tr>
                                <td style="padding: 22px;">
                                    {{range .Details}}
                                    <p style="margin: 0 0 5px 0; color: #5750F1; font-size: 12px;
                                              font-weight: 700; text-transform: uppercase; letter-spacing: 0.6px;">
                                        {{.Label}}
                                    </p>
                                    <p style="margin: 0 0 18px 0; color: #1c2434; font-size: 15px;">
                                        {{.Value}}
                                    </p>
                                    {{end}}
                                </td>
                            </tr>
                        </table>
                        {{end}}

                        <p style="margin: 24px 0 0 0; font-size: 14px; color: #64748b;">
                            This is an automated message from CARDex Smart Library.
                        </p>
                    </td>
                </tr>

                <tr>
                    <td style="background-color: #f8fafc; padding: 20px 30px; text-align: center;
                               border-top: 1px solid #e2e8f0;">
                        <p style="font-size: 12px; color: #94a3b8; margin: 0;">
                            &copy; {{.CurrentYear}} CARDex Smart Library. All rights reserved.
                        </p>
                    </td>
                </tr>

            </table>
        </td>
    </tr>
</table>
</body>
</html>
`

	tmpl, err := template.New("notification-email").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parse notification email template: %w", err)
	}

	var htmlBody bytes.Buffer
	if err := tmpl.Execute(&htmlBody, data); err != nil {
		return fmt.Errorf("render notification email template: %w", err)
	}

	plainText := data.Heading + "\r\n\r\n"
	if data.RecipientName != "" {
		plainText += "Hello " + data.RecipientName + ",\r\n\r\n"
	}
	plainText += data.Message + "\r\n\r\n"

	for _, detail := range data.Details {
		plainText += detail.Label + ": " + detail.Value + "\r\n"
	}

	fromHeader := (&mail.Address{Name: fromName, Address: from}).String()
	boundary := fmt.Sprintf("CARDexNotification_%d", time.Now().UnixNano())

	message := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: multipart/alternative; boundary=%q\r\n\r\n"+
			"--%s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 8bit\r\n\r\n"+
			"%s\r\n"+
			"--%s\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 8bit\r\n\r\n"+
			"%s\r\n"+
			"--%s--\r\n",
		fromHeader,
		recipient.String(),
		data.Subject,
		boundary,
		boundary,
		plainText,
		boundary,
		htmlBody.String(),
		boundary,
	)

	auth := smtp.PlainAuth("", login, password, smtpHost)

	return smtp.SendMail(
		net.JoinHostPort(smtpHost, smtpPort),
		auth,
		from,
		[]string{recipient.Address},
		[]byte(message),
	)
}
