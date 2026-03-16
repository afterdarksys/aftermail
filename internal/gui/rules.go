package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// buildRulesTab creates the Starlark Rules Studio UI
func buildRulesTab() fyne.CanvasObject {
	// A simple IDE-like interface for Starlark
	ruleEditor := widget.NewMultiLineEntry()
	ruleEditor.SetPlaceHolder("def evaluate():\n    # Write AfterSMTP MailScript (Starlark) here...\n    accept()\n")
	ruleEditor.Resize(fyne.NewSize(600, 400))

	saveBtn := widget.NewButton("Save Rule", func() {
		// In a real app, this would save to SQLite or send to meowmaild
		ruleEditor.SetText(ruleEditor.Text + "\n# Rule saved locally!")
	})

	syntaxHint := widget.NewLabel("Supported MailScript Functions:\n- accept()\n- discard()\n- fileinto(\"Folder\")\n- regex_match(pattern, text)\n- get_header(\"Name\")")

	top := container.NewVBox(
		widget.NewLabelWithStyle("Rules Studio (Starlark)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		syntaxHint,
	)

	return container.NewBorder(top, saveBtn, nil, nil, ruleEditor)
}
