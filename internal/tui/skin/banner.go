package skin

// WelcomePalette is the small set of colors the welcome panel needs. It is
// derived from the active HermesSkin's banner tokens so every built-in skin
// keeps theming the panel.
type WelcomePalette struct {
	Border string
	Title  string
	Accent string
	Dim    string
}

// WelcomePaletteFor returns the welcome-panel color tokens for a skin.
func WelcomePaletteFor(skin HermesSkin) WelcomePalette {
	skin = NormalizeStyleSkin(skin)
	return WelcomePalette{
		Border: skin.Colors.BannerBorder,
		Title:  skin.Colors.BannerTitle,
		Accent: skin.Colors.BannerAccent,
		Dim:    skin.Colors.BannerDim,
	}
}

// BannerLogoColors returns the per-line color sequence for the full Gormes
// ASCII logo, derived from the active skin's banner tokens.
func BannerLogoColors(skin HermesSkin) []string {
	skin = NormalizeStyleSkin(skin)
	c := skin.Colors
	return []string{c.BannerBorder, c.BannerAccent, c.BannerTitle, c.BannerText, c.BannerTitle, c.BannerAccent}
}

// BannerCaduceusColors returns the per-band color sequence for the compact
// caduceus art, derived from the active skin's banner tokens.
func BannerCaduceusColors(skin HermesSkin) []string {
	skin = NormalizeStyleSkin(skin)
	c := skin.Colors
	return []string{c.BannerBorder, c.BannerAccent, c.BannerTitle, c.BannerTitle, c.BannerAccent, c.BannerDim}
}
