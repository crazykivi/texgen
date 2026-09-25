package gui

import (
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type cyrillicTheme struct{}

var cyrillicFont fyne.Resource

func initCyrillicFont() {
	candidates := []string{
		`C:\Windows\Fonts\segoeui.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
		`C:\Windows\Fonts\calibri.ttf`,
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			cyrillicFont = fyne.NewStaticResource(filepath.Base(p), data)
			return
		}
	}
	cyrillicFont = nil
}

func (cyrillicTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}
func (cyrillicTheme) Font(_ fyne.TextStyle) fyne.Resource {
	if cyrillicFont != nil {
		return cyrillicFont
	}
	return theme.DefaultTheme().Font(fyne.TextStyle{})
}
func (cyrillicTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}
func (cyrillicTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}
