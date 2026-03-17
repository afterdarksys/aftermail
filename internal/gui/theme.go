package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// AfterMailTheme extends the base Fyne theme with custom branding colors.
type AfterMailTheme struct {
	baseTheme fyne.Theme
	isDark    bool
}

// NewAfterMailTheme creates a new custom theme instance.
// Set isDark to true for the dark UI, false for the light UI.
func NewAfterMailTheme(isDark bool) *AfterMailTheme {
	var base fyne.Theme
	if isDark {
		base = theme.DarkTheme()
	} else {
		base = theme.LightTheme()
	}
	
	return &AfterMailTheme{
		baseTheme: base,
		isDark:    isDark,
	}
}

var _ fyne.Theme = (*AfterMailTheme)(nil)

// Color returns customized colors for our premium aesthetic, falling back to base theme.
func (m *AfterMailTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	// A sleek macOS-style accent blue
	primaryColor := color.NRGBA{R: 0x00, G: 0x7A, B: 0xFF, A: 0xFF}

	switch n {
	case theme.ColorNamePrimary:
		return primaryColor
	case theme.ColorNameFocus:
		// Soft focus ring
		return color.NRGBA{R: 0x00, G: 0x7A, B: 0xFF, A: 0x40}
	case theme.ColorNameSelection:
		// Sleek selection color
		if m.isDark {
			return color.NRGBA{R: 0x00, G: 0x55, B: 0xB3, A: 0x80}
		}
		return color.NRGBA{R: 0x00, G: 0x7A, B: 0xFF, A: 0x20}
	case theme.ColorNameBackground:
		if m.isDark {
			// Rich, macOS-style dark mode window background
			return color.NRGBA{R: 0x1E, G: 0x1E, B: 0x1E, A: 0xFF}
		}
		// Clean macOS light mode window background
		return color.NRGBA{R: 0xF5, G: 0xF5, B: 0xF7, A: 0xFF}
	case theme.ColorNameButton:
		if m.isDark {
			return color.NRGBA{R: 0x33, G: 0x33, B: 0x36, A: 0xFF}
		}
		return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF} // Clean white buttons
	case theme.ColorNameInputBackground:
		if m.isDark {
			return color.NRGBA{R: 0x2A, G: 0x2A, B: 0x2C, A: 0xFF}
		}
		return color.NRGBA{R: 0xE8, G: 0xE8, B: 0xED, A: 0xFF}
	case theme.ColorNameHover:
		if m.isDark {
			return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x10}
		}
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x05}
	case theme.ColorNameHyperlink:
		return primaryColor
	case theme.ColorNameError:
		return color.NRGBA{R: 0xFF, G: 0x3B, B: 0x30, A: 0xFF} // macOS Red
	case theme.ColorNameSuccess:
		return color.NRGBA{R: 0x34, G: 0xC7, B: 0x59, A: 0xFF} // macOS Green
	case theme.ColorNameWarning:
		return color.NRGBA{R: 0xFF, G: 0x95, B: 0x00, A: 0xFF} // macOS Orange
	}

	// Fallback to the base theme
	return m.baseTheme.Color(n, v)
}

// Font overrides customized typography if necessary.
func (m *AfterMailTheme) Font(style fyne.TextStyle) fyne.Resource {
	// Here you would return custom bundled font resources like Inter or Roboto
	// For now, we fall back to standard Fyne fonts to ensure stability cross-platform
	return m.baseTheme.Font(style)
}

// Icon passes through icons seamlessly, allowing override potential later.
func (m *AfterMailTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return m.baseTheme.Icon(n)
}

// Size applies customized sizing constraints to Fyne containers.
func (m *AfterMailTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNamePadding:
		// Slightly thicker padding for a highly breathable, modern "puffy" aesthetic
		return 6.0
	case theme.SizeNameInlineIcon:
		return 24.0
	case theme.SizeNameHeadingText:
		return 28.0
	case theme.SizeNameSubHeadingText:
		return 20.0
	}
	return m.baseTheme.Size(n)
}
