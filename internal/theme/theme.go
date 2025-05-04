package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type SmallFontTheme struct{}

func NewSmallFontTheme(app fyne.App) fyne.App {
	th := &SmallFontTheme{}
	app.Settings().SetTheme(th)
	return app
}

func (s *SmallFontTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}

func (s *SmallFontTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (s *SmallFontTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (s *SmallFontTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameText:
		return 12
	default:
		return theme.DefaultTheme().Size(n)
	}
}
