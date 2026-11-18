//go:build windows

package ui

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

var (
	procBeginPaint     = user32.NewProc("BeginPaint")
	procEndPaint       = user32.NewProc("EndPaint")
	procIntersectClip  = gdi32.NewProc("IntersectClipRect")
	procSelectClipRgn  = gdi32.NewProc("SelectClipRgn")
	procSelectObject   = gdi32.NewProc("SelectObject")
	procCreatePen      = gdi32.NewProc("CreatePen")
	procRectangle      = gdi32.NewProc("Rectangle")
	procDrawText       = user32.NewProc("DrawTextW")
	procSetScrollInfo  = user32.NewProc("SetScrollInfo")
	procSetCursor      = user32.NewProc("SetCursor")
	procLoadHandCursor = user32.NewProc("LoadCursorW")
	procScreenToClient = user32.NewProc("ScreenToClient")
)

const (
	dtLeft        = 0x0000
	dtCenter      = 0x0001
	dtRight       = 0x0002
	dtVcenter     = 0x0004
	dtSingleLine  = 0x0020
	dtEndEllipsis = 0x8000
	dtWordBreak   = 0x0010
	dtNoPrefix    = 0x0800

	psSolid  = 0
	sbCtl    = 2
	sifRange = 0x0001
	sifPage  = 0x0002
	sifPos   = 0x0004
	sifAll   = sifRange | sifPage | sifPos

	idcHand = 32649
)

type mainPaintStruct struct {
	DC        uintptr
	Erase     int32
	Paint     RECT
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}

type mainScrollInfo struct {
	Size     uint32
	Mask     uint32
	Min      int32
	Max      int32
	Page     uint32
	Pos      int32
	TrackPos int32
}

var (
	mainFontMu    sync.Mutex
	mainFontCache = map[[3]int32]uintptr{}
)

func mainFont(hwnd uintptr, height, weight int32) uintptr {
	dpi := int32(96)
	if hwnd != 0 {
		if d, _, _ := procGetDpiForWindow.Call(hwnd); d != 0 {
			dpi = int32(d)
		}
	}
	key := [3]int32{dpi, height, weight}
	mainFontMu.Lock()
	defer mainFontMu.Unlock()
	if f := mainFontCache[key]; f != 0 {
		return f
	}
	face := strPtr("Microsoft YaHei UI")
	h := -int32(float64(height)*float64(dpi)/96 + 0.5)
	font, _, _ := procCreateFont.Call(uintptr(h), 0, 0, 0, uintptr(weight),
		0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(face)))
	mainFontCache[key] = font
	return font
}

type mainCanvas struct {
	hwnd uintptr
	dc   uintptr
	font uintptr
}

func (c mainCanvas) useFont(height, weight int32) mainCanvas {
	out := c
	if f := mainFont(c.hwnd, height, weight); f != 0 {
		out.font = f
		procSelectObject.Call(c.dc, f)
	}
	return out
}

func mainRect(x, y, right, bottom int32) RECT {
	return RECT{Left: x, Top: y, Right: right, Bottom: bottom}
}

// GDI 刷子 / 画笔进程级缓存：滚动等高频重绘下每次创建销毁是纯浪费，
// 调色板固定、数量有限，句柄可跨 DC 复用，随进程退出释放。
var (
	mainGdiMu   sync.Mutex
	mainBrushes = map[uintptr]uintptr{}
	mainPens    = map[uintptr]uintptr{}
)

func mainBrush(color uintptr) uintptr {
	mainGdiMu.Lock()
	defer mainGdiMu.Unlock()
	if b := mainBrushes[color]; b != 0 {
		return b
	}
	b, _, _ := procCreateSolidBr.Call(color)
	mainBrushes[color] = b
	return b
}

func mainPen(color uintptr) uintptr {
	mainGdiMu.Lock()
	defer mainGdiMu.Unlock()
	if p := mainPens[color]; p != 0 {
		return p
	}
	p, _, _ := procCreatePen.Call(psSolid, 1, color)
	mainPens[color] = p
	return p
}

func (c mainCanvas) fill(r RECT, color uintptr) {
	if brush := mainBrush(color); brush != 0 {
		procFillRect.Call(c.dc, uintptr(unsafe.Pointer(&r)), brush)
	}
}

func (c mainCanvas) line(x1, y1, x2, y2 int32, color uintptr) {
	pen := mainPen(color)
	if pen == 0 {
		return
	}
	old, _, _ := procSelectObject.Call(c.dc, pen)
	// MoveToEx + LineTo，保留旧画笔并在结尾恢复。
	procMoveToEx.Call(c.dc, uintptr(x1), uintptr(y1), 0)
	procLineTo.Call(c.dc, uintptr(x2), uintptr(y2))
	procSelectObject.Call(c.dc, old)
}

var (
	procMoveToEx = gdi32.NewProc("MoveToEx")
	procLineTo   = gdi32.NewProc("LineTo")
)

func (c mainCanvas) text(text string, r RECT, flags uint32, color uintptr) {
	if r.Right <= r.Left || r.Bottom <= r.Top {
		return
	}
	SetBkMode(c.dc, transparentMode)
	SetTextColor(c.dc, color)
	p := strPtr(text)
	procDrawText.Call(c.dc, uintptr(unsafe.Pointer(p)), ^uintptr(0),
		uintptr(unsafe.Pointer(&r)), uintptr(flags|dtNoPrefix))
}

func (c mainCanvas) label(text string, x, y, width, height int32, color uintptr) {
	c.text(text, mainRect(x, y, x+width, y+height), dtLeft|dtVcenter|dtSingleLine|dtEndEllipsis, color)
}

func (c mainCanvas) clip(r RECT) {
	procIntersectClip.Call(c.dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom))
}

func (c mainCanvas) clearClip() { procSelectClipRgn.Call(c.dc, 0) }

var (
	mainBackground = rgb(247, 248, 250)
	mainSidebar    = rgb(241, 244, 247)
	mainCard       = rgb(255, 255, 255)
	mainLine       = rgb(222, 227, 232)
	mainTextColor  = rgb(37, 48, 58)
	mainMuted      = rgb(105, 117, 127)
	mainAccent     = rgb(48, 104, 128)
)

func (w *mainWindow) paint() {
	var ps mainPaintStruct
	dc, _, _ := procBeginPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		return
	}
	defer procEndPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps)))

	var client RECT
	procGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&client)))
	c := mainCanvas{hwnd: w.hwnd, dc: dc}.useFont(15, 400)
	c.fill(client, mainBackground)
	c.fill(mainRect(0, 0, w.scale(194), client.Bottom), mainSidebar)
	c.line(w.scale(194), 0, w.scale(194), client.Bottom, mainLine)

	left, _, right, bottom := w.contentRect()
	brand := c.useFont(25, 700)
	brand.label("科贸生活", w.scale(22), w.scale(28), w.scale(156), w.scale(38), mainTextColor)
	c.useFont(13, 400).label("校园日常，安静呈现", w.scale(22), w.scale(69), w.scale(160), w.scale(22), mainMuted)

	switch w.page {
	case mainPageToday:
		w.paintToday(c, left, right, bottom)
	case mainPageTimetable:
		w.paintTimetable(c, left, right, bottom)
	case mainPageElectricity:
		w.paintElectricity(c, left, right, bottom)
	case mainPageCampus:
		w.paintCampus(c, left, right)
	case mainPageSettings:
		w.paintSettings(c, left, right)
	}
}

func (w *mainWindow) pageTitle(c mainCanvas, left, right int32, title, subtitle string) {
	c.useFont(24, 700).label(title, left, w.scale(24), right-left-w.scale(120), w.scale(36), mainTextColor)
	c.useFont(13, 400).label(subtitle, left, w.scale(65), right-left, w.scale(25), mainMuted)
}

func (w *mainWindow) card(c mainCanvas, r RECT, title string) {
	c.fill(r, mainCard)
	if pen := mainPen(mainLine); pen != 0 {
		old, _, _ := procSelectObject.Call(c.dc, pen)
		procRectangle.Call(c.dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom))
		procSelectObject.Call(c.dc, old)
	}
	c.useFont(16, 700).label(title, r.Left+w.scale(18), r.Top+w.scale(15),
		r.Right-r.Left-w.scale(36), w.scale(27), mainTextColor)
}

func (w *mainWindow) paintToday(c mainCanvas, left, right, bottom int32) {
	date := mainTextOr(w.state.TodayDate, w.state.Today.Date.String())
	subtitle := mainTextOr(date, "校园信息一览")
	// 周次 / 星期 / 是否学期内来自 DaySchedule DTO（docs/windows.md §3.1）。
	if w.state.HasToday {
		today := w.state.Today
		if today.Week > 0 {
			subtitle += " · 第 " + strconv.Itoa(today.Week) + " 周"
		}
		if today.WeekdayCn != "" {
			subtitle += " · 星期" + today.WeekdayCn
		}
		if !today.InSemester {
			subtitle += " · 不在学期内"
		}
	}
	w.pageTitle(c, left, right, "今天", subtitle)
	top := w.scale(112)
	cardBottom := bottom
	if cardBottom < top+w.scale(330) {
		cardBottom = top + w.scale(330)
	}
	gap := w.scale(16)
	width := (right - left - gap*2) / 3
	if width < 1 {
		return
	}
	for i := 0; i < 3; i++ {
		x := left + int32(i)*(width+gap)
		r := mainRect(x, top, x+width, cardBottom)
		switch i {
		case 0:
			w.card(c, r, "今日课表")
			w.paintTodaySchedule(c, r)
		case 1:
			w.card(c, r, "电费")
			w.paintTodayElectricity(c, r)
		case 2:
			w.card(c, r, "校园网")
			w.paintTodayCampus(c, r)
		}
	}
}

func (w *mainWindow) paintTodayCardStatus(c mainCanvas, card RECT, loading bool, errText string) {
	x := card.Left + w.scale(18)
	width := card.Right - card.Left - w.scale(36)
	if loading {
		c.useFont(12, 400).label("正在刷新…", x, card.Top+w.scale(40), width, w.scale(22), mainAccent)
	}
	if errText != "" {
		c.useFont(12, 400).text(errText, mainRect(x, card.Bottom-w.scale(74), card.Right-w.scale(18), card.Bottom-w.scale(12)),
			dtLeft|dtWordBreak|dtEndEllipsis, mainMuted)
	}
}

func (w *mainWindow) paintTodaySchedule(c mainCanvas, card RECT) {
	x, y, width := card.Left+w.scale(18), card.Top+w.scale(64), card.Right-card.Left-w.scale(36)
	if !w.state.HasToday {
		c.useFont(15, 400).label("暂无今日课表", x, y, width, w.scale(30), mainMuted)
		w.paintTodayCardStatus(c, card, w.state.TimetableLoading, w.state.TimetableError)
		return
	}
	entries := w.state.Today.Entries
	if !w.state.Today.InSemester {
		c.useFont(20, 600).label("不在学期内", x, y+w.scale(18), width, w.scale(38), mainTextColor)
		w.paintTodayCardStatus(c, card, w.state.TimetableLoading, w.state.TimetableError)
		return
	}
	c.useFont(14, 400).label("共 "+strconv.Itoa(len(entries))+" 节课", x, y, width, w.scale(28), mainAccent)
	y += w.scale(42)
	if len(entries) == 0 {
		c.useFont(20, 600).label("今天没有课", x, y+w.scale(18), width, w.scale(38), mainTextColor)
		return
	}
	for i, entry := range entries {
		if y+w.scale(48) > card.Bottom-w.scale(14) {
			c.useFont(13, 400).label("…", x, y, width, w.scale(22), mainMuted)
			break
		}
		c.useFont(14, 600).label(mainTextOr(entry.Name, "未命名课程"), x, y, width, w.scale(23), mainTextColor)
		detail := "第 " + mainTextOr(entry.Periods, "—") + " 节"
		if entry.Location != "" {
			detail += " · " + entry.Location
		}
		c.useFont(12, 400).label(detail, x, y+w.scale(23), width, w.scale(21), mainMuted)
		y += w.scale(50)
		if i >= 7 {
			break
		}
	}
}

func (w *mainWindow) paintTodayElectricity(c mainCanvas, card RECT) {
	x, y, width := card.Left+w.scale(18), card.Top+w.scale(72), card.Right-card.Left-w.scale(36)
	if !w.state.HasReport {
		c.useFont(15, 400).label("尚未获取电费数据", x, y, width, w.scale(32), mainMuted)
		w.paintTodayCardStatus(c, card, w.state.ElectricityLoading, w.state.ElectricityError)
		return
	}
	status := w.state.Report.Status
	c.useFont(29, 700).label(mainFormatFloat(status.Balance)+" "+mainTextOr(status.Unit, "度"),
		x, y, width, w.scale(48), mainAccent)
	c.useFont(13, 400).label("剩余电费", x, y+w.scale(51), width, w.scale(24), mainMuted)
	if status.EstimatedDays != nil {
		c.useFont(14, 400).label("预计可用 "+strconv.Itoa(*status.EstimatedDays)+" 天",
			x, y+w.scale(96), width, w.scale(28), mainTextColor)
	}
	c.useFont(12, 400).label("采集："+mainTextOr(status.CollectedAt, "—"),
		x, card.Bottom-w.scale(47), width, w.scale(24), mainMuted)
}

func (w *mainWindow) paintTodayCampus(c mainCanvas, card RECT) {
	x, y, width := card.Left+w.scale(18), card.Top+w.scale(76), card.Right-card.Left-w.scale(36)
	c.useFont(26, 700).label(mainStateLabel(w.state.Campus.State), x, y, width, w.scale(45), mainAccent)
	c.useFont(13, 400).label("当前状态", x, y+w.scale(51), width, w.scale(24), mainMuted)
	c.useFont(12, 400).label("检查："+mainTextOr(w.state.Campus.CheckedAt, "尚未检查"),
		x, y+w.scale(99), width, w.scale(24), mainMuted)
	w.paintTodayCardStatus(c, card, w.state.CampusLoading, w.state.CampusError)
}

type mainCourseBlock struct {
	day      int
	entry    model.Entry
	from, to int
	column   int
	columns  int
}

type mainCourseHit struct {
	rect  RECT
	day   model.DaySchedule
	entry model.Entry
}

func mainEntryRuns(entry model.Entry) [][2]int {
	s := strings.NewReplacer("，", ",", "、", ",", "—", "-", "－", "-", "~", "-").Replace(entry.Periods)
	var runs [][2]int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		bounds := strings.SplitN(part, "-", 2)
		from, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
		if err != nil || from < 1 || from > 12 {
			continue
		}
		to := from
		if len(bounds) == 2 {
			if v, e := strconv.Atoi(strings.TrimSpace(bounds[1])); e == nil {
				to = v
			}
		}
		if to < from {
			from, to = to, from
		}
		if to > 12 {
			to = 12
		}
		if from <= to {
			runs = append(runs, [2]int{from, to})
		}
	}
	return runs
}

func (w *mainWindow) courseBlocks() []mainCourseBlock {
	var all []mainCourseBlock
	for dayIndex := 0; dayIndex < 7; dayIndex++ {
		if dayIndex >= len(w.state.Days) {
			continue
		}
		day := w.state.Days[dayIndex]
		var blocks []mainCourseBlock
		for _, entry := range day.Entries {
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
		// 重叠连通分量是一组分栏；分量内贪心复用最早的空闲列。
		for start := 0; start < len(blocks); {
			end, clusterEnd := start+1, blocks[start].to
			for end < len(blocks) && blocks[end].from <= clusterEnd {
				if blocks[end].to > clusterEnd {
					clusterEnd = blocks[end].to
				}
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

var courseColors = []uintptr{
	rgb(225, 238, 243), rgb(232, 237, 249), rgb(239, 235, 247),
	rgb(237, 242, 229), rgb(249, 238, 224), rgb(247, 231, 234),
}

// courseColorFor 按课程名稳定取色：同一门课在任意周、任意天都是同一颜色，
// 便于跨周识别；名字为空时才用固定首色，不引入随机性。
func courseColorFor(name string) uintptr {
	if name == "" {
		return courseColors[0]
	}
	sum := 0
	for _, r := range name {
		sum += int(r)
	}
	return courseColors[sum%len(courseColors)]
}

func (w *mainWindow) paintTimetable(c mainCanvas, left, right, bottom int32) {
	week := "本周"
	if w.state.Week != 0 {
		week = "第 " + strconv.Itoa(w.state.Week) + " 周"
		if w.state.LastWeek > 0 {
			week += " / 共 " + strconv.Itoa(w.state.LastWeek) + " 周"
		}
	}
	w.pageTitle(c, left, right, "课表", week)
	if w.state.TimetableLoading {
		c.useFont(13, 400).label("正在加载…", right-w.scale(270), w.scale(70), w.scale(120), w.scale(25), mainAccent)
	}
	gridTop := w.scale(116)
	headerH, rowH := w.scale(52), w.scale(56)
	timeW := w.scale(58)
	gridRight := right - w.scale(21)
	dayW := (gridRight - left - timeW) / 7
	if dayW < 1 {
		return
	}
	gridRight = left + timeW + dayW*7
	bodyTop, bodyBottom := gridTop+headerH, bottom

	// 先画可滚动正文，再固定画表头，避免课程块滚到表头上。
	c.clip(mainRect(left, bodyTop, gridRight+1, bodyBottom))
	c.fill(mainRect(left, bodyTop-w.timetableScroll, gridRight, bodyTop+rowH*12-w.timetableScroll), mainCard)
	for i := 0; i < 12; i++ {
		y := bodyTop + int32(i)*rowH - w.timetableScroll
		c.useFont(13, 400).text(strconv.Itoa(i+1), mainRect(left, y, left+timeW, y+rowH),
			dtCenter|dtVcenter|dtSingleLine, mainMuted)
	}
	c.clearClip()

	w.courseHits = w.courseHits[:0]
	for _, block := range w.courseBlocks() {
		x0 := left + timeW + int32(block.day)*dayW
		columnW := dayW / int32(block.columns)
		x := x0 + int32(block.column)*columnW + w.scale(2)
		y := bodyTop + int32(block.from-1)*rowH - w.timetableScroll + w.scale(2)
		r := mainRect(x, y, x0+int32(block.column+1)*columnW-w.scale(2),
			bodyTop+int32(block.to)*rowH-w.timetableScroll-w.scale(2))
		if r.Bottom <= bodyTop || r.Top >= bodyBottom {
			continue
		}
		visible := r
		if visible.Top < bodyTop {
			visible.Top = bodyTop
		}
		if visible.Bottom > bodyBottom {
			visible.Bottom = bodyBottom
		}
		c.clip(visible)
		c.fill(r, courseColorFor(block.entry.Name))
		name := mainTextOr(block.entry.Name, "未命名课程")
		textRect := mainRect(r.Left+w.scale(7), r.Top+w.scale(5), r.Right-w.scale(6), r.Bottom-w.scale(4))
		c.useFont(13, 600).text(name, textRect, dtLeft|dtWordBreak|dtEndEllipsis, mainTextColor)
		if block.to-block.from >= 1 && columnW >= w.scale(78) && r.Bottom-r.Top >= w.scale(86) {
			loc := mainTextOr(block.entry.Location, "—")
			c.useFont(11, 400).text(loc, mainRect(r.Left+w.scale(7), r.Bottom-w.scale(29),
				r.Right-w.scale(6), r.Bottom-w.scale(6)), dtLeft|dtSingleLine|dtEndEllipsis, mainMuted)
		}
		c.clearClip()
		w.courseHits = append(w.courseHits, mainCourseHit{rect: visible, day: w.state.Days[block.day], entry: block.entry})
	}

	// 网格线最后画，保证课程块的边界仍清楚。
	c.clip(mainRect(left, bodyTop, gridRight+1, bodyBottom))
	for i := 0; i <= 12; i++ {
		y := bodyTop + int32(i)*rowH - w.timetableScroll
		c.line(left, y, gridRight, y, mainLine)
	}
	for i := 0; i <= 7; i++ {
		x := left + timeW + int32(i)*dayW
		c.line(x, bodyTop, x, bodyBottom, mainLine)
	}
	c.line(left, bodyTop, left, bodyBottom, mainLine)
	c.clearClip()

	c.fill(mainRect(left, gridTop, gridRight, bodyTop), rgb(250, 251, 252))
	c.useFont(13, 600).text("节次", mainRect(left, gridTop, left+timeW, bodyTop), dtCenter|dtVcenter|dtSingleLine, mainMuted)
	weekdays := []string{"一", "二", "三", "四", "五", "六", "日"}
	for i := 0; i < 7; i++ {
		label := "周" + weekdays[i]
		if i < len(w.state.Days) {
			day := w.state.Days[i]
			if day.WeekdayCn != "" {
				label = "周" + day.WeekdayCn
			}
			if !day.Date.IsZero() {
				label += "\n" + day.Date.String()[5:]
			}
		}
		x := left + timeW + int32(i)*dayW
		c.useFont(12, 600).text(label, mainRect(x, gridTop+w.scale(4), x+dayW, bodyTop-w.scale(3)),
			dtCenter|dtVcenter|dtWordBreak, mainTextColor)
	}
	c.line(left, gridTop, gridRight, gridTop, mainLine)
	c.line(left, bodyTop, gridRight, bodyTop, mainLine)
	for i := 0; i <= 7; i++ {
		x := left + timeW + int32(i)*dayW
		c.line(x, gridTop, x, bodyTop, mainLine)
	}

	status := "同步于 " + mainTextOr(w.state.TimetableSyncedAt, "—")
	if w.state.TimetableOffline {
		status = "离线数据 · " + status
	}
	if w.state.TimetableError != "" {
		status += " · " + w.state.TimetableError
	}
	if len(w.state.Days) == 0 && !w.state.TimetableLoading {
		c.useFont(18, 400).text("暂无课表，点击“刷新”同步", mainRect(left+w.scale(90), bodyTop+w.scale(90),
			gridRight-w.scale(40), bodyTop+w.scale(140)), dtCenter|dtSingleLine, mainMuted)
	}
	// 状态放在工具栏右侧，避免占用课表最后一行。
	c.useFont(11, 400).label(status, left+w.scale(350), w.scale(75), gridRight-left-w.scale(350), w.scale(20), mainMuted)
}

func (w *mainWindow) paintElectricity(c mainCanvas, left, right, bottom int32) {
	w.pageTitle(c, left, right, "电费", "余额、采集时间与每月每日用量")
	c.useFont(14, 400).label("月份", left, w.scale(75), w.scale(70), w.scale(24), mainMuted)
	if w.state.ElectricityLoading {
		c.useFont(13, 400).label("正在查询…", left+w.scale(466), w.scale(77), w.scale(130), w.scale(24), mainAccent)
	}
	card := mainRect(left, w.scale(118), right, w.scale(276))
	w.card(c, card, "电费摘要")
	if !w.state.HasReport {
		c.useFont(18, 400).label("暂无数据，请输入 YYYY-MM 后查询", left+w.scale(20), w.scale(174),
			right-left-w.scale(40), w.scale(36), mainMuted)
	} else {
		report := w.state.Report
		c.useFont(28, 700).label(mainFormatFloat(report.Status.Balance)+" "+mainTextOr(report.Status.Unit, "度"),
			left+w.scale(20), w.scale(166), w.scale(210), w.scale(42), mainAccent)
		c.useFont(13, 400).label("剩余", left+w.scale(22), w.scale(211), w.scale(180), w.scale(24), mainMuted)
		c.useFont(18, 600).label(mainFormatFloat(report.Usage.MonthUse),
			left+w.scale(250), w.scale(172), w.scale(145), w.scale(34), mainTextColor)
		c.useFont(12, 400).label("本月用量", left+w.scale(250), w.scale(211), w.scale(145), w.scale(24), mainMuted)
		daily := "—"
		if report.Status.DailyAvg != nil {
			daily = mainFormatFloat(*report.Status.DailyAvg)
		}
		c.useFont(18, 600).label(daily, left+w.scale(414), w.scale(172), w.scale(135), w.scale(34), mainTextColor)
		c.useFont(12, 400).label("近 7 日均量", left+w.scale(414), w.scale(211), w.scale(135), w.scale(24), mainMuted)
		// 「数据截至」绑定采集时刻 collectedAt（docs/windows.md §3.3）；
		// dataUntil 是用量统计的截止日期，措辞与 Android 端一致。
		meta := "数据截至：" + mainTextOr(report.Status.CollectedAt, "—") +
			"    用量统计至：" + mainTextOr(report.Usage.DataUntil, "—")
		c.useFont(12, 400).label(meta, left+w.scale(570), w.scale(176), right-left-w.scale(590), w.scale(50), mainMuted)
	}
	if w.state.ElectricityError != "" {
		c.useFont(12, 400).label(w.state.ElectricityError, left+w.scale(570), w.scale(230),
			right-left-w.scale(590), w.scale(25), mainMuted)
	}

	tableTop := w.scale(306)
	tableRight := right - w.scale(21)
	dateW := (tableRight - left) * 2 / 3
	c.fill(mainRect(left, tableTop, tableRight, tableTop+w.scale(32)), rgb(241, 244, 247))
	c.useFont(13, 600).label("日期", left+w.scale(14), tableTop, dateW-w.scale(18), w.scale(32), mainMuted)
	c.useFont(13, 600).text("用量", mainRect(left+dateW, tableTop, tableRight-w.scale(14), tableTop+w.scale(32)),
		dtRight|dtVcenter|dtSingleLine, mainMuted)
	bodyTop := tableTop + w.scale(32)
	rowH := w.scale(28)
	c.clip(mainRect(left, bodyTop, tableRight, bottom))
	for i, day := range w.state.Report.Usage.Days {
		y := bodyTop + int32(i)*rowH - w.electricScroll
		if y+rowH < bodyTop || y > bottom {
			continue
		}
		if i%2 == 1 {
			c.fill(mainRect(left, y, tableRight, y+rowH), rgb(251, 252, 253))
		}
		c.useFont(13, 400).label(day.Date, left+w.scale(14), y, dateW-w.scale(18), rowH, mainTextColor)
		c.useFont(13, 400).text(mainFormatFloat(day.Use), mainRect(left+dateW, y, tableRight-w.scale(14), y+rowH),
			dtRight|dtVcenter|dtSingleLine, mainTextColor)
		c.line(left, y+rowH, tableRight, y+rowH, mainLine)
	}
	c.clearClip()
	c.line(left, bodyTop, tableRight, bodyTop, mainLine)
	if w.state.HasReport && len(w.state.Report.Usage.Days) == 0 {
		c.useFont(15, 400).label("该月暂无每日用量", left, bodyTop+w.scale(28), tableRight-left, w.scale(30), mainMuted)
	}
}

func (w *mainWindow) paintCampus(c mainCanvas, left, right int32) {
	w.pageTitle(c, left, right, "校园网", "只展示连接状态与检查时间")
	card := mainRect(left, w.scale(112), right, w.scale(404))
	w.card(c, card, "连接状态")
	c.useFont(32, 700).label(mainStateLabel(w.state.Campus.State), left+w.scale(22), w.scale(164),
		right-left-w.scale(44), w.scale(52), mainAccent)
	if w.state.CampusLoading {
		c.useFont(13, 400).label("正在执行，请稍候…", left+w.scale(24), w.scale(224),
			right-left-w.scale(48), w.scale(25), mainMuted)
	}
	// 最近一次本机动作 / 轮询的脱敏错误（docs/windows.md §3.4）。
	if w.state.CampusError != "" {
		c.useFont(12, 400).text(w.state.CampusError, mainRect(left+w.scale(24), w.scale(254),
			right-w.scale(24), w.scale(300)), dtLeft|dtWordBreak|dtEndEllipsis, mainMuted)
	}
	c.useFont(13, 400).label("检查时间："+mainTextOr(w.state.Campus.CheckedAt, "尚未检查"),
		left+w.scale(24), w.scale(315), right-left-w.scale(48), w.scale(30), mainMuted)
	c.useFont(12, 400).label("账号、认证与轮询选项请在设置中修改。", left+w.scale(24), w.scale(354),
		right-left-w.scale(48), w.scale(24), mainMuted)
}

func (w *mainWindow) paintSettings(c mainCanvas, left, right int32) {
	// 表单本身是原生控件，这里只画页面标题；版本号放在提示行右侧，
	// 避免与控件区的「保存 / 还原」以及提示文本重叠。
	w.pageTitle(c, left, right, "设置", "账号、认证、轮询与提醒")
	if w.state.Version != "" {
		g := w.settingsGeometry()
		c.useFont(12, 400).text(w.state.Version,
			mainRect(right-w.scale(260), g.hintY, right, g.hintY+w.scale(24)),
			dtRight|dtSingleLine|dtEndEllipsis, mainMuted)
	}
}

func (w *mainWindow) courseAt(x, y int32) *mainCourseHit {
	for i := len(w.courseHits) - 1; i >= 0; i-- {
		hit := &w.courseHits[i]
		r := hit.rect
		if x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom {
			return hit
		}
	}
	return nil
}

func (w *mainWindow) updateScrollbars() {
	if w == nil {
		return
	}
	_, bodyTop, bottom, maximum := w.timetableMetrics()
	page := bottom - bodyTop
	if page < 0 {
		page = 0
	}
	if w.timetableScroll > maximum {
		w.timetableScroll = maximum
	}
	info := mainScrollInfo{Size: uint32(unsafe.Sizeof(mainScrollInfo{})), Mask: sifAll,
		Min: 0, Max: 12*w.scale(56) - 1, Page: uint32(page), Pos: w.timetableScroll}
	procSetScrollInfo.Call(w.ctl[IDMainTimetableScroll], sbCtl, uintptr(unsafe.Pointer(&info)), 1)
	Enable(w.ctl[IDMainTimetableScroll], maximum > 0)

	_, bodyTop, bottom, maximum = w.electricityMetrics()
	page = bottom - bodyTop
	if page < 0 {
		page = 0
	}
	if w.electricScroll > maximum {
		w.electricScroll = maximum
	}
	// 无用量行时 Max 不得为 -1（SCROLLINFO 范围非法）。
	electricMax := int32(len(w.state.Report.Usage.Days))*w.scale(28) - 1
	if electricMax < 0 {
		electricMax = 0
	}
	info = mainScrollInfo{Size: uint32(unsafe.Sizeof(mainScrollInfo{})), Mask: sifAll,
		Min: 0, Max: electricMax, Page: uint32(page), Pos: w.electricScroll}
	procSetScrollInfo.Call(w.ctl[IDMainElectricityScroll], sbCtl, uintptr(unsafe.Pointer(&info)), 1)
	Enable(w.ctl[IDMainElectricityScroll], maximum > 0)

	top, bottom := w.settingsViewport()
	g := w.settingsGeometry()
	viewport, maximum := settingsScrollMetrics(top, bottom, g.quietY+w.scale(30))
	if w.settingsScroll > maximum {
		w.settingsScroll = maximum
	}
	settingsMax := g.quietY + w.scale(30) - top - 1
	if settingsMax < 0 {
		settingsMax = 0
	}
	info = mainScrollInfo{Size: uint32(unsafe.Sizeof(mainScrollInfo{})), Mask: sifAll,
		Min: 0, Max: settingsMax, Page: uint32(viewport), Pos: w.settingsScroll}
	procSetScrollInfo.Call(w.ctl[IDMainSettingsScroll], sbCtl, uintptr(unsafe.Pointer(&info)), 1)
	Enable(w.ctl[IDMainSettingsScroll], maximum > 0)
}
