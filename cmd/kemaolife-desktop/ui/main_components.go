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
	myui.Row(c).AlignItems(myui.Start).Gap(16).Children(func() {
		myui.Column(c).Grow(1).Gap(5).Children(func() {
			myui.Text(c, title).FontSize(26).Bold()
			if subtitle != "" {
				myui.Text(c, subtitle).FontSize(12).TextColor(c.Theme().TextMuted)
			}
		})
		if actions != nil {
			myui.Row(c).Gap(8).Children(actions)
		}
	})
}

func mainCard(c *myui.Context) myui.Element {
	p := mainPalette(c)
	return myui.Column(c).Padding(20).Gap(14).Radius(10).Background(p.card).
		Border(1, p.border).Shadow(0, 2, 8, 0, myui.RGBA(0, 0, 0, 0.035))
}

func mainSectionTitle(c *myui.Context, title, detail string) {
	myui.Row(c).Gap(12).Children(func() {
		myui.Text(c, title).FontSize(15).Bold()
		myui.Spacer(c)
		if detail != "" {
			myui.Text(c, detail).FontSize(11).TextColor(c.Theme().TextMuted)
		}
	})
}

func mainLoading(c *myui.Context, loading bool, text string) {
	if !loading {
		return
	}
	myui.Row(c).Gap(8).Children(func() {
		myui.Spinner(c).Label(text)
		myui.Text(c, text).FontSize(12).TextColor(c.Theme().TextMuted)
	})
}

func mainError(c *myui.Context, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	myui.Text(c, text).FontSize(12).TextColor(c.Theme().Danger).Selectable()
}

func mainEmpty(c *myui.Context, title, detail string) {
	myui.Column(c).Padding(28, 12).Gap(8).AlignItems(myui.Center).Children(func() {
		myui.Text(c, title).FontSize(16).Bold().TextColor(c.Theme().TextMuted)
		if detail != "" {
			myui.Text(c, detail).FontSize(12).TextColor(c.Theme().TextMuted).TextAlign(myui.Center)
		}
	})
}

func mainLessonRow(c *myui.Context, entry model.Entry) {
	p := mainPalette(c)
	myui.Row(c).Padding(12).Gap(14).Radius(8).Background(p.subtle).AlignItems(myui.Start).Children(func() {
		myui.Column(c).Width(76).Shrink(0).Gap(4).Children(func() {
			myui.Text(c, "第 "+mainTextOr(entry.Periods, "—")+" 节").Bold().TextColor(tAccent(c))
			if entry.Time != "" {
				myui.Text(c, entry.Time).FontSize(10).TextColor(c.Theme().TextMuted)
			}
		})
		myui.Column(c).Grow(1).Gap(5).Children(func() {
			myui.Text(c, mainTextOr(entry.Name, "未命名课程")).FontSize(14).Bold()
			details := make([]string, 0, 2)
			if entry.Location != "" {
				details = append(details, entry.Location)
			}
			if entry.Teacher != "" {
				details = append(details, entry.Teacher)
			}
			if len(details) > 0 {
				myui.Text(c, strings.Join(details, " · ")).FontSize(12).TextColor(c.Theme().TextMuted)
			}
		})
	})
}

type courseColor struct{ fill, edge myui.Color }

func courseColorFor(name string, dark bool) courseColor {
	light := []courseColor{
		{myui.Hex("#e7f1f8"), myui.Hex("#24739b")},
		{myui.Hex("#ecebfa"), myui.Hex("#6658ad")},
		{myui.Hex("#e8f4ec"), myui.Hex("#3d8753")},
		{myui.Hex("#fbf0df"), myui.Hex("#ad7024")},
		{myui.Hex("#f9e9ed"), myui.Hex("#ad5068")},
	}
	darkColors := []courseColor{
		{myui.Hex("#203c4a"), myui.Hex("#58a6c9")},
		{myui.Hex("#302d4b"), myui.Hex("#9a8ed2")},
		{myui.Hex("#233d2c"), myui.Hex("#67ae78")},
		{myui.Hex("#453723"), myui.Hex("#d09a4e")},
		{myui.Hex("#442b34"), myui.Hex("#ce7b90")},
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

func mainMetric(c *myui.Context, label, value, unit string, color myui.Color) {
	myui.Column(c).Gap(5).Children(func() {
		myui.Text(c, label).FontSize(11).TextColor(c.Theme().TextMuted)
		myui.Row(c).AlignItems(myui.Baseline).Gap(5).Children(func() {
			myui.Text(c, value).FontSize(25).Bold().TextColor(color).FontFeatures("tnum")
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
