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

		return fmt.Errorf("SMTP configuration is incomplete")
	}

	/*
		HTML email template.
	*/
	const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <title>Reset Your Password - CARDex Smart Library</title>
</head>

<body style="
    margin: 0;
    padding: 0;
    background-color: #f4f6f9;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    -webkit-font-smoothing: antialiased;
">

<table role="presentation"
       width="100%"
       cellspacing="0"
       cellpadding="0"
       border="0"
       style="
           background-color: #f4f6f9;
           padding: 40px 0;
       ">

    <tr>
        <td align="center">

            <!-- Main Card -->
            <table role="presentation"
                   width="100%"
                   cellspacing="0"
                   cellpadding="0"
                   border="0"
                   style="
                       max-width: 560px;
                       background-color: #ffffff;
                       border-radius: 10px;
                       overflow: hidden;
                       box-shadow: 0 4px 16px rgba(0,0,0,0.06);
                   ">

                <!-- Header -->
                <tr>
                    <td style="
                        background-color: #5750F1;
                        padding: 32px 30px;
                        text-align: center;
                    ">

                        <h1 style="
                            color: #ffffff;
                            margin: 0;
                            font-size: 23px;
                            font-weight: 700;
                            letter-spacing: 0.5px;
                        ">
                            CARDex Smart Library
                        </h1>

                        <p style="
                            color: #e4e3ff;
                            margin: 7px 0 0 0;
                            font-size: 13px;
                        ">
                            Centralized Academic Resource Database &amp; Exchange
                        </p>

                    </td>
                </tr>

                <!-- Content -->
                <tr>
                    <td style="
                        padding: 40px 32px;
                        color: #333333;
                        font-size: 15px;
                        line-height: 1.6;
                    ">

                        <h2 style="
                            color: #1c2434;
                            font-size: 20px;
                            margin: 0 0 18px 0;
                        ">
                            Password Reset Request
                        </h2>

                        <p style="
                            margin: 0 0 18px 0;
                        ">
                            We received a request to reset the password
                            associated with your CARDex account.
                        </p>

                        <p style="
                            margin: 0 0 24px 0;
                        ">
                            Click the button below to continue and create
                            a new password.
                        </p>

                        <!-- Reset Password Button -->
                        <table role="presentation"
                               width="100%"
                               cellspacing="0"
                               cellpadding="0"
                               border="0"
                               style="
                                   margin: 0 0 24px 0;
                               ">

                            <tr>
                                <td align="center">

                                    <!--
                                        Normal HTML anchor styled as a button.
                                        This is intentionally NOT a Brevo button.
                                    -->
                                    <a href="{{.ResetURL}}"
                                       style="
                                           display: inline-block;
                                           background-color: #5750F1;
                                           color: #ffffff;
                                           text-decoration: none;
                                           padding: 13px 30px;
                                           border-radius: 7px;
                                           font-size: 14px;
                                           font-weight: 600;
                                           line-height: 1.4;
                                       ">
                                        Reset Password
                                    </a>

                                </td>
                            </tr>

                        </table>

                        <!-- Backup Link -->
                        <table role="presentation"
                               width="100%"
                               cellspacing="0"
                               cellpadding="0"
                               border="0"
                               style="
                                   background-color: #f8f9ff;
                                   border: 1px solid #dedcff;
                                   border-radius: 8px;
                               ">

                            <tr>
                                <td style="
                                    padding: 20px;
                                ">

                                    <p style="
                                        margin: 0 0 8px 0;
                                        color: #5750F1;
                                        font-size: 12px;
                                        font-weight: 700;
                                        text-transform: uppercase;
                                        letter-spacing: 0.6px;
                                    ">
                                        Alternative Reset Link
                                    </p>

                                    <p style="
                                        margin: 0 0 8px 0;
                                        color: #64748b;
                                        font-size: 12px;
                                        line-height: 1.5;
                                    ">
                                        If the button does not work,
                                        copy and paste this link into your browser:
                                    </p>

                                    <p style="
                                        margin: 0;
                                        color: #334155;
                                        font-size: 12px;
                                        line-height: 1.7;
                                        word-break: break-all;
                                    ">
                                        {{.ResetURL}}
                                    </p>

                                </td>
                            </tr>

                        </table>

                        <!-- Expiration Notice -->
                        <table role="presentation"
                               width="100%"
                               cellspacing="0"
                               cellpadding="0"
                               border="0"
                               style="
                                   margin-top: 20px;
                                   background-color: #fff8e7;
                                   border: 1px solid #f3dfaa;
                                   border-radius: 7px;
                               ">

                            <tr>
                                <td style="
                                    padding: 14px 16px;
                                    color: #795b16;
                                    font-size: 13px;
                                    line-height: 1.5;
                                ">

                                    <strong>
                                        This link expires in 15 minutes.
                                    </strong>

                                    <br>

                                    Please request a new password reset link
                                    if the current one expires.

                                </td>
                            </tr>

                        </table>

                        <p style="
                            margin: 24px 0 0 0;
                            font-size: 14px;
                            color: #64748b;
                        ">
                            If you did not request a password reset,
                            you can safely ignore this email.
                        </p>

                        <!-- Divider -->
                        <hr style="
                            border: none;
                            border-top: 1px solid #e2e8f0;
                            margin: 30px 0;
                        ">

                        <p style="
                            margin: 0;
                            font-size: 12px;
                            color: #94a3b8;
                            line-height: 1.5;
                        ">
                            For your security, never share this reset link
                            with anyone else.
                        </p>

                    </td>
                </tr>

                <!-- Footer -->
                <tr>
                    <td style="
                        background-color: #f8fafc;
                        padding: 20px 30px;
                        text-align: center;
                        border-top: 1px solid #e2e8f0;
                    ">

                        <p style="
                            font-size: 12px;
                            color: #94a3b8;
                            margin: 0;
                        ">
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
		"Reset your CARDex password using this link:\r\n\r\n"+
			"%s\r\n\r\n"+
			"This link expires in 15 minutes.\r\n\r\n"+
			"If you did not request a password reset, "+
			"you can safely ignore this email.\r\n\r\n"+
			"Never share this reset link with anyone else.",
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
