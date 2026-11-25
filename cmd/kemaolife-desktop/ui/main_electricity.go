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
	myui.Column(c).Fill().Padding(24, 28).Gap(16).Children(func() {
		pageHeader(c, "电费", "余额、采集时间与每月每日用量", func() {
			myui.Text(c, "月份").FontSize(12).TextColor(t.TextMuted)
			if myui.TextInput(c.Key("electricity-month"), &w.month).Placeholder("YYYY-MM").Label("电费月份").
				Width(124).Disabled(w.state.ElectricityLoading).Changed() {
				w.monthDirty = true
			}
			if myui.Button(c.Key("month-query"), "查询月份").Disabled(w.state.ElectricityLoading || w.state.Refreshing).
				Clicked() && w.handler.OnElectricityMonth != nil {
				w.handler.OnElectricityMonth(strings.TrimSpace(w.month))
			}
			if myui.PrimaryButton(c.Key("electricity-poll"), "").Disabled(w.state.ElectricityLoading || w.state.Refreshing).
				Children(func() {
					myui.Icon(c, mainIcons.refresh)
					myui.Text(c, "立即查询电费")
				}).Clicked() && w.handler.OnElectricityPoll != nil {
				w.handler.OnElectricityPoll()
			}
		})
		mainLoading(c, w.state.ElectricityLoading, "正在查询电费…")
		mainError(c, w.state.ElectricityError)

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

		// 左右双栏布局：左侧固定电费摘要，右侧上方柱形/折线趋势图 + 下方明细表格。
		myui.Row(c).Grow(1).Gap(16).AlignItems(myui.Stretch).Children(func() {
			myui.Column(c).Width(308).FillHeight().Children(func() {
				mainCard(c).Grow(1).Children(func() {
					myui.Row(c).AlignItems(myui.Center).Gap(8).Children(func() {
						myui.Text(c, "电费摘要").FontSize(14).Bold()
						myui.Spacer(c)
						mainBadge(c, mainTextOr(w.state.Report.Usage.YearMonth, "尚未查询"), t.TextMuted, p.cardSubtle, false)
					})

					myui.Column(c).Padding(16, 18).Gap(6).Radius(6).Background(p.accentSoft).Children(func() {
						myui.Text(c, "剩余电费").FontSize(12).TextColor(t.TextMuted)
						myui.Row(c).AlignItems(myui.Baseline).Gap(6).Children(func() {
							myui.Text(c, balance).FontSize(32).Bold().TextColor(t.Accent).FontFeatures("tnum")
							if w.state.HasReport {
								myui.Text(c, unit).FontSize(13).TextColor(t.TextMuted)
							}
						})
					})

					myui.Column(c).Gap(8).Children(func() {
						electricityStatRow(c, "本月用量", monthUse, unit)
						electricityStatRow(c, "近 7 日均量", daily, unit)
						electricityStatRow(c, "预计可用", daysLeft, "天")
					})

					myui.Spacer(c)
					myui.Box(c).Height(1).Background(p.hairline)
					myui.Column(c).Gap(4).Children(func() {
						if w.state.HasReport {
							myui.Text(c, "数据截至 "+mainTextOr(mainTimeText(w.state.Report.Status.CollectedAt), "—")).
								FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
							myui.Text(c, "用量统计至 "+mainTextOr(w.state.Report.Usage.DataUntil, "—")).
								FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
						}
						myui.Text(c, "电费数据约有 1 天延迟").FontSize(11).TextColor(t.TextMuted)
					})
				})
			})

			mainCard(c).Grow(1).Padding(18, 20).Gap(12).Children(func() {
				myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
					myui.Text(c, "每日用量").FontSize(14).Bold()
					if w.state.HasReport {
						mainBadge(c, strconv.Itoa(len(w.state.Report.Usage.Days))+" 天记录", t.TextMuted, p.cardSubtle, false)
					}
					myui.Spacer(c)
					if w.state.HasReport && len(w.state.Report.Usage.Days) > 0 {
						myui.Segmented(c.Key("electricity-chart-mode"), &w.electricityChart, "柱形图", "折线图").
							Label("图表样式")
					}
				})

				if !w.state.HasReport {
					myui.Spacer(c)
					mainEmpty(c, "暂无电费数据", "输入 YYYY-MM 查询指定月份，或点击右上角「立即查询电费」")
					myui.Spacer(c)
					return
				}
				if len(w.state.Report.Usage.Days) == 0 {
					myui.Spacer(c)
					mainEmpty(c, "该月暂无每日用量", "")
					myui.Spacer(c)
					return
				}

				w.paintElectricityChart(c, unit)
				w.paintElectricityTable(c, unit, monthUse)
			})
		})
	})
}

func electricityStatRow(c *myui.Context, label, value, unit string) {
	p := mainPalette(c)
	t := c.Theme()
	myui.Row(c).Padding(11, 14).Radius(6).Background(p.cardSubtle).AlignItems(myui.Baseline).Children(func() {
		myui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
		myui.Spacer(c)
		myui.Text(c, value).FontSize(18).Bold().TextColor(t.Text).FontFeatures("tnum")
		if value != "—" && unit != "" {
			myui.Text(c, unit).FontSize(11).TextColor(t.TextMuted).Padding(0, 0, 0, 4)
		}
	})
}

func (w *mainWindow) paintElectricityChart(c *myui.Context, unit string) {
	p := mainPalette(c)
	t := c.Theme()
	days := w.state.Report.Usage.Days
	// electricityHover 采用 1-based（0 表示未悬停），保证结构体零值安全。
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
		padL float32 = 42
		padR float32 = 14
		padT float32 = 16
		padB float32 = 24
	)

	myui.Column(c).Height(228).Padding(12, 14, 10, 14).Gap(6).Radius(6).
		Background(p.cardSubtle.Alpha(0.6)).Border(1, p.cardBorder).Children(func() {
		// 图表顶栏：悬停时显示精确日期与数值，平时显示趋势与日均参考说明。
		myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
			if hoverIdx >= 0 && hoverIdx < len(days) {
				d := days[hoverIdx]
				myui.Text(c, d.Date+" "+dateWeekdayCn(d.Date)).FontSize(12).Bold().TextColor(t.Text).FontFeatures("tnum")
				myui.Text(c, mainFormatFloat(d.Use)+" "+unit).FontSize(12).Bold().TextColor(t.Accent).FontFeatures("tnum")
			} else {
				myui.Text(c, "每日用电趋势").FontSize(12).Bold().TextColor(t.Text)
				if hasAvg {
					myui.Row(c).AlignItems(myui.Center).Gap(5).Children(func() {
						myui.Box(c).Width(14).Height(2).Radius(1).Background(t.Warning.Alpha(0.85))
						myui.Text(c, "近 7 日均 "+mainFormatFloat(dailyAvg)+" "+unit).FontSize(11).
							TextColor(t.TextMuted).FontFeatures("tnum")
					})
				}
			}
			myui.Spacer(c)
			myui.Text(c, "单位："+unit).FontSize(11).TextColor(t.TextMuted)
		})

		// 绘图画布 + 透明列悬停热区。
		hoveredNow := 0
		chartMode := w.electricityChart
		activeHover := hoverIdx
		accent := t.Accent
		textMuted := t.TextMuted
		textColor := t.Text
		cardBg := p.card
		gridColor := p.hairline
		avgColor := t.Warning

		myui.Box(c).Grow(1).FillWidth().Draw(func(pt *myui.Painter, r myui.Rect) {
			plotX := r.X + padL
			plotY := r.Y + padT
			plotW := r.W - padL - padR
			plotH := r.H - padT - padB
			if plotW <= 20 || plotH <= 20 || len(days) == 0 {
				return
			}

			// 水平网格线与 Y 轴刻度（4 档：0, 1/3, 2/3, max）。
			const ticks = 3
			for i := 0; i <= ticks; i++ {
				frac := float32(i) / float32(ticks)
				y := plotY + plotH*(1-frac)
				lineC := gridColor
				if i == 0 {
					lineC = p.cardBorder
				}
				pt.Line(plotX, y, plotX+plotW, y, 1, lineC)
				val := yMax * float64(frac)
				lbl := formatAxisValue(val)
				tw, _ := pt.MeasureText(padL-6, myui.Span{Text: lbl, Size: 10})
				pt.Text(plotX-tw-6, y-6, lbl, 10, textMuted)
			}

			// 近 7 日均量虚线参考线。
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

			// X 轴日期标签：按像素间距防碰撞，避免末项与前一项贴近。
			maxLabels := max(2, int(plotW/56))
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
	})
}

func drawElectricityBars(pt *myui.Painter, days []model.DayUsage, points [][2]float32, plotY, plotH, slotW float32, hover int, accent, textColor myui.Color) {
	barW := minFloat(32, maxFloat(5, slotW*0.56))
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
		fill := accent.Alpha(0.80)
		if i == hover {
			fill = accent
		}
		radius := minFloat(4, barW/2)
		pt.FillRadii(myui.Rect{X: cx - barW/2, Y: cy, W: barW, H: barH}, [4]float32{radius, radius, 0, 0}, fill)

		if showAllValues || i == hover {
			lbl := mainFormatFloat(d.Use)
			tw, _ := pt.MeasureText(56, myui.Span{Text: lbl, Size: 10})
			ty := maxFloat(plotY-13, cy-14)
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
		// 折线下方柔和面积渐变。
		var area myui.Path
		area.MoveTo(points[0][0], bottomY)
		for _, ptXY := range points {
			area.LineTo(ptXY[0], ptXY[1])
		}
		area.LineTo(points[n-1][0], bottomY).Close()
		pt.FillPathGradient(&area, myui.LinearGradient{
			From:  accent.Alpha(0.24),
			To:    accent.Alpha(0.01),
			Angle: 180,
		})

		// 主折线路径。
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
			ty := maxFloat(plotY-13, cy-16)
			cVal := textColor.Alpha(0.78)
			if i == hover {
				cVal = accent
			}
			pt.Text(cx-tw/2, ty, lbl, 10, cVal)
		}
	}
}

func (w *mainWindow) paintElectricityTable(c *myui.Context, unit, monthUse string) {
	p := mainPalette(c)
	t := c.Theme()
	days := w.state.Report.Usage.Days
	const idxW, weekW, useW float32 = 48, 84, 146

	myui.Column(c).Grow(1).MinHeight(140).Radius(6).Background(p.card).Border(1, p.cardBorder).Clip().Children(func() {
		// 表头行：带底色与纵横网格线，形成清晰的桌面表格结构。
		myui.Row(c).Height(34).Background(p.cardSubtle).
			BorderWidth(0, 0, 1, 0).BorderColor(p.cardBorder).AlignItems(myui.Stretch).Children(func() {
			tableCell(c, idxW, false, myui.Center, true, func() {
				myui.Text(c, "#").FontSize(12).Bold().TextColor(t.TextMuted)
			})
			tableCell(c, 0, true, myui.Start, true, func() {
				myui.Text(c, "日期").FontSize(12).Bold().TextColor(t.TextMuted)
			})
			tableCell(c, weekW, false, myui.Center, true, func() {
				myui.Text(c, "星期").FontSize(12).Bold().TextColor(t.TextMuted)
			})
			tableCell(c, useW, false, myui.End, false, func() {
				myui.Text(c, "当日用量（"+unit+"）").FontSize(12).Bold().TextColor(t.TextMuted)
			})
		})

		// 表体滚动区：斑马纹 + 细格线 + 等宽数字。
		myui.Scroll(c.Key("electricity-table-body")).Grow(1).Children(func() {
			for i, day := range days {
				bg := p.card
				if i%2 == 1 {
					bg = p.cardSubtle.Alpha(0.55)
				}
				if w.electricityHover > 0 && i == w.electricityHover-1 {
					bg = p.accentSoft
				}
				row := myui.Row(c.Key(day.Date)).Height(34).Background(bg).
					BorderWidth(0, 0, 1, 0).BorderColor(p.hairline).AlignItems(myui.Stretch)
				if row.Hovered() {
					row.Background(p.cardHover)
				}
				row.Children(func() {
					tableCell(c, idxW, false, myui.Center, true, func() {
						myui.Text(c, strconv.Itoa(i+1)).FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
					})
					tableCell(c, 0, true, myui.Start, true, func() {
						myui.Text(c, day.Date).FontSize(13).TextColor(t.Text).SingleLine().FontFeatures("tnum")
					})
					tableCell(c, weekW, false, myui.Center, true, func() {
						myui.Text(c, dateWeekdayCn(day.Date)).FontSize(12).TextColor(t.TextMuted).SingleLine()
					})
					tableCell(c, useW, false, myui.End, false, func() {
						myui.Text(c, mainFormatFloat(day.Use)).FontSize(13).Bold().TextColor(t.Text).
							SingleLine().FontFeatures("tnum")
					})
				})
			}
		})

		// 表尾汇总行：固定在表格底部，与上方列严格对齐。
		myui.Row(c).Height(36).Background(p.cardSubtle).
			BorderWidth(1, 0, 0, 0).BorderColor(p.cardBorder).AlignItems(myui.Stretch).Children(func() {
			tableCell(c, 0, true, myui.Start, true, func() {
				myui.Text(c, "本月合计").FontSize(12).Bold().TextColor(t.Text)
			})
			tableCell(c, useW, false, myui.End, false, func() {
				myui.Text(c, monthUse+" "+unit).FontSize(13).Bold().TextColor(t.Accent).
					SingleLine().FontFeatures("tnum")
			})
		})
	})
}

func tableCell(c *myui.Context, width float32, grow bool, justify myui.Align, rightBorder bool, content func()) {
	p := mainPalette(c)
	cell := myui.Row(c).Padding(0, 14).AlignItems(myui.Center).Justify(justify)
	if grow {
		cell.Grow(1)
	} else {
		cell.Width(width).Shrink(0)
	}
	if rightBorder {
		cell.BorderWidth(0, 1, 0, 0).BorderColor(p.hairline)
	}
	cell.Children(content)
}

func niceChartMax(maxVal float64) float64 {
	if maxVal <= 0 {
		return 1
	}
	// 留出少量顶部余白并取易读刻度上限。
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
