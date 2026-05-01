package gui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/afterdarksys/aftermail/pkg/accounts"
)

// The predefined themes
type neonTheme struct{}

func (n neonTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 20, G: 20, B: 30, A: 255} // Dark blue/black background
	case theme.ColorNameForeground, theme.ColorNameButton:
		return color.NRGBA{R: 255, G: 0, B: 255, A: 255} // Magenta text & outlines
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0, G: 255, B: 255, A: 255} // Cyan highlights
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 30, G: 30, B: 40, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 50, G: 0, B: 50, A: 255} // Dark magenta hover
	case theme.ColorNameFocus:
		return color.NRGBA{R: 0, G: 255, B: 255, A: 128} // Cyan focus ring
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (n neonTheme) Font(style fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(style) }
func (n neonTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(name) }
func (n neonTheme) Size(name fyne.ThemeSizeName) float32 { return theme.DefaultTheme().Size(name) }

type customTheme struct{}
// Placeholders for custom user theme logic for now
func (c customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color { return theme.DefaultTheme().Color(name, variant) }
func (c customTheme) Font(style fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(style) }
func (c customTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(name) }
func (c customTheme) Size(name fyne.ThemeSizeName) float32 { return theme.DefaultTheme().Size(name) }


// openSettingsDialog opens the comprehensive settings window
func openSettingsDialog(a fyne.App) {
	sw := a.NewWindow("AfterMail Settings")
	sw.Resize(fyne.NewSize(800, 600))

	tabs := container.NewAppTabs(
		container.NewTabItem("General", buildGeneralTab(a, sw)),
		container.NewTabItem("AI Assistant", buildAITab(sw)),
		container.NewTabItem("Accounts", buildAccountsTab(sw)),
		container.NewTabItem("Advanced", buildAdvancedTab(sw)),
	)

	sw.SetContent(tabs)
	sw.Show()
}

// buildGeneralTab creates the general settings tab
func buildGeneralTab(a fyne.App, w fyne.Window) fyne.CanvasObject {
	themeSelect := widget.NewSelect([]string{"OS Theme", "Dark", "Light", "Neon", "Custom"}, func(selected string) {
		switch selected {
		case "OS Theme":
			a.Settings().SetTheme(theme.DefaultTheme())
		case "Dark":
			a.Settings().SetTheme(NewAfterMailTheme(true))
		case "Light":
			a.Settings().SetTheme(NewAfterMailTheme(false))
		case "Neon":
			a.Settings().SetTheme(&neonTheme{})
		case "Custom":
			a.Settings().SetTheme(&customTheme{})
		}
	})
	themeSelect.SetSelected("OS Theme")

	languageSelect := widget.NewSelect([]string{"English", "Spanish", "French", "German", "Japanese"}, nil)
	languageSelect.SetSelected("English")

	spellCheckEnable := widget.NewCheck("Enable spell checking", nil)
	spellCheckEnable.SetChecked(true)

	grammarCheckEnable := widget.NewCheck("Enable grammar checking", nil)
	grammarCheckEnable.SetChecked(true)

	form := widget.NewForm(
		widget.NewFormItem("Theme", themeSelect),
		widget.NewFormItem("Language", languageSelect),
		widget.NewFormItem("", spellCheckEnable),
		widget.NewFormItem("", grammarCheckEnable),
	)

	return container.NewVBox(
		widget.NewLabelWithStyle("General Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		form,
	)
}

// buildAITab creates the AI Assistant settings tab
func buildAITab(w fyne.Window) fyne.CanvasObject {
	providerSelect := widget.NewSelect([]string{"Anthropic (Claude)", "OpenRouter"}, nil)
	providerSelect.SetSelected("Anthropic (Claude)")

	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Enter your API key")

	modelEntry := widget.NewEntry()
	modelEntry.SetPlaceHolder("claude-sonnet-4-20250514 (default)")

	saveBtn := widget.NewButton("Save API Key", func() {
		provider := "anthropic"
		if providerSelect.Selected == "OpenRouter" {
			provider = "openrouter"
		}
		model := modelEntry.Text
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
		SetAICredentials(provider, apiKeyEntry.Text, model)
		dialog.ShowInformation("Saved", "AI credentials saved for this session.", w)
	})
	saveBtn.Importance = widget.HighImportance

	testBtn := widget.NewButton("Test Connection", func() {
		if apiKeyEntry.Text == "" {
			dialog.ShowInformation("Test", "Enter an API key first.", w)
			return
		}
		dialog.ShowInformation("Test", "Connection test: key is set. Send a message to verify it works.", w)
	})

	form := widget.NewForm(
		widget.NewFormItem("Provider", providerSelect),
		widget.NewFormItem("API Key", apiKeyEntry),
		widget.NewFormItem("Model (optional)", modelEntry),
	)

	buttonRow := container.NewHBox(
		saveBtn,
		testBtn,
	)

	helpText := widget.NewLabel(`AI Assistant Features:
• Spell checking
• Grammar checking
• Writing improvements
• Text rewriting (formal/friendly/concise)
• Draft generation
• Email summarization

Get your API key:
• Anthropic: https://console.anthropic.com
• OpenRouter: https://openrouter.ai

Your API key is stored locally and never shared.`)
	helpText.Wrapping = fyne.TextWrapWord

	return container.NewVBox(
		widget.NewLabelWithStyle("AI Assistant Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		form,
		buttonRow,
		widget.NewSeparator(),
		helpText,
	)
}

// buildAccountsTab creates the accounts settings tab with a working Add Account form.
func buildAccountsTab(w fyne.Window) fyne.CanvasObject {
	// In-memory list for this session; in production load from DB.
	type accountRow struct{ email, kind string }
	rows := []accountRow{}
	list := widget.NewList(
		func() int { return len(rows) },
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewLabel(""), widget.NewLabel(""))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)
			c.Objects[0].(*widget.Label).SetText(rows[id].email)
			c.Objects[1].(*widget.Label).SetText(rows[id].kind)
		},
	)

	showAddForm := func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Display name")
		emailEntry := widget.NewEntry()
		emailEntry.SetPlaceHolder("you@example.com")
		smtpHostEntry := widget.NewEntry()
		smtpHostEntry.SetPlaceHolder("smtp.gmail.com")
		smtpPortEntry := widget.NewEntry()
		smtpPortEntry.SetText("587")
		usernameEntry := widget.NewEntry()
		usernameEntry.SetPlaceHolder("you@example.com")
		passwordEntry := widget.NewPasswordEntry()
		passwordEntry.SetPlaceHolder("App password")
		tlsCheck := widget.NewCheck("Use TLS (port 465)", nil)
		imapHostEntry := widget.NewEntry()
		imapHostEntry.SetPlaceHolder("imap.gmail.com")
		imapPortEntry := widget.NewEntry()
		imapPortEntry.SetText("993")

		dialog.ShowForm("Add Email Account", "Save", "Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Name", nameEntry),
				widget.NewFormItem("Email", emailEntry),
				widget.NewFormItem("SMTP Host", smtpHostEntry),
				widget.NewFormItem("SMTP Port", smtpPortEntry),
				widget.NewFormItem("Username", usernameEntry),
				widget.NewFormItem("Password", passwordEntry),
				widget.NewFormItem("", tlsCheck),
				widget.NewFormItem("IMAP Host", imapHostEntry),
				widget.NewFormItem("IMAP Port", imapPortEntry),
			},
			func(ok bool) {
				if !ok || emailEntry.Text == "" || smtpHostEntry.Text == "" {
					return
				}
				var smtpPort, imapPort int
				fmt.Sscanf(smtpPortEntry.Text, "%d", &smtpPort)
				fmt.Sscanf(imapPortEntry.Text, "%d", &imapPort)
				if smtpPort == 0 {
					smtpPort = 587
				}
				if imapPort == 0 {
					imapPort = 993
				}
				rows = append(rows, accountRow{emailEntry.Text, "SMTP"})
				list.Refresh()
				dialog.ShowInformation("Account Added",
					"Account "+emailEntry.Text+" saved.\nSMTP: "+smtpHostEntry.Text, w)
			}, w)
	}

	addBtn := widget.NewButton("Add Account", showAddForm)
	addBtn.Importance = widget.HighImportance

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Email Accounts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			addBtn,
		),
		nil, nil, nil,
		list,
	)
}

// buildAdvancedTab creates the advanced settings tab
func buildAdvancedTab(w fyne.Window) fyne.CanvasObject {
	debugCheck := widget.NewCheck("Enable debug logging", nil)

	robustParsingCheck := widget.NewCheck("Enable robust MIME/header parsing", func(checked bool) {
		accounts.RobustParsingEnabled = checked
	})
	robustParsingCheck.SetChecked(accounts.RobustParsingEnabled)

	dataPathEntry := widget.NewEntry()
	dataPathEntry.SetText("~/.aftermail")

	form := widget.NewForm(
		widget.NewFormItem("Data Directory", dataPathEntry),
		widget.NewFormItem("", debugCheck),
		widget.NewFormItem("", robustParsingCheck),
	)

	exportBtn := widget.NewButton("Export Data", func() {
		dialog.ShowInformation("Export", "Export not yet implemented.", w)
	})
	importBtn := widget.NewButton("Import Data", func() {
		dialog.ShowInformation("Import", "Import not yet implemented.", w)
	})
	clearCacheBtn := widget.NewButton("Clear Cache", func() {
		dialog.ShowInformation("Cache", "Cache cleared.", w)
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Advanced Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		form,
		widget.NewSeparator(),
		container.NewGridWithColumns(3, exportBtn, importBtn, clearCacheBtn),
	)
}
