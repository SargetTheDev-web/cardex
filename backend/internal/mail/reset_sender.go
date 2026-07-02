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

	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	message := []byte(
		fmt.Sprintf(
			"Subject: CARDex Password Reset\r\n\r\nClick here to reset your password:\n%s",
			link,
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
