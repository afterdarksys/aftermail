package telemetry

import (
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/posthog/posthog-go"
)

// Tracker handles unified privacy-preserving analytics and crash reports
type Tracker struct {
	posthogClient posthog.Client
	sentryActive  bool
}

// Config manages upstream DSN and API keys
type Config struct {
	SentryDSN        string
	PostHogAPIKey    string
	PostHogEndpoint  string
	EnableTracking   bool
	AppVersion       string
}

// Init creates physical bounds linking Sentry and PostHog globally
func Init(cfg Config) (*Tracker, error) {
	t := &Tracker{
		sentryActive: false,
	}

	if !cfg.EnableTracking {
		log.Println("[Telemetry] Tracking disabled globally by user config. Telemetry offline.")
		return t, nil
	}

	// 1. Initialize Sentry (Crash Reporting)
	if cfg.SentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Release:          "aftermail@" + cfg.AppVersion,
			TracesSampleRate: 0.1, // Only grab 10% of standard performance paths
			// Disable auto-attaching user IP addresses for strict privacy
			SendDefaultPII:   false,
		})
		if err != nil {
			log.Printf("[Telemetry] Sentry initialization failed: %v", err)
		} else {
			t.sentryActive = true
			log.Println("[Telemetry] Sentry Crash Reporting armed.")
		}
	}

	// 2. Initialize PostHog (Product Analytics)
	if cfg.PostHogAPIKey != "" {
		endpoint := "https://app.posthog.com"
		if cfg.PostHogEndpoint != "" {
			endpoint = cfg.PostHogEndpoint
		}
		
		client, _ := posthog.NewWithConfig(
			cfg.PostHogAPIKey,
			posthog.Config{
				Endpoint: endpoint,
			},
		)
		t.posthogClient = client
		log.Println("[Telemetry] PostHog Privacy Analytics armed.")
	}

	return t, nil
}

// CaptureError explicitly records unhandled logic exceptions safely
func (t *Tracker) CaptureError(err error) {
	if t.sentryActive {
		sentry.CaptureException(err)
	}
}

// CaptureEvent sends generic non-PII events to PostHog
func (t *Tracker) CaptureEvent(distinctID, eventName string, properties map[string]interface{}) {
	if t.posthogClient != nil {
		t.posthogClient.Enqueue(posthog.Capture{
			DistinctId: distinctID,
			Event:      eventName,
			Properties: properties,
		})
	}
}

// Flush ensures all metrics sync upstream before app termination
func (t *Tracker) Flush(timeout time.Duration) {
	if t.sentryActive {
		sentry.Flush(timeout)
	}
	if t.posthogClient != nil {
		t.posthogClient.Close()
	}
}
