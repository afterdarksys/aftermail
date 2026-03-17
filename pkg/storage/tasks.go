package storage

import (
	"time"
)

// Task represents a to-do item
type Task struct {
	ID          int64
	Title       string
	Description string
	DueDate     time.Time
	IsCompleted bool
	CreatedAt   time.Time
}

// AddTask inserts a new task
func (db *DB) AddTask(t *Task) (int64, error) {
	query := `INSERT INTO tasks (title, description, due_date, completed, created_at) VALUES (?, ?, ?, ?, ?)`
	now := time.Now()
	res, err := db.conn.Exec(query, t.Title, t.Description, t.DueDate, t.IsCompleted, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateTask modifies an existing task
func (db *DB) UpdateTask(t *Task) error {
	query := `UPDATE tasks SET title = ?, description = ?, due_date = ?, completed = ? WHERE id = ?`
	_, err := db.conn.Exec(query, t.Title, t.Description, t.DueDate, t.IsCompleted, t.ID)
	return err
}

// DeleteTask removes a task
func (db *DB) DeleteTask(id int64) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// ListTasks returns all tasks, optionally filtering out completed ones
func (db *DB) ListTasks(hideCompleted bool) ([]Task, error) {
	query := `SELECT id, title, description, due_date, completed, created_at FROM tasks`
	if hideCompleted {
		query += ` WHERE completed = 0`
	}
	query += ` ORDER BY due_date ASC, created_at DESC`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var due, created *time.Time
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &due, &t.IsCompleted, &created); err != nil {
			return nil, err
		}
		if due != nil {
			t.DueDate = *due
		}
		if created != nil {
			t.CreatedAt = *created
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
