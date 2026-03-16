package gui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Task struct {
	ID          int
	Title       string
	Description string
	DueDate     time.Time
	IsCompleted bool
}

// buildTasksTab creates the Tasks UI
func buildTasksTab() fyne.CanvasObject {
	tasks := []Task{
		{1, "Review AfterSMTP Security Audit", "Check the latest report for any regressions.", time.Now().Add(24 * time.Hour), false},
		{2, "Draft Mailblocks Whitepaper", "Include section on proof of stake spam prevention.", time.Now().Add(48 * time.Hour), false},
		{3, "Update DNS records", "Update SPF and DKIM for the new msgs.global servers.", time.Now().Add(-2 * time.Hour), true},
	}

	var list *widget.List
	var selectedIndex int = -1

	titleEntry := widget.NewEntry()
	descEntry := widget.NewMultiLineEntry()
	dueDateEntry := widget.NewEntry() // Simple string for now

	list = widget.NewList(
		func() int { return len(tasks) },
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

			t := &tasks[i]
			check.SetChecked(t.IsCompleted)
			check.OnChanged = func(checked bool) {
				tasks[i].IsCompleted = checked
				list.Refresh()
			}

			// Strike out if completed or add a tag if overdue
			text := t.Title
			if t.IsCompleted {
				text = "✓ " + text
			} else if t.DueDate.Before(time.Now()) {
				text = "[OVERDUE] " + text
			}
			label.SetText(text)
		},
	)

	detailsForm := widget.NewForm(
		widget.NewFormItem("Title", titleEntry),
		widget.NewFormItem("Due Date", dueDateEntry),
	)

	saveBtn := widget.NewButton("Save Task", func() {
		dt, err := time.Parse("2006-01-02 15:04", dueDateEntry.Text)
		if err != nil {
			dt = time.Now()
		}

		task := Task{
			ID:          len(tasks) + 1,
			Title:       titleEntry.Text,
			Description: descEntry.Text,
			DueDate:     dt,
			IsCompleted: false, // Default to incomplete on edit for simplicity
		}

		if selectedIndex >= 0 {
			task.ID = tasks[selectedIndex].ID
			task.IsCompleted = tasks[selectedIndex].IsCompleted
			tasks[selectedIndex] = task
		} else {
			tasks = append(tasks, task)
			selectedIndex = len(tasks) - 1
		}
		list.Refresh()
		list.Select(widget.ListItemID(selectedIndex))
	})

	rightPane := container.NewBorder(
		widget.NewLabelWithStyle("Task Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		saveBtn,
		nil, nil,
		container.NewVScroll(container.NewVBox(detailsForm, widget.NewLabel("Description:"), descEntry)),
	)

	list.OnSelected = func(id widget.ListItemID) {
		selectedIndex = int(id)
		selected := tasks[id]
		titleEntry.SetText(selected.Title)
		descEntry.SetText(selected.Description)
		dueDateEntry.SetText(selected.DueDate.Format("2006-01-02 15:04"))
	}

	if len(tasks) > 0 {
		list.Select(0)
	}

	header := container.NewHBox(
		widget.NewButton("Add Task", func() {
			selectedIndex = -1
			titleEntry.SetText("")
			descEntry.SetText("")
			dueDateEntry.SetText(time.Now().Format("2006-01-02 15:04"))
			titleEntry.SetPlaceHolder("New Task Title")
			list.UnselectAll()
		}),
		widget.NewCheck("Hide Completed", func(checked bool) {
			// Filtering logic would apply to the list data model
		}),
	)

	leftPane := container.NewBorder(header, nil, nil, nil, list)

	split := container.NewHSplit(leftPane, rightPane)
	split.SetOffset(0.4)

	return split
}
