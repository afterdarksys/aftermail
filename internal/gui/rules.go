package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// buildRulesTab creates the Starlark Rules Studio UI
func buildRulesTab() fyne.CanvasObject {
	// 1. Code Editor Setup
	ruleEditor := widget.NewMultiLineEntry()
	ruleEditor.SetPlaceHolder("def evaluate():\n    # Write AfterSMTP MailScript (Starlark) here...\n    accept()\n")
	ruleEditor.Resize(fyne.NewSize(600, 400))

	// 2. Visual Builder Setup
	// Condition row components
	targetHeader := widget.NewSelect([]string{"From", "To", "Subject", "Date"}, nil)
	targetHeader.SetSelected("Subject")
	matchType := widget.NewSelect([]string{"contains", "does not contain", "equals", "starts with"}, nil)
	matchType.SetSelected("contains")
	matchValue := widget.NewEntry()
	matchValue.SetPlaceHolder("e.g., 'viagra' or 'urgent'")

	conditionRow := container.NewGridWrap(fyne.NewSize(150, 40), targetHeader, matchType, container.NewGridWrap(fyne.NewSize(200, 40), matchValue))

	// Action row components
	actionSelect := widget.NewSelect([]string{"Move to Folder", "Delete (Discard)", "Forward to", "Mark as Read"}, nil)
	actionSelect.SetSelected("Move to Folder")
	actionValue := widget.NewEntry()
	actionValue.SetPlaceHolder("e.g., 'Junk' or 'user@example.com'")

	actionRow := container.NewGridWrap(fyne.NewSize(150, 40), actionSelect, container.NewGridWrap(fyne.NewSize(200, 40), actionValue))

	visualBuilder := container.NewVBox(
		widget.NewLabelWithStyle("1. If message matches:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		conditionRow,
		widget.NewLabelWithStyle("2. Then perform action:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		actionRow,
	)

	// Function to compile Visual Rule into Starlark Code
	compileRule := func() {
		header := targetHeader.Selected
		match := matchType.Selected
		val := matchValue.Text
		
		act := actionSelect.Selected
		actVal := actionValue.Text

		if val == "" {
			return // Don't compile if empty condition
		}

		var starlarkCode strings.Builder
		starlarkCode.WriteString("def evaluate():\n")
		starlarkCode.WriteString(fmt.Sprintf("    header_val = get_header(\"%s\").lower()\n", header))
		starlarkCode.WriteString("    \n")

		// Condition Logic
		lowerVal := strings.ToLower(val)
		if match == "contains" {
			starlarkCode.WriteString(fmt.Sprintf("    if \"%s\" in header_val:\n", lowerVal))
		} else if match == "equals" {
			starlarkCode.WriteString(fmt.Sprintf("    if header_val == \"%s\":\n", lowerVal))
		} else if match == "does not contain" {
			starlarkCode.WriteString(fmt.Sprintf("    if \"%s\" not in header_val:\n", lowerVal))
		} else if match == "starts with" {
			starlarkCode.WriteString(fmt.Sprintf("    if regex_match(\"^%s.*\", header_val):\n", lowerVal))
		}

		// Action Logic
		if act == "Move to Folder" {
			starlarkCode.WriteString(fmt.Sprintf("        fileinto(\"%s\")\n", actVal))
		} else if act == "Delete (Discard)" {
			starlarkCode.WriteString("        discard()\n")
		} else if act == "Forward to" {
			starlarkCode.WriteString(fmt.Sprintf("        redirect(\"%s\")\n", actVal))
		} else {
			starlarkCode.WriteString("        accept()\n")
		}
		
		starlarkCode.WriteString("    else:\n        accept()\n")

		ruleEditor.SetText(starlarkCode.String())
	}

	compileBtn := widget.NewButton("Generate MailScript", compileRule)
	visualBuilderContainer := container.NewBorder(nil, compileBtn, nil, nil, visualBuilder)

	// 3. Documentation Pane Setup
	docText := widget.NewRichTextFromMarkdown(`
### MailScript (Starlark) Reference

MailScript offers parity with legacy Sieve while exposing AfterSMTP features.

**Native Functions:**
*   ` + "`accept()`" + `: Accept the message (Keep).
*   ` + "`discard()`" + `: Silently drop the message.
*   ` + "`fileinto(\"Folder\")`" + `: Move the message.
*   ` + "`get_header(\"Name\")`" + `: Returns the value of a header layer.
*   ` + "`regex_match(pattern, text)`" + `: Regex evaluation on text.

**Example Script:**
` + "```python\ndef evaluate():\n    subj = get_header(\"Subject\").lower()\n    if \"bitcoin\" in subj:\n        fileinto(\"Junk\")\n    else:\n        accept()\n```")
	docScroll := container.NewVScroll(docText)

	tabs := container.NewAppTabs(
		container.NewTabItem("Visual Builder", visualBuilderContainer),
		container.NewTabItem("Code Editor (Starlark)", ruleEditor),
	)

	split := container.NewHSplit(tabs, docScroll)
	split.SetOffset(0.6) // Give more space to the editor

	saveBtn := widget.NewButton("Save Rule to engine", func() {
		// In a real app, this would save to SQLite or send to aftermaild
		ruleEditor.SetText(ruleEditor.Text + "\n# Rule saved locally!")
	})

	top := widget.NewLabelWithStyle("Rules Studio", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return container.NewBorder(top, saveBtn, nil, nil, split)
}

// ShowMailScriptPolicyDialog opens a dialog to configure MailScript execution policies.
func ShowMailScriptPolicyDialog(w fyne.Window) {
	// Simple JSON editor mock for MailScript policies
	policyEditor := widget.NewMultiLineEntry()
	policyEditor.SetText(`{
  "execution_order": ["system", "user_defined"],
  "execution_priority": "high",
  "hard_fail": {
    "action": "notify_me",
    "halt": true
  },
  "soft_fail": {
    "action": "log",
    "halt": false
  }
}`)
	policyEditor.Resize(fyne.NewSize(500, 300))

	saveBtn := widget.NewButton("Save Policy", func() {
		// Just a mock save for now
		dialog.ShowInformation("MailScript Policy", "Policy Settings Saved Successfully", w)
	})

	content := container.NewBorder(
		widget.NewLabelWithStyle("MailScript Engine Execution Policy (JSON)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		saveBtn,
		nil, nil,
		policyEditor,
	)

	dialog.ShowCustom("MailScript Policies", "Close", content, w)
}
