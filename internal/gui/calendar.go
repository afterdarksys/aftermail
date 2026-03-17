package gui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// CalendarTab maps calendar logic
type CalendarTab struct {
	db         *storage.DB
	events     []storage.CalendarEvent
	agendaList *widget.List
	window     fyne.Window
}

// buildCalendarTab creates the Calendar UI integrated with local DB
func buildCalendarTab(w fyne.Window, db *storage.DB) fyne.CanvasObject {
	tab := &CalendarTab{
		db:     db,
		window: w,
	}

	dateLabel := widget.NewLabelWithStyle("Agenda: "+time.Now().Format("Jan 02, 2006"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	header := container.NewHBox(
		widget.NewButton("<", func() {}),
		widget.NewButton("Today", func() {}),
		widget.NewButton(">", func() {}),
		layout.NewSpacer(),
		dateLabel,
		layout.NewSpacer(),
		widget.NewButton("Import iCal", func() {
			tab.importICal()
		}),
		widget.NewButton("Export iCal", func() {
			tab.exportICal()
		}),
		widget.NewButtonWithIcon("New Event", theme.ContentAddIcon(), func() {
			tab.showEventDialog(-1)
		}),
	)

	// Agenda List
	tab.agendaList = widget.NewList(
		func() int { return len(tab.events) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabelWithStyle("Time - Title", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Location"),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			box := o.(*fyne.Container)
			titleLabel := box.Objects[0].(*widget.Label)
			locLabel := box.Objects[1].(*widget.Label)

			e := tab.events[i]
			timeRange := fmt.Sprintf("%s - %s", e.StartTime.Format("Jan 02 15:04"), e.EndTime.Format("15:04"))
			titleLabel.SetText(fmt.Sprintf("%s | %s", timeRange, e.Title))
			
			locText := e.Location
			if locText == "" {
				locText = "No Location"
			}
			locLabel.SetText(locText)
		},
	)

	tab.agendaList.OnSelected = func(id widget.ListItemID) {
		tab.showEventDialog(tab.events[id].ID)
		tab.agendaList.UnselectAll()
	}

	miniCalendar := widget.NewLabel("S M T W T F S\n1 2 3 4 5 6 7\n8 9 10 11 12 13 14\n...")
	
	sidebar := container.NewVBox(
		widget.NewLabelWithStyle("Mini Calendar", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		miniCalendar,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("My Calendars", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewCheck("Personal", nil),
		widget.NewCheck("Work", nil),
		widget.NewCheck("Web3 / Mailblocks", nil),
	)

	split := container.NewHSplit(sidebar, tab.agendaList)
	split.SetOffset(0.2) 

	tab.Reload()
	return container.NewBorder(header, nil, nil, nil, split)
}

func (t *CalendarTab) Reload() {
	if t.db == nil {
		return
	}
	events, err := t.db.ListEvents()
	if err == nil {
		t.events = events
		t.agendaList.Refresh()
	}
}

func (t *CalendarTab) showEventDialog(id int64) {
	var e storage.CalendarEvent
	isNew := true

	if id > 0 {
		for _, ev := range t.events {
			if ev.ID == id {
				e = ev
				isNew = false
				break
			}
		}
	} else {
		e.Title = "New Meeting"
		e.StartTime = time.Now().Add(time.Hour)
		e.EndTime = time.Now().Add(2 * time.Hour)
	}

	titleEntry := widget.NewEntry()
	titleEntry.SetText(e.Title)
	locEntry := widget.NewEntry()
	locEntry.SetText(e.Location)
	descEntry := widget.NewMultiLineEntry()
	descEntry.SetText(e.Description)
	
	startEntry := widget.NewEntry()
	startEntry.SetText(e.StartTime.Format("2006-01-02 15:04"))
	endEntry := widget.NewEntry()
	endEntry.SetText(e.EndTime.Format("2006-01-02 15:04"))

	items := []*widget.FormItem{
		widget.NewFormItem("Title", titleEntry),
		widget.NewFormItem("Location", locEntry),
		widget.NewFormItem("Start Time", startEntry),
		widget.NewFormItem("End Time", endEntry),
		widget.NewFormItem("Description", descEntry),
	}

	var d dialog.Dialog
	saveFunc := func(saved bool) {
		if !saved {
			return
		}

		st, err := time.Parse("2006-01-02 15:04", startEntry.Text)
		if err == nil { e.StartTime = st }
		en, err := time.Parse("2006-01-02 15:04", endEntry.Text)
		if err == nil { e.EndTime = en }

		e.Title = titleEntry.Text
		e.Location = locEntry.Text
		e.Description = descEntry.Text

		if isNew {
			_, _ = t.db.AddEvent(&e)
		} else {
			_ = t.db.UpdateEvent(&e)
		}
		t.Reload()
	}

	d = dialog.NewForm("Event Details", "Save", "Cancel", items, saveFunc, t.window)
	d.Resize(fyne.NewSize(400, 300))
	d.Show()
}

func (t *CalendarTab) exportICal() {
	dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		var sb strings.Builder
		sb.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//MeowMail//Client//EN\r\n")
		
		for _, e := range t.events {
			sb.WriteString("BEGIN:VEVENT\r\n")
			sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", e.Title))
			sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", e.StartTime.UTC().Format("20060102T150405Z")))
			sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", e.EndTime.UTC().Format("20060102T150405Z")))
			if e.Location != "" {
				sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", e.Location))
			}
			if e.Description != "" {
				sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", strings.ReplaceAll(e.Description, "\n", "\\n")))
			}
			sb.WriteString("END:VEVENT\r\n")
		}
		
		sb.WriteString("END:VCALENDAR\r\n")
		uc.Write([]byte(sb.String()))
		dialog.ShowInformation("Export Successful", "Saved iCal to "+uc.URI().Name(), t.window)
	}, t.window)
}

func (t *CalendarTab) importICal() {
	dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		data, err := io.ReadAll(uc)
		if err != nil {
			dialog.ShowError(err, t.window)
			return
		}

		content := string(data)
		lines := strings.Split(content, "\n")
		
		var currentEvent *storage.CalendarEvent
		importCount := 0
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "BEGIN:VEVENT" {
				currentEvent = &storage.CalendarEvent{}
			} else if line == "END:VEVENT" && currentEvent != nil {
				t.db.AddEvent(currentEvent)
				importCount++
				currentEvent = nil
			} else if currentEvent != nil {
				if strings.HasPrefix(line, "SUMMARY:") {
					currentEvent.Title = strings.TrimPrefix(line, "SUMMARY:")
				} else if strings.HasPrefix(line, "LOCATION:") {
					currentEvent.Location = strings.TrimPrefix(line, "LOCATION:")
				} else if strings.HasPrefix(line, "DTSTART:") {
					tm, err := time.Parse("20060102T150405Z", strings.TrimPrefix(line, "DTSTART:"))
					if err == nil {
						currentEvent.StartTime = tm
					}
				} else if strings.HasPrefix(line, "DTEND:") {
					tm, err := time.Parse("20060102T150405Z", strings.TrimPrefix(line, "DTEND:"))
					if err == nil {
						currentEvent.EndTime = tm
					}
				}
			}
		}

		t.Reload()
		dialog.ShowInformation("Import Successful", fmt.Sprintf("Imported %d events from %s", importCount, uc.URI().Name()), t.window)
	}, t.window)
}
