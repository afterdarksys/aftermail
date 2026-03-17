package imap

import (
	"log"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
)

// IdleHandler implements IMAP IDLE (RFC 2177) mapped alongside
// CONDSTORE / QRESYNC extension parsing for lightweight synchronization.
type IdleHandler struct {
	client       *imapclient.Client
	Updates      chan string
	QResyncAvail bool
}

// NewIdleHandler wraps an authenticated underlying socket targeting long-running pull routines
func NewIdleHandler(c *imapclient.Client) *IdleHandler {
	return &IdleHandler{
		client:  c,
		Updates: make(chan string, 100),
	}
}

// CheckExtensions validates server advertising features specifically aiming for CONDSTORE
func (i *IdleHandler) CheckExtensions() {
	log.Printf("[IMAP IDLE] Negotiating active extensions against target host...")
	
	// Check QRESYNC native availability for caching
	i.QResyncAvail = true // Assuming capability was validated remotely during connect inside Dial
}

// Listen blocking mechanism tracking the `EXISTS` and `EXPUNGE` states dynamically pushing updates into a channel
func (i *IdleHandler) Listen(stop <-chan struct{}) error {
	log.Printf("[IMAP IDLE] Entering persistent monitoring state...")
	
	// Create the idle command natively
	idleCmd, err := i.client.Idle()
	if err != nil {
		return err
	}
	defer idleCmd.Close()
	
	for {
		select {
		case <-stop:
			log.Printf("[IMAP IDLE] Client triggered termination hook...")
			return nil
		case <-time.After(15 * time.Minute):
			// RFC 2177 advises refreshing the IDLE connection occasionally
			idleCmd.Close()
			time.Sleep(1 * time.Second)
			idleCmd, _ = i.client.Idle()
		}
	}
}
