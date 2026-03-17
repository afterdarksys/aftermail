package gui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// TasksTab holds the state for the Tasks UI
type TasksTab struct {
	db       *storage.DB
	tasks    []storage.Task
	list     *widget.List
	selected int64
	
	titleEntry   *widget.Entry
	descEntry    *widget.Entry
	dueDateEntry *widget.Entry
	hideDone     bool
}

// buildTasksTab creates the Tasks UI integrated with SQLite
func buildTasksTab(db *storage.DB) fyne.CanvasObject {
	t := &TasksTab{
		db:       db,
		hideDone: false,
		selected: -1,
	}

	t.titleEntry = widget.NewEntry()
	t.descEntry = widget.NewMultiLineEntry()
	t.dueDateEntry = widget.NewEntry()

	t.list = widget.NewList(
		func() int { return len(t.tasks) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewCheck("", nil),
				widget.NewLabel("Task Title Template"),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			box := o.(*fyne.Container)
			check := box.Objects[0].(*widget.Check)
			label := box.Objects[1].(*widget.Label)

			task := t.tasks[i]
			check.SetChecked(task.IsCompleted)
			
			// Handle completion toggle
			check.OnChanged = func(checked bool) {
				task.IsCompleted = checked
				if t.db != nil {
					_ = t.db.UpdateTask(&task)
				}
				t.Reload()
			}

			// Format title
			text := task.Title
			if task.IsCompleted {
				text = "✓ " + text
			} else if !task.DueDate.IsZero() && task.DueDate.Before(time.Now()) {
				text = "[OVERDUE] " + text
			}
			label.SetText(text)
		},
	)

	t.list.OnSelected = func(id widget.ListItemID) {
		t.selected = t.tasks[id].ID
		selected := t.tasks[id]
		t.titleEntry.SetText(selected.Title)
		t.descEntry.SetText(selected.Description)
		if !selected.DueDate.IsZero() {
			t.dueDateEntry.SetText(selected.DueDate.Format("2006-01-02 15:04"))
		} else {
			t.dueDateEntry.SetText("")
		}
	}

	detailsForm := widget.NewForm(
		widget.NewFormItem("Title", t.titleEntry),
		widget.NewFormItem("Due Date", t.dueDateEntry),
	)

	saveBtn := widget.NewButtonWithIcon("Save Task", theme.DocumentSaveIcon(), func() {
		var dt time.Time
		if t.dueDateEntry.Text != "" {
			parsed, err := time.Parse("2006-01-02 15:04", t.dueDateEntry.Text)
			if err == nil {
				dt = parsed
			}
		}

		task := storage.Task{
			ID:          t.selected,
			Title:       t.titleEntry.Text,
			Description: t.descEntry.Text,
			DueDate:     dt,
			IsCompleted: false,
		}

		if t.selected > 0 {
			// Find existing completion state
			for _, existing := range t.tasks {
				if existing.ID == t.selected {
					task.IsCompleted = existing.IsCompleted
					break
				}
			}
			_ = t.db.UpdateTask(&task)
		} else {
			id, _ := t.db.AddTask(&task)
			t.selected = id
		}
		t.Reload()
	})

	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		if t.selected > 0 {
			_ = t.db.DeleteTask(t.selected)
			t.clearForm()
			t.Reload()
		}
	})

	actionsRow := container.NewHBox(saveBtn, deleteBtn)

	rightPane := container.NewBorder(
		widget.NewLabelWithStyle("Task Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		actionsRow,
		nil, nil,
		container.NewVScroll(container.NewVBox(detailsForm, widget.NewLabel("Description:"), t.descEntry)),
	)

	hideDoneCheck := widget.NewCheck("Hide Completed", func(checked bool) {
		t.hideDone = checked
		t.Reload()
	})
	hideDoneCheck.SetChecked(t.hideDone)

	header := container.NewHBox(
		widget.NewButtonWithIcon("Add Task", theme.ContentAddIcon(), func() {
			t.clearForm()
			t.list.UnselectAll()
		}),
		hideDoneCheck,
	)

	leftPane := container.NewBorder(header, nil, nil, nil, t.list)

	split := container.NewHSplit(leftPane, rightPane)
	split.SetOffset(0.4)

	t.Reload()
	return split
}

func (t *TasksTab) clearForm() {
	t.selected = -1
	t.titleEntry.SetText("")
	t.descEntry.SetText("")
	t.dueDateEntry.SetText(time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04"))
	t.titleEntry.SetPlaceHolder("New Task Title")
}

func (t *TasksTab) Reload() {
	if t.db == nil {
		return
	}
	tasks, err := t.db.ListTasks(t.hideDone)
	if err == nil {
		t.tasks = tasks
		t.list.Refresh()
	}
}
