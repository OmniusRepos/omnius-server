package services

import (
	"fmt"
	"net/smtp"
	"strings"
)

// EmailService sends license emails via SMTP
type EmailService struct {
	host string
	port string
	user string
	pass string
	from string
}

// NewEmailService creates an email service. Returns nil if SMTP is not configured.
func NewEmailService(host, port, user, pass, from string) *EmailService {
	if host == "" || user == "" || pass == "" {
		return nil
	}
	if from == "" {
		from = user
	}
	return &EmailService{
		host: host,
		port: port,
		user: user,
		pass: pass,
		from: from,
	}
}

// SendLicenseEmail sends a license key to the customer
func (s *EmailService) SendLicenseEmail(to, name, licenseKey, plan string) error {
	subject := "Your Omnius License Key"

	displayName := name
	if displayName == "" {
		displayName = "there"
	}

	planDisplay := strings.ToUpper(plan[:1]) + plan[1:]
	deployments := "1"
	if plan == "business" {
		deployments = "5"
	}

	body := fmt.Sprintf(`Hi %s,

Thank you for purchasing Omnius (%s plan)!

Your license key:

    %s

Deploy with Docker:

    docker run -d \
      -e LICENSE_KEY=%s \
      -v omnius-data:/app/data \
      -p 8080:8080 \
      omnius/server:latest

Plan details:
  - Plan: %s
  - Max deployments: %s

Need help? Visit https://omnius.stream/setup or reply to this email.

— Omnius Team
`, displayName, planDisplay, licenseKey, licenseKey, planDisplay, deployments)

	msg := fmt.Sprintf("From: Omnius <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)

	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
