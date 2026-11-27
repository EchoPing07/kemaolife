//go:build windows

package ui

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	myui "github.com/egoist/mygo/ui"
)

const (
	electricityChartBar = iota
	electricityChartLine
)

func (w *mainWindow) paintElectricity(c *myui.Context) {
	t := c.Theme()
	p := mainPalette(c)
	windowW, _ := c.Size()
	contentW := windowW - w.sidebarWidth() - 56
	busy := w.state.ElectricityLoading || w.state.Refreshing

	subtitle := "余额、采集时间与每月每日用量（约有 1 天延迟）"
	if w.state.HasReport {
		subtitle = "数据截至 " + mainTextOr(mainTimeText(w.state.Report.Status.CollectedAt), "—") +
			" · 用量统计至 " + mainTextOr(w.state.Report.Usage.DataUntil, "—") + " · 约有 1 天延迟"
	}

	unit := "度"
	balance, monthUse, daily, daysLeft := "—", "—", "—", "—"
	if w.state.HasReport {
		unit = mainTextOr(w.state.Report.Status.Unit, "度")
		balance = mainFormatFloat(w.state.Report.Status.Balance)
		monthUse = mainFormatFloat(w.state.Report.Usage.MonthUse)
		if w.state.Report.Status.DailyAvg != nil {
			daily = mainFormatFloat(*w.state.Report.Status.DailyAvg)
		}
		if w.state.Report.Status.EstimatedDays != nil {
			daysLeft = strconv.Itoa(*w.state.Report.Status.EstimatedDays)
		}
	}

	myui.Column(c).Fill().Padding(22, 28, 18, 28).Gap(14).Children(func() {
		// 1. 干净页眉：只有标题/采集时间 + 单一主操作按钮。
		myui.Column(c).Shrink(0).FillWidth().Gap(10).Children(func() {
			pageHeader(c, "电费", subtitle, func() {
				if fluentPrimaryButton(c, "electricity-poll", "").Disabled(busy).
					Children(func() {
						myui.Icon(c, mainIcons.refresh).FontSize(14)
						myui.Text(c, "立即查询电费").FontSize(13).FontWeight(500)
					}).Clicked() && w.handler.OnElectricityPoll != nil {
					w.handler.OnElectricityPoll()
				}
			})
			mainLoading(c, w.state.ElectricityLoading, "正在查询电费…")
			mainError(c, w.state.ElectricityError)
		})

		// 2. 统一单卡 4 格摘要条：用极细竖线分格替代 4 个独立盒子的视觉割裂感。
		myui.Column(c).Shrink(0).FillWidth().Radius(8).Background(p.card).
			Border(1, p.cardBorder).Shadow(0, 1, 2, 0, p.cardShadow).Children(func() {
			myui.Row(c).FillWidth().Padding(10, 18).AlignItems(myui.Center).Gap(10).
				BorderWidth(0, 0, 1, 0).BorderColor(p.hairline).Children(func() {
				myui.Icon(c, mainIcons.electricity).FontSize(15).TextColor(t.Text)
				myui.Text(c, "电费摘要").FontSize(13).FontWeight(600).TextColor(t.Text)
				mainBadge(c, mainTextOr(w.state.Report.Usage.YearMonth, "尚未查询"), t.TextMuted, p.cardSubtle, false)
				myui.Spacer(c)
				myui.Text(c, "数据约有 1 天延迟").FontSize(11).TextColor(t.TextMuted)
			})
			myui.Row(c).FillWidth().AlignItems(myui.Stretch).Children(func() {
				electricityMetricCell(c, "剩余电费", balance, unit, true, true)
				electricityMetricCell(c, "本月用量", monthUse, unit, false, true)
				electricityMetricCell(c, "近 7 日均量", daily, unit, false, true)
				electricityMetricCell(c, "预计可用", daysLeft, "天", false, false)
			})
		})

		// 3. 主工作区：左侧趋势图 + 右侧每日明细，自适应窄/宽窗口。
		sideListW := float32(296)
		if contentW < 820 {
			sideListW = 248
		}
		chartW := contentW - sideListW - 14

		myui.Row(c).Grow(1).Height(0).FillWidth().Gap(14).AlignItems(myui.Stretch).Children(func() {
			// 左卡：趋势图与月份控制
			mainCard(c).Grow(1).MinWidth(0).Padding(16, 18).Gap(12).Children(func() {
				w.paintElectricityTrendHeader(c, unit, busy, chartW)

				if !w.state.HasReport {
					myui.Spacer(c)
					mainEmpty(c, "暂无电费数据", "可选择右上角月份点击「查询」，或点页面右上角「立即查询电费」")
					myui.Spacer(c)
					return
				}
				if len(w.state.Report.Usage.Days) == 0 {
					myui.Spacer(c)
					mainEmpty(c, "该月暂无每日用量", "试试切换到其他月份查看历史记录")
					myui.Spacer(c)
					return
				}

				w.paintElectricityChart(c, unit)
				w.paintElectricityStatsFooter(c, unit)
			})

			// 右卡：每日明细（窄表无竖线，悬浮与左侧图表联动高亮）
			mainCard(c).Width(sideListW).Shrink(0).Padding(16, 0, 0, 0).Gap(0).Clip().Children(func() {
				myui.Row(c).Shrink(0).FillWidth().Padding(0, 16, 12, 16).AlignItems(myui.Center).Gap(8).
					BorderWidth(0, 0, 1, 0).BorderColor(p.hairline).Children(func() {
					myui.Text(c, "每日明细").FontSize(14).FontWeight(600).TextColor(t.Text)
					if w.state.HasReport {
						mainBadge(c, strconv.Itoa(len(w.state.Report.Usage.Days))+" 天", t.TextMuted, p.cardSubtle, false)
					}
					myui.Spacer(c)
					myui.Text(c, unit).FontSize(11).TextColor(t.TextMuted)
				})

				if !w.state.HasReport || len(w.state.Report.Usage.Days) == 0 {
					myui.Column(c).Grow(1).Center().Padding(16).Children(func() {
						myui.Text(c, "暂无记录").FontSize(12).TextColor(t.TextMuted)
					})
					return
				}

				w.paintElectricityDetailList(c, unit, monthUse)
			})
		})
	})
}

func electricityMetricCell(c *myui.Context, label, value, unit string, emphasize, dividerRight bool) {
	p := mainPalette(c)
	t := c.Theme()
	cell := myui.Column(c).Grow(1).MinWidth(0).Padding(14, 18).Gap(4)
	if dividerRight {
		cell.BorderWidth(0, 1, 0, 0).BorderColor(p.hairline)
	}
	cell.Children(func() {
		myui.Text(c, label).FontSize(11).TextColor(t.TextMuted)
		myui.Row(c).AlignItems(myui.Baseline).Gap(4).Children(func() {
			valSize := float32(22)
			if emphasize {
				valSize = 26
			}
			myui.Text(c, value).FontSize(valSize).FontWeight(600).TextColor(t.Text).FontFeatures("tnum")
			if value != "—" && unit != "" {
				myui.Text(c, unit).FontSize(11).TextColor(t.TextMuted)
			}
		})
	})
}

func (w *mainWindow) paintElectricityTrendHeader(c *myui.Context, unit string, busy bool, cardW float32) {
	t := c.Theme()
	days := w.state.Report.Usage.Days
	hoverIdx := w.electricityHover - 1
	hasHover := hoverIdx >= 0 && hoverIdx < len(days)

	var dailyAvg float64
	hasAvg := w.state.Report.Status.DailyAvg != nil && *w.state.Report.Status.DailyAvg > 0
	if hasAvg {
		dailyAvg = *w.state.Report.Status.DailyAvg
	}

	renderTitle := func() {
		myui.Row(c).Shrink(0).AlignItems(myui.Center).Gap(10).Children(func() {
			myui.Text(c, "每日用电趋势").FontSize(14).FontWeight(600).TextColor(t.Text).SingleLine()
			if hasHover {
				d := days[hoverIdx]
				myui.Text(c, d.Date+" "+dateWeekdayCn(d.Date)+" · "+mainFormatFloat(d.Use)+" "+unit).
					FontSize(12).FontWeight(600).TextColor(t.Accent).SingleLine().FontFeatures("tnum")
			} else if hasAvg {
				myui.Row(c).AlignItems(myui.Center).Gap(5).Children(func() {
					myui.Box(c).Width(12).Height(1.5).Radius(1).Background(t.TextMuted.Alpha(0.65)).Shrink(0)
					myui.Text(c, "近 7 日均 "+mainFormatFloat(dailyAvg)+" "+unit).
						FontSize(11).TextColor(t.TextMuted).SingleLine().FontFeatures("tnum")
				})
			}
		})
	}

	renderMonthControls := func() {
		myui.Row(c).Shrink(0).AlignItems(myui.Center).Gap(6).Children(func() {
			if fluentIconButton(c, "month-prev", mainIcons.left, "上一月", false).Disabled(busy).Clicked() {
				w.shiftElectricityMonth(-1)
			}
			if fluentTextInput(c, "electricity-month", &w.month).Placeholder("YYYY-MM").Label("电费月份").
				Width(88).Disabled(w.state.ElectricityLoading).Changed() {
				w.monthDirty = true
			}
			if fluentIconButton(c, "month-next", mainIcons.right, "下一月", false).Disabled(busy).Clicked() {
				w.shiftElectricityMonth(1)
			}
			if fluentButton(c, "month-query", "查询").Disabled(busy).
				Clicked() && w.handler.OnElectricityMonth != nil {
				w.handler.OnElectricityMonth(strings.TrimSpace(w.month))
			}
		})
	}

	renderChartMode := func() {
		if w.state.HasReport && len(days) > 0 {
			fluentSegmented(c, "electricity-chart-mode", &w.electricityChart, "柱形图", "折线图").
				Label("图表样式")
		}
	}

	if cardW < 680 {
		myui.Column(c).Shrink(0).FillWidth().Gap(8).Children(func() {
			renderTitle()
			myui.Row(c).FillWidth().AlignItems(myui.Center).Gap(8).Children(func() {
				renderMonthControls()
				myui.Spacer(c)
				renderChartMode()
			})
		})
		return
	}
	myui.Row(c).Shrink(0).FillWidth().AlignItems(myui.Center).Gap(10).Children(func() {
		renderTitle()
		myui.Spacer(c)
		myui.Row(c).Shrink(0).AlignItems(myui.Center).Gap(8).Children(func() {
			renderMonthControls()
			renderChartMode()
		})
	})
}

func (w *mainWindow) shiftElectricityMonth(delta int) {
	base := strings.TrimSpace(w.month)
	if base == "" {
		base = mainDisplayMonth(w.state)
	}
	parsed, err := time.Parse("2006-01", base)
	if err != nil {
		return
	}
	next := parsed.AddDate(0, delta, 0).Format("2006-01")
	w.month = next
	w.monthDirty = true
	if w.handler.OnElectricityMonth != nil {
		w.handler.OnElectricityMonth(next)
	}
}

func (w *mainWindow) paintElectricityChart(c *myui.Context, unit string) {
	p := mainPalette(c)
	t := c.Theme()
	days := w.state.Report.Usage.Days
	hoverIdx := w.electricityHover - 1
	if hoverIdx < 0 || hoverIdx >= len(days) {
		hoverIdx = -1
	}

	maxUse := 0.0
	for _, d := range days {
		if d.Use > maxUse {
			maxUse = d.Use
		}
	}
	var dailyAvg float64
	hasAvg := w.state.Report.Status.DailyAvg != nil && *w.state.Report.Status.DailyAvg > 0
	if hasAvg {
		dailyAvg = *w.state.Report.Status.DailyAvg
		if dailyAvg > maxUse {
			maxUse = dailyAvg
		}
	}
	yMax := niceChartMax(maxUse)

	const (
		padL float32 = 38
		padR float32 = 10
		padT float32 = 18
		padB float32 = 24
	)

	hoveredNow := 0
	chartMode := w.electricityChart
	activeHover := hoverIdx
	accent := t.Accent
	textMuted := t.TextMuted
	textColor := t.Text
	cardBg := p.card
	gridColor := p.hairline
	axisLineColor := p.cardBorder
	avgColor := t.TextMuted

	myui.Box(c).Grow(1).MinHeight(140).FillWidth().Draw(func(pt *myui.Painter, r myui.Rect) {
		plotX := r.X + padL
		plotY := r.Y + padT
		plotW := r.W - padL - padR
		plotH := r.H - padT - padB
		if plotW <= 20 || plotH <= 20 || len(days) == 0 {
			return
		}

		const ticks = 3
		for i := 0; i <= ticks; i++ {
			frac := float32(i) / float32(ticks)
			y := plotY + plotH*(1-frac)
			lineC := gridColor
			if i == 0 {
				lineC = axisLineColor
			}
			pt.Line(plotX, y, plotX+plotW, y, 1, lineC)
			val := yMax * float64(frac)
			lbl := formatAxisValue(val)
			tw, _ := pt.MeasureText(padL-6, myui.Span{Text: lbl, Size: 10})
			pt.Text(plotX-tw-6, y-6, lbl, 10, textMuted)
		}

		if hasAvg && dailyAvg <= yMax {
			avgY := plotY + plotH*(1-float32(dailyAvg/yMax))
			var dash myui.Path
			dash.MoveTo(plotX, avgY).LineTo(plotX+plotW, avgY).Dash(0, 4, 4)
			pt.StrokePath(&dash, 1.2, avgColor.Alpha(0.75))
		}

		n := len(days)
		slotW := plotW / float32(n)
		points := make([][2]float32, n)
		for i, d := range days {
			cx := plotX + (float32(i)+0.5)*slotW
			ratio := float32(math.Max(0, math.Min(1, d.Use/yMax)))
			cy := plotY + plotH*(1-ratio)
			points[i] = [2]float32{cx, cy}
		}

		if chartMode == electricityChartBar {
			drawElectricityBars(pt, days, points, plotY, plotH, slotW, activeHover, accent, textColor)
		} else {
			drawElectricityLine(pt, days, points, plotY, plotH, activeHover, accent, cardBg, textColor, gridColor)
		}

		maxLabels := max(2, int(plotW/54))
		step := max(1, (n+maxLabels-1)/maxLabels)
		lastRight := plotX - 100
		for i, d := range days {
			if i%step != 0 && i != n-1 {
				continue
			}
			lbl := shortChartDate(d.Date)
			tw, _ := pt.MeasureText(50, myui.Span{Text: lbl, Size: 10})
			cx := points[i][0]
			lx := cx - tw/2
			if lx < plotX {
				lx = plotX
			}
			if lx+tw > plotX+plotW {
				lx = plotX + plotW - tw
			}
			if lx < lastRight+10 {
				continue
			}
			lastRight = lx + tw
			lblColor := textMuted
			if i == activeHover {
				lblColor = accent
			}
			pt.Text(lx, plotY+plotH+6, lbl, 10, lblColor)
		}
	}).Children(func() {
		myui.Row(c).Absolute().Left(padL).Right(padR).Top(padT).Bottom(padB).Children(func() {
			for i, d := range days {
				tip := d.Date + " " + dateWeekdayCn(d.Date) + "：" + mainFormatFloat(d.Use) + " " + unit
				slot := myui.Box(c.Key(struct {
					k string
					i int
				}{"chart-slot", i})).
					Grow(1).FillHeight().Tooltip(tip)
				if slot.Hovered() {
					hoveredNow = i + 1
				}
			}
		})
	})
	w.electricityHover = hoveredNow
}

func (w *mainWindow) paintElectricityStatsFooter(c *myui.Context, unit string) {
	p := mainPalette(c)
	t := c.Theme()
	days := w.state.Report.Usage.Days
	if len(days) == 0 {
		return
	}
	minDay, maxDay := days[0], days[0]
	for _, d := range days[1:] {
		if d.Use < minDay.Use {
			minDay = d
		}
		if d.Use > maxDay.Use {
			maxDay = d
		}
	}
	myui.Row(c).Shrink(0).FillWidth().Padding(10, 0, 0, 0).Gap(12).AlignItems(myui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(p.hairline).Children(func() {
		myui.Text(c, "峰值 "+mainFormatFloat(maxDay.Use)+" "+unit+"（"+shortChartDate(maxDay.Date)+"）").
			FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
		myui.Text(c, "·").FontSize(11).TextColor(t.TextMuted)
		myui.Text(c, "最低 "+mainFormatFloat(minDay.Use)+" "+unit+"（"+shortChartDate(minDay.Date)+"）").
			FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
		myui.Spacer(c)
		myui.Text(c, "共 "+strconv.Itoa(len(days))+" 天记录").FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
	})
}

func drawElectricityBars(pt *myui.Painter, days []model.DayUsage, points [][2]float32, plotY, plotH, slotW float32, hover int, accent, textColor myui.Color) {
	barW := minFloat(28, maxFloat(5, slotW*0.56))
	bottomY := plotY + plotH
	showAllValues := len(days) <= 10

	for i, d := range days {
		cx, cy := points[i][0], points[i][1]
		if i == hover {
			pt.Fill(myui.Rect{X: cx - slotW/2 + 1, Y: plotY, W: maxFloat(1, slotW-2), H: plotH}, accent.Alpha(0.08), 4)
		}
		barH := maxFloat(2, bottomY-cy)
		if d.Use <= 0 {
			barH = 2
			cy = bottomY - 2
		}
		fill := accent.Alpha(0.82)
		if i == hover {
			fill = accent
		}
		radius := minFloat(4, barW/2)
		pt.FillRadii(myui.Rect{X: cx - barW/2, Y: cy, W: barW, H: barH}, [4]float32{radius, radius, 0, 0}, fill)

		if showAllValues || i == hover {
			lbl := mainFormatFloat(d.Use)
			tw, _ := pt.MeasureText(56, myui.Span{Text: lbl, Size: 10})
			ty := maxFloat(plotY-14, cy-14)
			cVal := textColor.Alpha(0.78)
			if i == hover {
				cVal = accent
			}
			pt.Text(cx-tw/2, ty, lbl, 10, cVal)
		}
	}
}

func drawElectricityLine(pt *myui.Painter, days []model.DayUsage, points [][2]float32, plotY, plotH float32, hover int, accent, cardBg, textColor, gridColor myui.Color) {
	bottomY := plotY + plotH
	n := len(points)
	if n >= 2 {
		var area myui.Path
		area.MoveTo(points[0][0], bottomY)
		for _, ptXY := range points {
			area.LineTo(ptXY[0], ptXY[1])
		}
		area.LineTo(points[n-1][0], bottomY).Close()
		pt.FillPathGradient(&area, myui.LinearGradient{
			From:  accent.Alpha(0.22),
			To:    accent.Alpha(0.01),
			Angle: 180,
		})

		var line myui.Path
		line.Cap(myui.RoundCap).Join(myui.RoundJoin)
		line.MoveTo(points[0][0], points[0][1])
		for i := 1; i < n; i++ {
			line.LineTo(points[i][0], points[i][1])
		}
		pt.StrokePath(&line, 2.2, accent)
	}

	if hover >= 0 && hover < n {
		hx := points[hover][0]
		pt.Line(hx, plotY, hx, bottomY, 1, gridColor)
	}

	showAllValues := n <= 8
	for i, ptXY := range points {
		cx, cy := ptXY[0], ptXY[1]
		if i == hover {
			var outer myui.Path
			outer.Circle(cx, cy, 6.5)
			pt.FillPath(&outer, accent.Alpha(0.22))
			var dot myui.Path
			dot.Circle(cx, cy, 4)
			pt.FillPath(&dot, accent)
		} else if n <= 20 || i == 0 || i == n-1 {
			var dot myui.Path
			dot.Circle(cx, cy, 3.2)
			pt.FillPath(&dot, cardBg)
			pt.StrokePath(&dot, 1.8, accent)
		}

		if showAllValues || i == hover {
			lbl := mainFormatFloat(days[i].Use)
			tw, _ := pt.MeasureText(56, myui.Span{Text: lbl, Size: 10})
			ty := maxFloat(plotY-14, cy-16)
			cVal := textColor.Alpha(0.78)
			if i == hover {
				cVal = accent
			}
			pt.Text(cx-tw/2, ty, lbl, 10, cVal)
		}
	}
}

func (w *mainWindow) paintElectricityDetailList(c *myui.Context, unit, monthUse string) {
	p := mainPalette(c)
	t := c.Theme()
	days := w.state.Report.Usage.Days

	myui.Scroll(c.Key("electricity-table-body")).Grow(1).Height(0).FillWidth().Children(func() {
		for i, day := range days {
			bg := myui.Transparent
			if w.electricityHover > 0 && i == w.electricityHover-1 {
				bg = p.accentSoft
			}
			row := myui.Row(c.Key(day.Date)).FillWidth().Height(36).Padding(0, 16).Background(bg).
				BorderWidth(0, 0, 1, 0).BorderColor(p.hairline.Alpha(0.6)).
				AlignItems(myui.Center).Gap(8)
			if row.Hovered() {
				row.Background(p.cardHover)
			}
			row.Children(func() {
				myui.Text(c, day.Date).FontSize(12).TextColor(t.Text).SingleLine().FontFeatures("tnum")
				myui.Text(c, dateWeekdayCn(day.Date)).FontSize(11).TextColor(t.TextMuted).SingleLine()
				myui.Spacer(c)
				myui.Text(c, mainFormatFloat(day.Use)).FontSize(13).FontWeight(600).TextColor(t.Text).
					SingleLine().FontFeatures("tnum")
			})
		}
	})

	myui.Row(c).Shrink(0).FillWidth().Height(40).Padding(0, 16).Background(p.cardSecondary).
		BorderWidth(1, 0, 0, 0).BorderColor(p.hairline).AlignItems(myui.Center).Children(func() {
		myui.Text(c, "本月合计").FontSize(12).FontWeight(600).TextColor(t.Text)
		myui.Spacer(c)
		myui.Text(c, monthUse+" "+unit).FontSize(13).FontWeight(600).TextColor(t.Text).
			SingleLine().FontFeatures("tnum")
	})
}

func niceChartMax(maxVal float64) float64 {
	if maxVal <= 0 {
		return 1
	}
	target := maxVal * 1.15
	exp := math.Pow(10, math.Floor(math.Log10(target)))
	norm := target / exp
	switch {
	case norm <= 1.2:
		return 1.2 * exp
	case norm <= 1.5:
		return 1.5 * exp
	case norm <= 2:
		return 2 * exp
	case norm <= 3:
		return 3 * exp
	case norm <= 4.5:
		return 4.5 * exp
	case norm <= 6:
		return 6 * exp
	case norm <= 9:
		return 9 * exp
	default:
		return 12 * exp
	}
}

func formatAxisValue(v float64) string {
	if v == 0 {
		return "0"
	}
	if v >= 10 || math.Abs(v-math.Round(v)) < 0.05 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func shortChartDate(ymd string) string {
	s := strings.TrimSpace(ymd)
	if len(s) >= 10 {
		return s[5:10]
	}
	return s
}

func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func dateWeekdayCn(ymd string) string {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(ymd))
	if err != nil {
		return "—"
	}
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return names[parsed.Weekday()]
}
