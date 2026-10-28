package notifications

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host    string
	Port    int
	From    string
	Timeout time.Duration
}

type SMTPMailer struct {
	config SMTPConfig
}

func NewSMTPMailer(config SMTPConfig) *SMTPMailer {
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	return &SMTPMailer{config: config}
}

func (m *SMTPMailer) Send(ctx context.Context, message Message) error {
	address := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	connection, err := (&net.Dialer{Timeout: m.config.Timeout}).DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("dial SMTP: %w", err)
	}
	defer connection.Close()
	deadline := time.Now().Add(m.config.Timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}
	client, err := smtp.NewClient(connection, m.config.Host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()
	if err := client.Mail(m.config.From); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(message.To); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP body: %w", err)
	}
	headers := strings.Builder{}
	fmt.Fprintf(&headers, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n", m.config.From, message.To, message.Subject)
	if _, err := writer.Write([]byte(headers.String())); err != nil {
		return fmt.Errorf("write SMTP headers: %w", err)
	}
	if _, err := writer.Write([]byte(message.Text)); err != nil {
		return fmt.Errorf("write SMTP body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close SMTP body: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP session: %w", err)
	}
	return nil
}
