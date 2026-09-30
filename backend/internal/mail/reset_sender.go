// internal/mail/reset_sender.go

package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"time"
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

	if smtpHost == "" ||
		smtpPort == "" ||
		login == "" ||
		password == "" ||
		from == "" {

		return fmt.Errorf(
			"SMTP configuration is incomplete",
		)
	}

	/*
		HTML email template.
	*/
	const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">

    <meta
        name="viewport"
        content="width=device-width, initial-scale=1.0"
    >

    <title>Reset Your Password - CARDex Smart Library</title>
</head>

<body style="
    margin: 0;
    padding: 0;
    background-color: #f4f6f9;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    -webkit-font-smoothing: antialiased;
">

<table
    role="presentation"
    width="100%"
    cellspacing="0"
    cellpadding="0"
    border="0"
    style="background-color: #f4f6f9; padding: 40px 0;"
>
    <tr>
        <td align="center">

            <table
                role="presentation"
                width="100%"
                style="
                    max-width: 560px;
                    background-color: #ffffff;
                    border-radius: 8px;
                    overflow: hidden;
                    box-shadow: 0 4px 12px rgba(0,0,0,0.05);
                "
                cellspacing="0"
                cellpadding="0"
                border="0"
            >

                <!-- Header -->

                <tr>
                    <td
                        style="
                            background-color: #5750F1;
                            padding: 30px;
                            text-align: center;
                        "
                    >

                        <h1
                            style="
                                color: #ffffff;
                                margin: 0;
                                font-size: 22px;
                                font-weight: 700;
                                letter-spacing: 0.5px;
                            "
                        >
                            CARDex Smart Library
                        </h1>

                        <p
                            style="
                                color: #e0e0ff;
                                margin: 5px 0 0 0;
                                font-size: 13px;
                            "
                        >
                            Centralized Academic Resource Database & Exchange
                        </p>

                    </td>
                </tr>

                <!-- Body -->

                <tr>
                    <td
                        style="
                            padding: 40px 30px;
                            color: #333333;
                            font-size: 15px;
                            line-height: 1.6;
                        "
                    >

                        <h2
                            style="
                                color: #1c2434;
                                font-size: 18px;
                                margin-top: 0;
                                margin-bottom: 16px;
                            "
                        >
                            Password Reset Request
                        </h2>

                        <p style="margin-bottom: 20px;">
                            We received a request to reset the password
                            associated with your account.
                        </p>

                        <p>
                            Click the button below to set a new password:
                        </p>

                        <table
                            role="presentation"
                            cellspacing="0"
                            cellpadding="0"
                            border="0"
                            style="margin: 30px auto;"
                        >

                            <tr>

                                <td
                                    align="center"
                                    style="
                                        border-radius: 6px;
                                        background-color: #5750F1;
                                    "
                                >

                                    <a
                                        href="{{.ResetURL}}"
                                        target="_blank"
                                        style="
                                            display: inline-block;
                                            padding: 14px 28px;
                                            font-size: 15px;
                                            color: #ffffff;
                                            font-weight: 600;
                                            text-decoration: none;
                                            border-radius: 6px;
                                        "
                                    >
                                        Reset Password
                                    </a>

                                </td>

                            </tr>

                        </table>

                        <p
                            style="
                                margin-bottom: 20px;
                                font-size: 14px;
                                color: #64748b;
                            "
                        >
                            This link is valid for
                            <strong>15 minutes</strong>.
                        </p>

                        <p
                            style="
                                font-size: 14px;
                                color: #64748b;
                            "
                        >
                            If you did not request a password reset,
                            you can safely ignore this email.
                        </p>

                        <hr
                            style="
                                border: none;
                                border-top: 1px solid #e2e8f0;
                                margin: 30px 0;
                            "
                        />

                        <p
                            style="
                                font-size: 12px;
                                color: #94a3b8;
                                word-break: break-all;
                                margin: 0;
                            "
                        >
                            If the button above does not work,
                            copy and paste this URL into your browser:
                            <br><br>

                            <a
                                href="{{.ResetURL}}"
                                style="
                                    color: #5750F1;
                                    text-decoration: underline;
                                "
                            >
                                {{.ResetURL}}
                            </a>

                        </p>

                    </td>
                </tr>

                <!-- Footer -->

                <tr>

                    <td
                        style="
                            background-color: #f8fafc;
                            padding: 20px 30px;
                            text-align: center;
                            border-top: 1px solid #e2e8f0;
                        "
                    >

                        <p
                            style="
                                font-size: 12px;
                                color: #94a3b8;
                                margin: 0;
                            "
                        >
                            &copy; {{.CurrentYear}}
                            CARDex Smart Library.
                            All rights reserved.
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

	tmpl, err := template.New(
		"password-reset",
	).Parse(htmlTemplate)

	if err != nil {
		return err
	}

	var htmlBody bytes.Buffer

	err = tmpl.Execute(
		&htmlBody,
		struct {
			ResetURL    string
			CurrentYear int
		}{
			ResetURL:    link,
			CurrentYear: time.Now().Year(),
		},
	)

	if err != nil {
		return err
	}

	/*
		Plain-text fallback.
	*/
	plainText := fmt.Sprintf(
		"Reset your CARDex password using this link:\r\n\r\n%s\r\n\r\n"+
			"This link expires in 15 minutes.",
		link,
	)

	/*
		Multipart/alternative email.

		Mail clients can choose HTML or plain text.
	*/
	boundary := "CARDexResetBoundary"

	message := fmt.Sprintf(
		"From: %s <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: CARDex Password Reset\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: multipart/alternative; boundary=\"%s\"\r\n"+
			"\r\n"+

			"--%s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 8bit\r\n"+
			"\r\n"+
			"%s\r\n"+

			"--%s\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 8bit\r\n"+
			"\r\n"+
			"%s\r\n"+

			"--%s--\r\n",

		fromName,
		from,
		to,

		boundary,

		boundary,
		plainText,

		boundary,
		htmlBody.String(),

		boundary,
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
		[]byte(message),
	)
}
