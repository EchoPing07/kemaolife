//go:build windows

package ui

import (
	"strconv"

	myui "github.com/egoist/mygo/ui"
)

func (w *mainWindow) paintToday(c *myui.Context) {
	t := c.Theme()
	p := mainPalette(c)
	myui.Column(c).Fill().Padding(24, 28).Gap(18).Children(func() {
		subtitle := mainTextOr(w.state.TodayDate, "校园信息一览")
		if w.state.HasToday {
			if w.state.Today.Week > 0 {
				subtitle += " · 第 " + strconv.Itoa(w.state.Today.Week) + " 周"
			}
			if w.state.Today.WeekdayCn != "" {
				subtitle += " · 星期" + w.state.Today.WeekdayCn
			}
			if !w.state.Today.InSemester {
				subtitle += " · 不在学期内"
			}
		}
		pageHeader(c, "今天", subtitle, func() {
			if myui.PrimaryButton(c.Key("refresh-all"), "").Disabled(w.state.Refreshing).
				Children(func() {
					myui.Icon(c, mainIcons.refresh)
					myui.Text(c, "刷新全部")
				}).Clicked() && w.handler.OnRefreshAll != nil {
				w.handler.OnRefreshAll()
			}
		})
		mainLoading(c, w.state.Refreshing, "正在刷新课表、电费与校园网…")

		myui.Row(c).Grow(1).Gap(16).AlignItems(myui.Stretch).Children(func() {
			// 主区域给当天课节；右侧放固定宽度摘要卡，保持桌面主次分明。
			mainCard(c).Grow(1).Children(func() {
				myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
					myui.Text(c, "今日课节").FontSize(15).Bold()
					if w.state.HasToday {
						mainBadge(c, strconv.Itoa(len(w.state.Today.Entries))+" 节课", t.Accent, p.accentSoft, false)
					}
					myui.Spacer(c)
					if mainSubtleLink(c, "today-to-week", "查看完整课表") {
						w.page = mainPageTimetable
					}
				})
				switch {
				case !w.state.HasToday:
					myui.Spacer(c)
					mainEmpty(c, "暂无今日课表", "点击右上角「刷新全部」同步本地课表快照")
					myui.Spacer(c)
				case !w.state.Today.InSemester:
					myui.Spacer(c)
					mainEmpty(c, "不在学期内", "当前日期没有学期内课程安排")
					myui.Spacer(c)
				case len(w.state.Today.Entries) == 0:
					myui.Spacer(c)
					mainEmpty(c, "今天没有课", "好好安排今天的时间")
					myui.Spacer(c)
				default:
					myui.Scroll(c.Key("today-lessons")).Grow(1).Gap(8).Children(func() {
						for i, entry := range w.state.Today.Entries {
							entry := entry
							mainLessonRow(c, i, entry, func() {
								w.openCourseDetail(w.state.Today, entry)
							})
						}
					})
				}
			})

			myui.Column(c).Width(316).FillHeight().Gap(16).Children(func() {
				mainCard(c).Grow(1).Children(func() {
					myui.Row(c).AlignItems(myui.Center).Gap(8).Children(func() {
						myui.Box(c).Size(28, 28).Radius(6).Center().Background(p.accentSoft).Children(func() {
							myui.Icon(c, mainIcons.electricity).FontSize(15).TextColor(t.Accent)
						})
						myui.Text(c, "电费").FontSize(14).Bold()
						myui.Spacer(c)
						myui.Text(c, "约有 1 天延迟").FontSize(11).TextColor(t.TextMuted)
					})

					balance, unit := "—", ""
					if w.state.HasReport {
						balance = mainFormatFloat(w.state.Report.Status.Balance)
						unit = mainTextOr(w.state.Report.Status.Unit, "度")
					}
					myui.Column(c).Gap(6).Padding(4, 0, 0, 0).Children(func() {
						myui.Text(c, "剩余电费").FontSize(12).TextColor(t.TextMuted)
						myui.Row(c).AlignItems(myui.Baseline).Gap(6).Children(func() {
							myui.Text(c, balance).FontSize(32).Bold().TextColor(t.Accent).FontFeatures("tnum")
							if unit != "" {
								myui.Text(c, unit).FontSize(13).TextColor(t.TextMuted)
							}
						})
					})

					if w.state.HasReport {
						if w.state.Report.Status.EstimatedDays != nil {
							myui.Row(c).Children(func() {
								mainBadge(c, "预计可用 "+strconv.Itoa(*w.state.Report.Status.EstimatedDays)+" 天", t.Text, p.cardSubtle, false)
							})
						}
						myui.Text(c, "数据截至 "+mainTextOr(mainTimeText(w.state.Report.Status.CollectedAt), "—")).
							FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
					}
					mainLoading(c, w.state.ElectricityLoading && !w.state.Refreshing, "电费查询中…")
					mainError(c, w.state.ElectricityError)
					myui.Spacer(c)
					myui.Box(c).Height(1).Background(p.hairline)
					myui.Row(c).Justify(myui.End).Children(func() {
						if mainSubtleLink(c, "today-to-electricity", "电费详情") {
							w.page = mainPageElectricity
						}
					})
				})

				mainCard(c).Grow(1).Children(func() {
					stateColor := campusStateColor(c, w.state.Campus.State)
					stateBg := campusStateSoftBg(c, w.state.Campus.State)
					myui.Row(c).AlignItems(myui.Center).Gap(8).Children(func() {
						myui.Box(c).Size(28, 28).Radius(6).Center().Background(stateBg).Children(func() {
							myui.Icon(c, mainIcons.network).FontSize(15).TextColor(stateColor)
						})
						myui.Text(c, "校园网").FontSize(14).Bold()
						myui.Spacer(c)
						mainBadge(c, mainBoolText(w.state.AutoMode, "自动模式", "手动模式"), t.TextMuted, p.cardSubtle, false)
					})

					myui.Column(c).Gap(8).Padding(4, 0, 0, 0).Children(func() {
						myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
							myui.Box(c).Size(10, 10).Radius(5).Background(stateColor)
							myui.Text(c, mainStateLabel(w.state.Campus.State)).FontSize(26).Bold().TextColor(stateColor)
						})
						myui.Text(c, "检查时间 "+mainTextOr(mainTimeText(w.state.Campus.CheckedAt), "尚未检查")).
							FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
					})

					mainLoading(c, w.state.CampusLoading && !w.state.Refreshing, "状态检查中…")
					mainError(c, w.state.CampusError)
					myui.Spacer(c)
					myui.Box(c).Height(1).Background(p.hairline)
					myui.Row(c).Justify(myui.End).Children(func() {
						if mainSubtleLink(c, "today-to-campus", "连接详情") {
							w.page = mainPageCampus
						}
					})
				})
			})
		})
	})
}
