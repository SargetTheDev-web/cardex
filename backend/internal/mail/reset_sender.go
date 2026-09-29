// internal/mail/reset_sender.go

package mail

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendResetLink(
	to string,
	link string,
) error {

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	login := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	from := os.Getenv("SMTP_FROM_EMAIL")
	fromName := os.Getenv("SMTP_FROM_NAME")

	message := []byte(
		fmt.Sprintf(
			"From: %s <%s>\r\n"+
				"To: %s\r\n"+
				"Subject: CARDex Password Reset\r\n"+
				"MIME-Version: 1.0\r\n"+
				"Content-Type: text/plain; charset=UTF-8\r\n"+
				"\r\n"+
				"Click here to reset your password:\r\n%s",
			fromName,
			from,
			to,
			link,
		),
	)

	auth := smtp.PlainAuth(
		"",
		login,
		password,
		smtpHost,
	)

	return smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		from,
		[]string{to},
		message,
	)
}
