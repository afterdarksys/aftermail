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
	"fyne.io/fyne/v2/widget"
)

type Event struct {
	Title     string
	StartTime time.Time
	EndTime   time.Time
	Location  string
	Attendees []string
}

// buildCalendarTab creates the Calendar UI
func buildCalendarTab(w fyne.Window) fyne.CanvasObject {
	// A simple agenda view for the prototype
	events := []Event{
		{"Standup", time.Now().Add(time.Hour), time.Now().Add(time.Hour + 30*time.Minute), "Video Call", []string{"Ryan", "Brenda"}},
		{"Protocol Architecture Review", time.Now().Add(3 * time.Hour), time.Now().Add(4 * time.Hour), "Conference Room A", []string{"Engineering Team"}},
		{"Lunch with investors", time.Now().Add(24 * time.Hour), time.Now().Add(25 * time.Hour), "Downtown", []string{"VC Partners"}},
	}

	// Month/Week selector header
	dateLabel := widget.NewLabelWithStyle("Today: "+time.Now().Format("Monday, January 2"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	var agendaList *widget.List
	
	header := container.NewHBox(
		widget.NewButton("<", func() {}),
		widget.NewButton("Today", func() {}),
		widget.NewButton(">", func() {}),
		layout.NewSpacer(),
		dateLabel,
		layout.NewSpacer(),
		widget.NewButton("Import iCal", func() {
			importICal(w, &events, func() { agendaList.Refresh() })
		}),
		widget.NewButton("Export iCal", func() {
			exportICal(w, events)
		}),
		widget.NewSelect([]string{"Agenda", "Day", "Week", "Month"}, func(s string) {}),
		widget.NewButton("New Event", func() {
			newEvent := Event{
				Title:     "New Meeting",
				StartTime: time.Now().Add(time.Hour),
				EndTime:   time.Now().Add(2 * time.Hour),
				Location:  "TBD",
				Attendees: []string{},
			}
			events = append(events, newEvent)
			agendaList.Refresh()
		}),
	)

	// Agenda List
	agendaList = widget.NewList(
		func() int { return len(events) },
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

			e := events[i]
			timeRange := fmt.Sprintf("%s - %s", e.StartTime.Format("15:04"), e.EndTime.Format("15:04"))
			titleLabel.SetText(fmt.Sprintf("%s | %s", timeRange, e.Title))
			
			locText := e.Location
			if len(e.Attendees) > 0 {
				locText += fmt.Sprintf(" (%d attendees)", len(e.Attendees))
			}
			locLabel.SetText(locText)
		},
	)

	// Quick event creation sidebar (mock)
	miniCalendar := widget.NewLabel("S M T W T F S\n1 2 3 4 5 6 7\n8 9 10 11 12 13 14\n...")
	
	sidebar := container.NewVBox(
		widget.NewLabelWithStyle("Mini Calendar", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		miniCalendar,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("My Calendars", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewCheck("Personal", nil),
		widget.NewCheck("Work", nil),
		widget.NewCheck("Msgs.Global Ops", nil),
	)

	split := container.NewHSplit(sidebar, agendaList)
	split.SetOffset(0.2) // Sidebar takes 20%

	return container.NewBorder(header, nil, nil, nil, split)
}

func exportICal(w fyne.Window, events []Event) {
	dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		var sb strings.Builder
		sb.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//MeowMail//Client//EN\r\n")
		
		for _, e := range events {
			sb.WriteString("BEGIN:VEVENT\r\n")
			sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", e.Title))
			sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", e.StartTime.UTC().Format("20060102T150405Z")))
			sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", e.EndTime.UTC().Format("20060102T150405Z")))
			if e.Location != "" {
				sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", e.Location))
			}
			sb.WriteString("END:VEVENT\r\n")
		}
		
		sb.WriteString("END:VCALENDAR\r\n")
		
		uc.Write([]byte(sb.String()))
		dialog.ShowInformation("Export Successful", "Saved iCal to "+uc.URI().Name(), w)

	}, w)
}

func importICal(w fyne.Window, events *[]Event, refresh func()) {
	dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		data, err := io.ReadAll(uc)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		// Very basic iCal parser for demonstration purposes
		content := string(data)
		lines := strings.Split(content, "\n")
		
		var currentEvent *Event
		importCount := 0
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "BEGIN:VEVENT" {
				currentEvent = &Event{}
			} else if line == "END:VEVENT" && currentEvent != nil {
				*events = append(*events, *currentEvent)
				importCount++
				currentEvent = nil
			} else if currentEvent != nil {
				if strings.HasPrefix(line, "SUMMARY:") {
					currentEvent.Title = strings.TrimPrefix(line, "SUMMARY:")
				} else if strings.HasPrefix(line, "LOCATION:") {
					currentEvent.Location = strings.TrimPrefix(line, "LOCATION:")
				} else if strings.HasPrefix(line, "DTSTART:") {
					t, err := time.Parse("20060102T150405Z", strings.TrimPrefix(line, "DTSTART:"))
					if err == nil {
						currentEvent.StartTime = t
					}
				} else if strings.HasPrefix(line, "DTEND:") {
					t, err := time.Parse("20060102T150405Z", strings.TrimPrefix(line, "DTEND:"))
					if err == nil {
						currentEvent.EndTime = t
					}
				}
			}
		}

		refresh()
		dialog.ShowInformation("Import Successful", fmt.Sprintf("Imported %d events from %s", importCount, uc.URI().Name()), w)

	}, w)
}
