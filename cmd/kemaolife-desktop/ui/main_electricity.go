//go:build windows

package ui

import (
	"strconv"
	"strings"

	myui "github.com/egoist/mygo/ui"
)

func (w *mainWindow) paintElectricity(c *myui.Context) {
	t := c.Theme()
	myui.Column(c).Fill().Padding(26).Gap(15).Children(func() {
		pageHeader(c, "电费", "余额、采集时间与每月每日用量", nil)
		myui.Row(c).Gap(10).Children(func() {
			myui.Text(c, "月份").TextColor(t.TextMuted)
			if myui.TextInput(c.Key("electricity-month"), &w.month).Placeholder("YYYY-MM").Label("电费月份").
				Width(140).Disabled(w.state.ElectricityLoading).Changed() {
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

		mainCard(c).Children(func() {
			mainSectionTitle(c, "电费摘要", mainTextOr(w.state.Report.Usage.YearMonth, "尚未查询"))
			myui.Grid(c).Columns(4).Gap(16).Children(func() {
				balance, unit := "—", ""
				monthUse, daily, days := "—", "—", "—"
				if w.state.HasReport {
					balance = mainFormatFloat(w.state.Report.Status.Balance)
					unit = mainTextOr(w.state.Report.Status.Unit, "度")
					monthUse = mainFormatFloat(w.state.Report.Usage.MonthUse)
					if w.state.Report.Status.DailyAvg != nil {
						daily = mainFormatFloat(*w.state.Report.Status.DailyAvg)
					}
					if w.state.Report.Status.EstimatedDays != nil {
						days = strconv.Itoa(*w.state.Report.Status.EstimatedDays)
					}
				}
				mainMetric(c, "剩余电费", balance, unit, t.Accent)
				mainMetric(c, "本月用量", monthUse, unit, t.Text)
				mainMetric(c, "近 7 日均量", daily, unit, t.Text)
				mainMetric(c, "预计可用", days, "天", t.Text)
			})
			if w.state.HasReport {
				myui.Divider(c)
				myui.Text(c, "数据截至 "+mainTextOr(mainTimeText(w.state.Report.Status.CollectedAt), "—")+
					"　·　用量统计至 "+mainTextOr(w.state.Report.Usage.DataUntil, "—")).
					FontSize(11).TextColor(t.TextMuted)
			}
		})

		if !w.state.HasReport {
			mainCard(c).Grow(1).Children(func() {
				mainEmpty(c, "暂无电费数据", "输入 YYYY-MM 查询指定月份，或直接查询当前电费")
			})
			return
		}
		mainCard(c).Grow(1).Padding(0).Gap(0).Children(func() {
			myui.Row(c).Padding(16, 18, 6, 18).Children(func() {
				myui.Text(c, "每日用量").FontSize(15).Bold()
				myui.Spacer(c)
				myui.Text(c, "按日期升序").FontSize(11).TextColor(t.TextMuted)
			})
			columns := []myui.TableColumn{
				{Title: "日期", Width: 220},
				{Title: "用量（" + mainTextOr(w.state.Report.Status.Unit, "度") + "）", Align: myui.End},
			}
			w.electricityRows.Key = func(i int) any { return w.state.Report.Usage.Days[i].Date }
			w.electricityRows.RowHeight = 32
			w.electricityRows.RoundRows = true
			myui.Table(c, &w.electricityRows, columns, len(w.state.Report.Usage.Days), func(row, col int) {
				day := w.state.Report.Usage.Days[row]
				switch col {
				case 0:
					myui.Text(c, day.Date).Padding(5, 14).SingleLine().FontFeatures("tnum")
				case 1:
					myui.Text(c, mainFormatFloat(day.Use)).Padding(5, 14).SingleLine().FontFeatures("tnum")
				}
			}).Grow(1).MinHeight(150).Children(func() {
				if len(w.state.Report.Usage.Days) == 0 {
					mainEmpty(c, "该月暂无每日用量", "")
				}
			})
		})
	})
}
