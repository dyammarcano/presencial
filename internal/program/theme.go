package program

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type smallFontTheme struct{}

func newSmallFontTheme(app fyne.App) fyne.App {
	th := &smallFontTheme{}
	app.Settings().SetTheme(th)
	return app
}

func (s *smallFontTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}

func (s *smallFontTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (s *smallFontTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (s *smallFontTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameText:
		return 12
	default:
		return theme.DefaultTheme().Size(n)
	}
}
