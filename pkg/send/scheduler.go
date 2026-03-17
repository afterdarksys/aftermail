package send

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/afterdarksys/aftermail/pkg/accounts"
)
type ScheduledMessage struct {
	ID        string
	Account   *accounts.Account
	To        []string
	Cc        []string
	Bcc       []string
	Subject   string
	Body      string // Rich Markdown / HTML body
	SendAt    time.Time
}

// ScheduledDispatcher manages background cron jobs for sending deferred emails
type ScheduledDispatcher struct {
	queue []ScheduledMessage
	mu    sync.Mutex
	quit  chan struct{}
	wg    sync.WaitGroup
	ctx   context.Context
	cancel context.CancelFunc
}

// NewScheduledDispatcher creates a background worker running every minute
func NewScheduledDispatcher() *ScheduledDispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	sd := &ScheduledDispatcher{
		queue:  make([]ScheduledMessage, 0),
		quit:   make(chan struct{}),
		ctx:    ctx,
		cancel: cancel,
	}
	go sd.loop()
	return sd
}

// QueueMessage adds an email to the dispatch queue
func (sd *ScheduledDispatcher) QueueMessage(msg ScheduledMessage) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	// Create a unique ID fallback
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("sched-%d", time.Now().UnixNano())
	}
	
	sd.queue = append(sd.queue, msg)
	log.Printf("Message scheduled to %s at %v", msg.To, msg.SendAt)
}

// CancelMessage removes a message from the queue before it sends
func (sd *ScheduledDispatcher) CancelMessage(id string) bool {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	for i, msg := range sd.queue {
		if msg.ID == id {
			sd.queue = append(sd.queue[:i], sd.queue[i+1:]...)
			log.Printf("Scheduled message %s cancelled.", id)
			return true
		}
	}
	return false
}

func (sd *ScheduledDispatcher) loop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sd.processQueue()
		case <-sd.quit:
			return
		}
	}
}

// processQueue iterates through pending messages and fires them if Time >= SendAt
func (sd *ScheduledDispatcher) processQueue() {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	now := time.Now()
	var pending []ScheduledMessage

	for _, msg := range sd.queue {
		if now.After(msg.SendAt) || now.Equal(msg.SendAt) {
			// Dispatch via standard Send logic
			log.Printf("Transmitting scheduled message %s to %v...", msg.ID, msg.To)

			// Fire asynchronously to avoid blocking cron loop
			// Track goroutines with WaitGroup to prevent leaks
			sd.wg.Add(1)
			go func(m ScheduledMessage) {
				defer sd.wg.Done()

				select {
				case <-sd.ctx.Done():
					log.Printf("Scheduled message %s cancelled due to shutdown", m.ID)
					return
				default:
				}

				log.Printf("Executing scheduled message %s to %v: %s", m.ID, m.To, m.Subject)
				// Here we would dispatch to the correct account handler
				// TODO: Add actual dispatch logic with error handling and retry
			}(msg)
		} else {
			// Keep in queue if it hasn't fired
			pending = append(pending, msg)
		}
	}

	sd.queue = pending
}

// Stop cleanly shuts down the dispatcher and waits for all in-flight sends
func (sd *ScheduledDispatcher) Stop() {
	close(sd.quit)
	sd.cancel() // Cancel context to stop any running goroutines
	sd.wg.Wait() // Wait for all send goroutines to complete
	log.Printf("Scheduled dispatcher stopped, all pending sends completed")
}
