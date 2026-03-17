package storage

import (
	"time"
)

// CalendarEvent represents a scheduled calendar event
type CalendarEvent struct {
	ID          int64
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
	CreatedAt   time.Time
}

// AddEvent inserts a new calendar event
func (db *DB) AddEvent(e *CalendarEvent) (int64, error) {
	query := `INSERT INTO calendar_events (title, description, start_time, end_time, location, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now()
	res, err := db.conn.Exec(query, e.Title, e.Description, e.StartTime, e.EndTime, e.Location, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateEvent modifies an existing event
func (db *DB) UpdateEvent(e *CalendarEvent) error {
	query := `UPDATE calendar_events SET title = ?, description = ?, start_time = ?, end_time = ?, location = ? WHERE id = ?`
	_, err := db.conn.Exec(query, e.Title, e.Description, e.StartTime, e.EndTime, e.Location, e.ID)
	return err
}

// DeleteEvent removes an event
func (db *DB) DeleteEvent(id int64) error {
	query := `DELETE FROM calendar_events WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// ListEvents retrieves calendar events ordered by start time
func (db *DB) ListEvents() ([]CalendarEvent, error) {
	query := `SELECT id, title, description, start_time, end_time, location, created_at FROM calendar_events ORDER BY start_time ASC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		var start, end, created *time.Time
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &start, &end, &e.Location, &created); err != nil {
			return nil, err
		}
		if start != nil {
			e.StartTime = *start
		}
		if end != nil {
			e.EndTime = *end
		}
		if created != nil {
			e.CreatedAt = *created
		}
		events = append(events, e)
	}
	return events, nil
}
