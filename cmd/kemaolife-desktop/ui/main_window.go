//go:build windows

package ui

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/desktop"
)

// MainWindowState 是主窗口的完整只读快照。
//
// 状态只由 main 在 UI 线程通过 UpdateMainWindow 写入。UI 不查询业务接口，
// 也不从 desktop.Status 反推 DTO；切片会在接收时复制，避免调用方复用缓冲。
type MainWindowState struct {
	TodayDate string
	Today     model.DaySchedule
	HasToday  bool

	Week              int
	LastWeek          int
	Days              []model.DaySchedule
	TimetableSyncedAt string
	TimetableOffline  bool
	TimetableLoading  bool
	TimetableError    string

	Report             model.ElectricityReport
	HasReport          bool
	ElectricityLoading bool
	ElectricityError   string

	Campus        model.CampusStatus
	CampusLoading bool
	CampusError   string
	AutoMode      bool
	// Version 是可选的展示文本，不参与任何业务状态判断。
	Version string
	// SettingsForm 是设置页当前生效的取值为基准；密码字段恒为空。
	SettingsForm desktop.SettingsForm
	SettingsHint string

	Refreshing bool
}

// MainWindowHandler 是 main 注入的命令出口。
//
// 回调在 UI 线程触发，只能快速修改状态或启动 goroutine；禁止在回调里同步
// 做网络 IO。后台结果必须先回到 UI 线程，再调用 UpdateMainWindow。
type MainWindowHandler struct {
	OnRefreshAll       func()
	OnTimetableWeek    func(int)
	OnTimetableRefresh func()
	OnElectricityMonth func(string)
	OnElectricityPoll  func()       // 电费页「立即查询电费」：走 PollOnce
	OnCampusAction     func(string) // refresh / login / logout / relogin
	// OnSaveSettings 在 UI 线程做校验与落盘，返回要显示在设置页的提示文本。
	OnSaveSettings func(desktop.SettingsResult) string
}

// 主窗口命令 ID。课表课程块与表格行由 GDI 命中测试，不占用命令 ID。
const (
	IDMainToday       = 3001
	IDMainTimetable   = 3002
	IDMainElectricity = 3003
	IDMainCampus      = 3004
	IDMainSettings    = 3005

	IDMainRefreshAll   = 3101
	IDMainTodayRefresh = 3102

	IDMainTimetablePrev    = 3201
	IDMainTimetableCurrent = 3202
	IDMainTimetableNext    = 3203
	IDMainTimetableRefresh = 3204

	IDMainElectricityMonth  = 3301
	IDMainElectricityQuery  = 3302
	IDMainElectricityScroll = 3303
	IDMainElectricityPoll   = 3304

	IDMainCampusRefresh = 3401
	IDMainCampusLogin   = 3402
	IDMainCampusLogout  = 3403
	IDMainCampusRelogin = 3404

	IDMainTimetableScroll = 3451
	IDMainSettingsScroll  = 3452
)

// MainCls 是主窗口类名；与托盘、设置窗口共用 RegisterClass 注册表。
const MainCls = "KemaolifeMainWindow"

// 对外只暴露需要由托盘和启动流程选择的页面。
const (
	MainPageToday    = mainPageToday
	MainPageSettings = mainPageSettings
)

const (
	mainPageToday = iota
	mainPageTimetable
	mainPageElectricity
	mainPageCampus
	mainPageSettings
)

const (
	wsThickFrame   = 0x00040000
	wsMaximizeBox  = 0x00010000
	wsClipChildren = 0x02000000

	wmCreate        = 0x0001
	wmDestroy       = WM_DESTROY
	wmSize          = 0x0005
	wmPaint         = 0x000F
	wmGetMinMaxInfo = 0x0024
	wmSetCursor     = 0x0020
	wmMouseWheel    = 0x020A
	wmVScroll       = 0x0115
	wmDPIChanged    = 0x02E0
	wmResizeToDpi   = WM_APP + 21

	sbsVert = 0x0001

	sbLineUp        = 0
	sbLineDown      = 1
	sbPageUp        = 2
	sbPageDown      = 3
	sbThumbPosition = 4
	sbThumbTrack    = 5
	sbTop           = 6
	sbBottom        = 7

	swShowNA = 8
)

type mainWindow struct {
	hwnd    uintptr
	icon    uintptr
	state   MainWindowState
	handler MainWindowHandler
	page    int
	ctl     map[int]uintptr

	// dpi 在窗口创建后固定一次，尺寸计算与控件摆放共用它。
	dpi             uint32
	timetableScroll int32
	electricScroll  int32
	settingsScroll  int32
	monthShown      string
	courseHits      []mainCourseHit
}

type mainPoint struct {
	X, Y int32
}

type mainMinMaxInfo struct {
	Reserved     mainPoint
	MaxSize      mainPoint
	MaxPosition  mainPoint
	MinTrackSize mainPoint
	MaxTrackSize mainPoint
}

var (
	mainMu  sync.Mutex
	mainWnd *mainWindow
)

// OpenMainWindow 创建（或复用）唯一主窗口并显示它。
// owner 保留给 main 组装语义；主窗口是独立顶层窗，不能挂在隐藏托盘窗下。
func OpenMainWindow(owner, icon uintptr, state MainWindowState, h MainWindowHandler) error {
	_ = owner
	mainMu.Lock()
	if w := mainWnd; w != nil && w.hwnd != 0 {
		w.handler = h
		w.state = cloneMainState(state)
		mainMu.Unlock()
		w.applyState()
		w.layout()
		// 窗口可能被最小化：SW_SHOW 不会恢复最小化窗口，先 Restore。
		Restore(w.hwnd)
		Show(w.hwnd, true)
		return nil
	}
	mainMu.Unlock()

	const style = WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX | wsMaximizeBox |
		wsThickFrame | wsClipChildren
	hwnd, err := CreateWindow(MainCls, "科贸生活", style, 0, 0, 0)
	if err != nil {
		return err
	}
	w := &mainWindow{
		hwnd: hwnd, icon: icon, state: cloneMainState(state), handler: h,
		page: mainPageToday, ctl: make(map[int]uintptr),
	}
	mainMu.Lock()
	mainWnd = w
	mainMu.Unlock()

	SetWindowIcon(hwnd, icon)
	if err := w.buildControls(); err != nil {
		return err
	}
	w.applyState()
	// 客户区尺寸必须是「逻辑 1120×720 按当前 DPI 放大后的物理像素」。
	// 只在创建时读一次 DPI 不够：窗口尚未经过 CenterOnScreen 落到目标显示器，
	// GetDpiForWindow 可能返回 96，于是 150% 缩放下会得到 1120 物理像素的窗口，
	// 而所有控件都按 144 DPI 布局，右列直接被裁到窗口外。
	// 这里先创建并落位，再按真实 DPI 校正一次尺寸。
	w.dpi = monitorDPIFor(hwnd)
	cw, ch := scaleByDPI(1120, w.dpi), scaleByDPI(720, w.dpi)
	ww, wh := AdjustToClient(hwnd, style, 0, cw, ch)
	CenterOnScreen(hwnd, ww, wh)
	w.layout()
	w.setPage(mainPageToday)
	Show(hwnd, true)
	// 显示之后再校正一次尺寸：此刻窗口真正落到显示器上，
	// GetDpiForWindow 才返回真实 DPI。创建阶段读到 96 是常见情况。
	procPostMessage.Call(hwnd, wmResizeToDpi, 0, 0)
	return nil
}

// resizeMainToDesign 按当前 DPI 把客户区校正为设计尺寸（1120×720 逻辑像素）。
//
// 只在窗口真正落到显示器之后调用：创建阶段 GetDpiForWindow 常返回 96，
// 按它算出的物理尺寸会让 150% 缩放下的窗口只有设计宽度的 2/3，
// 而控件按真实 DPI 布局，右列就会被裁掉。
func resizeMainToDesign(hwnd uintptr) {
	const style = WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX | wsMaximizeBox |
		wsThickFrame | wsClipChildren
	dpi := monitorDPIFor(hwnd)
	cw, ch := scaleByDPI(1120, dpi), scaleByDPI(720, dpi)
	w := currentMainWindow(hwnd)
	if w != nil {
		w.dpi = dpi
	}
	var r RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if r.Right == cw && r.Bottom == ch {
		return
	}
	ww, wh := AdjustToClient(hwnd, style, 0, cw, ch)
	CenterOnScreen(hwnd, ww, wh)
}

// OpenMainPage 切换已创建主窗口的页面，不新开窗口。
func OpenMainPage(page int) {
	mainMu.Lock()
	w := mainWnd
	mainMu.Unlock()
	if w != nil && w.hwnd != 0 {
		w.setPage(page)
	}
}

// ShowMainWindow 显示并置前已经创建的主窗口；尚未创建时为空操作。
func ShowMainWindow() {
	mainMu.Lock()
	w := mainWnd
	mainMu.Unlock()
	if w != nil && w.hwnd != 0 {
		Restore(w.hwnd)
		Show(w.hwnd, true)
	}
}

// UpdateMainWindow 原子替换展示快照并重绘。
// 必须在创建窗口的 UI 线程调用；本函数不会替后台 goroutine 做线程封送。
func UpdateMainWindow(state MainWindowState) {
	mainMu.Lock()
	w := mainWnd
	if w != nil {
		if w.state.Week != state.Week {
			w.timetableScroll = 0
		}
		w.state = cloneMainState(state)
	}
	mainMu.Unlock()
	if w == nil || w.hwnd == 0 {
		return
	}
	w.applyState()
	w.layout()
	procInvalidateRect.Call(w.hwnd, 0, 0)
}

func cloneMainState(s MainWindowState) MainWindowState {
	out := s
	out.Today.Entries = append([]model.Entry(nil), s.Today.Entries...)
	out.Days = make([]model.DaySchedule, len(s.Days))
	for i, day := range s.Days {
		out.Days[i] = day
		out.Days[i].Entries = append([]model.Entry(nil), day.Entries...)
	}
	out.Report.Usage.Days = append([]model.DayUsage(nil), s.Report.Usage.Days...)
	if s.Report.Status.DailyAvg != nil {
		v := *s.Report.Status.DailyAvg
		out.Report.Status.DailyAvg = &v
	}
	if s.Report.Status.EstimatedDays != nil {
		v := *s.Report.Status.EstimatedDays
		out.Report.Status.EstimatedDays = &v
	}
	return out
}

func currentMainWindow(hwnd uintptr) *mainWindow {
	mainMu.Lock()
	defer mainMu.Unlock()
	if mainWnd != nil && mainWnd.hwnd == hwnd {
		return mainWnd
	}
	return nil
}

// scale 把逻辑像素换算成物理像素。
//
// DPI 固定在窗口上（创建后由布局流程写入一次），尺寸计算与控件摆放
// 共用同一个值 —— 否则会出现「按 144 排版、按 96 开窗」的错位：
// 窗口只有 1120 物理像素宽，而控件按 144 DPI 铺到 1680，右列直接被裁掉。
func (w *mainWindow) scale(dp int32) int32 {
	if w.dpi == 0 {
		w.dpi = monitorDPIFor(w.hwnd)
	}
	return scaleByDPI(dp, w.dpi)
}

func (w *mainWindow) addButton(id int, text string, kind int) {
	h := CreateChild(classNameButton, text, BS_PUSHBUTTON|WS_TABSTOP, w.hwnd, id)
	w.ctl[id] = h
	registerMainButton(h, kind)
}

// 焦点控件的滚轮转发：WM_MOUSEWHEEL 发给焦点窗口而不是光标下的窗口，
// 点过按钮 / 输入框后焦点留在控件上，控件默认过程不处理滚轮，页面就滚不动。
// 这里给可聚焦子控件换一层转发壳：滚轮转给父窗口，其余消息原样放行。
// 主窗口只隐藏不销毁，控件与进程同寿，原过程表无需清理。
var (
	childProcMu        sync.Mutex
	childOrigProcs     = map[uintptr]uintptr{}
	childWheelCallback uintptr
)

const gwlpWndProc = ^uintptr(3) // GWLP_WNDPROC = -4

func subclassChildWheel(h uintptr) {
	if h == 0 {
		return
	}
	childProcMu.Lock()
	defer childProcMu.Unlock()
	if _, ok := childOrigProcs[h]; ok {
		return
	}
	if childWheelCallback == 0 {
		childWheelCallback = syscall.NewCallback(func(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
			if msg == wmMouseWheel {
				if parent, _, _ := procGetParent.Call(hwnd); parent != 0 {
					r, _, _ := procSendMessage.Call(parent, uintptr(msg), wParam, lParam)
					return r
				}
			}
			childProcMu.Lock()
			orig := childOrigProcs[hwnd]
			childProcMu.Unlock()
			if orig != 0 {
				r, _, _ := procCallWindowProc.Call(orig, hwnd, uintptr(msg), wParam, lParam)
				return r
			}
			r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
			return r
		})
	}
	orig, _, _ := procSetWindowLongPtr.Call(h, gwlpWndProc, childWheelCallback)
	childOrigProcs[h] = orig
}

func (w *mainWindow) buildControls() error {
	for _, b := range []struct {
		id   int
		text string
		kind int
	}{
		{IDMainToday, "今天", btnNav},
		{IDMainTimetable, "课表", btnNav},
		{IDMainElectricity, "电费", btnNav},
		{IDMainCampus, "校园网", btnNav},
		{IDMainSettings, "设置", btnNav},
		{IDMainRefreshAll, "刷新全部", btnPrimary},
		{IDMainTodayRefresh, "刷新课表", btnGhost},
		{IDMainTimetablePrev, "上一周", btnGhost},
		{IDMainTimetableCurrent, "本周", btnGhost},
		{IDMainTimetableNext, "下一周", btnGhost},
		{IDMainTimetableRefresh, "刷新", btnGhost},
		{IDMainElectricityQuery, "查询", btnGhost},
		{IDMainElectricityPoll, "立即查询电费", btnPrimary},
		{IDMainCampusRefresh, "刷新", btnGhost},
		{IDMainCampusLogin, "登录", btnPrimary},
		{IDMainCampusLogout, "下线", btnDanger},
		{IDMainCampusRelogin, "重登", btnGhost},
	} {
		w.addButton(b.id, b.text, b.kind)
	}
	month := CreateChild(classNameEdit, "", ES_AUTOHSCROLL|WS_BORDER|WS_TABSTOP,
		w.hwnd, IDMainElectricityMonth)
	w.ctl[IDMainElectricityMonth] = month
	if font := mainFont(w.hwnd, 14, 400); font != 0 {
		procSendMessage.Call(month, WM_SETFONT, font, 1)
	}
	registerCueEdit(month, "YYYY-MM")
	for _, id := range []int{IDMainTimetableScroll, IDMainElectricityScroll, IDMainSettingsScroll} {
		w.ctl[id] = CreateChild("SCROLLBAR", "", sbsVert, w.hwnd, id)
	}
	return w.buildSettingsControls()
}

func (w *mainWindow) contentRect() (left, top, right, bottom int32) {
	var r RECT
	procGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r)))
	return w.scale(216), w.scale(20), r.Right - w.scale(28), r.Bottom - w.scale(24)
}

func (w *mainWindow) move(id int, x, y, width, height int32) {
	if h := w.ctl[id]; h != 0 {
		Move(h, x, y, width, height)
	}
}

func (w *mainWindow) show(id int, visible bool) {
	if h := w.ctl[id]; h != 0 {
		// 子控件只切换可见性，不置前整个应用。
		cmd := uintptr(SW_HIDE)
		if visible {
			cmd = swShowNA
		}
		procShowWindow.Call(h, cmd)
	}
}

func (w *mainWindow) layout() {
	if w == nil || w.hwnd == 0 {
		return
	}
	s := w.scale
	left, _, right, bottom := w.contentRect()
	navX, navY := s(18), s(104)
	navW, navH, gap := s(166), s(42), s(9)
	for i, id := range []int{IDMainToday, IDMainTimetable, IDMainElectricity, IDMainCampus, IDMainSettings} {
		w.move(id, navX, navY+int32(i)*(navH+gap), navW, navH)
	}

	w.move(IDMainRefreshAll, right-s(104), s(26), s(104), s(34))
	w.move(IDMainTodayRefresh, right-s(216), s(26), s(104), s(34))

	y := s(96)
	x := left
	for _, item := range []struct {
		id    int
		width int32
	}{{IDMainTimetablePrev, 78}, {IDMainTimetableCurrent, 66},
		{IDMainTimetableNext, 78}, {IDMainTimetableRefresh, 72}} {
		w.move(item.id, x, y, s(item.width), s(32))
		x += s(item.width + 8)
	}

	w.move(IDMainElectricityMonth, left+s(78), y, s(142), s(30))
	w.move(IDMainElectricityQuery, left+s(230), y, s(88), s(32))
	w.move(IDMainElectricityPoll, left+s(326), y, s(128), s(32))

	buttonY := s(322)
	x = left
	for _, item := range []struct {
		id    int
		width int32
	}{{IDMainCampusRefresh, 82}, {IDMainCampusLogin, 82},
		{IDMainCampusLogout, 82}, {IDMainCampusRelogin, 82}} {
		w.move(item.id, x, buttonY, s(item.width), s(34))
		x += s(item.width + 10)
	}

	gridTop, gridBottom := s(140), bottom
	w.move(IDMainTimetableScroll, right-s(15), gridTop, s(15), gridBottom-gridTop)
	tableTop := s(356)
	w.move(IDMainElectricityScroll, right-s(15), tableTop, s(15), bottom-tableTop)
	if w.page == mainPageSettings {
		g := w.settingsGeometry()
		w.move(IDMainSettingsScroll, right-s(15), s(96), s(15), g.saveY-s(8)-s(96))
	}
	// 先更新滚动条范围（可能钳制滚动位置），再按钳制后的位置摆设置页控件。
	w.updateScrollbars()
	if w.page == mainPageSettings {
		w.layoutSettings()
	}
	// 移动控件后强制全部子控件按新矩形重绘：自绘按钮若沿用旧矩形画过
	// 一次，移动后会在让出的边缘留下色条残影。
	for _, h := range w.ctl {
		if h != 0 {
			procInvalidateRect.Call(h, 0, 1)
		}
	}
}

func (w *mainWindow) setPage(page int) {
	w.page = page
	navPages := map[int]int{IDMainToday: mainPageToday, IDMainTimetable: mainPageTimetable,
		IDMainElectricity: mainPageElectricity, IDMainCampus: mainPageCampus,
		IDMainSettings: mainPageSettings}
	for id, p := range navPages {
		w.show(id, true)
		setMainButtonActive(w.ctl[id], p == page)
	}
	for id := range w.ctl {
		if _, nav := navPages[id]; !nav {
			w.show(id, false)
		}
	}
	switch page {
	case mainPageToday:
		w.show(IDMainRefreshAll, true)
		w.show(IDMainTodayRefresh, true)
	case mainPageTimetable:
		for _, id := range []int{IDMainTimetablePrev, IDMainTimetableCurrent,
			IDMainTimetableNext, IDMainTimetableRefresh, IDMainTimetableScroll} {
			w.show(id, true)
		}
	case mainPageElectricity:
		for _, id := range []int{IDMainElectricityMonth, IDMainElectricityQuery,
			IDMainElectricityPoll, IDMainElectricityScroll} {
			w.show(id, true)
		}
	case mainPageCampus:
		for _, id := range []int{IDMainCampusRefresh, IDMainCampusLogin, IDMainCampusLogout,
			IDMainCampusRelogin} {
			w.show(id, true)
		}
	case mainPageSettings:
		w.show(IDMainSettingsScroll, true)
	}
	// 切页也要重摆控件：设置页滚动条只在 layout 里按当前页定位，
	// 不重摆会停留在创建时的 0 尺寸，表现为「设置页滚不动」。
	w.layout()
	w.applyState()
	w.updateScrollbars()
	if page == mainPageSettings {
		w.layoutSettings()
		w.fillSettingsForm(w.state.SettingsForm)
	}
	procInvalidateRect.Call(w.hwnd, 0, 1)
}

func (w *mainWindow) applyState() {
	if w == nil {
		return
	}
	Enable(w.ctl[IDMainRefreshAll], !w.state.Refreshing)
	if w.state.Refreshing {
		SetText(w.ctl[IDMainRefreshAll], "刷新中…")
	} else {
		SetText(w.ctl[IDMainRefreshAll], "刷新全部")
	}
	for _, id := range []int{IDMainTimetablePrev, IDMainTimetableCurrent,
		IDMainTimetableNext, IDMainTimetableRefresh, IDMainTodayRefresh} {
		Enable(w.ctl[id], !w.state.TimetableLoading && !w.state.Refreshing)
	}
	Enable(w.ctl[IDMainElectricityQuery], !w.state.ElectricityLoading && !w.state.Refreshing)
	Enable(w.ctl[IDMainElectricityPoll], !w.state.ElectricityLoading && !w.state.Refreshing)
	Enable(w.ctl[IDMainElectricityMonth], !w.state.ElectricityLoading)
	campusReady := !w.state.CampusLoading && !w.state.Refreshing
	for _, id := range []int{IDMainCampusRefresh, IDMainCampusRelogin} {
		Enable(w.ctl[id], campusReady)
	}
	// 自动模式只保留重登，与托盘菜单的操作口径一致；离线时两种模式都没有可执行动作。
	canAuthenticate := campusReady && w.state.Campus.State != model.CampusOffline
	Enable(w.ctl[IDMainCampusLogin], canAuthenticate && !w.state.AutoMode)
	Enable(w.ctl[IDMainCampusLogout], canAuthenticate && !w.state.AutoMode)
	Enable(w.ctl[IDMainCampusRelogin], canAuthenticate)
	month := mainDisplayMonth(w.state)
	if month != "" && month != w.monthShown {
		SetText(w.ctl[IDMainElectricityMonth], month)
		w.monthShown = month
	}
}

func mainDisplayMonth(s MainWindowState) string {
	if month := strings.TrimSpace(s.Report.Usage.YearMonth); month != "" {
		return month
	}
	if len(s.TodayDate) >= 7 {
		return s.TodayDate[:7]
	}
	if !s.Today.Date.IsZero() {
		v := s.Today.Date.String()
		if len(v) >= 7 {
			return v[:7]
		}
	}
	return ""
}

// MainProc 是主窗口过程。WM_CLOSE 只隐藏，绝不销毁窗口或结束消息循环。
func MainProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	w := currentMainWindow(hwnd)
	switch msg {
	case wmCreate, wmDestroy:
		if msg == wmDestroy {
			mainMu.Lock()
			if mainWnd != nil && mainWnd.hwnd == hwnd {
				mainWnd = nil
			}
			mainMu.Unlock()
		}
		return 0
	case WM_CLOSE:
		Show(hwnd, false)
		return 0
	case WM_QUERYENDSESSION:
		return 1 // 允许注销 / 关机；托盘图标由托盘窗口的 WM_ENDSESSION 摘除
	case wmGetMinMaxInfo:
		if w != nil && lParam != 0 {
			applyMainMinTrackSize(lParam, Scale(hwnd, 960), Scale(hwnd, 640))
			return 0
		}
	case wmResizeToDpi:
		resizeMainToDesign(hwnd)
		if w != nil {
			w.layout()
			procInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case wmSize:
		if w != nil {
			w.layout()
			procInvalidateRect.Call(hwnd, 0, 0)
			return 0
		}
	case wmDPIChanged:
		if w != nil && lParam != 0 {
			w.dpi = uint32(wParam & 0xFFFF)
			left, top, width, height := mainSuggestedWindowRect(lParam)
			procSetWindowPos.Call(hwnd, 0, uintptr(left), uintptr(top),
				uintptr(width), uintptr(height), SWP_NOZORDER|SWP_NOACTIVATE)
			activeNav := map[int]bool{IDMainToday: w.page == mainPageToday,
				IDMainTimetable:   w.page == mainPageTimetable,
				IDMainElectricity: w.page == mainPageElectricity,
				IDMainCampus:      w.page == mainPageCampus, IDMainSettings: w.page == mainPageSettings}
			for id, h := range w.ctl {
				weight := int32(400)
				if activeNav[id] {
					weight = 700
				}
				if font := mainFont(hwnd, 14, weight); font != 0 {
					procSendMessage.Call(h, WM_SETFONT, font, 1)
				}
			}
			w.layout()
			return 0
		}
	case WM_COMMAND:
		if w != nil {
			id := int(wParam & 0xFFFF)
			notify := uint32(wParam >> 16)
			if id == IDMainElectricityMonth {
				return 0
			}
			if notify == 0 { // BN_CLICKED / 键盘激活
				w.command(id)
			}
			return 0
		}
	case wmVScroll:
		if w != nil {
			switch uintptr(lParam) {
			case w.ctl[IDMainTimetableScroll]:
				w.scrollTimetable(uint32(wParam&0xFFFF), int32(uint16(wParam>>16)))
			case w.ctl[IDMainElectricityScroll]:
				w.scrollElectricity(uint32(wParam&0xFFFF), int32(uint16(wParam>>16)))
			case w.ctl[IDMainSettingsScroll]:
				w.scrollSettings(uint32(wParam&0xFFFF), int32(uint16(wParam>>16)))
			}
			return 0
		}
	case wmMouseWheel:
		if w != nil {
			delta := int32(int16(wParam >> 16))
			steps := -delta / 120
			if steps == 0 {
				if delta < 0 {
					steps = -1
				} else if delta > 0 {
					steps = 1
				}
			}
			switch w.page {
			case mainPageTimetable:
				w.setTimetableScroll(w.timetableScroll + steps*w.scale(56*2))
			case mainPageElectricity:
				w.setElectricScroll(w.electricScroll + steps*w.scale(28*3))
			case mainPageSettings:
				w.setSettingsScroll(w.settingsScroll + steps*w.scale(38))
			}
			return 0
		}
	case wmSetCursor:
		if w != nil && w.page == mainPageTimetable && uint32(lParam&0xFFFF) == 1 {
			p := cursorPos()
			procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&p)))
			if w.courseAt(p.X, p.Y) != nil {
				hand, _, _ := procLoadHandCursor.Call(0, idcHand)
				procSetCursor.Call(hand)
				return 1
			}
		}
	case WM_LBUTTONUP:
		if w != nil && w.page == mainPageTimetable {
			x := int32(int16(lParam & 0xFFFF))
			y := int32(int16((lParam >> 16) & 0xFFFF))
			if hit := w.courseAt(x, y); hit != nil {
				mainOwnedMessageBox(w.hwnd, "课程详情", mainDetailText(hit.day, hit.entry))
			}
			return 0
		}
	case wmPaint:
		if w != nil {
			w.paint()
			return 0
		}
	case WM_ERASEBKGND:
		return 1
	case WM_CTLCOLORSTATIC, WM_CTLCOLORBTN:
		// 设置页标签 / 分组标题 / 复选框：背景与自绘页面同色，否则留下灰块；
		// 文字颜色按控件角色区分（分组标题用主色，提示行用弱化色）。
		if brush := controlBackground(); brush != 0 {
			SetBkMode(wParam, transparentMode)
			color := mainTextColor
			if w != nil {
				switch wParam {
				case w.ctl[IDSetGroupFirst], w.ctl[IDSetGroupSecond],
					w.ctl[IDSetGroupThird], w.ctl[IDSetGroupThird+1]:
					color = mainAccent
				case w.ctl[IDSetHint], w.ctl[IDSetQuietSep]:
					color = mainMuted
				}
			}
			SetTextColor(wParam, color)
			return brush
		}
	}
	r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func (w *mainWindow) command(id int) {
	switch id {
	case IDMainToday, IDMainTimetable, IDMainElectricity, IDMainCampus, IDMainSettings:
		pages := map[int]int{IDMainToday: mainPageToday, IDMainTimetable: mainPageTimetable,
			IDMainElectricity: mainPageElectricity, IDMainCampus: mainPageCampus,
			IDMainSettings: mainPageSettings}
		w.setPage(pages[id])
	case IDMainRefreshAll:
		if w.handler.OnRefreshAll != nil {
			w.handler.OnRefreshAll()
		}
	case IDMainTimetablePrev:
		// 周次从 1 起：第 1 周再往前没有合法周次（0 是「当周」语义，
		// 传下去会跳回当前周而不是上一周）。
		if w.state.Week > 1 && w.handler.OnTimetableWeek != nil {
			w.handler.OnTimetableWeek(w.state.Week - 1)
		}
	case IDMainTimetableCurrent:
		if w.handler.OnTimetableWeek != nil {
			w.handler.OnTimetableWeek(0)
		}
	case IDMainTimetableNext:
		if w.handler.OnTimetableWeek != nil {
			w.handler.OnTimetableWeek(w.state.Week + 1)
		}
	case IDMainTimetableRefresh, IDMainTodayRefresh:
		if w.handler.OnTimetableRefresh != nil {
			w.handler.OnTimetableRefresh()
		}
	case IDMainElectricityQuery:
		if w.handler.OnElectricityMonth != nil {
			w.handler.OnElectricityMonth(strings.TrimSpace(Text(w.ctl[IDMainElectricityMonth])))
		}
	case IDMainElectricityPoll:
		if w.handler.OnElectricityPoll != nil {
			w.handler.OnElectricityPoll()
		}
	case IDMainCampusRefresh, IDMainCampusLogin, IDMainCampusLogout, IDMainCampusRelogin:
		actions := map[int]string{IDMainCampusRefresh: "refresh", IDMainCampusLogin: "login",
			IDMainCampusLogout: "logout", IDMainCampusRelogin: "relogin"}
		if w.handler.OnCampusAction != nil {
			w.handler.OnCampusAction(actions[id])
		}
	case IDSetSave, IDSetRevert:
		w.commandSettings(id)
	case IDSetAutoMode, IDSetAutoStart, IDSetStudentID, IDSetEduPass, IDSetCustomer,
		IDSetCampusUser, IDSetCampusPass, IDSetCampusFing, IDSetPollSecs, IDSetProbeSecs,
		IDSetBalance, IDSetDays, IDSetQuietFrom, IDSetQuietTo, IDSetHint:
		// 输入框 / 复选框自身不需要派生动作，吞掉通知避免 PlaySound。
		return
	}
}

func (w *mainWindow) scrollValue(code uint32, pos, current, page, maximum int32) int32 {
	switch code {
	case sbLineUp:
		current -= w.scale(40)
	case sbLineDown:
		current += w.scale(40)
	case sbPageUp:
		current -= page
	case sbPageDown:
		current += page
	case sbThumbPosition, sbThumbTrack:
		current = pos
	case sbTop:
		current = 0
	case sbBottom:
		current = maximum
	}
	if current < 0 {
		current = 0
	}
	if current > maximum {
		current = maximum
	}
	return current
}

func (w *mainWindow) timetableMetrics() (top, bodyTop, bottom, maxScroll int32) {
	_, _, _, bottom = w.contentRect()
	top = w.scale(140)
	bodyTop = top + w.scale(52)
	bodyHeight := w.scale(12 * 56)
	viewport := bottom - bodyTop
	if viewport < 1 {
		viewport = 1
	}
	maxScroll = bodyHeight - viewport
	if maxScroll < 0 {
		maxScroll = 0
	}
	return
}

func (w *mainWindow) electricityMetrics() (top, bodyTop, bottom, maxScroll int32) {
	_, _, _, bottom = w.contentRect()
	top = w.scale(356)
	bodyTop = top + w.scale(32)
	rows := len(w.state.Report.Usage.Days)
	bodyHeight := w.scale(int32(rows * 28))
	viewport := bottom - bodyTop
	if viewport < 1 {
		viewport = 1
	}
	maxScroll = bodyHeight - viewport
	if maxScroll < 0 {
		maxScroll = 0
	}
	return
}

func (w *mainWindow) setTimetableScroll(v int32) {
	_, _, _, maximum := w.timetableMetrics()
	if v < 0 {
		v = 0
	}
	if v > maximum {
		v = maximum
	}
	w.timetableScroll = v
	w.updateScrollbars()
	procInvalidateRect.Call(w.hwnd, 0, 0)
}

// settingsViewport 返回设置页表单可视区的上下边界（物理像素）。
// 表单从标题下方开始，到保存按钮上方结束；保存 / 提示固定在底部不滚动。
func (w *mainWindow) settingsViewport() (top, bottom int32) {
	g := w.settingsGeometry()
	return w.scale(96), g.saveY - w.scale(8)
}

// settingsMaxScroll 返回设置页可滚动的最大距离。
//
// 窗口最小 960×640 时表单底部会超出可视区，因此设置页也参与滚动；
// 只有内容真的放不下时才允许滚动，避免大窗口下滚出空白。
func (w *mainWindow) settingsMaxScroll() int32 {
	top, bottom := w.settingsViewport()
	g := w.settingsGeometry()
	_, maxScroll := settingsScrollMetrics(top, bottom, g.quietY+w.scale(30))
	return maxScroll
}

func (w *mainWindow) scrollSettings(code uint32, pos int32) {
	top, bottom := w.settingsViewport()
	viewport, maximum := settingsScrollMetrics(top, bottom, w.settingsGeometry().quietY+w.scale(30))
	w.setSettingsScroll(w.scrollValue(code, pos, w.settingsScroll, viewport, maximum))
}

func (w *mainWindow) setSettingsScroll(v int32) {
	maximum := w.settingsMaxScroll()
	if v < 0 {
		v = 0
	}
	if v > maximum {
		v = maximum
	}
	if w.settingsScroll == v {
		return
	}
	w.settingsScroll = v
	w.layoutSettings()
	procInvalidateRect.Call(w.hwnd, 0, 0)
}

func (w *mainWindow) setElectricScroll(v int32) {
	_, _, _, maximum := w.electricityMetrics()
	if v < 0 {
		v = 0
	}
	if v > maximum {
		v = maximum
	}
	w.electricScroll = v
	w.updateScrollbars()
	procInvalidateRect.Call(w.hwnd, 0, 0)
}

func (w *mainWindow) scrollTimetable(code uint32, pos int32) {
	_, bodyTop, bottom, maximum := w.timetableMetrics()
	w.setTimetableScroll(w.scrollValue(code, pos, w.timetableScroll, bottom-bodyTop, maximum))
}

func (w *mainWindow) scrollElectricity(code uint32, pos int32) {
	_, bodyTop, bottom, maximum := w.electricityMetrics()
	w.setElectricScroll(w.scrollValue(code, pos, w.electricScroll, bottom-bodyTop, maximum))
}

// mainStateLabel 复用 desktop 的三态文案，避免 UI 与托盘各写一套；
// 零值（尚未判态）是 UI 独有的第四种展示。
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

// mainTimeText 把 DTO 的 RFC3339 时刻转成本机可读格式（呈现层格式化，
// 不改 DTO）；解析失败时原样展示，不丢信息。
func mainTimeText(value string) string {
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err == nil {
		return t.Local().Format("2006-01-02 15:04")
	}
	return value
}

func mainDetailText(day model.DaySchedule, entry model.Entry) string {
	lines := []string{mainTextOr(entry.Name, "未命名课程"), ""}
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
	lines = append(lines, "教师："+mainTextOr(entry.Teacher, "—"),
		"地点："+mainTextOr(entry.Location, "—"))
	return strings.Join(lines, "\r\n")
}

func mainOwnedMessageBox(owner uintptr, title, text string) {
	pTitle, pText := strPtr(title), strPtr(text)
	procMessageBox.Call(owner, uintptr(unsafe.Pointer(pText)), uintptr(unsafe.Pointer(pTitle)),
		MB_OK|MB_ICONINFO)
}

func mainFormatFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
