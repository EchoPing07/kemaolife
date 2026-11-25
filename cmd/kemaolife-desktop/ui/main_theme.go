//go:build windows

package ui

import myui "github.com/egoist/mygo/ui"

type mainColors struct {
	shell        myui.Color
	layer        myui.Color
	layerStroke  myui.Color
	card         myui.Color
	cardSubtle   myui.Color
	cardHover    myui.Color
	cardBorder   myui.Color
	hairline     myui.Color
	navHover     myui.Color
	navActive    myui.Color
	navPressed   myui.Color
	accentSoft   myui.Color
	accentBorder myui.Color
	successSoft  myui.Color
	warningSoft  myui.Color
	dangerSoft   myui.Color
	dangerBorder myui.Color
	backdrop     myui.Color
}

func applyMainTheme(c *myui.Context) {
	base := c.Theme()
	accent := c.Preferences().Accent
	if accent.A == 0 {
		if base.Dark {
			accent = myui.Hex("#60cdff")
		} else {
			accent = myui.Hex("#0067c0")
		}
	}

	t := *base.WithAccent(accent)
	// 留空跟随系统默认字体：Windows 11 为 Segoe UI Variable，Win10 自动回落 Segoe UI。
	t.Font = ""
	t.FontSize = 14
	t.Radius = 6
	t.Spacing = 4
	t.InactiveAccent = t.Accent

	p := buildMainPalette(t.Dark, t.Accent, c.Vibrancy())
	t.Background = p.shell
	if t.Dark {
		t.Surface = myui.Hex("#323232")
		t.SurfaceHover = myui.Hex("#3b3b3b")
		t.SurfacePressed = myui.Hex("#2b2b2b")
		t.Border = myui.RGBA(255, 255, 255, 0.11)
		t.Text = myui.Hex("#f5f5f5")
		t.TextMuted = myui.Hex("#a6a6a6")
		t.Success = myui.Hex("#6ccb5f")
		t.Warning = myui.Hex("#fce100")
		t.Danger = myui.Hex("#ff99a4")
	} else {
		t.Surface = myui.Hex("#ffffff")
		t.SurfaceHover = myui.Hex("#f6f6f6")
		t.SurfacePressed = myui.Hex("#eeeeee")
		t.Border = myui.RGBA(0, 0, 0, 0.11)
		t.Text = myui.Hex("#1a1a1a")
		t.TextMuted = myui.Hex("#5f5f5f")
		t.Success = myui.Hex("#0f7b0f")
		t.Warning = myui.Hex("#9d5d00")
		t.Danger = myui.Hex("#c42b1c")
	}
	c.SetTheme(&t)
}

func mainPalette(c *myui.Context) mainColors {
	t := c.Theme()
	return buildMainPalette(t.Dark, t.Accent, c.Vibrancy())
}

func buildMainPalette(dark bool, accent myui.Color, vibrancy bool) mainColors {
	if dark {
		layer := myui.Hex("#272727")
		if vibrancy {
			layer = myui.RGBA(39, 39, 39, 0.72)
		}
		return mainColors{
			shell:        myui.Hex("#202020"),
			layer:        layer,
			layerStroke:  myui.RGBA(255, 255, 255, 0.065),
			card:         myui.Hex("#2d2d2d"),
			cardSubtle:   myui.Hex("#252525"),
			cardHover:    myui.Hex("#353535"),
			cardBorder:   myui.RGBA(255, 255, 255, 0.075),
			hairline:     myui.RGBA(255, 255, 255, 0.07),
			navHover:     myui.RGBA(255, 255, 255, 0.055),
			navActive:    myui.RGBA(255, 255, 255, 0.075),
			navPressed:   myui.RGBA(255, 255, 255, 0.04),
			accentSoft:   accent.Alpha(0.16),
			accentBorder: accent.Alpha(0.32),
			successSoft:  myui.RGBA(108, 203, 95, 0.15),
			warningSoft:  myui.RGBA(252, 225, 0, 0.14),
			dangerSoft:   myui.RGBA(255, 153, 164, 0.14),
			dangerBorder: myui.RGBA(255, 153, 164, 0.28),
			backdrop:     myui.RGBA(0, 0, 0, 0.48),
		}
	}
	layer := myui.Hex("#f9f9f9")
	if vibrancy {
		layer = myui.RGBA(250, 250, 250, 0.72)
	}
	return mainColors{
		shell:        myui.Hex("#f3f3f3"),
		layer:        layer,
		layerStroke:  myui.RGBA(0, 0, 0, 0.055),
		card:         myui.Hex("#ffffff"),
		cardSubtle:   myui.Hex("#f6f6f6"),
		cardHover:    myui.Hex("#f3f3f3"),
		cardBorder:   myui.RGBA(0, 0, 0, 0.068),
		hairline:     myui.RGBA(0, 0, 0, 0.06),
		navHover:     myui.RGBA(0, 0, 0, 0.036),
		navActive:    myui.RGBA(0, 0, 0, 0.052),
		navPressed:   myui.RGBA(0, 0, 0, 0.025),
		accentSoft:   accent.Alpha(0.10),
		accentBorder: accent.Alpha(0.24),
		successSoft:  myui.Hex("#dff6dd"),
		warningSoft:  myui.Hex("#fff4ce"),
		dangerSoft:   myui.Hex("#fde7e9"),
		dangerBorder: myui.RGBA(196, 43, 28, 0.22),
		backdrop:     myui.RGBA(0, 0, 0, 0.30),
	}
}

func tAccent(c *myui.Context) myui.Color { return c.Theme().Accent }

type mainIconSet struct {
	app, today, calendar, electricity, network, settings *myui.SVG
	refresh, left, right, chevronRight, close            *myui.SVG
	alert, info, clock, pin, user, update                *myui.SVG
}

var mainIcons = mainIconSet{
	app:          mainIcon(`<path d="m3 10 9-5 9 5-9 5-9-5Z"/><path d="M7 12.2V17c2.8 2 7.2 2 10 0v-4.8"/><path d="M21 10v6"/>`),
	today:        mainIcon(`<rect x="3" y="4" width="18" height="17" rx="2"/><path d="M16 2v4M8 2v4M3 9h18"/><path d="m9 15 2 2 4-4"/>`),
	calendar:     mainIcon(`<rect x="3" y="4" width="18" height="17" rx="2"/><path d="M16 2v4M8 2v4M3 9h18M8 13h.01M12 13h.01M16 13h.01M8 17h.01M12 17h.01"/>`),
	electricity:  mainIcon(`<path d="m13 2-9 12h7l-1 8 9-12h-7l1-8Z"/>`),
	network:      mainIcon(`<path d="M5 12.5a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0M2 9a14 14 0 0 1 20 0"/><circle cx="12" cy="20" r="1"/>`),
	settings:     mainIcon(`<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-2.8 2.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6v.2h-4V21a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1L4.2 17l.1-.1a1.7 1.7 0 0 0 .3-1.9A1.7 1.7 0 0 0 3 14H2.8v-4H3a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9L4.2 7 7 4.2l.1.1a1.7 1.7 0 0 0 1.9.3A1.7 1.7 0 0 0 10 3V2.8h4V3a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1L19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.2v4H21a1.7 1.7 0 0 0-1.6 1Z"/>`),
	refresh:      mainIcon(`<path d="M20 6v5h-5M4 18v-5h5"/><path d="M6.1 9a7 7 0 0 1 11.5-2.6L20 11M4 13l2.4 4.6A7 7 0 0 0 17.9 15"/>`),
	left:         mainIcon(`<path d="m15 18-6-6 6-6"/>`),
	right:        mainIcon(`<path d="m9 18 6-6-6-6"/>`),
	chevronRight: mainIcon(`<path d="m9 6 6 6-6 6"/>`),
	close:        mainIcon(`<path d="M18 6 6 18M6 6l12 12"/>`),
	alert:        mainIcon(`<circle cx="12" cy="12" r="9"/><path d="M12 8v4M12 16h.01"/>`),
	info:         mainIcon(`<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>`),
	clock:        mainIcon(`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`),
	pin:          mainIcon(`<path d="M12 21s-6-5.3-6-10a6 6 0 1 1 12 0c0 4.7-6 10-6 10Z"/><circle cx="12" cy="11" r="2"/>`),
	user:         mainIcon(`<circle cx="12" cy="8" r="4"/><path d="M5 20a7 7 0 0 1 14 0"/>`),
	update:       mainIcon(`<path d="M12 3v12m0 0-4-4m4 4 4-4"/><path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2"/>`),
}

func mainIcon(shapes string) *myui.SVG {
	return myui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}
