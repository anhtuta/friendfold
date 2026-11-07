package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTP delivers mail through a local or remote SMTP server (Mailpit in dev).
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	StartTLS bool
}

// Send transmits msg. The request/process context bounds dial and I/O.
func (s *SMTP) Send(ctx context.Context, msg Message) error {
	if s == nil {
		return fmt.Errorf("mail smtp: sender is nil")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mail smtp: %w", err)
	}
	to := strings.TrimSpace(msg.To)
	if to == "" {
		return fmt.Errorf("mail smtp: missing recipient")
	}
	from := envelopeAddr(s.From)
	if from == "" {
		return fmt.Errorf("mail smtp: missing from")
	}

	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mail smtp dial: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return fmt.Errorf("mail smtp hello: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsCfg := &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}
			if err := client.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("mail smtp starttls: %w", err)
			}
		}
	}
	if strings.TrimSpace(s.Username) != "" {
		auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mail smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail smtp from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mail smtp rcpt: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail smtp data: %w", err)
	}
	headerFrom := strings.TrimSpace(s.From)
	if headerFrom == "" {
		headerFrom = from
	}
	payload := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		headerFrom, to, sanitizeHeader(msg.Subject), msg.Body)
	if _, err := wc.Write([]byte(payload)); err != nil {
		_ = wc.Close()
		return fmt.Errorf("mail smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("mail smtp write: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("mail smtp quit: %w", err)
	}
	return nil
}

func envelopeAddr(from string) string {
	from = strings.TrimSpace(from)
	if i := strings.LastIndex(from, "<"); i >= 0 {
		rest := from[i+1:]
		if j := strings.Index(rest, ">"); j > 0 {
			return strings.TrimSpace(rest[:j])
		}
	}
	return from
}

func sanitizeHeader(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, s)
}
