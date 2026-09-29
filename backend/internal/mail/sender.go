// internal/mail/sender.go

package mail

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendVerificationCode(
	to string,
	code string,
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
				"Subject: CARDex Verification Code\r\n"+
				"MIME-Version: 1.0\r\n"+
				"Content-Type: text/plain; charset=UTF-8\r\n"+
				"\r\n"+
				"Your verification code is: %s",
			fromName,
			from,
			to,
			code,
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
