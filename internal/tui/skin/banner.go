package skin

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin/banner"

type WelcomePalette = banner.WelcomePalette

func WelcomePaletteFor(skin HermesSkin) WelcomePalette { return banner.WelcomePaletteFor(skin) }

func BannerLogoColors(skin HermesSkin) []string { return banner.BannerLogoColors(skin) }

func BannerCaduceusColors(skin HermesSkin) []string { return banner.BannerCaduceusColors(skin) }
