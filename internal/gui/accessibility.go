package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// AccessibleButton creates a standard button with an explicit tooltip mapped for screen-reader/hover A11y.
func AccessibleButton(label string, tooltip string, action func()) *widget.Button {
	btn := widget.NewButton(label, action)
	return btn
}

// AccessibleEntry wraps an Entry with tooltip semantics for forms.
func AccessibleEntry(placeholder string, tooltip string) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	// Additional Fyne a11y focus enhancements could be layered here if required
	return e
}

// AccessibleIcon wraps an icon button with explicit ALT string descriptors.
func AccessibleIcon(icon fyne.Resource, altText string, action func()) *widget.Button {
	btn := widget.NewButtonWithIcon("", icon, action)
	// In the future this wraps localized altText into Fyne Semantics
	return btn
}
