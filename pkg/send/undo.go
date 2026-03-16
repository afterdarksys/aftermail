package send

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/afterdarksys/aftermail/pkg/accounts"
)

// UndoSendManager handles delayed sending with undo capability
type UndoSendManager struct {
	mu             sync.RWMutex
	pendingSends   map[string]*PendingSend
	defaultDelay   time.Duration
	onSend         func(*accounts.Message) error
	onCancel       func(string)
}

// PendingSend represents a message waiting to be sent
type PendingSend struct {
	ID            string
	Message       *accounts.Message
	ScheduledTime time.Time
	CancelFunc    context.CancelFunc
	Status        SendStatus
}

// SendStatus represents the status of a send operation
type SendStatus string

const (
	StatusPending   SendStatus = "pending"
	StatusCancelled SendStatus = "cancelled"
	StatusSending   SendStatus = "sending"
	StatusSent      SendStatus = "sent"
	StatusFailed    SendStatus = "failed"
)

// NewUndoSendManager creates a new undo send manager
func NewUndoSendManager(defaultDelay time.Duration) *UndoSendManager {
	return &UndoSendManager{
		pendingSends: make(map[string]*PendingSend),
		defaultDelay: defaultDelay,
	}
}

// ScheduleSend schedules a message to be sent after the undo delay
func (m *UndoSendManager) ScheduleSend(msg *accounts.Message, delay time.Duration) (string, error) {
	if delay == 0 {
		delay = m.defaultDelay
	}

	// Generate unique ID
	sendID := fmt.Sprintf("send-%d-%d", time.Now().Unix(), msg.ID)

	ctx, cancel := context.WithCancel(context.Background())

	pending := &PendingSend{
		ID:            sendID,
		Message:       msg,
		ScheduledTime: time.Now().Add(delay),
		CancelFunc:    cancel,
		Status:        StatusPending,
	}

	m.mu.Lock()
	m.pendingSends[sendID] = pending
	m.mu.Unlock()

	// Start the countdown
	go m.countdown(ctx, pending, delay)

	return sendID, nil
}

// CancelSend cancels a pending send
func (m *UndoSendManager) CancelSend(sendID string) error {
	m.mu.Lock()
	pending, exists := m.pendingSends[sendID]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("send ID not found: %s", sendID)
	}

	if pending.Status != StatusPending {
		return fmt.Errorf("cannot cancel: message is %s", pending.Status)
	}

	// Cancel the context
	pending.CancelFunc()

	m.mu.Lock()
	pending.Status = StatusCancelled
	m.mu.Unlock()

	if m.onCancel != nil {
		m.onCancel(sendID)
	}

	return nil
}

// GetPending returns all pending sends
func (m *UndoSendManager) GetPending() []*PendingSend {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pending := make([]*PendingSend, 0, len(m.pendingSends))
	for _, p := range m.pendingSends {
		if p.Status == StatusPending {
			pending = append(pending, p)
		}
	}

	return pending
}

// GetStatus returns the status of a send
func (m *UndoSendManager) GetStatus(sendID string) (SendStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pending, exists := m.pendingSends[sendID]
	if !exists {
		return "", fmt.Errorf("send ID not found: %s", sendID)
	}

	return pending.Status, nil
}

// SetOnSend sets the callback for when a message is actually sent
func (m *UndoSendManager) SetOnSend(fn func(*accounts.Message) error) {
	m.onSend = fn
}

// SetOnCancel sets the callback for when a send is cancelled
func (m *UndoSendManager) SetOnCancel(fn func(string)) {
	m.onCancel = fn
}

// countdown waits for the delay and then sends the message
func (m *UndoSendManager) countdown(ctx context.Context, pending *PendingSend, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		// Send was cancelled
		return

	case <-timer.C:
		// Time to send!
		m.mu.Lock()
		pending.Status = StatusSending
		m.mu.Unlock()

		// Actually send the message
		err := m.sendMessage(pending.Message)

		m.mu.Lock()
		if err != nil {
			pending.Status = StatusFailed
		} else {
			pending.Status = StatusSent
		}
		m.mu.Unlock()

		// Clean up after a while
		time.AfterFunc(5*time.Minute, func() {
			m.mu.Lock()
			delete(m.pendingSends, pending.ID)
			m.mu.Unlock()
		})
	}
}

// sendMessage actually sends the message
func (m *UndoSendManager) sendMessage(msg *accounts.Message) error {
	if m.onSend != nil {
		return m.onSend(msg)
	}

	// Default: no-op (caller should set onSend callback)
	return nil
}

// GetTimeRemaining returns the time remaining before send
func (m *UndoSendManager) GetTimeRemaining(sendID string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pending, exists := m.pendingSends[sendID]
	if !exists {
		return 0, fmt.Errorf("send ID not found: %s", sendID)
	}

	if pending.Status != StatusPending {
		return 0, nil
	}

	remaining := time.Until(pending.ScheduledTime)
	if remaining < 0 {
		return 0, nil
	}

	return remaining, nil
}

// CleanupOld removes old completed sends
func (m *UndoSendManager) CleanupOld(olderThan time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	for id, pending := range m.pendingSends {
		if pending.Status != StatusPending && pending.ScheduledTime.Before(cutoff) {
			delete(m.pendingSends, id)
		}
	}
}
