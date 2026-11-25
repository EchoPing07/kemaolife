//go:build windows

package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/desktop"
	myui "github.com/egoist/mygo/ui"
)

func pageHeader(c *myui.Context, title, subtitle string, actions func()) {
	myui.Row(c).AlignItems(myui.Center).Gap(16).Children(func() {
		myui.Column(c).Grow(1).Gap(4).Children(func() {
			myui.Text(c, title).FontSize(26).Bold()
			if subtitle != "" {
				myui.Text(c, subtitle).FontSize(12).TextColor(c.Theme().TextMuted)
			}
		})
		if actions != nil {
			myui.Row(c).AlignItems(myui.Center).Gap(8).Children(actions)
		}
	})
}

func mainCard(c *myui.Context) myui.Element {
	p := mainPalette(c)
	return myui.Column(c).Padding(18, 20).Gap(14).Radius(8).Background(p.card).
		Border(1, p.cardBorder)
}

func mainSectionTitle(c *myui.Context, title, detail string) {
	myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
		myui.Text(c, title).FontSize(14).Bold()
		myui.Spacer(c)
		if detail != "" {
			myui.Text(c, detail).FontSize(11).TextColor(c.Theme().TextMuted)
		}
	})
}

func mainBadge(c *myui.Context, label string, textColor, bgColor myui.Color, withDot bool) {
	myui.Row(c).Padding(3, 9).Gap(6).Radius(12).Background(bgColor).AlignItems(myui.Center).Children(func() {
		if withDot {
			myui.Box(c).Size(6, 6).Radius(3).Background(textColor).Shrink(0)
		}
		myui.Text(c, label).FontSize(11).Bold().TextColor(textColor).SingleLine()
	})
}

func mainInfoBar(c *myui.Context, icon *myui.SVG, text string, fg, bg, border myui.Color) {
	if strings.TrimSpace(text) == "" {
		return
	}
	myui.Row(c).Padding(10, 14).Gap(10).Radius(6).Background(bg).Border(1, border).
		AlignItems(myui.Center).Children(func() {
		if icon != nil {
			myui.Icon(c, icon).FontSize(15).TextColor(fg).Shrink(0)
		}
		myui.Text(c, text).FontSize(12).TextColor(fg).Selectable().Grow(1)
	})
}

func mainLoading(c *myui.Context, loading bool, text string) {
	if !loading {
		return
	}
	p := mainPalette(c)
	myui.Row(c).Padding(8, 12).Gap(8).Radius(6).Background(p.accentSoft).Border(1, p.accentBorder).
		AlignItems(myui.Center).Children(func() {
		myui.Spinner(c).Label(text)
		myui.Text(c, text).FontSize(12).TextColor(c.Theme().Text)
	})
}

func mainError(c *myui.Context, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	p := mainPalette(c)
	mainInfoBar(c, mainIcons.alert, text, c.Theme().Danger, p.dangerSoft, p.dangerBorder)
}

func mainEmpty(c *myui.Context, title, detail string) {
	myui.Column(c).Padding(32, 16).Gap(6).AlignItems(myui.Center).Children(func() {
		myui.Text(c, title).FontSize(15).Bold().TextColor(c.Theme().Text)
		if detail != "" {
			myui.Text(c, detail).FontSize(12).TextColor(c.Theme().TextMuted).TextAlign(myui.Center)
		}
	})
}

func mainSubtleLink(c *myui.Context, key, label string) bool {
	p := mainPalette(c)
	t := c.Theme()
	btn := myui.ButtonBase(c.Key(key)).Height(30).Padding(0, 10).Gap(4).Radius(5).
		AlignItems(myui.Center).Justify(myui.Center)
	btn.Children(func() {
		myui.Text(c, label).FontSize(12).TextColor(t.Accent).SingleLine()
		myui.Icon(c, mainIcons.chevronRight).FontSize(13).TextColor(t.Accent)
	})
	if btn.Pressed() {
		btn.Background(p.navPressed)
	} else if btn.Hovered() {
		btn.Background(p.navHover)
	}
	return btn.Clicked()
}

func mainLessonRow(c *myui.Context, key any, entry model.Entry, onClick func()) {
	p := mainPalette(c)
	t := c.Theme()
	cc := courseColorFor(entry.Name, t.Dark)
	row := myui.ButtonBase(c.Key(key)).Padding(10, 12).Gap(14).Radius(6).
		Background(p.cardSubtle).AlignItems(myui.Center).Justify(myui.Start)
	if row.Pressed() {
		row.Background(p.navPressed)
	} else if row.Hovered() {
		row.Background(p.cardHover)
	}
	row.Children(func() {
		// 用淡底色节次胶囊关联课表色系，避免生硬的左侧竖条装饰。
		myui.Column(c).Width(94).Shrink(0).Padding(6, 8).Gap(2).Radius(6).
			Background(cc.fill).Border(1, cc.border).AlignItems(myui.Center).Children(func() {
			myui.Text(c, "第 "+mainTextOr(entry.Periods, "—")+" 节").FontSize(12).Bold().TextColor(cc.text)
			if entry.Time != "" {
				myui.Text(c, entry.Time).FontSize(10).TextColor(cc.text.Alpha(0.78)).FontFeatures("tnum")
			}
		})
		myui.Column(c).Grow(1).Gap(4).Children(func() {
			myui.Text(c, mainTextOr(entry.Name, "未命名课程")).FontSize(14).Bold()
			details := make([]string, 0, 2)
			if entry.Location != "" {
				details = append(details, entry.Location)
			}
			if entry.Teacher != "" {
				details = append(details, entry.Teacher)
			}
			if len(details) > 0 {
				myui.Text(c, strings.Join(details, " · ")).FontSize(12).TextColor(t.TextMuted)
			}
		})
		if onClick != nil {
			myui.Icon(c, mainIcons.chevronRight).FontSize(14).TextColor(t.TextMuted).Shrink(0)
		}
	})
	if onClick != nil && row.Clicked() {
		onClick()
	}
}

type courseColor struct{ fill, hover, border, text myui.Color }

// 课表 8 色调色板与 Android 端保持同一色相语言，纯色块 + 极淡同色描边，不加左侧竖条。
func courseColorFor(name string, dark bool) courseColor {
	light := []courseColor{
		{myui.RGB(255, 236, 239), myui.Hex("#ffe0e5"), myui.RGBA(209, 75, 98, 0.22), myui.Hex("#7a1f31")}, // 玫瑰
		{myui.Hex("#fff1e6"), myui.Hex("#ffe4d1"), myui.RGBA(212, 107, 40, 0.22), myui.Hex("#733409")},    // 蜜桃
		{myui.Hex("#fff6df"), myui.Hex("#ffedc2"), myui.RGBA(196, 138, 20, 0.24), myui.Hex("#634400")},    // 琥珀
		{myui.Hex("#eaf7e6"), myui.Hex("#dcf0d5"), myui.RGBA(71, 145, 52, 0.22), myui.Hex("#204d16")},     // 新芽
		{myui.Hex("#e4f6ef"), myui.Hex("#d2efe4"), myui.RGBA(30, 140, 110, 0.22), myui.Hex("#0c4536")},    // 薄荷
		{myui.Hex("#e7f3ff"), myui.Hex("#d6eaff"), myui.RGBA(29, 114, 194, 0.22), myui.Hex("#0d3c6b")},    // 天蓝
		{myui.Hex("#f0ecff"), myui.Hex("#e3dcff"), myui.RGBA(108, 83, 201, 0.22), myui.Hex("#362673")},    // 淡紫
		{myui.Hex("#fcedf6"), myui.Hex("#f8deed"), myui.RGBA(184, 71, 138, 0.22), myui.Hex("#631d47")},    // 樱粉
	}
	darkColors := []courseColor{
		{myui.Hex("#43262c"), myui.Hex("#502e35"), myui.RGBA(240, 128, 148, 0.24), myui.Hex("#ffd9df")},
		{myui.Hex("#432f21"), myui.Hex("#503928"), myui.RGBA(242, 156, 97, 0.24), myui.Hex("#ffe4d1")},
		{myui.Hex("#3e341e"), myui.Hex("#4b3f24"), myui.RGBA(230, 185, 76, 0.24), myui.Hex("#fff0c9")},
		{myui.Hex("#263c23"), myui.Hex("#2f492b"), myui.RGBA(126, 203, 104, 0.24), myui.Hex("#dcf5d3")},
		{myui.Hex("#1e3b34"), myui.Hex("#264840"), myui.RGBA(86, 199, 168, 0.24), myui.RGB(206, 243, 231)},
		{myui.Hex("#20364a"), myui.Hex("#28425a"), myui.RGBA(107, 180, 245, 0.24), myui.Hex("#d4ebff")},
		{myui.Hex("#312b4c"), myui.Hex("#3b345c"), myui.RGBA(166, 149, 240, 0.24), myui.Hex("#e6e0ff")},
		{myui.Hex("#412538"), myui.Hex("#4e2d43"), myui.RGBA(226, 130, 189, 0.24), myui.Hex("#fbdcf0")},
	}
	colors := light
	if dark {
		colors = darkColors
	}
	if name == "" {
		return colors[0]
	}
	sum := 0
	for _, r := range name {
		sum += int(r)
	}
	return colors[sum%len(colors)]
}

func campusStateColor(c *myui.Context, state model.CampusState) myui.Color {
	switch state {
	case model.CampusOnline:
		return c.Theme().Success
	case model.CampusUnauth:
		return c.Theme().Warning
	default:
		return c.Theme().TextMuted
	}
}

func campusStateSoftBg(c *myui.Context, state model.CampusState) myui.Color {
	p := mainPalette(c)
	switch state {
	case model.CampusOnline:
		return p.successSoft
	case model.CampusUnauth:
		return p.warningSoft
	default:
		return p.cardSubtle
	}
}

func mainStateLabel(state model.CampusState) string {
	if state == "" {
		return "尚未检查"
	}
	return desktop.CampusStateText(state)
}

func mainTextOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func mainTimeText(value string) string {
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err == nil {
		return t.Local().Format("2006-01-02 15:04")
	}
	return value
}

func mainFormatFloat(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

func mainMetric(c *myui.Context, label, value, unit string, color myui.Color) {
	myui.Column(c).Gap(6).Children(func() {
		myui.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted)
		myui.Row(c).AlignItems(myui.Baseline).Gap(5).Children(func() {
			myui.Text(c, value).FontSize(28).Bold().TextColor(color).FontFeatures("tnum")
			if unit != "" {
				myui.Text(c, unit).FontSize(12).TextColor(c.Theme().TextMuted)
			}
		})
	})
}

func mainBoolText(v bool, yes, no string) string {
	if v {
		return yes
	}
	return no
}
