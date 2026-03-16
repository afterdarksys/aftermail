package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func buildFoldersTab() fyne.CanvasObject {
	// List of folders fetched from meowmaild loopback API
	folderList := widget.NewList(
		func() int { return 4 }, // Mock default length
		func() fyne.CanvasObject {
			return widget.NewLabel("Folder Name")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			names := []string{"Inbox", "Sent", "Trash", "Important"}
			o.(*widget.Label).SetText(names[i])
		},
	)

	// List of emails in the selected folder
	emailList := widget.NewList(
		func() int { return 1 },
		func() fyne.CanvasObject {
			return widget.NewLabel("Sender - Subject")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText("boss@company.com - Important update (IMAP)")
		},
	)

	messagePreview := widget.NewMultiLineEntry()
	messagePreview.Disable()
	messagePreview.SetText("Please read this...\n\n(Synced via meowmaild)")

	emailList.OnSelected = func(id widget.ListItemID) {
		messagePreview.SetText(fmt.Sprintf("Previewing Message %d...\n\nContent goes here.", id))
	}

	refreshBtn := widget.NewButton("Sync with meowmaild", func() {
		// Fetch from local HTTP loopback API running on :4460
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://127.0.0.1:4460/api/folders")
		if err == nil {
			defer resp.Body.Close()
			var data map[string][]string
			if json.NewDecoder(resp.Body).Decode(&data) == nil {
				messagePreview.SetText("Successfully communicated with meowmaild background service.")
			}
		} else {
			messagePreview.SetText(fmt.Sprintf("Failed to reach meowmaild HTTP loopback: %v\nMake sure meowmaild is running.", err))
		}
	})

	leftPanel := container.NewVBox(
		widget.NewLabelWithStyle("Mailboxes", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		refreshBtn,
		container.NewMax(folderList), // Need a better sizing container in real app
	)

	rightPanel := container.NewVSplit(
		emailList,
		messagePreview,
	)

	return container.NewHSplit(leftPanel, rightPanel)
}
