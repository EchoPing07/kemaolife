//go:build windows

package ui

import (
	"sort"
	"strconv"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	coreTimetable "github.com/EchoPing07/kemaolife/internal/core/timetable"
	myui "github.com/egoist/mygo/ui"
)

func (w *mainWindow) paintTimetable(c *myui.Context) {
	t := c.Theme()
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
	myui.Column(c).Fill().Padding(24, 28, 18, 28).Gap(14).Children(func() {
		pageHeader(c, "课表", week, func() {
			if myui.Button(c.Key("week-prev"), "").Label("上一周").Tooltip("上一周").
				Disabled(w.state.Week <= 1 || w.state.TimetableLoading || w.state.Refreshing).
				Children(func() { myui.Icon(c, mainIcons.left) }).Clicked() {
				w.selectWeek(w.state.Week - 1)
			}
			if myui.Button(c.Key("week-current"), "当周").
				Disabled(w.state.TimetableLoading || w.state.Refreshing).Clicked() {
				w.selectWeek(0)
			}
			if myui.Button(c.Key("week-next"), "").Label("下一周").Tooltip("下一周").
				Disabled(w.state.TimetableLoading || w.state.Refreshing || (w.state.LastWeek > 0 && w.state.Week >= w.state.LastWeek)).
				Children(func() { myui.Icon(c, mainIcons.right) }).Clicked() {
				w.selectWeek(w.state.Week + 1)
			}
			if myui.PrimaryButton(c.Key("week-refresh"), "").Disabled(w.state.TimetableLoading || w.state.Refreshing).
				Children(func() {
					myui.Icon(c, mainIcons.refresh)
					myui.Text(c, "刷新课表")
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
	const sidebarW, pagePadX, titleH = 208, 56, 44
	gridW := maxFloat(820, windowW-sidebarW-pagePadX-2)
	gridH := maxFloat(540, windowH-titleH-148)
	const headerH, timeW = 54, 44
	rowH := (gridH - headerH) / 12
	dayW := (gridW - timeW) / 7

	myui.Column(c).Grow(1).Radius(8).Background(p.card).Border(1, p.cardBorder).Clip().Children(func() {
		myui.ScrollBoth(c).Grow(1).TrackScroll(&w.timetableScroll).Children(func() {
			myui.Box(c).Width(gridW).Height(gridH).Background(p.card).Children(func() {
				// 顶部表头底色条。
				myui.Box(c).Absolute().Left(0).Top(0).Width(gridW).Height(headerH).Background(p.cardSubtle)

				// 今天列的极淡底色贯穿全部节次，跨周查看时也能快速定位。
				for i := 0; i < 7 && i < len(w.state.Days); i++ {
					if !w.state.Days[i].Date.IsZero() && w.state.Days[i].Date.String() == w.state.TodayDate {
						myui.Box(c).Absolute().Left(timeW + float32(i)*dayW).Top(headerH).
							Width(dayW).Height(gridH - headerH).Background(p.accentSoft.Alpha(0.45))
					}
				}

				// 仅绘制轻盈水平辅助线与极淡纵向分隔，避免 Excel 式重网格线。
				myui.Box(c).Absolute().Left(0).Top(headerH).Width(gridW).Height(1).Background(p.hairline)
				for period := 0; period < 12; period++ {
					y := headerH + float32(period)*rowH
					myui.Text(c, strconv.Itoa(period+1)).Absolute().Left(0).Top(y + rowH/2 - 8).
						Width(timeW).TextAlign(myui.Center).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
					if period > 0 {
						lineColor := p.hairline.Alpha(0.7)
						if period == 4 || period == 8 {
							// 午休与晚课分界略微加重，方便快速扫读上午/下午/晚上。
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
						Width(dayW - 20).FontSize(12).Bold().TextColor(weekdayColor)
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
						Padding(7, 8).Gap(2).Radius(6).
						Border(1, cc.border).
						Clip().Label(detail).Tooltip(detail)
					if course.Hovered() {
						course.Background(cc.hover)
					} else {
						course.Background(cc.fill)
					}
					course.Children(func() {
						nameText := myui.Text(c, mainTextOr(block.entry.Name, "未命名课程")).FontSize(12).Bold().
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
	cc := courseColorFor(entry.Name, t.Dark)

	myui.DialogBase(c.Key("course-detail-dialog"), &w.courseDialogOpen, func(backdrop, panel myui.Element) {
		backdrop.Background(p.backdrop)
		panel.Width(420).Padding(22, 24).Gap(16).Radius(8).Background(p.card).
			Border(1, p.cardBorder).Shadow(0, 16, 36, 0, myui.RGBA(0, 0, 0, 0.22)).Children(func() {
			myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
				myui.Text(c, mainTextOr(entry.Name, "未命名课程")).FontSize(18).Bold().Grow(1)
				mainBadge(c, "第 "+mainTextOr(entry.Periods, "—")+" 节", cc.text, cc.fill, false)
				closeBtn := myui.ButtonBase(c.Key("course-dialog-x")).Size(28, 28).Radius(6).
					Center().Label("关闭弹窗")
				closeBtn.Children(func() {
					myui.Icon(c, mainIcons.close).FontSize(15).TextColor(t.TextMuted)
				})
				if closeBtn.Hovered() {
					closeBtn.Background(p.navHover)
				}
				if closeBtn.Clicked() {
					w.courseDialogOpen = false
				}
			})

			myui.Column(c).Padding(12, 14).Gap(10).Radius(6).Background(p.cardSubtle).Children(func() {
				if !day.Date.IsZero() {
					courseMetaRow(c, "日期", day.Date.String()+"  星期"+mainTextOr(day.WeekdayCn, "—"))
				}
				courseMetaRow(c, "节次", "第 "+mainTextOr(entry.Periods, "—")+" 节")
				if entry.Time != "" {
					courseMetaRow(c, "时间", entry.Time)
				}
				if entry.WeeksText != "" {
					courseMetaRow(c, "周次", entry.WeeksText)
				}
				courseMetaRow(c, "教师", mainTextOr(entry.Teacher, "—"))
				courseMetaRow(c, "地点", mainTextOr(entry.Location, "—"))
			})

			myui.Box(c).Height(1).Background(p.hairline)
			myui.Row(c).Justify(myui.End).Children(func() {
				if myui.PrimaryButton(c.Key("course-dialog-close"), "关闭").Clicked() {
					w.courseDialogOpen = false
				}
			})
		})
	})
}

func courseMetaRow(c *myui.Context, label, value string) {
	t := c.Theme()
	myui.Row(c).AlignItems(myui.Start).Gap(12).Children(func() {
		myui.Text(c, label).Width(46).Shrink(0).FontSize(12).TextColor(t.TextMuted)
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
