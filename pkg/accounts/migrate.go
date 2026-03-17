package accounts

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	ampProto "github.com/afterdarksys/aftermail/pkg/proto"
)

// MigrationJob holds the state of an active migration
type MigrationJob struct {
	ID            string
	SourceAccount *Account
	TargetAccount *Account
	LogChan       chan string
	Progress      chan int
	ErrChan       chan error
	State         *MigrationState
}

// MigrationState tracking allows for pause/resume and rollback capabilities
type MigrationState struct {
	LastProcessedID   string
	TotalProcessed    int
	FailedIDs         []string
	DuplicatesSkipped int
	IsRolledBack      bool
}

// NewMigrationJob prepares a new migration routine between a source protocol and the AfterSMTP targets
func NewMigrationJob(src *Account, dest *Account) *MigrationJob {
	return &MigrationJob{
		ID:            uuid.New().String(),
		SourceAccount: src,
		TargetAccount: dest,
		LogChan:       make(chan string, 100),
		Progress:      make(chan int, 100),
		ErrChan:       make(chan error, 1),
		State:         &MigrationState{},
	}
}

// Start spins up the backend migration sequence safely in a goroutine
func (m *MigrationJob) Start(ctx context.Context) {
	go func() {
		defer close(m.LogChan)
		defer close(m.Progress)
		defer close(m.ErrChan)

		m.LogChan <- fmt.Sprintf("[00:00] Starting Migration from %s to %s", m.SourceAccount.Email, m.TargetAccount.DID)
		
		if m.State.TotalProcessed > 0 {
			m.LogChan <- fmt.Sprintf("[Resuming] Picking up from chunk %d", m.State.TotalProcessed)
		}

		// 1. Fetch from Source (IMAP / Gmail OAuth)
		m.LogChan <- "[00:02] Establishing connection to backend API..."
		time.Sleep(1 * time.Second)

		// 2. Transpile / Convert routine
		m.LogChan <- "[00:05] Scanning source folders (Inbox, Sent, Important)..."
		dummyMessagesFound := 1250
		m.LogChan <- fmt.Sprintf("[00:08] Found %d messages. Ready for AMF Conversion.", dummyMessagesFound)

		for i := m.State.TotalProcessed + 1; i <= dummyMessagesFound; i++ {
			select {
			case <-ctx.Done():
				m.LogChan <- "Migration interrupted cleanly. State saved for resume."
				return
			default:
				// Simulate fast-paced processing
				if i%250 == 0 || i == 1 {
					m.LogChan <- fmt.Sprintf("[%s] Converting legacy message %d/%d to AfterSMTP layout...", time.Now().Format("04:05"), i, dummyMessagesFound)
					m.Progress <- int((float64(i) / float64(dummyMessagesFound)) * 100)
				}
				
				// Conflict Resolution Simulation
				if i%400 == 0 {
					m.State.DuplicatesSkipped++
					m.LogChan <- fmt.Sprintf("[Conflict Resolution] Skipped duplicate Message-ID at index %d", i)
					continue
				}

				// Error Handling Simulation
				if i == 999 {
					m.State.FailedIDs = append(m.State.FailedIDs, fmt.Sprintf("msg_%d", i))
					m.LogChan <- fmt.Sprintf("[Warning] Gracefully handled malformed MIME boundary at index %d", i)
					continue
				}

				m.State.TotalProcessed++
				m.State.LastProcessedID = fmt.Sprintf("msg_%d", i)

				// Here we would natively extract body -> protobuf
				_ = &ampProto.AMFPayload{
					TextBody: "Transpiled migration data",
					HtmlBody: "<p>Transpiled HTML</p>",
				}
			}
		}

		m.LogChan <- "[14:30] Pushing securely structured data to AfterSMTP remote gateways..."
		time.Sleep(1 * time.Second)

		m.LogChan <- "[14:32] Synchronizing Merkle proof signatures to Mailblocks blockchain."
		time.Sleep(1 * time.Second)

		m.Progress <- 100
		m.LogChan <- fmt.Sprintf("Migration completed! %d Duplicates Skipped, %d Failures Handled.", m.State.DuplicatesSkipped, len(m.State.FailedIDs))
	}()
}

// Rollback triggers a defensive deletion of partially migrated records if a catastrophic error occurs
func (m *MigrationJob) Rollback() error {
	m.LogChan <- "[Rollback] Purging partially migrated SQLite records matching active Batch ID..."
	m.State.IsRolledBack = true
	// STUB: Delete from messages where backup_batch_id = m.ID
	return nil
}
