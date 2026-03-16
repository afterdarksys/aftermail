package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Reminder struct {
	ID        int
	Text      string
	TriggerAt time.Time
	Snoozed   bool
}

// buildRemindersTab creates the Reminders UI
func buildRemindersTab() fyne.CanvasObject {
	reminders := []Reminder{
		{1, "Join Web3 Email Consortium Call", time.Now().Add(15 * time.Minute), false},
		{2, "Renew AfterSMTP TLS Certificates", time.Now().Add(7 * 24 * time.Hour), false},
		{3, "Follow up with Brenda regarding spam rules", time.Now().Add(-1 * time.Hour), true}, // Snoozed/Missed
	}

	var list *widget.List

	list = widget.NewList(
		func() int { return len(reminders) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewButton("Snooze", nil),
				widget.NewButton("Done", nil),
				widget.NewLabel("Reminder Text Template"),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			box := o.(*fyne.Container)
			snoozeBtn := box.Objects[0].(*widget.Button)
			doneBtn := box.Objects[1].(*widget.Button)
			label := box.Objects[2].(*widget.Label)

			r := &reminders[i]
			
			snoozeBtn.OnTapped = func() {
				// Snooze logic (push forward 1 hour)
				reminders[i].Snoozed = true
				reminders[i].TriggerAt = time.Now().Add(1 * time.Hour)
				list.Refresh()
			}
			
			doneBtn.OnTapped = func() {
				// Dismiss logic (remove from slice)
				reminders = append(reminders[:i], reminders[i+1:]...)
				list.Refresh()
			}

			timeStr := r.TriggerAt.Format("15:04 (Jan 2)")
			text := fmt.Sprintf("[%s] %s", timeStr, r.Text)
			
			if r.Snoozed {
				text = "[SNOOZED] " + text
			} else if time.Now().After(r.TriggerAt) {
				text = "⚠️ " + text
			}
			
			label.SetText(text)
		},
	)

	addEntry := widget.NewEntry()
	addEntry.SetPlaceHolder("Type a reminder and hit enter... (defaults to 10 mins from now)")
	
	addBtn := widget.NewButton("Add Reminder", func() {
		if addEntry.Text != "" {
			newReminder := Reminder{
				ID:        len(reminders) + 1,
				Text:      addEntry.Text,
				TriggerAt: time.Now().Add(10 * time.Minute),
				Snoozed:   false,
			}
			reminders = append(reminders, newReminder)
			addEntry.SetText("")
			list.Refresh()
		}
	})

	topBar := container.NewBorder(nil, nil, nil, addBtn, addEntry)

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Upcoming Reminders", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			topBar,
		),
		nil, nil, nil,
		list,
	)
}
