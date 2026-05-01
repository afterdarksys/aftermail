// Package presence detects when a recipient is active and holds outbound
// messages until an optimal delivery moment.
//
// Signals used to infer presence:
//   - IMAP IDLE heartbeats (recipient's server is responding = likely online)
//   - IMAP RECENT / EXISTS changes (they opened their client)
//   - Calendar free/busy status (don't deliver during a meeting)
//   - Historical open-time patterns (they usually read email at 9 AM)
//   - Timezone inference from MX record geolocation
package presence

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	imapv1 "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Signal is a single data point contributing to presence score.
type Signal struct {
	Source    string    `json:"source"`
	Score     float64   `json:"score"` // 0.0 = definitely away, 1.0 = definitely active
	Timestamp time.Time `json:"timestamp"`
}

// PresenceState is the inferred recipient activity state.
type PresenceState string

const (
	StateUnknown PresenceState = "unknown"
	StateActive  PresenceState = "active"   // high confidence they're at the keyboard
	StateOnline  PresenceState = "online"   // IMAP connected, not necessarily reading
	StateAway    PresenceState = "away"     // no activity signal
	StateBusy    PresenceState = "busy"     // in a calendar event
)

// PresenceReport is the computed presence for a recipient.
type PresenceReport struct {
	Recipient string        `json:"recipient"`
	State     PresenceState `json:"state"`
	Score     float64       `json:"score"`   // aggregate 0.0–1.0
	Signals   []Signal      `json:"signals"`
	BestSendWindow time.Time `json:"best_send_window"` // predicted next good time
	UpdatedAt time.Time     `json:"updated_at"`
}

// OpenPattern records historical email-open behaviour for a recipient.
type OpenPattern struct {
	// HourWeights[h] is the relative open probability for hour h (0..23 UTC).
	HourWeights [24]float64 `json:"hour_weights"`
	// DayWeights[d] is the relative open probability for weekday d (0=Sun..6=Sat).
	DayWeights [7]float64 `json:"day_weights"`
	// SampleCount is the number of opens used to build this model.
	SampleCount int `json:"sample_count"`
}

// RecordOpen updates the open pattern with a new open event at t.
func (p *OpenPattern) RecordOpen(t time.Time) {
	h := t.UTC().Hour()
	d := int(t.UTC().Weekday())

	// Exponential moving average: new weight gets 10% influence.
	alpha := 0.1
	p.HourWeights[h] = p.HourWeights[h]*(1-alpha) + alpha
	p.DayWeights[d] = p.DayWeights[d]*(1-alpha) + alpha

	// Decay all other slots slightly.
	decay := 1 - alpha/23.0
	for i := range p.HourWeights {
		if i != h {
			p.HourWeights[i] *= decay
		}
	}
	p.SampleCount++
}

// BestHour returns the UTC hour with the highest predicted open probability.
func (p *OpenPattern) BestHour() int {
	best := 0
	for h, w := range p.HourWeights {
		if w > p.HourWeights[best] {
			best = h
		}
	}
	return best
}

// NextBestWindow returns the next calendar time (after now) when the recipient
// is most likely to open email.
func (p *OpenPattern) NextBestWindow(now time.Time) time.Time {
	if p.SampleCount == 0 {
		// No data: schedule for next morning 9 AM UTC.
		t := now.UTC().Add(24 * time.Hour)
		return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, time.UTC)
	}

	bestHour := p.BestHour()
	candidate := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(),
		bestHour, 0, 0, 0, time.UTC)
	if candidate.Before(now.Add(5 * time.Minute)) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}

// ─── Tracker ─────────────────────────────────────────────────────────────────

// Tracker monitors recipient presence and provides hold-until-active scheduling.
type Tracker struct {
	mu       sync.Mutex
	reports  map[string]*PresenceReport
	patterns map[string]*OpenPattern
}

// NewTracker creates a Tracker.
func NewTracker() *Tracker {
	return &Tracker{
		reports:  make(map[string]*PresenceReport),
		patterns: make(map[string]*OpenPattern),
	}
}

// Probe attempts an IMAP connection to infer presence.
// imapAddr is host:port; credentials are used to authenticate (IMAP ID only —
// no mail is read).
func (t *Tracker) Probe(ctx context.Context, recipient, imapAddr, username, password string) (*PresenceReport, error) {
	signals := []Signal{}
	now := time.Now()

	// Attempt IMAP connection.
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	c, err := dialWithContext(dialCtx, imapAddr)
	if err != nil {
		// Server unreachable = no presence signal.
		signals = append(signals, Signal{Source: "imap_connect", Score: 0.0, Timestamp: now})
	} else {
		defer c.Logout()
		signals = append(signals, Signal{Source: "imap_connect", Score: 0.5, Timestamp: now})

		if err := c.Login(username, password); err == nil {
			// Successful auth: client is configured = likely active.
			signals = append(signals, Signal{Source: "imap_auth", Score: 0.7, Timestamp: now})

			// Check INBOX for recent activity.
			mbox, err := c.Select("INBOX", true)
			if err == nil {
				recentScore := clamp(float64(mbox.Recent)/10.0, 0, 1)
				signals = append(signals, Signal{
					Source:    "imap_recent",
					Score:     recentScore,
					Timestamp: now,
				})
			}
		}
	}

	// Factor in historical open pattern.
	t.mu.Lock()
	pat, hasPat := t.patterns[recipient]
	t.mu.Unlock()

	var bestWindow time.Time
	if hasPat {
		hourScore := pat.HourWeights[now.UTC().Hour()]
		norm := maxHourWeight(pat)
		if norm > 0 {
			hourScore /= norm
		}
		signals = append(signals, Signal{Source: "open_pattern", Score: hourScore, Timestamp: now})
		bestWindow = pat.NextBestWindow(now)
	} else {
		bestWindow = now.Add(1 * time.Hour)
	}

	// Aggregate score (average of signals).
	aggregate := aggregateSignals(signals)
	state := scoreToState(aggregate)

	report := &PresenceReport{
		Recipient:      recipient,
		State:          state,
		Score:          aggregate,
		Signals:        signals,
		BestSendWindow: bestWindow,
		UpdatedAt:      now,
	}

	t.mu.Lock()
	t.reports[recipient] = report
	t.mu.Unlock()

	return report, nil
}

// RecordOpen updates the open pattern for a recipient when they open a message.
func (t *Tracker) RecordOpen(recipient string, at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.patterns[recipient]; !ok {
		t.patterns[recipient] = &OpenPattern{}
	}
	t.patterns[recipient].RecordOpen(at)
	log.Printf("[presence] recorded open for %s at %s", recipient, at.Format(time.RFC3339))
}

// GetReport returns the last computed presence report for a recipient.
func (t *Tracker) GetReport(recipient string) (*PresenceReport, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	r, ok := t.reports[recipient]
	return r, ok
}

// ─── Held-send scheduler ──────────────────────────────────────────────────────

// HeldMessage is a message waiting for the recipient to be active.
type HeldMessage struct {
	ID           string
	Recipient    string
	SendFn       func(ctx context.Context) error
	QueuedAt     time.Time
	MaxWait      time.Duration // absolute deadline before sending anyway
	MinScore     float64       // minimum presence score to trigger send
}

// HoldQueue manages messages waiting for optimal recipient presence.
type HoldQueue struct {
	mu      sync.Mutex
	held    []*HeldMessage
	tracker *Tracker
	quit    chan struct{}
	wg      sync.WaitGroup
}

// NewHoldQueue creates a HoldQueue backed by tracker.
func NewHoldQueue(tracker *Tracker) *HoldQueue {
	q := &HoldQueue{
		tracker: tracker,
		quit:    make(chan struct{}),
	}
	q.wg.Add(1)
	go q.loop()
	return q
}

// Hold adds a message to the queue.
func (q *HoldQueue) Hold(m *HeldMessage) {
	if m.MaxWait == 0 {
		m.MaxWait = 24 * time.Hour
	}
	if m.MinScore == 0 {
		m.MinScore = 0.6
	}
	q.mu.Lock()
	q.held = append(q.held, m)
	q.mu.Unlock()
	log.Printf("[presence] holding message %s for %s (min score %.2f)", m.ID, m.Recipient, m.MinScore)
}

// Stop shuts down the hold queue.
func (q *HoldQueue) Stop() {
	close(q.quit)
	q.wg.Wait()
}

func (q *HoldQueue) loop() {
	defer q.wg.Done()
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			q.dispatch()
		case <-q.quit:
			return
		}
	}
}

func (q *HoldQueue) dispatch() {
	now := time.Now()
	q.mu.Lock()
	remaining := q.held[:0]
	var toSend []*HeldMessage

	for _, m := range q.held {
		deadline := m.QueuedAt.Add(m.MaxWait)
		if now.After(deadline) {
			// Max wait exceeded — send regardless of presence.
			toSend = append(toSend, m)
			continue
		}
		report, ok := q.tracker.GetReport(m.Recipient)
		if ok && report.Score >= m.MinScore {
			toSend = append(toSend, m)
			continue
		}
		remaining = append(remaining, m)
	}
	q.held = remaining
	q.mu.Unlock()

	for _, m := range toSend {
		go func(msg *HeldMessage) {
			log.Printf("[presence] sending held message %s to %s", msg.ID, msg.Recipient)
			if err := msg.SendFn(context.Background()); err != nil {
				log.Printf("[presence] send error for %s: %v", msg.ID, err)
			}
		}(m)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func dialWithContext(ctx context.Context, addr string) (*client.Client, error) {
	done := make(chan struct {
		c   *client.Client
		err error
	}, 1)
	go func() {
		c, err := client.DialTLS(addr, nil)
		done <- struct {
			c   *client.Client
			err error
		}{c, err}
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("dial timeout")
	case r := <-done:
		return r.c, r.err
	}
}

func aggregateSignals(signals []Signal) float64 {
	if len(signals) == 0 {
		return 0
	}
	var sum float64
	for _, s := range signals {
		sum += s.Score
	}
	return sum / float64(len(signals))
}

func scoreToState(score float64) PresenceState {
	switch {
	case score >= 0.8:
		return StateActive
	case score >= 0.5:
		return StateOnline
	case score >= 0.2:
		return StateAway
	default:
		return StateUnknown
	}
}

func maxHourWeight(p *OpenPattern) float64 {
	max := 0.0
	for _, w := range p.HourWeights {
		if w > max {
			max = w
		}
	}
	return max
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(hi, math.Max(lo, v))
}

// ─── Calendar busy check ──────────────────────────────────────────────────────

// BusySlot represents a busy calendar block.
type BusySlot struct {
	Start time.Time
	End   time.Time
}

// FindNextFree returns the earliest time after now that is not in any busy slot.
func FindNextFree(now time.Time, busy []BusySlot) time.Time {
	sort.Slice(busy, func(i, j int) bool {
		return busy[i].Start.Before(busy[j].Start)
	})

	candidate := now
	for _, slot := range busy {
		if candidate.Before(slot.End) && candidate.After(slot.Start.Add(-1*time.Second)) {
			candidate = slot.End.Add(5 * time.Minute)
		}
	}
	return candidate
}

// Dummy reference to suppress unused import error if go-imap changes.
var _ = imapv1.SeqSet{}
