package tlsconn

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

// Session represents an active raw TCP or TLS connection.
type Session struct {
	conn net.Conn
	reader *bufio.Reader
}

// Connect dial a host, optionally using TLS, with a timeout.
func Connect(host string, useTLS bool, timeout time.Duration) (*Session, error) {
	var conn net.Conn
	var err error

	if useTLS {
		// Use InsecureSkipVerify for a debugger tool so we can test self-signed certs
		config := &tls.Config{InsecureSkipVerify: true}
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", host, config)
	} else {
		conn, err = net.DialTimeout("tcp", host, timeout)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", host, err)
	}

	return &Session{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

// ReadLine reads a single line from the connection.
func (s *Session) ReadLine() (string, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line, nil
}

// WriteLine writes a string command ending with \r\n to the connection.
func (s *Session) WriteLine(cmd string) error {
	_, err := fmt.Fprintf(s.conn, "%s\r\n", cmd)
	return err
}

// Close terminates the connection.
func (s *Session) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
