package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Contact struct {
	Name      string
	Email     string
	DID       string // For AfterSMTP native DID
	Phone     string
	Company   string
	IsStarred bool
}

// buildContactsTab creates the Address Book UI
func buildContactsTab() fyne.CanvasObject {
	// Dummy data for now
	contacts := []Contact{
		{"Alice Smith", "alice@example.com", "did:aftersmtp:msgs.global:alice", "555-0100", "Acme Corp", true},
		{"Bob Jones", "bob@example.com", "", "555-0101", "Widgets Inc", false},
		{"Charlie Brown", "charlie@example.com", "did:aftersmtp:msgs.global:cbrown", "555-0102", "Peanuts LLC", false},
	}

	var list *widget.List
	var selectedIndex int = -1

	// List to display contacts
	list = widget.NewList(
		func() int {
			return len(contacts)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Name For Contact") // Template
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(contacts[i].Name)
		},
	)

	// Details pane
	nameEntry := widget.NewEntry()
	emailEntry := widget.NewEntry()
	didEntry := widget.NewEntry()
	phoneEntry := widget.NewEntry()
	companyEntry := widget.NewEntry()

	detailsForm := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Email", emailEntry),
		widget.NewFormItem("AfterSMTP DID", didEntry),
		widget.NewFormItem("Phone", phoneEntry),
		widget.NewFormItem("Company", companyEntry),
	)

	list.OnSelected = func(id widget.ListItemID) {
		selectedIndex = int(id)
		selected := contacts[id]
		nameEntry.SetText(selected.Name)
		emailEntry.SetText(selected.Email)
		didEntry.SetText(selected.DID)
		phoneEntry.SetText(selected.Phone)
		companyEntry.SetText(selected.Company)
	}

	if len(contacts) > 0 {
		list.Select(0)
	}

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search contacts...")

	filterToolbar := container.NewHBox(
		searchEntry,
		widget.NewButton("Add Contact", func() {
			selectedIndex = -1
			nameEntry.SetText("")
			emailEntry.SetText("")
			didEntry.SetText("")
			phoneEntry.SetText("")
			companyEntry.SetText("")
			nameEntry.SetPlaceHolder("New Contact Name")
			list.UnselectAll()
		}),
	)

	leftPane := container.NewBorder(filterToolbar, nil, nil, nil, list)
	
	saveBtn := widget.NewButton("Save Changes", func() {
		contact := Contact{
			Name:    nameEntry.Text,
			Email:   emailEntry.Text,
			DID:     didEntry.Text,
			Phone:   phoneEntry.Text,
			Company: companyEntry.Text,
		}

		if selectedIndex >= 0 {
			contacts[selectedIndex] = contact
		} else {
			contacts = append(contacts, contact)
			selectedIndex = len(contacts) - 1
		}
		list.Refresh()
		list.Select(widget.ListItemID(selectedIndex))
	})
	
	rightPane := container.NewBorder(
		widget.NewLabelWithStyle("Contact Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		saveBtn,
		nil, nil,
		detailsForm,
	)

	split := container.NewHSplit(leftPane, rightPane)
	split.SetOffset(0.3) // Left side gets 30% of width

	return split
}
