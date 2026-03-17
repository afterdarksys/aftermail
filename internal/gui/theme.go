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
	// Our core branding color (a sleek electric blue / indigo)
	primaryColor := color.NRGBA{R: 0x4F, G: 0x46, B: 0xE5, A: 0xFF} // Indigo 600

	switch n {
	case theme.ColorNamePrimary:
		return primaryColor
	case theme.ColorNameFocus:
		// Slightly transparent primary color for focus rings
		return color.NRGBA{R: 0x4F, G: 0x46, B: 0xE5, A: 0x80}
	case theme.ColorNameSelection:
		// More transparent primary for list selections
		if m.isDark {
			return color.NRGBA{R: 0x4F, G: 0x46, B: 0xE5, A: 0x40}
		}
		return color.NRGBA{R: 0x4F, G: 0x46, B: 0xE5, A: 0x20}
	case theme.ColorNameBackground:
		if m.isDark {
			// A richer, darker solid background instead of standard gray
			return color.NRGBA{R: 0x11, G: 0x18, B: 0x27, A: 0xFF} // Gray 900
		}
		// A softer off-white for light mode
		return color.NRGBA{R: 0xF9, G: 0xFA, B: 0xFB, A: 0xFF} // Gray 50
	case theme.ColorNameButton:
		if m.isDark {
			return color.NRGBA{R: 0x1F, G: 0x29, B: 0x37, A: 0xFF} // Gray 800
		}
		return color.NRGBA{R: 0xF3, G: 0xF4, B: 0xF6, A: 0xFF} // Gray 100
	case theme.ColorNameHyperlink:
		// A brighter blue for links
		return color.NRGBA{R: 0x3B, G: 0x82, B: 0xF6, A: 0xFF} // Blue 500
	case theme.ColorNameError:
		// Vibrant red for errors
		return color.NRGBA{R: 0xEF, G: 0x44, B: 0x44, A: 0xFF} // Red 500
	case theme.ColorNameSuccess:
		// Clean green for success markers (like Verified DIDs)
		return color.NRGBA{R: 0x10, G: 0xB9, B: 0x81, A: 0xFF} // Emerald 500
	}

	// Fallback to the base theme (light or dark depending on variant)
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
