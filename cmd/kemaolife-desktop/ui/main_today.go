//go:build windows

package ui

import (
	"strconv"

	myui "github.com/egoist/mygo/ui"
)

func (w *mainWindow) paintToday(c *myui.Context) {
	t := c.Theme()
	myui.Column(c).Fill().Padding(24).Gap(18).Children(func() {
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

		myui.Row(c).Grow(1).Gap(18).AlignItems(myui.Stretch).Children(func() {
			// 主区域给当天课节；两个状态摘要放在固定宽度侧栏，避免首屏全是小卡片。
			mainCard(c).Grow(1).Children(func() {
				detail := ""
				if w.state.HasToday {
					detail = strconv.Itoa(len(w.state.Today.Entries)) + " 节课"
				}
				mainSectionTitle(c, "今日课节", detail)
				switch {
				case !w.state.HasToday:
					mainEmpty(c, "暂无今日课表", "可通过「刷新全部」同步本地课表快照")
				case !w.state.Today.InSemester:
					mainEmpty(c, "不在学期内", "当前日期没有学期内课程安排")
				case len(w.state.Today.Entries) == 0:
					mainEmpty(c, "今天没有课", "好好安排今天的时间")
				default:
					myui.Scroll(c.Key("today-lessons")).Grow(1).Gap(10).Children(func() {
						for _, entry := range w.state.Today.Entries {
							mainLessonRow(c, entry)
						}
					})
				}
				myui.Spacer(c)
				if myui.Button(c.Key("today-to-week"), "查看完整课表").Clicked() {
					w.page = mainPageTimetable
				}
			})

			myui.Column(c).Width(292).FillHeight().Gap(18).Children(func() {
				mainCard(c).Grow(1).Children(func() {
					mainSectionTitle(c, "电费", "约有 1 天延迟")
					balance, unit := "—", ""
					if w.state.HasReport {
						balance = mainFormatFloat(w.state.Report.Status.Balance)
						unit = mainTextOr(w.state.Report.Status.Unit, "度")
					}
					mainMetric(c, "剩余电费", balance, unit, t.Accent)
					if w.state.HasReport {
						if w.state.Report.Status.EstimatedDays != nil {
							myui.Text(c, "预计可用 "+strconv.Itoa(*w.state.Report.Status.EstimatedDays)+" 天").
								FontSize(13).TextColor(t.Text)
						}
						myui.Text(c, "数据截至\n"+mainTextOr(mainTimeText(w.state.Report.Status.CollectedAt), "—")).
							FontSize(11).TextColor(t.TextMuted)
					}
					mainLoading(c, w.state.ElectricityLoading && !w.state.Refreshing, "电费查询中…")
					mainError(c, w.state.ElectricityError)
					myui.Spacer(c)
					if myui.Button(c.Key("today-to-electricity"), "电费详情").Clicked() {
						w.page = mainPageElectricity
					}
				})
				mainCard(c).Grow(1).Children(func() {
					mainSectionTitle(c, "校园网", mainBoolText(w.state.AutoMode, "自动模式", "手动模式"))
					myui.Text(c, mainStateLabel(w.state.Campus.State)).FontSize(30).Bold().
						TextColor(campusStateColor(c, w.state.Campus.State))
					myui.Text(c, "检查时间\n"+mainTextOr(mainTimeText(w.state.Campus.CheckedAt), "尚未检查")).
						FontSize(11).TextColor(t.TextMuted)
					mainLoading(c, w.state.CampusLoading && !w.state.Refreshing, "状态检查中…")
					mainError(c, w.state.CampusError)
					myui.Spacer(c)
					if myui.Button(c.Key("today-to-campus"), "连接详情").Clicked() {
						w.page = mainPageCampus
					}
				})
			})
		})
	})
}
