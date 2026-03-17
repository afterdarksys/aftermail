package caldav

import (
	"context"
	"fmt"
	"log"
)

// Client manages synchronization with an upstream CalDAV server
type Client struct {
	Endpoint string
	Username string
	Password string
}

// NewClient returns an initialized CalDAV client
func NewClient(endpoint, username, password string) *Client {
	return &Client{
		Endpoint: endpoint,
		Username: username,
		Password: password,
	}
}

// Sync events pulling from remote and pushing local changes
func (c *Client) Sync(ctx context.Context) error {
	log.Printf("[CalDAV] Synchronizing calendars with %s", c.Endpoint)
	// STUB: Implement `emersion/go-webdav/caldav` `Client.Propfind` and `Client.Report`.
	// Convert upstream `.ics` blocks into SQLite `calendar_events` and upload internal changes upstream via `Put`.
	return fmt.Errorf("CalDAV synchronization requires importing emersion/go-webdav")
}
