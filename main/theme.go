package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	accentColor  = color.NRGBA{0x7C, 0x5C, 0xFF, 0xFF}
	accentCyan   = color.NRGBA{0x22, 0xD3, 0xEE, 0xFF}
	accentGreen  = color.NRGBA{0x34, 0xD3, 0x99, 0xFF}
	accentAmber  = color.NRGBA{0xF5, 0x9E, 0x0B, 0xFF}
	bgColor      = color.NRGBA{0x0E, 0x0F, 0x16, 0xFF}
	surfaceColor = color.NRGBA{0x1A, 0x1C, 0x27, 0xFF}
	surfaceHover = color.NRGBA{0x24, 0x27, 0x36, 0xFF}
	fgColor      = color.NRGBA{0xED, 0xED, 0xF2, 0xFF}
	mutedColor   = color.NRGBA{0x8A, 0x8D, 0xA0, 0xFF}
	borderColor  = color.NRGBA{0x2A, 0x2D, 0x3B, 0xFF}
	errorColor   = color.NRGBA{0xE7, 0x4C, 0x5B, 0xFF}
)

type cubieTheme struct{}

func (cubieTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return bgColor
	case theme.ColorNameForeground:
		return fgColor
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
	case theme.ColorNamePrimary:
		return accentColor
	case theme.ColorNameButton:
		return surfaceColor
	case theme.ColorNameHover:
		return surfaceHover
	case theme.ColorNameInputBackground:
		return color.NRGBA{0x15, 0x17, 0x20, 0xFF}
	case theme.ColorNameInputBorder:
		return borderColor
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{0x64, 0x67, 0x7A, 0xFF}
	case theme.ColorNameSeparator:
		return borderColor
	case theme.ColorNameDisabled:
		return color.NRGBA{0x45, 0x48, 0x57, 0xFF}
	case theme.ColorNameError:
		return errorColor
	case theme.ColorNameScrollBar:
		return color.NRGBA{0x00, 0x00, 0x00, 0x66}
	case theme.ColorNameShadow:
		return color.NRGBA{0x00, 0x00, 0x00, 0x88}
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (cubieTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (cubieTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (cubieTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 10
	case theme.SizeNameInputRadius:
		return 10
	case theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameScrollBar:
		return 10
	}
	return theme.DefaultTheme().Size(name)
}
