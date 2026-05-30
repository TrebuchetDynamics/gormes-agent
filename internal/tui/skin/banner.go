package skin

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
