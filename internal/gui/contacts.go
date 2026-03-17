package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// ContactsTab manages the UI for the address book
type ContactsTab struct {
	db           *storage.DB
	contacts     []storage.Contact
	list         *widget.List
	detailsPanel *fyne.Container
	selectedID   int64
	window       fyne.Window
}

// buildContactsTab constructs the Fyne UI for managing contacts
func buildContactsTab(window fyne.Window, db *storage.DB) fyne.CanvasObject {
	tab := &ContactsTab{
		db:     db,
		window: window,
	}

	tab.list = widget.NewList(
		func() int { return len(tab.contacts) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.AccountIcon()),
				widget.NewLabel("Name placeholder..."),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			c := tab.contacts[i]
			box := o.(*fyne.Container)
			label := box.Objects[1].(*widget.Label)
			label.SetText(fmt.Sprintf("%s (%s)", c.Name, c.Email))
		},
	)

	tab.list.OnSelected = func(id widget.ListItemID) {
		tab.selectedID = tab.contacts[id].ID
		tab.refreshDetailsBox(tab.contacts[id])
	}

	tab.detailsPanel = container.NewVBox(
		widget.NewLabelWithStyle("Select a contact to view details.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
	)

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			tab.showAddContactDialog()
		}),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			if tab.selectedID > 0 {
				tab.deleteSelectedContact()
			}
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			tab.Reload()
		}),
	)

	leftPanel := container.NewBorder(toolbar, nil, nil, nil, tab.list)
	
	split := container.NewHSplit(leftPanel, container.NewScroll(tab.detailsPanel))
	split.Offset = 0.3

	// Initial load
	tab.Reload()

	return split
}

// Reload pulls fresh contacts from the SQLite database
func (t *ContactsTab) Reload() {
	if t.db == nil {
		return
	}
	contacts, err := t.db.ListContacts("")
	if err == nil {
		t.contacts = contacts
		t.list.Refresh()
		t.selectedID = 0
		t.detailsPanel.Objects = []fyne.CanvasObject{
			widget.NewLabelWithStyle("Select a contact to view details.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		}
		t.detailsPanel.Refresh()
	}
}

// showAddContactDialog raises a form to insert a new address book entry
func (t *ContactsTab) showAddContactDialog() {
	nameEntry := AccessibleEntry("Full Name", "Contact's full name")
	emailEntry := AccessibleEntry("Email Address", "Contact's primary email")
	publicKeyEntry := AccessibleEntry("Public Key (Optional Web3)", "X25519/Ed25519 Key for AfterSMTP")
	groupEntry := AccessibleEntry("Group Tag", "E.g., Family, Work, Crypto")
	notesEntry := widget.NewMultiLineEntry()

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Email", emailEntry),
		widget.NewFormItem("Public Key", publicKeyEntry),
		widget.NewFormItem("Group", groupEntry),
		widget.NewFormItem("Notes", notesEntry),
	}

	dialog.ShowForm("Add Contact", "Save", "Cancel", items, func(saved bool) {
		if !saved {
			return
		}
		if nameEntry.Text == "" || emailEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("name and email are required fields"), t.window)
			return
		}

		newContact := &storage.Contact{
			Name:      nameEntry.Text,
			Email:     emailEntry.Text,
			PublicKey: publicKeyEntry.Text,
			GroupTag:  groupEntry.Text,
			Notes:     notesEntry.Text,
		}

		_, err := t.db.AddContact(newContact)
		if err != nil {
			dialog.ShowError(err, t.window)
			return
		}
		t.Reload()
	}, t.window)
}

// deleteSelectedContact removes the actively highlighted contact
func (t *ContactsTab) deleteSelectedContact() {
	dialog.ShowConfirm("Delete Contact", "Are you sure you want to permanently delete this contact?", func(b bool) {
		if b {
			err := t.db.DeleteContact(t.selectedID)
			if err != nil {
				dialog.ShowError(err, t.window)
				return
			}
			t.Reload()
		}
	}, t.window)
}

// refreshDetailsBox updates the right-hand panel with the selected contact's meta
func (t *ContactsTab) refreshDetailsBox(c storage.Contact) {
	nameTitle := widget.NewLabelWithStyle(c.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	emailLabel := widget.NewLabel(fmt.Sprintf("Email: %s", c.Email))
	
	pubKeyDisplay := "None"
	if c.PublicKey != "" {
		pubKeyDisplay = c.PublicKey
	}
	pubKeyLabel := widget.NewLabel(fmt.Sprintf("Public Key: %s", pubKeyDisplay))
	
	groupDisplay := "Ungrouped"
	if c.GroupTag != "" {
		groupDisplay = c.GroupTag
	}
	groupLabel := widget.NewLabel(fmt.Sprintf("Group: %s", groupDisplay))
	
	notesCard := widget.NewCard("Notes", "", widget.NewLabel(c.Notes))

	// Bind composer hook
	composeBtn := widget.NewButtonWithIcon("Compose Message", theme.MailSendIcon(), func() {
		if composerToEntry != nil {
			composerToEntry.SetText(c.Email)
			if globalTabs != nil && composerTabItem != nil {
				globalTabs.Select(composerTabItem)
			}
		}
	})

	t.detailsPanel.Objects = []fyne.CanvasObject{
		nameTitle,
		widget.NewSeparator(),
		emailLabel,
		pubKeyLabel,
		groupLabel,
		widget.NewSeparator(),
		notesCard,
		widget.NewSeparator(),
		composeBtn,
	}
	t.detailsPanel.Refresh()
}
