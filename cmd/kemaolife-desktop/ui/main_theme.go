//go:build windows

package ui

import (
	"github.com/EchoPing07/kemaolife/internal/desktop"
	"github.com/egoist/mygo"
	myui "github.com/egoist/mygo/ui"
)

// ApplyThemeMode 将设置中的外观主题（system / light / dark）同步到 MyGo 全局主题源。
func ApplyThemeMode(mode string) {
	switch desktop.NormalizeThemeMode(mode) {
	case desktop.ThemeLight:
		mygo.Theme.SetSource(mygo.ThemeLight)
	case desktop.ThemeDark:
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

type mainColors struct {
	shell         myui.Color
	layer         myui.Color
	layerStroke   myui.Color
	card          myui.Color
	cardSecondary myui.Color
	cardSubtle    myui.Color
	cardHover     myui.Color
	cardBorder    myui.Color
	cardHighlight myui.Color
	cardShadow    myui.Color

	controlBg           myui.Color
	controlHover        myui.Color
	controlPressed      myui.Color
	controlBorderTop    myui.Color
	controlBorderBottom myui.Color

	inputBg         myui.Color
	inputHover      myui.Color
	inputFocus      myui.Color
	inputBorder     myui.Color
	inputBottomRest myui.Color

	accentHover        myui.Color
	accentPressed      myui.Color
	accentBorderTop    myui.Color
	accentBorderBottom myui.Color

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
	dialogFooter myui.Color
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
	// Fluent 2 控件层圆角 4px，容器/卡片层显式使用 8px。
	t.Radius = 4
	t.Spacing = 4
	t.InactiveAccent = t.Accent

	p := buildMainPalette(t.Dark, t.Accent, c.Vibrancy())
	t.Background = p.shell
	if t.Dark {
		t.Surface = myui.Hex("#2d2d2d")
		t.SurfaceHover = myui.Hex("#363636")
		t.SurfacePressed = myui.Hex("#262626")
		t.Border = myui.RGBA(255, 255, 255, 0.095)
		t.Text = myui.Hex("#f3f3f3")
		t.TextMuted = myui.Hex("#9e9e9e")
		t.Success = myui.Hex("#6ccb5f")
		t.Warning = myui.Hex("#fce100")
		t.Danger = myui.Hex("#ff99a4")
		t.ScrollbarStyle = myui.ScrollbarStyle{
			TrackWidth: 10,
			ThumbWidth: 3,
			Radius:     2,
			MinLength:  28,
			Thumb:      myui.RGBA(255, 255, 255, 0.24),
			Hover:      myui.RGBA(255, 255, 255, 0.42),
			Active:     myui.RGBA(255, 255, 255, 0.56),
		}
	} else {
		t.Surface = myui.Hex("#ffffff")
		t.SurfaceHover = myui.Hex("#f7f7f7")
		t.SurfacePressed = myui.Hex("#efefef")
		t.Border = myui.RGBA(0, 0, 0, 0.085)
		t.Text = myui.Hex("#1a1a1a")
		t.TextMuted = myui.Hex("#616161")
		t.Success = myui.Hex("#0f7b0f")
		t.Warning = myui.Hex("#9d5d00")
		t.Danger = myui.Hex("#c42b1c")
		t.ScrollbarStyle = myui.ScrollbarStyle{
			TrackWidth: 10,
			ThumbWidth: 3,
			Radius:     2,
			MinLength:  28,
			Thumb:      myui.RGBA(0, 0, 0, 0.22),
			Hover:      myui.RGBA(0, 0, 0, 0.40),
			Active:     myui.RGBA(0, 0, 0, 0.54),
		}
	}
	c.SetTheme(&t)
}

func mainPalette(c *myui.Context) mainColors {
	t := c.Theme()
	return buildMainPalette(t.Dark, t.Accent, c.Vibrancy())
}

func buildMainPalette(dark bool, accent myui.Color, vibrancy bool) mainColors {
	if dark {
		layer := myui.Hex("#202020")
		card := myui.Hex("#2b2b2b")
		cardSecondary := myui.Hex("#252525")
		if vibrancy {
			layer = myui.Transparent
			card = myui.RGBA(255, 255, 255, 0.055)
			cardSecondary = myui.RGBA(255, 255, 255, 0.03)
		}
		return mainColors{
			shell:               myui.Hex("#202020"),
			layer:               layer,
			layerStroke:         myui.RGBA(255, 255, 255, 0.065),
			card:                card,
			cardSecondary:       cardSecondary,
			cardSubtle:          myui.RGBA(255, 255, 255, 0.04),
			cardHover:           myui.RGBA(255, 255, 255, 0.08),
			cardBorder:          myui.RGBA(255, 255, 255, 0.085),
			cardHighlight:       myui.RGBA(255, 255, 255, 0.085),
			cardShadow:          myui.RGBA(0, 0, 0, 0.18),
			controlBg:           myui.RGBA(255, 255, 255, 0.055),
			controlHover:        myui.RGBA(255, 255, 255, 0.082),
			controlPressed:      myui.RGBA(255, 255, 255, 0.04),
			controlBorderTop:    myui.RGBA(255, 255, 255, 0.095),
			controlBorderBottom: myui.RGBA(255, 255, 255, 0.065),
			inputBg:             myui.RGBA(255, 255, 255, 0.055),
			inputHover:          myui.RGBA(255, 255, 255, 0.078),
			inputFocus:          myui.RGBA(30, 30, 30, 0.82),
			inputBorder:         myui.RGBA(255, 255, 255, 0.08),
			inputBottomRest:     myui.RGBA(255, 255, 255, 0.54),
			accentHover:         accent.Mix(myui.RGB(0, 0, 0), 0.08),
			accentPressed:       accent.Mix(myui.RGB(0, 0, 0), 0.16),
			accentBorderTop:     myui.RGBA(255, 255, 255, 0.14),
			accentBorderBottom:  myui.RGBA(0, 0, 0, 0.20),
			hairline:            myui.RGBA(255, 255, 255, 0.075),
			navHover:            myui.RGBA(255, 255, 255, 0.05),
			navActive:           myui.RGBA(255, 255, 255, 0.072),
			navPressed:          myui.RGBA(255, 255, 255, 0.035),
			accentSoft:          accent.Alpha(0.12),
			accentBorder:        accent.Alpha(0.28),
			successSoft:         myui.RGBA(108, 203, 95, 0.14),
			warningSoft:         myui.RGBA(252, 225, 0, 0.12),
			dangerSoft:          myui.RGBA(255, 153, 164, 0.12),
			dangerBorder:        myui.RGBA(255, 153, 164, 0.26),
			backdrop:            myui.RGBA(0, 0, 0, 0.50),
			dialogFooter:        myui.Hex("#202020"),
		}
	}

	layer := myui.Hex("#f3f3f3")
	card := myui.Hex("#ffffff")
	cardSecondary := myui.Hex("#fafafa")
	if vibrancy {
		layer = myui.Transparent
		card = myui.RGBA(255, 255, 255, 0.78)
		cardSecondary = myui.RGBA(255, 255, 255, 0.50)
	}
	return mainColors{
		shell:               myui.Hex("#f3f3f3"),
		layer:               layer,
		layerStroke:         myui.RGBA(0, 0, 0, 0.06),
		card:                card,
		cardSecondary:       cardSecondary,
		cardSubtle:          myui.RGBA(0, 0, 0, 0.03),
		cardHover:           myui.RGBA(255, 255, 255, 0.92),
		cardBorder:          myui.RGBA(0, 0, 0, 0.085),
		cardHighlight:       myui.RGBA(255, 255, 255, 0.90),
		cardShadow:          myui.RGBA(0, 0, 0, 0.025),
		controlBg:           myui.RGBA(255, 255, 255, 0.78),
		controlHover:        myui.RGBA(255, 255, 255, 0.94),
		controlPressed:      myui.RGBA(245, 245, 245, 0.76),
		controlBorderTop:    myui.RGBA(0, 0, 0, 0.065),
		controlBorderBottom: myui.RGBA(0, 0, 0, 0.15),
		inputBg:             myui.RGBA(255, 255, 255, 0.76),
		inputHover:          myui.RGBA(255, 255, 255, 0.90),
		inputFocus:          myui.Hex("#ffffff"),
		inputBorder:         myui.RGBA(0, 0, 0, 0.075),
		inputBottomRest:     myui.RGBA(0, 0, 0, 0.45),
		accentHover:         accent.Mix(myui.RGB(255, 255, 255), 0.08),
		accentPressed:       accent.Mix(myui.RGB(0, 0, 0), 0.10),
		accentBorderTop:     myui.RGBA(255, 255, 255, 0.16),
		accentBorderBottom:  myui.RGBA(0, 0, 0, 0.36),
		hairline:            myui.RGBA(0, 0, 0, 0.07),
		navHover:            myui.RGBA(0, 0, 0, 0.035),
		navActive:           myui.RGBA(0, 0, 0, 0.052),
		navPressed:          myui.RGBA(0, 0, 0, 0.025),
		accentSoft:          accent.Alpha(0.08),
		accentBorder:        accent.Alpha(0.22),
		successSoft:         myui.Hex("#dff6dd"),
		warningSoft:         myui.Hex("#fff4ce"),
		dangerSoft:          myui.Hex("#fde7e9"),
		dangerBorder:        myui.RGBA(196, 43, 28, 0.22),
		backdrop:            myui.RGBA(0, 0, 0, 0.32),
		dialogFooter:        myui.Hex("#f3f3f3"),
	}
}

func tAccent(c *myui.Context) myui.Color { return c.Theme().Accent }

type mainIconSet struct {
	app, home, calendar, calendarEvent, electricity, network, settings *myui.SVG
	refresh, left, right, chevronRight, close                          *myui.SVG
	alert, info, clock, pin, user, update                              *myui.SVG
	menu, palette, key, building, layout                               *myui.SVG
	shield, cable, power, bell, timer, moonClock                       *myui.SVG
	bookOpen, check, externalLink, github, usersGroup                  *myui.SVG
	scale, tag, chartBar, chartLine                                    *myui.SVG
}

// 全部图标取自 Tabler Icons outline v3.35.0（24×24 viewBox，统一 1.5px 线条）。
var mainIcons = mainIconSet{
	app:           mainIcon(`<path d="M22 9l-10 -4l-10 4l10 4l10 -4v6"/><path d="M6 10.6v5.4a6 3 0 0 0 12 0v-5.4"/>`),
	home:          mainIcon(`<path d="M5 12l-2 0l9 -9l9 9l-2 0"/><path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7"/><path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6"/>`),
	calendar:      mainIcon(`<path d="M4 7a2 2 0 0 1 2 -2h12a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2v-12z"/><path d="M16 3v4"/><path d="M8 3v4"/><path d="M4 11h16"/><path d="M7 14h.013"/><path d="M10.01 14h.005"/><path d="M13.01 14h.005"/><path d="M16.015 14h.005"/><path d="M13.015 17h.005"/><path d="M7.01 17h.005"/><path d="M10.01 17h.005"/>`),
	calendarEvent: mainIcon(`<path d="M4 5m0 2a2 2 0 0 1 2 -2h12a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2z"/><path d="M16 3l0 4"/><path d="M8 3l0 4"/><path d="M4 11l16 0"/><path d="M8 15h2v2h-2z"/>`),
	electricity:   mainIcon(`<path d="M13 3l0 7l6 0l-8 11l0 -7l-6 0l8 -11"/>`),
	network:       mainIcon(`<path d="M12 18l.01 0"/><path d="M9.172 15.172a4 4 0 0 1 5.656 0"/><path d="M6.343 12.343a8 8 0 0 1 11.314 0"/><path d="M3.515 9.515c4.686 -4.687 12.284 -4.687 17 0"/>`),
	settings:      mainIcon(`<path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z"/><path d="M9 12a3 3 0 1 0 6 0a3 3 0 0 0 -6 0"/>`),
	refresh:       mainIcon(`<path d="M20 11a8.1 8.1 0 0 0 -15.5 -2m-.5 -4v4h4"/><path d="M4 13a8.1 8.1 0 0 0 15.5 2m.5 4v-4h-4"/>`),
	left:          mainIcon(`<path d="M15 6l-6 6l6 6"/>`),
	right:         mainIcon(`<path d="M9 6l6 6l-6 6"/>`),
	chevronRight:  mainIcon(`<path d="M9 6l6 6l-6 6"/>`),
	close:         mainIcon(`<path d="M18 6l-12 12"/><path d="M6 6l12 12"/>`),
	alert:         mainIcon(`<path d="M3 12a9 9 0 1 0 18 0a9 9 0 0 0 -18 0"/><path d="M12 8v4"/><path d="M12 16h.01"/>`),
	info:          mainIcon(`<path d="M3 12a9 9 0 1 0 18 0a9 9 0 0 0 -18 0"/><path d="M12 9h.01"/><path d="M11 12h1v4h1"/>`),
	clock:         mainIcon(`<path d="M3 12a9 9 0 1 0 18 0a9 9 0 0 0 -18 0"/><path d="M12 7v5l3 3"/>`),
	pin:           mainIcon(`<path d="M9 11a3 3 0 1 0 6 0a3 3 0 0 0 -6 0"/><path d="M17.657 16.657l-4.243 4.243a2 2 0 0 1 -2.827 0l-4.244 -4.243a8 8 0 1 1 11.314 0z"/>`),
	user:          mainIcon(`<path d="M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0"/><path d="M6 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"/>`),
	update:        mainIcon(`<path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2 -2v-2"/><path d="M7 11l5 5l5 -5"/><path d="M12 4l0 12"/>`),
	menu:          mainIcon(`<path d="M4 6l16 0"/><path d="M4 12l16 0"/><path d="M4 18l16 0"/>`),
	palette:       mainIcon(`<path d="M12 21a9 9 0 0 1 0 -18c4.97 0 9 3.582 9 8c0 1.06 -.474 2.078 -1.318 2.828c-.844 .75 -1.989 1.172 -3.182 1.172h-2.5a2 2 0 0 0 -1 3.75a1.3 1.3 0 0 1 -1 2.25"/><path d="M8.5 10.5m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"/><path d="M12.5 7.5m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"/><path d="M16.5 10.5m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"/>`),
	key:           mainIcon(`<path d="M16.555 3.843l3.602 3.602a2.877 2.877 0 0 1 0 4.069l-2.643 2.643a2.877 2.877 0 0 1 -4.069 0l-.301 -.301l-6.558 6.558a2 2 0 0 1 -1.239 .578l-.175 .008h-1.172a1 1 0 0 1 -.993 -.883l-.007 -.117v-1.172a2 2 0 0 1 .467 -1.284l.119 -.13l.414 -.414h2v-2h2v-2l2.144 -2.144l-.301 -.301a2.877 2.877 0 0 1 0 -4.069l2.643 -2.643a2.877 2.877 0 0 1 4.069 0z"/><path d="M15 9h.01"/>`),
	building:      mainIcon(`<path d="M3 21l18 0"/><path d="M9 8l1 0"/><path d="M9 12l1 0"/><path d="M9 16l1 0"/><path d="M14 8l1 0"/><path d="M14 12l1 0"/><path d="M14 16l1 0"/><path d="M5 21v-16a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v16"/>`),
	layout:        mainIcon(`<path d="M4 4m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v4a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M14 4m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v4a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M4 14m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v4a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M14 14m0 1a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v4a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/>`),
	shield:        mainIcon(`<path d="M12 3a12 12 0 0 0 8.5 3a12 12 0 0 1 -8.5 15a12 12 0 0 1 -8.5 -15a12 12 0 0 0 8.5 -3"/>`),
	cable:         mainIcon(`<path d="M9.785 6l8.215 8.215l-2.054 2.054a5.81 5.81 0 1 1 -8.215 -8.215l2.054 -2.054z"/><path d="M4 20l3.5 -3.5"/><path d="M15 4l-3.5 3.5"/><path d="M20 9l-3.5 3.5"/>`),
	power:         mainIcon(`<path d="M7 6a7.75 7.75 0 1 0 10 0"/><path d="M12 4l0 8"/>`),
	bell:          mainIcon(`<path d="M10 5a2 2 0 1 1 4 0a7 7 0 0 1 4 6v3a4 4 0 0 0 2 3h-16a4 4 0 0 0 2 -3v-3a7 7 0 0 1 4 -6"/><path d="M9 17v1a3 3 0 0 0 6 0v-1"/>`),
	timer:         mainIcon(`<path d="M6.5 7h11"/><path d="M6.5 17h11"/><path d="M6 20v-2a6 6 0 1 1 12 0v2a1 1 0 0 1 -1 1h-10a1 1 0 0 1 -1 -1z"/><path d="M6 4v2a6 6 0 1 0 12 0v-2a1 1 0 0 0 -1 -1h-10a1 1 0 0 0 -1 1z"/>`),
	moonClock:     mainIcon(`<path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z"/>`),
	bookOpen:      mainIcon(`<path d="M3 19a9 9 0 0 1 9 0a9 9 0 0 1 9 0"/><path d="M3 6a9 9 0 0 1 9 0a9 9 0 0 1 9 0"/><path d="M3 6l0 13"/><path d="M12 6l0 13"/><path d="M21 6l0 13"/>`),
	check:         mainIcon(`<path d="M5 12l5 5l10 -10"/>`),
	externalLink:  mainIcon(`<path d="M12 6h-6a2 2 0 0 0 -2 2v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-6"/><path d="M11 13l9 -9"/><path d="M15 4h5v5"/>`),
	github:        mainIcon(`<path d="M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5"/>`),
	usersGroup:    mainIcon(`<path d="M10 13a2 2 0 1 0 4 0a2 2 0 0 0 -4 0"/><path d="M8 21v-1a2 2 0 0 1 2 -2h4a2 2 0 0 1 2 2v1"/><path d="M15 5a2 2 0 1 0 4 0a2 2 0 0 0 -4 0"/><path d="M17 10h2a2 2 0 0 1 2 2v1"/><path d="M5 5a2 2 0 1 0 4 0a2 2 0 0 0 -4 0"/><path d="M3 13v-1a2 2 0 0 1 2 -2h2"/>`),
	scale:         mainIcon(`<path d="M7 20l10 0"/><path d="M6 6l6 -1l6 1"/><path d="M12 3l0 17"/><path d="M9 12l-3 -6l-3 6a3 3 0 0 0 6 0"/><path d="M21 12l-3 -6l-3 6a3 3 0 0 0 6 0"/>`),
	tag:           mainIcon(`<path d="M7.5 7.5m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"/><path d="M3 6v5.172a2 2 0 0 0 .586 1.414l7.71 7.71a2.41 2.41 0 0 0 3.408 0l5.592 -5.592a2.41 2.41 0 0 0 0 -3.408l-7.71 -7.71a2 2 0 0 0 -1.414 -.586h-5.172a3 3 0 0 0 -3 3z"/>`),
	chartBar:      mainIcon(`<path d="M3 13a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v6a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M15 9a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v10a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M9 5a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v14a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1z"/><path d="M4 20h14"/>`),
	chartLine:     mainIcon(`<path d="M4 19l16 0"/><path d="M4 15l4 -6l4 2l4 -5l4 4"/>`),
}

func mainIcon(shapes string) *myui.SVG {
	return myui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}
