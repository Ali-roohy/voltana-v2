// Package mailer provides email delivery for account verification. The concrete
// types here satisfy service.Mailer structurally (no import of service), so the
// service layer stays mockable and SMTP is never reached in unit tests.
package mailer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

// SMTPMailer sends verification emails through a configured SMTP relay.
type SMTPMailer struct {
	addr string // host:port
	auth smtp.Auth
	from string
}

// NewSMTP builds an SMTPMailer. Auth is omitted when user is empty (e.g. an
// unauthenticated local relay).
func NewSMTP(host, port, user, password, from string) *SMTPMailer {
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, password, host)
	}
	return &SMTPMailer{addr: net.JoinHostPort(host, port), auth: auth, from: from}
}

func (m *SMTPMailer) SendVerificationEmail(_ context.Context, toEmail, verifyURL string) error {
	subject := "Verify your Voltana account"
	body := "Welcome to Voltana!\r\n\r\n" +
		"Please verify your email address by opening this link:\r\n" +
		verifyURL + "\r\n\r\n" +
		"This link expires in 24 hours. If you did not create an account, ignore this email.\r\n"
	msg := buildMessage(m.from, toEmail, subject, body)
	if err := smtp.SendMail(m.addr, m.auth, m.from, []string{toEmail}, msg); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func (m *SMTPMailer) SendOTPEmail(_ context.Context, toEmail, code string) error {
	subject := "تست OTP ولتانا"
	body := "کد تست ولتانا: " + code + "\r\n"
	msg := buildMessage(m.from, toEmail, subject, body)
	if err := smtp.SendMail(m.addr, m.auth, m.from, []string{toEmail}, msg); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// LogMailer is used when SMTP is not configured (dev). It never logs the raw
// token / verify URL (which carries the token) or the full recipient address.
type LogMailer struct{}

func (LogMailer) SendVerificationEmail(_ context.Context, toEmail, _ string) error {
	log.Printf("mailer: SMTP not configured — verification email skipped for %s", maskEmail(toEmail))
	return nil
}

func (LogMailer) SendOTPEmail(_ context.Context, toEmail, code string) error {
	log.Printf("mailer: SMTP not configured — OTP test email skipped for %s (code=%s)", maskEmail(toEmail), code)
	return nil
}

func maskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 1 {
		return "***"
	}
	return string(email[0]) + "***" + email[at:]
}
