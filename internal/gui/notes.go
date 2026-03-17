package gui

import (


	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// NotesTab manages the UI for Markdown rich-text notes
type NotesTab struct {
	db          *storage.DB
	notes       []storage.Note
	list        *widget.List
	editorBox   *fyne.Container
	selectedID  int64
	window      fyne.Window
	
	titleEntry   *widget.Entry
	contentEntry *widget.Entry
	previewArea  *container.Scroll
	isPreview    bool
}

// buildNotesTab constructs the Fyne UI for managing Notes
func buildNotesTab(window fyne.Window, db *storage.DB) fyne.CanvasObject {
	tab := &NotesTab{
		db:        db,
		window:    window,
		isPreview: false,
	}

	tab.list = widget.NewList(
		func() int { return len(tab.notes) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.DocumentIcon()),
				widget.NewLabel("Note title..."),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			n := tab.notes[i]
			box := o.(*fyne.Container)
			label := box.Objects[1].(*widget.Label)
			label.SetText(n.Title)
		},
	)

	tab.list.OnSelected = func(id widget.ListItemID) {
		tab.selectedID = tab.notes[id].ID
		tab.loadNoteDetails(tab.notes[id])
	}

	tab.editorBox = container.NewVBox(
		widget.NewLabelWithStyle("Select a note or create a new one.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
	)

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			tab.createNewNote()
		}),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			if tab.selectedID > 0 {
				tab.deleteSelectedNote()
			}
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			tab.Reload()
		}),
	)

	leftPanel := container.NewBorder(toolbar, nil, nil, nil, tab.list)
	
	split := container.NewHSplit(leftPanel, container.NewBorder(nil, nil, nil, nil, tab.editorBox))
	split.Offset = 0.3

	// Initial load
	tab.Reload()

	return split
}

// Reload pulls fresh notes from the database
func (t *NotesTab) Reload() {
	if t.db == nil {
		return
	}
	notes, err := t.db.ListNotes()
	if err == nil {
		t.notes = notes
		t.list.Refresh()
		t.selectedID = 0
		t.editorBox.Objects = []fyne.CanvasObject{
			widget.NewLabelWithStyle("Select a note or create a new one.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		}
		t.editorBox.Refresh()
	}
}

// createNewNote initializes a blank note in the editor view
func (t *NotesTab) createNewNote() {
	t.selectedID = -1 // Indicates new unsaved note
	emptyNote := storage.Note{Title: "Untitled Note", Content: ""}
	t.loadNoteDetails(emptyNote)
}

// deleteSelectedNote removes the actively highlighted note
func (t *NotesTab) deleteSelectedNote() {
	dialog.ShowConfirm("Delete Note", "Are you sure you want to permanently delete this note?", func(b bool) {
		if b {
			err := t.db.DeleteNote(t.selectedID)
			if err != nil {
				dialog.ShowError(err, t.window)
				return
			}
			t.Reload()
		}
	}, t.window)
}

// loadNoteDetails prepares the middle editor panel for reading/writing markdown
func (t *NotesTab) loadNoteDetails(n storage.Note) {
	t.titleEntry = widget.NewEntry()
	t.titleEntry.SetText(n.Title)
	
	t.contentEntry = widget.NewMultiLineEntry()
	t.contentEntry.SetText(n.Content)
	t.contentEntry.Wrapping = fyne.TextWrapWord
	t.contentEntry.SetMinRowsVisible(15)

	t.previewArea = container.NewScroll(widget.NewRichTextFromMarkdown(n.Content))
	t.previewArea.Hide()
	
	editorContainer := container.NewMax(t.contentEntry, t.previewArea)

	saveBtn := widget.NewButtonWithIcon("Save Note", theme.DocumentSaveIcon(), func() {
		if t.selectedID == -1 {
			// Create new
			newNote := &storage.Note{
				Title:   t.titleEntry.Text,
				Content: t.contentEntry.Text,
			}
			id, err := t.db.AddNote(newNote)
			if err != nil {
				dialog.ShowError(err, t.window)
				return
			}
			t.selectedID = id
		} else {
			// Update existing
			err := t.db.UpdateNote(t.selectedID, t.titleEntry.Text, t.contentEntry.Text)
			if err != nil {
				dialog.ShowError(err, t.window)
				return
			}
		}
		t.Reload()
	})

	togglePreviewBtn := widget.NewButtonWithIcon("Toggle Markdown Preview", theme.FileTextIcon(), func() {
		t.isPreview = !t.isPreview
		if t.isPreview {
			// Update preview content before showing
			t.previewArea.Content = widget.NewRichTextFromMarkdown(t.contentEntry.Text)
			t.previewArea.Refresh()
			t.contentEntry.Hide()
			t.previewArea.Show()
		} else {
			t.previewArea.Hide()
			t.contentEntry.Show()
		}
	})

	actionsRow := container.NewHBox(saveBtn, togglePreviewBtn)

	// Build layout
	t.editorBox.Objects = []fyne.CanvasObject{
		container.NewBorder(
			container.NewVBox(
				widget.NewLabelWithStyle("Note Title", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				t.titleEntry,
				widget.NewSeparator(),
			),
			actionsRow,
			nil, nil,
			editorContainer,
		),
	}
	t.editorBox.Refresh()
}
