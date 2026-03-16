package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
	w := a.NewWindow("AfterMail Settings")
	w.Resize(fyne.NewSize(800, 600))

	tabs := container.NewAppTabs(
		container.NewTabItem("General", buildGeneralTab(a)),
		container.NewTabItem("AI Assistant", buildAITab()),
		container.NewTabItem("Accounts", buildAccountsTab()),
		container.NewTabItem("Advanced", buildAdvancedTab()),
	)

	w.SetContent(tabs)
	w.Show()
}

// buildGeneralTab creates the general settings tab
func buildGeneralTab(a fyne.App) fyne.CanvasObject {
	themeSelect := widget.NewSelect([]string{"OS Theme", "Dark", "Light", "Neon", "Custom"}, func(selected string) {
		switch selected {
		case "OS Theme":
			a.Settings().SetTheme(theme.DefaultTheme())
		case "Dark":
			a.Settings().SetTheme(theme.DarkTheme())
		case "Light":
			a.Settings().SetTheme(theme.LightTheme())
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
func buildAITab() fyne.CanvasObject {
	providerSelect := widget.NewSelect([]string{"Anthropic (Claude)", "OpenRouter"}, nil)
	providerSelect.SetSelected("Anthropic (Claude)")

	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder("Enter your API key")

	modelEntry := widget.NewEntry()
	modelEntry.SetPlaceHolder("claude-sonnet-4-20250514 (default)")

	testBtn := widget.NewButton("Test Connection", func() {
		// TODO: Test API connection
	})

	form := widget.NewForm(
		widget.NewFormItem("Provider", providerSelect),
		widget.NewFormItem("API Key", apiKeyEntry),
		widget.NewFormItem("Model (optional)", modelEntry),
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
		testBtn,
		widget.NewSeparator(),
		helpText,
	)
}

// buildAccountsTab creates the accounts settings tab
func buildAccountsTab() fyne.CanvasObject {
	accountsList := widget.NewList(
		func() int { return 3 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Account"),
				widget.NewLabel("Type"),
				widget.NewButton("Edit", nil),
				widget.NewButton("Remove", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			accounts := []string{"work@company.com", "personal@gmail.com", "did:aftersmtp:msgs.global:ryan"}
			types := []string{"IMAP", "Gmail", "AfterSMTP"}

			c := obj.(*fyne.Container)
			c.Objects[0].(*widget.Label).SetText(accounts[id])
			c.Objects[1].(*widget.Label).SetText(types[id])
		},
	)

	addBtn := widget.NewButton("Add Account", func() {})
	addBtn.Importance = widget.HighImportance

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Email Accounts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			addBtn,
		),
		nil, nil, nil,
		accountsList,
	)
}

// buildAdvancedTab creates the advanced settings tab
func buildAdvancedTab() fyne.CanvasObject {
	debugCheck := widget.NewCheck("Enable debug logging", nil)

	dataPathEntry := widget.NewEntry()
	dataPathEntry.SetText("~/.aftermail")

	form := widget.NewForm(
		widget.NewFormItem("Data Directory", dataPathEntry),
		widget.NewFormItem("", debugCheck),
	)

	exportBtn := widget.NewButton("Export Data", func() {})
	importBtn := widget.NewButton("Import Data", func() {})
	clearCacheBtn := widget.NewButton("Clear Cache", func() {})

	return container.NewVBox(
		widget.NewLabelWithStyle("Advanced Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		form,
		widget.NewSeparator(),
		container.NewGridWithColumns(3, exportBtn, importBtn, clearCacheBtn),
	)
}
