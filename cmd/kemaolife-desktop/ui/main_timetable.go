//go:build windows

package ui

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	coreTimetable "github.com/EchoPing07/kemaolife/internal/core/timetable"
	myui "github.com/egoist/mygo/ui"
)

func (w *mainWindow) paintTimetable(c *myui.Context) {
	t := c.Theme()
	p := mainPalette(c)
	week := "本周"
	if w.state.Week > 0 {
		week = "第 " + strconv.Itoa(w.state.Week) + " 周"
		if w.state.LastWeek > 0 {
			week += " / 共 " + strconv.Itoa(w.state.LastWeek) + " 周"
		}
	}
	if w.state.TimetableSyncedAt != "" {
		week += " · 同步于 " + w.state.TimetableSyncedAt
	}
	if w.state.TimetableOffline {
		week += " · 离线数据"
	}
	myui.Column(c).Fill().Padding(22, 28, 16, 28).Gap(14).Children(func() {
		pageHeader(c, "课表", week, func() {
			busy := w.state.TimetableLoading || w.state.Refreshing
			myui.Row(c).AlignItems(myui.Center).Gap(2).Padding(2).Radius(6).
				Background(p.cardSubtle).Border(1, p.hairline).Children(func() {
				if fluentIconButton(c, "week-prev", mainIcons.left, "上一周", true).
					Disabled(w.state.Week <= 1 || busy).Clicked() {
					w.selectWeek(w.state.Week - 1)
				}
				if fluentSubtleButton(c, "week-current", "当周").Disabled(busy).Clicked() {
					w.selectWeek(0)
				}
				if fluentIconButton(c, "week-next", mainIcons.right, "下一周", true).
					Disabled(busy || (w.state.LastWeek > 0 && w.state.Week >= w.state.LastWeek)).Clicked() {
					w.selectWeek(w.state.Week + 1)
				}
			})
			if fluentPrimaryButton(c, "week-refresh", "").Disabled(busy).
				Children(func() {
					myui.Icon(c, mainIcons.refresh).FontSize(14)
					myui.Text(c, "刷新课表").FontSize(13).FontWeight(500)
				}).Clicked() && w.handler.OnTimetableRefresh != nil {
				w.handler.OnTimetableRefresh()
			}
		})
		mainLoading(c, w.state.TimetableLoading, "正在读取或同步课表…")
		mainError(c, w.state.TimetableError)
		if len(w.state.Days) == 0 {
			mainCard(c).Grow(1).Children(func() {
				myui.Spacer(c)
				if !w.state.TimetableLoading {
					mainEmpty(c, "暂无课表", "点击「刷新课表」同步；已有快照时切周不会联网")
				}
				myui.Spacer(c)
			})
			return
		}
		w.timetableGrid(c)
		myui.Text(c, "切周只读本地快照 · 点击课程查看完整信息").FontSize(11).TextColor(t.TextMuted)
	})
}

func (w *mainWindow) selectWeek(week int) {
	if week < 0 || (week == 0 && w.state.Week == 0) {
		return
	}
	if w.handler.OnTimetableWeek != nil {
		w.handler.OnTimetableWeek(week)
	}
}

type mainCourseBlock struct {
	day             int
	entry           model.Entry
	from, to        int
	column, columns int
}

func mainEntryRuns(entry model.Entry) [][2]int {
	periods := coreTimetable.ExpandPeriods(entry.Periods)
	var runs [][2]int
	// DTO 的节次由 core 解析；UI 只把连续节号组成绘制用的区间。
	for start := 0; start < len(periods); {
		end := start
		for end+1 < len(periods) && periods[end+1] == periods[end]+1 {
			end++
		}
		from, to := max(periods[start], 1), min(periods[end], 12)
		if from <= to {
			runs = append(runs, [2]int{from, to})
		}
		start = end + 1
	}
	return runs
}

func (w *mainWindow) courseBlocks() []mainCourseBlock {
	var all []mainCourseBlock
	for dayIndex := 0; dayIndex < 7 && dayIndex < len(w.state.Days); dayIndex++ {
		var blocks []mainCourseBlock
		for _, entry := range w.state.Days[dayIndex].Entries {
			for _, run := range mainEntryRuns(entry) {
				blocks = append(blocks, mainCourseBlock{day: dayIndex, entry: entry, from: run[0], to: run[1]})
			}
		}
		sort.SliceStable(blocks, func(i, j int) bool {
			if blocks[i].from != blocks[j].from {
				return blocks[i].from < blocks[j].from
			}
			return blocks[i].to < blocks[j].to
		})
		// 重叠课程组成一个连通分量，在分量内复用最早空闲的列。
		for start := 0; start < len(blocks); {
			end, clusterEnd := start+1, blocks[start].to
			for end < len(blocks) && blocks[end].from <= clusterEnd {
				clusterEnd = max(clusterEnd, blocks[end].to)
				end++
			}
			var columnEnds []int
			for i := start; i < end; i++ {
				column := -1
				for j, last := range columnEnds {
					if blocks[i].from > last {
						column = j
						break
					}
				}
				if column < 0 {
					column = len(columnEnds)
					columnEnds = append(columnEnds, blocks[i].to)
				} else {
					columnEnds[column] = blocks[i].to
				}
				blocks[i].column = column
			}
			for i := start; i < end; i++ {
				blocks[i].columns = len(columnEnds)
			}
			start = end
		}
		all = append(all, blocks...)
	}
	return all
}

func (w *mainWindow) timetableGrid(c *myui.Context) {
	p := mainPalette(c)
	t := c.Theme()
	windowW, windowH := c.Size()
	sidebarW := w.sidebarWidth()
	const pagePadX = 56
	titleH := titleBarDefaultH
	gridW := maxFloat(680, windowW-sidebarW-pagePadX-2)
	gridH := maxFloat(430, windowH-titleH-148)
	const headerH, timeW = 54, 44
	rowH := (gridH - headerH) / 12
	dayW := (gridW - timeW) / 7

	myui.Column(c).Grow(1).Radius(8).Background(p.card).Border(1, p.cardBorder).
		Shadow(0, 1, 3, 0, p.cardShadow).Clip().Children(func() {
		myui.ScrollBoth(c).Grow(1).TrackScroll(&w.timetableScroll).Children(func() {
			myui.Box(c).Width(gridW).Height(gridH).Children(func() {
				// 顶部表头底色条。
				myui.Box(c).Absolute().Left(0).Top(0).Width(gridW).Height(headerH).Background(p.cardSecondary)

				// 今天列的极淡底色贯穿全部节次，跨周查看时也能快速定位。
				for i := 0; i < 7 && i < len(w.state.Days); i++ {
					if !w.state.Days[i].Date.IsZero() && w.state.Days[i].Date.String() == w.state.TodayDate {
						myui.Box(c).Absolute().Left(timeW + float32(i)*dayW).Top(headerH).
							Width(dayW).Height(gridH - headerH).Background(p.accentSoft.Alpha(0.45))
					}
				}

				// 轻盈水平辅助线与极淡纵向分隔。
				myui.Box(c).Absolute().Left(0).Top(headerH).Width(gridW).Height(1).Background(p.hairline)
				for period := 0; period < 12; period++ {
					y := headerH + float32(period)*rowH
					myui.Text(c, strconv.Itoa(period+1)).Absolute().Left(0).Top(y + rowH/2 - 8).
						Width(timeW).TextAlign(myui.Center).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
					if period > 0 {
						lineColor := p.hairline.Alpha(0.75)
						if period == 4 || period == 8 {
							lineColor = p.cardBorder
						}
						myui.Box(c).Absolute().Left(timeW).Top(y).Width(gridW - timeW).Height(1).Background(lineColor)
					}
				}
				for i := 0; i < 7; i++ {
					x := timeW + float32(i)*dayW
					myui.Box(c).Absolute().Left(x).Top(headerH).Width(1).Height(gridH - headerH).Background(p.hairline.Alpha(0.55))
				}
				myui.Text(c, "节次").Absolute().Left(0).Top(19).Width(timeW).TextAlign(myui.Center).
					FontSize(11).TextColor(t.TextMuted)

				weekdays := []string{"一", "二", "三", "四", "五", "六", "日"}
				for i := 0; i < 7; i++ {
					day := model.DaySchedule{WeekdayCn: weekdays[i]}
					if i < len(w.state.Days) {
						day = w.state.Days[i]
					}
					today := !day.Date.IsZero() && day.Date.String() == w.state.TodayDate
					x := timeW + float32(i)*dayW
					date := day.Date.String()
					if len(date) >= 10 {
						date = date[5:]
					}
					if today {
						myui.Box(c).Absolute().Left(x).Top(0).Width(dayW).Height(headerH).Background(p.accentSoft)
						myui.Box(c).Absolute().Left(x + 12).Top(headerH - 2).Width(dayW - 24).Height(2).
							Radius(1).Background(t.Accent)
					}
					weekdayColor := t.Text
					dateColor := t.TextMuted
					if today {
						weekdayColor = t.Accent
						dateColor = t.Accent
					}
					myui.Text(c, "周"+mainTextOr(day.WeekdayCn, "—")).Absolute().Left(x + 10).Top(9).
						Width(dayW - 20).FontSize(12).FontWeight(600).TextColor(weekdayColor)
					myui.Text(c, date).Absolute().Left(x + 10).Top(28).Width(dayW - 20).
						FontSize(11).TextColor(dateColor).FontFeatures("tnum")
				}

				displayOpt := w.timetableDisplay()
				for blockIndex, block := range w.courseBlocks() {
					block := block
					columnW := dayW / float32(block.columns)
					x := timeW + float32(block.day)*dayW + float32(block.column)*columnW + 4
					y := headerH + float32(block.from-1)*rowH + 3
					width := columnW - 8
					height := float32(block.to-block.from+1)*rowH - 6
					spanRows := block.to - block.from + 1
					cc := courseColorFor(block.entry.Name, t.Dark)
					detail := mainDetailText(w.state.Days[block.day], block.entry)
					shownLocation := mainEntryLocation(block.entry, displayOpt)
					course := myui.ButtonBase(c.Key(struct{ day, from, column, index int }{
						block.day, block.from, block.column, blockIndex,
					})).Absolute().
						Left(x).Top(y).Width(width).Height(height).Column().
						AlignItems(myui.Start).Justify(myui.Start).
						Padding(7, 8, 7, 10).Gap(2).Radius(6).
						Border(1, cc.border).
						Clip().Label(detail).Tooltip(detail)
					if course.Hovered() {
						course.Background(cc.hover).Shadow(0, 2, 6, 0, p.cardShadow)
					} else {
						course.Background(cc.fill)
					}
					accentStrip := cc.accent
					course.DrawOver(func(pt *myui.Painter, r myui.Rect) {
						pt.FillRadii(myui.Rect{X: r.X, Y: r.Y, W: 2, H: r.H}, [4]float32{6, 0, 0, 6}, accentStrip)
					})
					course.Children(func() {
						nameText := myui.Text(c, mainTextOr(block.entry.Name, "未命名课程")).FontSize(12).FontWeight(600).
							TextColor(cc.text)
						if displayOpt.hideOverflow {
							switch {
							case spanRows < 2:
								nameText.MaxLines(1)
							case spanRows >= 4:
								nameText.MaxLines(4)
							default:
								nameText.MaxLines(2)
							}
						}
						myui.Text(c, mainTextOr(block.entry.Periods, "—")+" 节").FontSize(10).
							TextColor(cc.text.Alpha(0.76)).FontFeatures("tnum")
						if height >= 70 && shownLocation != "" {
							locText := myui.Text(c, shownLocation).FontSize(10).
								TextColor(cc.text.Alpha(0.72))
							if displayOpt.hideOverflow {
								if spanRows >= 4 {
									locText.MaxLines(2)
								} else {
									locText.MaxLines(1)
								}
							}
						}
					})
					if course.Clicked() {
						w.openCourseDetail(w.state.Days[block.day], block.entry)
					}
				}
			})
		})
	})
}

func (w *mainWindow) paintCourseDialog(c *myui.Context) {
	if !w.courseDialogOpen {
		return
	}
	p := mainPalette(c)
	t := c.Theme()
	entry := w.courseDialogItem
	day := w.courseDialogDay

	myui.DialogBase(c.Key("course-detail-dialog"), &w.courseDialogOpen, func(backdrop, panel myui.Element) {
		backdrop.Background(p.backdrop)
		panel.Width(440).Radius(8).Background(p.card).
			Border(1, p.cardBorder).Shadow(0, 16, 40, 0, myui.RGBA(0, 0, 0, 0.26)).Clip().
			Transition(myui.ElementTransition{
				Duration: 150 * time.Millisecond,
				Enter:    &myui.Motion{Scale: 0.96, Opacity: 0},
			}).Children(func() {
			myui.Column(c).Padding(22, 24).Gap(16).Children(func() {
				myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
					myui.Text(c, mainTextOr(entry.Name, "未命名课程")).FontSize(18).FontWeight(600).Grow(1)
					mainBadge(c, "第 "+mainTextOr(entry.Periods, "—")+" 节", t.TextMuted, p.cardSubtle, false)
					if fluentIconButton(c, "course-dialog-x", mainIcons.close, "关闭弹窗", true).Clicked() {
						w.courseDialogOpen = false
					}
				})

				myui.Column(c).Padding(12, 14).Gap(10).Radius(6).Background(p.cardSecondary).
					Border(1, p.hairline).Children(func() {
					if !day.Date.IsZero() {
						courseMetaRow(c, mainIcons.calendar, "日期", day.Date.String()+"  星期"+mainTextOr(day.WeekdayCn, "—"))
					}
					courseMetaRow(c, mainIcons.clock, "节次", "第 "+mainTextOr(entry.Periods, "—")+" 节")
					if entry.Time != "" {
						courseMetaRow(c, mainIcons.timer, "时间", entry.Time)
					}
					if entry.WeeksText != "" {
						courseMetaRow(c, mainIcons.bookOpen, "周次", entry.WeeksText)
					}
					courseMetaRow(c, mainIcons.user, "教师", mainTextOr(entry.Teacher, "—"))
					courseMetaRow(c, mainIcons.pin, "地点", mainTextOr(entry.Location, "—"))
				})
			})

			myui.Row(c).Padding(14, 24).Justify(myui.End).Background(p.dialogFooter).
				BorderWidth(1, 0, 0, 0).BorderColor(p.hairline).Children(func() {
				if fluentPrimaryButton(c, "course-dialog-close", "关闭").Clicked() {
					w.courseDialogOpen = false
				}
			})
		})
	})
}

func courseMetaRow(c *myui.Context, icon *myui.SVG, label, value string) {
	t := c.Theme()
	myui.Row(c).AlignItems(myui.Start).Gap(10).Children(func() {
		if icon != nil {
			myui.Icon(c, icon).FontSize(14).TextColor(t.TextMuted).Margin(2, 0, 0, 0).Shrink(0)
		}
		myui.Text(c, label).Width(42).Shrink(0).FontSize(12).TextColor(t.TextMuted)
		myui.Text(c, value).FontSize(13).TextColor(t.Text).Selectable().Grow(1).FontFeatures("tnum")
	})
}

func mainDetailText(day model.DaySchedule, entry model.Entry) string {
	lines := []string{mainTextOr(entry.Name, "未命名课程")}
	if !day.Date.IsZero() {
		lines = append(lines, "日期："+day.Date.String()+"  星期"+day.WeekdayCn)
	}
	lines = append(lines, "节次：第 "+mainTextOr(entry.Periods, "—")+" 节")
	if entry.Time != "" {
		lines = append(lines, "时间："+entry.Time)
	}
	if entry.WeeksText != "" {
		lines = append(lines, "周次："+entry.WeeksText)
	}
	lines = append(lines, "教师："+mainTextOr(entry.Teacher, "—"), "地点："+mainTextOr(entry.Location, "—"))
	return strings.Join(lines, "\n")
}
