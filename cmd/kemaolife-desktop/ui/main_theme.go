//go:build windows

package ui

import myui "github.com/egoist/mygo/ui"

type mainColors struct {
	canvas       myui.Color
	sidebar      myui.Color
	sidebarHover myui.Color
	card         myui.Color
	subtle       myui.Color
	border       myui.Color
	accentSoft   myui.Color
	accentBorder myui.Color
}

func applyMainTheme(c *myui.Context) {
	t := *c.Theme().WithAccent(myui.Hex("#0f6cbd"))
	t.Font = "Segoe UI"
	t.FontSize = 14
	t.Radius = 8
	t.Spacing = 4
	t.InactiveAccent = t.Accent
	p := mainPalette(c)
	t.Background = p.canvas
	t.Surface = p.card
	t.Border = p.border
	if t.Dark {
		t.SurfaceHover = myui.Hex("#343b43")
		t.SurfacePressed = myui.Hex("#3d464f")
	} else {
		t.SurfaceHover = myui.Hex("#f0f4f7")
		t.SurfacePressed = myui.Hex("#e4ebf0")
	}
	c.SetTheme(&t)
}

func mainPalette(c *myui.Context) mainColors {
	if c.Theme().Dark {
		return mainColors{
			canvas: myui.Hex("#1c2127"), sidebar: myui.Hex("#22282f"),
			sidebarHover: myui.Hex("#2d353e"), card: myui.Hex("#272e36"),
			subtle: myui.Hex("#20262d"), border: myui.Hex("#3a434d"),
			accentSoft: myui.Hex("#173b57"), accentBorder: myui.Hex("#285c80"),
		}
	}
	return mainColors{
		canvas: myui.Hex("#f4f6f8"), sidebar: myui.Hex("#edf1f4"),
		sidebarHover: myui.Hex("#e2e9ee"), card: myui.Hex("#ffffff"),
		subtle: myui.Hex("#f7f9fa"), border: myui.Hex("#dce3e8"),
		accentSoft: myui.Hex("#e5f1fa"), accentBorder: myui.Hex("#bfd9ec"),
	}
}

func tAccent(c *myui.Context) myui.Color { return c.Theme().Accent }

type mainIconSet struct {
	app, today, calendar, electricity, network, settings *myui.SVG
	refresh, left, right                                 *myui.SVG
}

var mainIcons = mainIconSet{
	app:         mainIcon(`<path d="m3 10 9-5 9 5-9 5-9-5Z"/><path d="M7 12.2V17c2.8 2 7.2 2 10 0v-4.8"/><path d="M21 10v6"/>`),
	today:       mainIcon(`<path d="M3 9h18M9 3v6M3 5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5Z"/><path d="m9 15 2 2 4-5"/>`),
	calendar:    mainIcon(`<rect x="3" y="4" width="18" height="17" rx="2"/><path d="M16 2v4M8 2v4M3 9h18M8 13h.01M12 13h.01M16 13h.01M8 17h.01M12 17h.01"/>`),
	electricity: mainIcon(`<path d="m13 2-9 12h7l-1 8 9-12h-7l1-8Z"/>`),
	network:     mainIcon(`<path d="M5 12.5a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0M2 9a14 14 0 0 1 20 0"/><circle cx="12" cy="20" r="1"/>`),
	settings:    mainIcon(`<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-2.8 2.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6v.2h-4V21a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1L4.2 17l.1-.1a1.7 1.7 0 0 0 .3-1.9A1.7 1.7 0 0 0 3 14H2.8v-4H3a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9L4.2 7 7 4.2l.1.1a1.7 1.7 0 0 0 1.9.3A1.7 1.7 0 0 0 10 3V2.8h4V3a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1L19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.2v4H21a1.7 1.7 0 0 0-1.6 1Z"/>`),
	refresh:     mainIcon(`<path d="M20 6v5h-5M4 18v-5h5"/><path d="M6.1 9a7 7 0 0 1 11.5-2.6L20 11M4 13l2.4 4.6A7 7 0 0 0 17.9 15"/>`),
	left:        mainIcon(`<path d="m15 18-6-6 6-6"/>`),
	right:       mainIcon(`<path d="m9 18 6-6-6-6"/>`),
}

func mainIcon(shapes string) *myui.SVG {
	return myui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}
