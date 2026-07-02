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

	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	message := []byte(
		fmt.Sprintf(
			"Subject: CARDex Verification Code\r\n\r\nYour verification code is: %s",
			code,
		),
	)

	auth := smtp.PlainAuth(
		"",
		from,
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
