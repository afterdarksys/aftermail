package accounts

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
)

// quicConn implements net.Conn over a QUIC stream
type quicConn struct {
	*quic.Stream
	conn *quic.Conn
}

func (q *quicConn) LocalAddr() net.Addr {
	return q.conn.LocalAddr()
}

func (q *quicConn) RemoteAddr() net.Addr {
	return q.conn.RemoteAddr()
}

func (q *quicConn) SetDeadline(t time.Time) error {
	if err := q.Stream.SetDeadline(t); err != nil {
		return err
	}
	return nil
}

func (q *quicConn) SetReadDeadline(t time.Time) error {
	return q.Stream.SetReadDeadline(t)
}

func (q *quicConn) SetWriteDeadline(t time.Time) error {
	return q.Stream.SetWriteDeadline(t)
}

// quicDialer establishes a QUIC connection and opens a bidirectional stream
func quicDialer(ctx context.Context, target string) (net.Conn, error) {
	// Strip quic:// prefix if present
	target = strings.TrimPrefix(target, "quic://")

	// Extract hostname for TLS configuration
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		// If no port, use target as hostname
		host = target
	}

	// Only skip verification for test/development domains
	// In production, proper certificate verification is enforced
	insecureSkipVerify := strings.HasSuffix(host, ".test") ||
		strings.HasSuffix(host, ".test.local") ||
		strings.HasSuffix(host, ".localhost") ||
		host == "localhost" ||
		strings.HasPrefix(host, "127.")

	tlsConf := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		ServerName:         host,
		NextProtos:         []string{"aftersmtp-quic"},
	}

	quicConfig := &quic.Config{
		MaxIdleTimeout:     30 * time.Second,
		KeepAlivePeriod:    15 * time.Second,
		EnableDatagrams:    true,
	}

	// Dial the QUIC server
	conn, err := quic.DialAddr(ctx, target, tlsConf, quicConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial quic: %w", err)
	}

	// Open a bidirectional stream for gRPC to multiplex over
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		_ = conn.CloseWithError(1, "failed to open stream")
		return nil, fmt.Errorf("failed to open quic stream: %w", err)
	}

	return &quicConn{
		Stream: stream,
		conn:   conn,
	}, nil
}
