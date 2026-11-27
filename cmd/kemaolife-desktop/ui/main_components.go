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
	myui.Row(c).FillWidth().AlignItems(myui.Center).Gap(16).Children(func() {
		myui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			myui.Text(c, title).FontSize(26).FontWeight(600)
			if subtitle != "" {
				myui.Text(c, subtitle).FontSize(12).TextColor(c.Theme().TextMuted)
			}
		})
		if actions != nil {
			myui.Row(c).Shrink(0).AlignItems(myui.Center).Gap(8).Children(actions)
		}
	})
}

func mainCard(c *myui.Context) myui.Element {
	p := mainPalette(c)
	return myui.Column(c).Padding(18, 20).Gap(14).Radius(8).Background(p.card).
		Border(1, p.cardBorder).Shadow(0, 1, 2, 0, p.cardShadow)
}

func mainSectionTitle(c *myui.Context, title, detail string) {
	myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
		myui.Text(c, title).FontSize(14).FontWeight(600)
		myui.Spacer(c)
		if detail != "" {
			myui.Text(c, detail).FontSize(11).TextColor(c.Theme().TextMuted)
		}
	})
}

func mainBadge(c *myui.Context, label string, textColor, bgColor myui.Color, withDot bool) {
	p := mainPalette(c)
	myui.Row(c).Padding(2, 8).Gap(6).Radius(4).Background(bgColor).Border(1, p.hairline).
		AlignItems(myui.Center).Children(func() {
		if withDot {
			myui.Box(c).Size(6, 6).Radius(3).Background(textColor).Shrink(0)
		}
		myui.Text(c, label).FontSize(11).FontWeight(500).TextColor(textColor).SingleLine()
	})
}

func mainInfoBar(c *myui.Context, icon *myui.SVG, text string, fg, bg, border myui.Color) {
	if strings.TrimSpace(text) == "" {
		return
	}
	t := c.Theme()
	myui.Row(c).Padding(10, 14).Gap(10).Radius(6).Background(bg).Border(1, border).
		AlignItems(myui.Center).Children(func() {
		if icon != nil {
			myui.Icon(c, icon).FontSize(15).TextColor(fg).Shrink(0)
		}
		myui.Text(c, text).FontSize(12).TextColor(t.Text).Selectable().Grow(1)
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
	myui.Column(c).Padding(28, 16).Gap(6).AlignItems(myui.Center).Children(func() {
		myui.Text(c, title).FontSize(15).FontWeight(600).TextColor(c.Theme().Text)
		if detail != "" {
			myui.Text(c, detail).FontSize(12).TextColor(c.Theme().TextMuted).TextAlign(myui.Center)
		}
	})
}

func mainSubtleLink(c *myui.Context, key, label string) bool {
	p := mainPalette(c)
	t := c.Theme()
	btn := myui.ButtonBase(c.Key(key)).Height(28).Padding(0, 10).Gap(4).Radius(4).
		AlignItems(myui.Center).Justify(myui.Center)
	btn.Children(func() {
		myui.Text(c, label).FontSize(12).FontWeight(500).TextColor(t.Accent).SingleLine()
		myui.Icon(c, mainIcons.chevronRight).FontSize(13).TextColor(t.Accent)
	})
	if btn.Pressed() {
		btn.Background(p.navPressed)
	} else if btn.Hovered() {
		btn.Background(p.navHover)
	}
	return btn.Clicked()
}

type timetableDisplayOptions struct {
	optimizeLocation bool
	prefixLocationAt bool
	hideOverflow     bool
}

func (w *mainWindow) timetableDisplay() timetableDisplayOptions {
	return timetableDisplayOptions{
		optimizeLocation: w.state.SettingsForm.OptimizeLocationDisplay,
		prefixLocationAt: w.state.SettingsForm.PrefixLocationAt,
		hideOverflow:     w.state.SettingsForm.HideTimetableOverflow,
	}
}

// mainEntryLocation 返回课表网格与今日课节展示用的课室名：
// 开启优化且 core 提供了 ShortLocation 时使用精简名，否则回退原始地点（AGENTS A3）；
// 开启 prefixAt 时在非空课室前加 @，已有 @ 前缀不重复加。
func mainEntryLocation(entry model.Entry, opt timetableDisplayOptions) string {
	raw := strings.TrimSpace(entry.Location)
	if opt.optimizeLocation && strings.TrimSpace(entry.ShortLocation) != "" {
		raw = strings.TrimSpace(entry.ShortLocation)
	}
	if raw == "" {
		return ""
	}
	if opt.prefixLocationAt && !strings.HasPrefix(raw, "@") {
		return "@" + raw
	}
	return raw
}

func mainLessonRow(c *myui.Context, key any, entry model.Entry, opt timetableDisplayOptions, onClick func()) {
	p := mainPalette(c)
	t := c.Theme()
	shownLocation := mainEntryLocation(entry, opt)
	row := myui.ButtonBase(c.Key(key)).Padding(10, 14).Gap(14).Radius(6).
		Background(p.cardSecondary).Border(1, p.hairline).
		AlignItems(myui.Center).Justify(myui.Start)
	if row.Pressed() {
		row.Background(p.navPressed)
	} else if row.Hovered() {
		row.Background(p.cardHover).BorderColor(p.cardBorder)
	}
	row.Children(func() {
		// 极简黑灰/少蓝线条层次：左侧 2px 主题蓝细线 + 单色节次时间 + 高对比课程名。
		myui.Box(c).Width(2).Height(32).Radius(1).Background(t.Accent.Alpha(0.75)).Shrink(0)
		myui.Column(c).Width(86).Shrink(0).Gap(2).Children(func() {
			myui.Text(c, "第 "+mainTextOr(entry.Periods, "—")+" 节").FontSize(12).FontWeight(600).TextColor(t.Text)
			if entry.Time != "" {
				myui.Text(c, entry.Time).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
			}
		})
		myui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			nameText := myui.Text(c, mainTextOr(entry.Name, "未命名课程")).FontSize(14).FontWeight(600)
			if opt.hideOverflow {
				nameText.MaxLines(1)
			}
			details := make([]string, 0, 2)
			if shownLocation != "" {
				details = append(details, shownLocation)
			}
			if entry.Teacher != "" {
				details = append(details, entry.Teacher)
			}
			if len(details) > 0 {
				detailText := myui.Text(c, strings.Join(details, " · ")).FontSize(12).TextColor(t.TextMuted)
				if opt.hideOverflow {
					detailText.MaxLines(1)
				}
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

type courseColor struct{ fill, hover, border, accent, text myui.Color }

// 极简素色 + 少数蓝/冷灰线条方案：去掉糖果多色，保持纯白/深灰卡片 + 清晰细线框 + 2px 克制强调边。
func courseColorFor(name string, dark bool) courseColor {
	light := []courseColor{
		{myui.Hex("#ffffff"), myui.Hex("#f5f7fa"), myui.RGBA(0, 0, 0, 0.095), myui.Hex("#0067c0"), myui.Hex("#1a1a1a")},
		{myui.Hex("#fafbfc"), myui.Hex("#f2f4f7"), myui.RGBA(0, 0, 0, 0.095), myui.Hex("#3b4252"), myui.Hex("#1a1a1a")},
		{myui.Hex("#f6f9fd"), myui.Hex("#edf3fb"), myui.RGBA(0, 103, 192, 0.16), myui.Hex("#2b6cb0"), myui.Hex("#1a1a1a")},
		{myui.Hex("#f8f9fa"), myui.Hex("#eff1f4"), myui.RGBA(0, 0, 0, 0.095), myui.Hex("#5f6b7a"), myui.Hex("#1a1a1a")},
	}
	darkColors := []courseColor{
		{myui.Hex("#2c2c2c"), myui.Hex("#343434"), myui.RGBA(255, 255, 255, 0.095), myui.Hex("#60cdff"), myui.Hex("#f3f3f3")},
		{myui.Hex("#292b2f"), myui.Hex("#32353a"), myui.RGBA(255, 255, 255, 0.095), myui.Hex("#8c96a4"), myui.Hex("#f3f3f3")},
		{myui.Hex("#262d36"), myui.Hex("#2e3742"), myui.RGBA(96, 205, 255, 0.18), myui.Hex("#4ea8de"), myui.Hex("#f3f3f3")},
		{myui.Hex("#2b2b2b"), myui.Hex("#333333"), myui.RGBA(255, 255, 255, 0.095), myui.Hex("#a0a8b0"), myui.Hex("#f3f3f3")},
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

func mainBoolText(v bool, yes, no string) string {
	if v {
		return yes
	}
	return no
}
