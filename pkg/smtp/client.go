package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"time"
)

// Client wraps an outbound SMTP submission agent
type Client struct {
	Host     string
	Port     int
	UseTLS   bool
	Username string
	Password string
}

// SendMessage acts as the formal execution path pumping RFC5322 payloads to upstream relays
func (c *Client) SendMessage(ctx context.Context, from string, to []string, payload []byte) error {
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	log.Printf("[SMTP] Connecting to submission proxy %s...", addr)

	// Enforce context timeout
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled before dial: %w", err)
	}

	auth := smtp.PlainAuth("", c.Username, c.Password, c.Host)

	// Since Go 1.21, net/smtp detects and transparently upgrades to STARTTLS
	// We handle explicit TLS directly if using port 465.
	if c.UseTLS && c.Port == 465 {
		dialer := &net.Dialer{Timeout: 30 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: c.Host})
		if err != nil {
			return fmt.Errorf("implicit TLS connection failed: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, c.Host)
		if err != nil {
			return fmt.Errorf("SMTP handshake rejected: %w", err)
		}
		
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
		
		if err := client.Mail(from); err != nil {
			return err
		}
		for _, rcpt := range to {
			if err := client.Rcpt(rcpt); err != nil {
				return err
			}
		}

		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(payload)
		if err != nil {
			return err
		}
		
		err = w.Close()
		if err != nil {
			return err
		}
		return client.Quit()
	}

	// STARTTLS handling via standard lib for port 587
	// We use a custom dialer to respect the context timeout
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("SMTP dial failed: %w", err)
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP NewClient failed: %w", err)
	}
	
	// Try STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: host}
		if err = client.StartTLS(config); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}
	
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}
	if err = client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err = client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(payload)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return client.Quit()
}
