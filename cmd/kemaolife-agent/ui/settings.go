package ui

import (
	"strconv"
	"sync"

	"github.com/EchoPing07/kemaolife/internal/agent"
)

// 设置窗口：纯 Win32 控件，不做自定义绘制。
//
// 布局用「逻辑像素 + DPI 缩放」：在 150% / 200% 缩放的屏幕上写死像素会把文字裁掉。
// 所有行距、边距都在 design* 常量里集中定义，改一处即可整体调松紧。

// 控件 ID（窗口内唯一）。
const (
	IDStudentID = 2001
	IDCustomer  = 2002
	IDCampusU   = 2003
	IDCampusP   = 2004
	IDFinger    = 2005
	IDAutoMode  = 2006
	IDAutoStart = 2007
	IDPollSecs  = 2008
	IDProbeSecs = 2013
	IDBalance   = 2009
	IDDays      = 2010
	IDQuietFrom = 2011
	IDQuietTo   = 2012
	IDSave      = 2101
	IDCancel    = 2102
	IDHint      = 2201
)

// 设计尺寸（逻辑像素，96 DPI 基准）。
const (
	designClientW = 640
	designClientH = 726
	designMargin  = 16
	designLabelW  = 100
	designFieldW  = 236
	designNoteW   = 150
	designRowH    = 22
	designGroupH  = 26 // 分组框标题占用的高度
)

var (
	settingsMu   sync.Mutex
	settingsWnd  uintptr
	settingsCtl  map[int]uintptr
	settingsSave func(agent.SettingsResult) error
)

// OpenSettings 显示设置窗口（单实例：再次打开只把已有窗口提到前台）。
func OpenSettings(owner, icon uintptr, form agent.SettingsForm, onSave func(agent.SettingsResult) error) {
	// 锁只在读写共享状态时持有，绝不跨越 Win32 调用。
	//
	// 为什么：创建控件会同步派发 WM_COMMAND 到窗口过程，那里会再调
	// handleSettingsCommand 去锁同一把锁 —— Go 的 sync.Mutex 不可重入，
	// 一旦跨越 Win32 调用就会自死锁（表现为「点设置没反应」）。
	settingsMu.Lock()
	existing := settingsWnd
	settingsSave = onSave
	if existing != 0 {
		ctl := settingsCtl
		settingsMu.Unlock()
		fillSettings(ctl, form)
		SetSettingsHint("")
		Show(existing, true)
		return
	}
	settingsMu.Unlock()

	const style = WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX
	// 父窗口传 0：设置窗口是独立顶层窗口，挂在托盘隐藏窗下会跟着一起被隐藏。
	hwnd, err := CreateWindow(SettingsCls, "科贸生活 · 设置", style, 0, 0, 0)
	if err != nil {
		MessageBox("科贸生活", "打开设置窗口失败："+err.Error(), MB_OK|MB_ICONERROR)
		return
	}

	// 先登记句柄：控件创建期间可能派发 WM_COMMAND，处理函数需要它。
	settingsMu.Lock()
	settingsWnd = hwnd
	settingsMu.Unlock()

	ctl := buildSettings(hwnd, form)

	settingsMu.Lock()
	settingsCtl = ctl
	settingsMu.Unlock()

	SetWindowIcon(hwnd, icon)

	// 先定尺寸再显示：控件坐标按客户区算，窗口没尺寸时会被裁成 0。
	cw, ch := Scale(hwnd, designClientW), Scale(hwnd, designClientH)
	w, h := AdjustToClient(hwnd, style, 0, cw, ch)
	CenterOnScreen(hwnd, w, h)
	Show(hwnd, true)
	if btn := ctl[IDSave]; btn != 0 {
		Focus(btn)
	}
}

// SetSettingsHint 在窗口底部显示一行提示（保存结果 / 钳制说明）。
func SetSettingsHint(text string) {
	settingsMu.Lock()
	h := settingsCtl[IDHint]
	settingsMu.Unlock()
	if h != 0 {
		SetText(h, text)
	}
}

// buildSettings 创建所有控件并返回句柄表。
func buildSettings(hwnd uintptr, form agent.SettingsForm) map[int]uintptr {
	ctl := make(map[int]uintptr)
	s := func(dp int32) int32 { return Scale(hwnd, dp) }

	var (
		margin = s(designMargin)
		labelW = s(designLabelW)
		fieldX = margin + labelW
		fieldW = s(designFieldW)
		rowH   = s(designRowH)
		y      = margin
	)

	// 分节标题用 STATIC + 加粗，不用 GROUPBOX。
	//
	// 为什么放弃 GROUPBOX：它内部会再画一层背景，与透明 STATIC 叠加后
	// 文字出现重影（实测在 Win11 上明显）；而且它还会把标题压到边框上。
	// 纯文字标题视觉更干净，也不需要算内边距。
	group := func(text string, gy int32) {
		g := CreateChild(classNameStatic, text, SS_LEFT, hwnd, 0)
		SetFontBold(g, true)
		Move(g, margin, gy, s(designClientW)-2*margin, rowH)
	}
	// 行内标签：垂直居中到输入框。
	label := func(text string, ly int32) {
		l := CreateChild(classNameStatic, text, 0, hwnd, 0)
		Move(l, margin+s(10), ly+s(3), labelW, rowH)
	}
	// 右侧说明文字：单独给宽度，避免被分组框裁掉。
	note := func(text string, ny int32) {
		n := CreateChild(classNameStatic, text, 0, hwnd, 0)
		Move(n, fieldX+fieldW+s(10), ny+s(3), s(designNoteW), rowH)
	}
	field := func(id int, value string, ly int32, width int32, password bool) uintptr {
		st := uint32(ES_AUTOHSCROLL | WS_BORDER | WS_TABSTOP)
		if password {
			st |= ES_PASSWORD
		}
		e := CreateChild(classNameEdit, value, st, hwnd, id)
		Move(e, fieldX, ly, width, rowH)
		ctl[id] = e
		return e
	}
	checkbox := func(id int, text string, on bool, cy int32) {
		c := CreateChild(classNameButton, text, BS_AUTOCHECKBOX|WS_TABSTOP, hwnd, id)
		Move(c, margin+s(16), cy, s(designClientW)-2*margin-s(32), rowH)
		SetChecked(c, on)
		ctl[id] = c
	}

	// —— 电费 ——
	group("电费查询（电费接口无密码）", y)
	label("学号", y+s(30))
	field(IDStudentID, form.StudentID, y+s(30), fieldW, false)
	label("学校代码", y+s(62))
	field(IDCustomer, form.CustomerCode, y+s(62), fieldW, false)
	note("留空用默认", y+s(62))

	y += s(96)
	// —— 校园网 ——
	group("校园网认证（独立凭据，不复用教务密码）", y)
	label("账号", y+s(30))
	field(IDCampusU, form.CampusUser, y+s(30), fieldW, false)
	label("密码", y+s(62))
	field(IDCampusP, form.CampusPassword, y+s(62), fieldW, true)
	label("指纹参数", y+s(94))
	field(IDFinger, form.CampusFingerprint, y+s(94), fieldW, false)
	note("可留空", y+s(94))

	y += s(128)
	// —— 运行模式 ——
	group("运行模式", y)
	checkbox(IDAutoMode, "自动模式：未认证时自动登录（默认手动）", form.AutoMode, y+s(30))
	checkbox(IDAutoStart, "开机登录时自启动（只启动一次，不保活）", form.AutoStart, y+s(56))

	y += s(88)
	// —— 轮询与提醒 ——
	group("轮询与提醒", y)
	label("电费间隔", y+s(30))
	field(IDPollSecs, strconv.Itoa(form.PollSeconds), y+s(30), s(80), false)
	note("秒，默认 14400", y+s(30))

	// 校园网判态比电费轻得多（1 个探测请求），所以单独一个更短的周期。
	label("探测间隔", y+s(62))
	field(IDProbeSecs, strconv.Itoa(form.CampusProbeSecs), y+s(62), s(80), false)
	note("秒，默认 120", y+s(62))

	label("余额阈值", y+s(94))
	field(IDBalance, form.BalanceThreshold, y+s(94), s(80), false)
	note("度", y+s(94))

	label("可用天数", y+s(126))
	field(IDDays, strconv.Itoa(form.DaysThreshold), y+s(126), s(80), false)
	note("天", y+s(126))

	label("静默时段", y+s(158))
	field(IDQuietFrom, strconv.Itoa(form.QuietFrom), y+s(158), s(44), false)
	dash := CreateChild(classNameStatic, "—", 0, hwnd, 0)
	Move(dash, fieldX+s(50), y+s(162), s(14), rowH)
	// 第二个输入框要显式指定 x：field() 固定从 fieldX 起，直接用会重叠。
	to := CreateChild(classNameEdit, strconv.Itoa(form.QuietTo),
		ES_AUTOHSCROLL|WS_BORDER|WS_TABSTOP, hwnd, IDQuietTo)
	Move(to, fieldX+s(68), y+s(158), s(44), rowH)
	ctl[IDQuietTo] = to
	note("时，相同=不静默", y+s(158))

	y += s(196)
	hint := CreateChild(classNameStatic, "", 0, hwnd, IDHint)
	Move(hint, margin, y, s(designClientW)-2*margin, s(34))
	ctl[IDHint] = hint

	y += s(34)
	save := CreateChild(classNameButton, "保存", BS_PUSHBUTTON|WS_TABSTOP, hwnd, IDSave)
	Move(save, margin+s(280), y, s(100), s(28))
	ctl[IDSave] = save
	cancel := CreateChild(classNameButton, "取消", BS_PUSHBUTTON|WS_TABSTOP, hwnd, IDCancel)
	Move(cancel, margin+s(390), y, s(100), s(28))
	ctl[IDCancel] = cancel

	return ctl
}

// fillSettings 把最新配置写回已存在的控件（重复打开设置窗口时用）。
func fillSettings(ctl map[int]uintptr, form agent.SettingsForm) {
	set := func(id int, v string) {
		if h := ctl[id]; h != 0 {
			SetText(h, v)
		}
	}
	set(IDStudentID, form.StudentID)
	set(IDCustomer, form.CustomerCode)
	set(IDCampusU, form.CampusUser)
	set(IDCampusP, form.CampusPassword)
	set(IDFinger, form.CampusFingerprint)
	set(IDPollSecs, strconv.Itoa(form.PollSeconds))
	set(IDProbeSecs, strconv.Itoa(form.CampusProbeSecs))
	set(IDBalance, form.BalanceThreshold)
	set(IDDays, strconv.Itoa(form.DaysThreshold))
	set(IDQuietFrom, strconv.Itoa(form.QuietFrom))
	set(IDQuietTo, strconv.Itoa(form.QuietTo))
	if h := ctl[IDAutoMode]; h != 0 {
		SetChecked(h, form.AutoMode)
	}
	if h := ctl[IDAutoStart]; h != 0 {
		SetChecked(h, form.AutoStart)
	}
}

// collectSettings 读取控件当前值。
func collectSettings(ctl map[int]uintptr) agent.SettingsResult {
	get := func(id int) string {
		if h := ctl[id]; h != 0 {
			return Text(h)
		}
		return ""
	}
	checked := func(id int) bool {
		if h := ctl[id]; h != 0 {
			return Checked(h)
		}
		return false
	}
	return agent.SettingsResult{
		StudentID:         get(IDStudentID),
		CustomerCode:      get(IDCustomer),
		CampusUser:        get(IDCampusU),
		CampusPassword:    get(IDCampusP),
		CampusFingerprint: get(IDFinger),
		AutoMode:          checked(IDAutoMode),
		AutoStart:         checked(IDAutoStart),
		PollSeconds:       get(IDPollSecs),
		CampusProbeSecs:   get(IDProbeSecs),
		BalanceThreshold:  get(IDBalance),
		DaysThreshold:     get(IDDays),
		QuietFrom:         get(IDQuietFrom),
		QuietTo:           get(IDQuietTo),
	}
}

// handleSettingsCommand 处理设置窗口里的按钮点击，返回是否已消费该命令。
//
// 注意：函数内所有 Win32 调用都在**无锁**状态下进行（见 OpenSettings 的说明）。
func handleSettingsCommand(id int) bool {
	settingsMu.Lock()
	hint := settingsCtl[IDHint]
	saveFn := settingsSave
	ctl := settingsCtl
	win := settingsWnd
	settingsMu.Unlock()

	switch id {
	case IDSave:
		if saveFn == nil {
			if hint != 0 {
				SetText(hint, "内部错误：未绑定保存处理函数")
			}
			return true
		}
		if err := saveFn(collectSettings(ctl)); err != nil {
			// 校验失败留在窗口里，让用户改，不要关窗丢掉已填内容。
			if hint != 0 {
				SetText(hint, "保存失败："+err.Error())
			}
			return true
		}
		if win != 0 {
			Show(win, false)
		}
		return true
	case IDCancel:
		if win != 0 {
			Show(win, false)
		}
		return true
	}
	return false
}

// closeSettings 在窗口销毁时清理句柄（下次打开重新建控件）。
func closeSettings() {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	settingsWnd = 0
	settingsCtl = nil
}
