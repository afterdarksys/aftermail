package dispatcher

import (
	"log"
	"time"

	"github.com/afterdarksys/aftermail/pkg/storage"
)

// Dispatcher periodically checks the database for pending scheduled messages and sends them
type Dispatcher struct {
	DB *storage.DB
}

func NewDispatcher(db *storage.DB) *Dispatcher {
	return &Dispatcher{DB: db}
}

func (d *Dispatcher) Start() {
	go func() {
		log.Println("[Dispatcher] Scheduled sending engine started...")
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			d.RunTick()
		}
	}()
}

func (d *Dispatcher) RunTick() {
	// STUB: Query scheduled_messages where string(dispatch_at) <= time.Now() AND status = 'pending'
	// Then loop through records and dial SMTP via pkg/smtp
	// Finally mark status as 'sent'
}
