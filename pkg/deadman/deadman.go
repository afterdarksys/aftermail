// Package deadman implements Dead Man's Switch delivery for AfterMail.
//
// A switch arms itself when created. The owner must call CheckIn() before the
// deadline or the switch fires — sending the configured message via the
// MailScript-aware scheduler.  Use cases: key handoff, legal disclosure,
// last-will messages, dramatic effect.
package deadman

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TriggerFn is called when a switch fires.  It receives the Switch payload
// and should enqueue the message for delivery.
type TriggerFn func(ctx context.Context, s *Switch) error

// Status of a dead man's switch.
type Status string

const (
	StatusArmed     Status = "armed"
	StatusDisarmed  Status = "disarmed"
	StatusFired     Status = "fired"
	StatusCancelled Status = "cancelled"
)

// Switch is a single dead man's switch instance.
type Switch struct {
	// ID is a unique identifier.
	ID string `json:"id"`

	// Label is a human-readable name shown in the UI.
	Label string `json:"label"`

	// Deadline is when the switch fires if no check-in is received.
	Deadline time.Time `json:"deadline"`

	// CheckInInterval is how often the user must check in.
	// Each successful check-in resets Deadline to now+CheckInInterval.
	CheckInInterval time.Duration `json:"check_in_interval"`

	// LastCheckIn is the timestamp of the most recent check-in.
	LastCheckIn time.Time `json:"last_check_in"`

	// Status is the current lifecycle state.
	Status Status `json:"status"`

	// Recipients is the list of email addresses to notify when the switch fires.
	Recipients []string `json:"recipients"`

	// Subject is the email subject sent on trigger.
	Subject string `json:"subject"`

	// Body is the email body (may be a MailScript template string).
	Body string `json:"body"`

	// MailScript is an optional Starlark script executed on trigger to
	// transform the message before delivery.
	MailScript string `json:"mailscript,omitempty"`

	// GracePeriod is an additional delay after the deadline before firing.
	// Gives time for late check-ins to arrive.
	GracePeriod time.Duration `json:"grace_period"`

	// CreatedAt records when the switch was armed.
	CreatedAt time.Time `json:"created_at"`
}

// Manager tracks all switches and runs the background monitor.
type Manager struct {
	mu        sync.Mutex
	switches  map[string]*Switch
	trigger   TriggerFn
	stateDir  string
	quit      chan struct{}
	wg        sync.WaitGroup
}

// NewManager creates a Manager.  stateDir is where switch state is persisted
// across daemon restarts.  trigger is called when a switch fires.
func NewManager(stateDir string, trigger TriggerFn) (*Manager, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("state dir: %w", err)
	}
	m := &Manager{
		switches: make(map[string]*Switch),
		trigger:  trigger,
		stateDir: stateDir,
		quit:     make(chan struct{}),
	}
	if err := m.load(); err != nil {
		log.Printf("[deadman] warning: could not load state: %v", err)
	}
	m.wg.Add(1)
	go m.monitor()
	return m, nil
}

// Arm creates and registers a new armed switch.
func (m *Manager) Arm(s *Switch) error {
	if s.ID == "" {
		return fmt.Errorf("switch ID is required")
	}
	if s.Deadline.IsZero() && s.CheckInInterval == 0 {
		return fmt.Errorf("deadline or check_in_interval is required")
	}
	if len(s.Recipients) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	now := time.Now()
	s.Status = StatusArmed
	s.CreatedAt = now
	s.LastCheckIn = now
	if s.Deadline.IsZero() {
		s.Deadline = now.Add(s.CheckInInterval)
	}

	m.mu.Lock()
	m.switches[s.ID] = s
	m.mu.Unlock()

	log.Printf("[deadman] armed switch %q — fires at %s", s.ID, s.Deadline.Format(time.RFC3339))
	return m.save(s)
}

// CheckIn resets the deadline for the given switch ID.
// Returns an error if the switch does not exist or has already fired.
func (m *Manager) CheckIn(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.switches[id]
	if !ok {
		return fmt.Errorf("switch %q not found", id)
	}
	if s.Status != StatusArmed {
		return fmt.Errorf("switch %q is %s, not armed", id, s.Status)
	}

	now := time.Now()
	s.LastCheckIn = now
	if s.CheckInInterval > 0 {
		s.Deadline = now.Add(s.CheckInInterval)
	}
	log.Printf("[deadman] check-in for %q — new deadline %s", id, s.Deadline.Format(time.RFC3339))
	return m.save(s)
}

// Disarm permanently cancels a switch without firing it.
func (m *Manager) Disarm(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.switches[id]
	if !ok {
		return fmt.Errorf("switch %q not found", id)
	}
	s.Status = StatusCancelled
	log.Printf("[deadman] disarmed switch %q", id)
	return m.save(s)
}

// List returns all registered switches.
func (m *Manager) List() []*Switch {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Switch, 0, len(m.switches))
	for _, s := range m.switches {
		out = append(out, s)
	}
	return out
}

// Get returns a single switch by ID.
func (m *Manager) Get(id string) (*Switch, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.switches[id]
	return s, ok
}

// Stop shuts down the background monitor.
func (m *Manager) Stop() {
	close(m.quit)
	m.wg.Wait()
}

// monitor ticks every minute and fires any overdue switches.
func (m *Manager) monitor() {
	defer m.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.tick()
		case <-m.quit:
			return
		}
	}
}

func (m *Manager) tick() {
	now := time.Now()
	m.mu.Lock()
	var toFire []*Switch
	for _, s := range m.switches {
		if s.Status == StatusArmed && now.After(s.Deadline.Add(s.GracePeriod)) {
			s.Status = StatusFired
			toFire = append(toFire, s)
		}
	}
	m.mu.Unlock()

	for _, s := range toFire {
		log.Printf("[deadman] switch %q FIRED — sending to %v", s.ID, s.Recipients)
		if m.trigger != nil {
			if err := m.trigger(context.Background(), s); err != nil {
				log.Printf("[deadman] trigger error for %q: %v", s.ID, err)
				// Put back to armed so we retry next tick.
				m.mu.Lock()
				s.Status = StatusArmed
				m.mu.Unlock()
				continue
			}
		}
		m.save(s) //nolint:errcheck
	}
}

// ─── Persistence ─────────────────────────────────────────────────────────────

func (m *Manager) save(s *Switch) error {
	path := filepath.Join(m.stateDir, s.ID+".json")
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func (m *Manager) load() error {
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(m.stateDir, e.Name()))
		if err != nil {
			continue
		}
		var s Switch
		if err := json.Unmarshal(b, &s); err != nil {
			continue
		}
		// Re-arm switches that were armed before the daemon restarted.
		if s.Status == StatusArmed {
			m.switches[s.ID] = &s
			log.Printf("[deadman] reloaded switch %q (deadline %s)", s.ID, s.Deadline.Format(time.RFC3339))
		}
	}
	return nil
}

// TimeUntilFire returns how long until the switch fires (or negative if overdue).
func (s *Switch) TimeUntilFire() time.Duration {
	return time.Until(s.Deadline.Add(s.GracePeriod))
}

// IsOverdue returns true if the switch has passed its deadline+grace period.
func (s *Switch) IsOverdue() bool {
	return s.TimeUntilFire() < 0
}
