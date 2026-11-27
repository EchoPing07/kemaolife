//go:build windows

package ui

import (
	"strconv"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	myui "github.com/egoist/mygo/ui"
)

func (w *mainWindow) paintHome(c *myui.Context) {
	t := c.Theme()
	p := mainPalette(c)
	windowW, _ := c.Size()
	contentW := windowW - w.sidebarWidth() - 56
	sideW := float32(320)
	if contentW < 820 {
		sideW = 288
	}

	myui.Column(c).Fill().Padding(22, 28).Gap(16).Children(func() {
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
		myui.Column(c).Shrink(0).FillWidth().Gap(10).Children(func() {
			pageHeader(c, "首页", subtitle, func() {
				if fluentPrimaryButton(c, "refresh-all", "").Disabled(w.state.Refreshing).
					Children(func() {
						myui.Icon(c, mainIcons.refresh).FontSize(14)
						myui.Text(c, "刷新全部").FontSize(13).FontWeight(500)
					}).Clicked() && w.handler.OnRefreshAll != nil {
					w.handler.OnRefreshAll()
				}
			})
			mainLoading(c, w.state.Refreshing, "正在刷新课表、电费与校园网…")
		})

		myui.Row(c).Grow(1).Height(0).FillWidth().Gap(16).AlignItems(myui.Stretch).Children(func() {
			// 主区域给当天课节；右侧放自然高度的紧凑摘要卡，消除内部空洞留白。
			mainCard(c).Grow(1).MinWidth(0).Children(func() {
				myui.Row(c).Shrink(0).AlignItems(myui.Center).Gap(10).Children(func() {
					myui.Text(c, "今日课节").FontSize(15).FontWeight(600)
					if w.state.HasToday {
						myui.Text(c, strconv.Itoa(len(w.state.Today.Entries))+" 节课").FontSize(12).TextColor(t.TextMuted)
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
					displayOpt := w.timetableDisplay()
					myui.Scroll(c.Key("today-lessons")).Grow(1).Height(0).FillWidth().Gap(8).Children(func() {
						for i, entry := range w.state.Today.Entries {
							entry := entry
							mainLessonRow(c, i, entry, displayOpt, func() {
								w.openCourseDetail(w.state.Today, entry)
							})
						}
					})
				}
			})

			myui.Column(c).Width(sideW).Shrink(0).Gap(14).Children(func() {
				// 电费摘要卡（自然高度，黑白线条 + 高对比数值）
				mainCard(c).FillWidth().Padding(16, 18).Gap(12).Children(func() {
					myui.Row(c).AlignItems(myui.Center).Gap(8).Children(func() {
						myui.Icon(c, mainIcons.electricity).FontSize(16).TextColor(t.Text)
						myui.Text(c, "电费").FontSize(14).FontWeight(600)
						myui.Spacer(c)
						myui.Text(c, "约有 1 天延迟").FontSize(11).TextColor(t.TextMuted)
					})

					balance, unit := "—", ""
					if w.state.HasReport {
						balance = mainFormatFloat(w.state.Report.Status.Balance)
						unit = mainTextOr(w.state.Report.Status.Unit, "度")
					}
					myui.Row(c).AlignItems(myui.Baseline).Gap(6).Children(func() {
						myui.Text(c, balance).FontSize(30).FontWeight(600).TextColor(t.Text).FontFeatures("tnum")
						if unit != "" {
							myui.Text(c, unit).FontSize(13).TextColor(t.TextMuted)
						}
						myui.Spacer(c)
						if w.state.HasReport && w.state.Report.Status.EstimatedDays != nil {
							mainBadge(c, "预计可用 "+strconv.Itoa(*w.state.Report.Status.EstimatedDays)+" 天", t.TextMuted, p.cardSubtle, false)
						}
					})

					if w.state.HasReport {
						if days := w.state.Report.Usage.Days; len(days) > 0 {
							paintTodayMiniBars(c, days)
						}
						myui.Text(c, "数据截至 "+mainTextOr(mainTimeText(w.state.Report.Status.CollectedAt), "—")).
							FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
					}
					mainLoading(c, w.state.ElectricityLoading && !w.state.Refreshing, "电费查询中…")
					mainError(c, w.state.ElectricityError)
					myui.Box(c).Height(1).Background(p.hairline)
					myui.Row(c).AlignItems(myui.Center).Children(func() {
						if w.state.HasReport && w.state.Report.Status.DailyAvg != nil {
							myui.Text(c, "近 7 日均 "+mainFormatFloat(*w.state.Report.Status.DailyAvg)+" "+unit).
								FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
						}
						myui.Spacer(c)
						if mainSubtleLink(c, "today-to-electricity", "电费详情") {
							w.page = mainPageElectricity
						}
					})
				})

				// 校园网摘要卡（自然高度，黑白线条 + 仅小圆点状态指示）
				mainCard(c).FillWidth().Padding(16, 18).Gap(12).Children(func() {
					stateColor := campusStateColor(c, w.state.Campus.State)
					myui.Row(c).AlignItems(myui.Center).Gap(8).Children(func() {
						myui.Icon(c, mainIcons.network).FontSize(16).TextColor(t.Text)
						myui.Text(c, "校园网").FontSize(14).FontWeight(600)
						myui.Spacer(c)
						mainBadge(c, mainBoolText(w.state.AutoMode, "自动模式", "手动模式"), t.TextMuted, p.cardSubtle, false)
					})

					myui.Column(c).Gap(6).Children(func() {
						myui.Row(c).AlignItems(myui.Center).Gap(8).Children(func() {
							myui.Box(c).Size(8, 8).Radius(4).Background(stateColor)
							myui.Text(c, mainStateLabel(w.state.Campus.State)).FontSize(22).FontWeight(600).TextColor(t.Text)
						})
						myui.Text(c, "检查时间 "+mainTextOr(mainTimeText(w.state.Campus.CheckedAt), "尚未检查")).
							FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
					})

					mainLoading(c, w.state.CampusLoading && !w.state.Refreshing, "状态检查中…")
					mainError(c, w.state.CampusError)

					ready := !w.state.CampusLoading && !w.state.Refreshing
					canAuth := ready && w.state.Campus.State != model.CampusOffline
					myui.Row(c).Gap(8).AlignItems(myui.Center).Children(func() {
						if !w.state.AutoMode {
							if fluentButton(c, "today-campus-login", "登录").Disabled(!canAuth).Clicked() {
								w.campusAction("login")
							}
						}
						if fluentButton(c, "today-campus-probe", "立即探测").Disabled(!ready).Clicked() {
							w.campusAction("refresh")
						}
						myui.Spacer(c)
						if mainSubtleLink(c, "today-to-campus", "连接详情") {
							w.page = mainPageCampus
						}
					})
				})
			})
		})
	})
}

func paintTodayMiniBars(c *myui.Context, days []model.DayUsage) {
	p := mainPalette(c)
	t := c.Theme()
	start := max(0, len(days)-7)
	recent := days[start:]
	maxUse := 0.0
	for _, d := range recent {
		if d.Use > maxUse {
			maxUse = d.Use
		}
	}
	if maxUse <= 0 {
		maxUse = 1
	}
	accent := t.Accent
	track := p.cardSubtle
	myui.Column(c).Padding(8, 10).Gap(6).Radius(6).Background(p.cardSecondary).Border(1, p.hairline).Children(func() {
		myui.Row(c).AlignItems(myui.Center).Children(func() {
			myui.Text(c, "近期每日用电").FontSize(11).TextColor(t.TextMuted)
			myui.Spacer(c)
			last := recent[len(recent)-1]
			myui.Text(c, shortChartDate(last.Date)+" · "+mainFormatFloat(last.Use)).
				FontSize(11).TextColor(t.Text).FontFeatures("tnum")
		})
		myui.Box(c).Height(28).FillWidth().Draw(func(pt *myui.Painter, r myui.Rect) {
			n := len(recent)
			if n == 0 || r.W <= 0 {
				return
			}
			gap := float32(6)
			barW := (r.W - gap*float32(max(0, n-1))) / float32(n)
			if barW < 4 {
				barW = 4
			}
			for i, d := range recent {
				x := r.X + float32(i)*(barW+gap)
				pt.Fill(myui.Rect{X: x, Y: r.Y, W: barW, H: r.H}, track, 3)
				h := float32(d.Use/maxUse) * r.H
				if h < 3 && d.Use > 0 {
					h = 3
				}
				if h > r.H {
					h = r.H
				}
				col := accent.Alpha(0.55)
				if i == n-1 {
					col = accent
				}
				if h > 0 {
					pt.Fill(myui.Rect{X: x, Y: r.Y + r.H - h, W: barW, H: h}, col, 3)
				}
			}
		})
	})
}
