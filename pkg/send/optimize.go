package send

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// ResponseProfile captures a contact's historical email response behaviour.
type ResponseProfile struct {
	// ContactEmail is the email address this profile describes.
	ContactEmail string

	// SampleCount is the number of message pairs analysed.
	SampleCount int

	// MedianResponseMinutes is the median time (in minutes) from send to reply.
	MedianResponseMinutes float64

	// HourWeights is a 24-element array. Index = hour of day (UTC).
	// Each entry is the probability the contact responds within 2h if sent at that hour.
	HourWeights [24]float64

	// DayWeights is a 7-element array. Index 0 = Sunday, 6 = Saturday.
	DayWeights [7]float64

	// BestSendHour is the hour of day (UTC) with the highest predicted response rate.
	BestSendHour int

	// BestSendDay is the weekday with the highest predicted response rate.
	BestSendDay time.Weekday

	// TypicalTZ is a rough UTC offset inferred from when they normally send/reply.
	TypicalTZOffset int
}

// SendTimeSuggestion is the output of SuggestSendTime.
type SendTimeSuggestion struct {
	// RecommendedAt is the suggested send timestamp (in the caller's local time).
	RecommendedAt time.Time

	// Reason is a human-readable explanation.
	Reason string

	// Confidence is 0.0–1.0; low when there's insufficient data.
	Confidence float64
}

// MessageRecord is a minimal sent/received message for analysis.
type MessageRecord struct {
	// SentAt is when the user sent a message to this contact.
	SentAt time.Time
	// RepliedAt is when the contact replied. Zero if no reply observed.
	RepliedAt time.Time
	// HasReply indicates whether a reply was received.
	HasReply bool
}

// BuildProfile analyses a slice of sent messages and their replies to build
// a ResponseProfile for the given contact.
func BuildProfile(contactEmail string, records []MessageRecord) *ResponseProfile {
	p := &ResponseProfile{ContactEmail: contactEmail}
	if len(records) < 3 {
		return p
	}

	var responseTimes []float64
	hourCounts := [24]int{}
	hourReplies := [24]int{}
	dayCounts := [7]int{}
	dayReplies := [7]int{}
	var tzOffsets []int

	for _, r := range records {
		if r.SentAt.IsZero() {
			continue
		}

		sendHour := r.SentAt.UTC().Hour()
		sendDay := int(r.SentAt.UTC().Weekday())
		hourCounts[sendHour]++
		dayCounts[sendDay]++

		if r.HasReply && !r.RepliedAt.IsZero() {
			minutes := r.RepliedAt.Sub(r.SentAt).Minutes()
			if minutes > 0 && minutes < 60*72 { // ignore > 3 days
				responseTimes = append(responseTimes, minutes)
				hourReplies[sendHour]++
				dayReplies[sendDay]++

				// Infer TZ: replies tend to cluster in 8am–6pm local time.
				// If they reply at UTC hour H, their local noon is ~12, so offset ≈ 12-H mod 12.
				replyHourUTC := r.RepliedAt.UTC().Hour()
				approxOffset := 12 - replyHourUTC
				if approxOffset < -12 {
					approxOffset += 24
				} else if approxOffset > 12 {
					approxOffset -= 24
				}
				tzOffsets = append(tzOffsets, approxOffset)
			}
		}
	}

	p.SampleCount = len(records)

	// Median response time
	if len(responseTimes) > 0 {
		sort.Float64s(responseTimes)
		p.MedianResponseMinutes = responseTimes[len(responseTimes)/2]
	}

	// Hour weights: P(reply within 2h | sent at this hour)
	bestHourWeight := 0.0
	for h := 0; h < 24; h++ {
		if hourCounts[h] > 0 {
			w := float64(hourReplies[h]) / float64(hourCounts[h])
			p.HourWeights[h] = w
			if w > bestHourWeight {
				bestHourWeight = w
				p.BestSendHour = h
			}
		}
	}

	// Day weights
	bestDayWeight := 0.0
	for d := 0; d < 7; d++ {
		if dayCounts[d] > 0 {
			w := float64(dayReplies[d]) / float64(dayCounts[d])
			p.DayWeights[d] = w
			if w > bestDayWeight {
				bestDayWeight = w
				p.BestSendDay = time.Weekday(d)
			}
		}
	}

	// Median TZ offset
	if len(tzOffsets) > 0 {
		sort.Ints(tzOffsets)
		p.TypicalTZOffset = tzOffsets[len(tzOffsets)/2]
	}

	return p
}

// SuggestSendTime recommends when to send a message to maximise the chance of
// a timely response, given the current time and the contact's ResponseProfile.
func SuggestSendTime(p *ResponseProfile, now time.Time) *SendTimeSuggestion {
	if p.SampleCount < 3 {
		return &SendTimeSuggestion{
			RecommendedAt: now,
			Reason:        "Not enough response history to make a recommendation. Sending now.",
			Confidence:    0.1,
		}
	}

	// Find the next occurrence of (BestSendDay, BestSendHour) in UTC from now.
	targetUTC := nextOccurrence(now.UTC(), p.BestSendDay, p.BestSendHour)
	localTarget := targetUTC.In(now.Location())

	// Confidence: based on how many samples we have and how clear the best window is.
	confidence := math.Min(float64(p.SampleCount)/20.0, 1.0) * 0.85

	// Build a human description of the contact's typical timezone.
	tzDesc := describeOffset(p.TypicalTZOffset)

	reason := fmt.Sprintf(
		"%s typically replies within %.0f min. Best response rate is on %ss around %s (%s local). "+
			"Sending at %s gives the highest chance of a timely reply.",
		p.ContactEmail,
		p.MedianResponseMinutes,
		p.BestSendDay.String(),
		fmt.Sprintf("%d:00 UTC", p.BestSendHour),
		tzDesc,
		localTarget.Format("Mon Jan 2 at 3:04 PM"),
	)

	// If the best time is less than 30 min away, just say "send now".
	if targetUTC.Sub(now.UTC()) < 30*time.Minute {
		return &SendTimeSuggestion{
			RecommendedAt: now,
			Reason:        "You're already in their optimal response window — send now.",
			Confidence:    confidence,
		}
	}

	return &SendTimeSuggestion{
		RecommendedAt: localTarget,
		Reason:        reason,
		Confidence:    confidence,
	}
}

// ProfileSummary returns a short human-readable description of the profile.
func ProfileSummary(p *ResponseProfile) string {
	if p.SampleCount < 3 {
		return fmt.Sprintf("Insufficient data (%d message(s) observed)", p.SampleCount)
	}
	return fmt.Sprintf(
		"Based on %d emails: typically replies in ~%.0f min · best day %s · best hour %d:00 UTC · approx %s",
		p.SampleCount,
		p.MedianResponseMinutes,
		p.BestSendDay.String(),
		p.BestSendHour,
		describeOffset(p.TypicalTZOffset),
	)
}

func nextOccurrence(from time.Time, day time.Weekday, hour int) time.Time {
	// Move to target hour today (UTC)
	candidate := time.Date(from.Year(), from.Month(), from.Day(), hour, 0, 0, 0, from.Location())

	// Advance days until we land on the right weekday
	for candidate.Before(from) || candidate.Weekday() != day {
		candidate = candidate.Add(24 * time.Hour)
		if candidate.Weekday() == day && candidate.After(from) {
			break
		}
	}
	return candidate
}

func describeOffset(offset int) string {
	if offset == 0 {
		return "UTC"
	}
	if offset > 0 {
		return fmt.Sprintf("UTC+%d", offset)
	}
	return fmt.Sprintf("UTC%d", offset)
}
