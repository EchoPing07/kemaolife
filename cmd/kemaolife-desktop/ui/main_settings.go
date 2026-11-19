//go:build windows

package ui

import (
	"strconv"
	"strings"
	"sync"

	"github.com/EchoPing07/kemaolife/internal/desktop"
)

// 设置页直接长在主窗口里，不再有独立设置窗口。
// 控件只做「收集字符串 + 显示提示」，校验与落盘仍由 desktop.ParseSettingsResult /
// SettingsStore 负责（AGENTS §3：UI 不重写业务规则）。

const (
	// IDSetPanel 是表单滚动容器：表单控件都是它的子窗口，
	// 滚动时被容器客户区裁剪，不会溢进标题区；保存 / 还原 / 提示不在此列。
	IDSetPanel       = 4000
	IDSetStudentID   = 4001
	IDSetEduPass     = 4002
	IDSetCustomer    = 4003
	IDSetCampusUser  = 4010
	IDSetCampusPass  = 4011
	IDSetCampusFing  = 4012
	IDSetAutoMode    = 4020
	IDSetAutoStart   = 4021
	IDSetWiredOnly   = 4022
	IDSetPollSecs    = 4030
	IDSetProbeSecs   = 4031
	IDSetBalance     = 4032
	IDSetDays        = 4033
	IDSetQuietFrom   = 4034
	IDSetQuietTo     = 4035
	IDSetQuietSep    = 4036
	IDSetSave        = 4101
	IDSetRevert      = 4102
	IDSetHint        = 4201
	IDSetGroupFirst  = 4301
	IDSetGroupSecond = 4302
	IDSetGroupThird  = 4303
	IDSetLabelFirst  = 4311
)

const settingsPanelClass = "KemaolifeSettingsPanel"

const wmCtlColorEdit = 0x0133

// settingsPanelProc 只做转发与底色：面板是裁剪容器，不含任何业务。
// 子控件的 WM_COMMAND / WM_CTLCOLOR* 必须转给主窗口，否则按钮点击与
// 背景同色处理都会失效；滚轮同样转给主窗口统一处理。
func settingsPanelProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_COMMAND, WM_CTLCOLORSTATIC, WM_CTLCOLORBTN, wmCtlColorEdit, wmMouseWheel:
		if parent, _, _ := procGetParent.Call(hwnd); parent != 0 {
			r, _, _ := procSendMessage.Call(parent, uintptr(msg), wParam, lParam)
			return r
		}
	case WM_ERASEBKGND:
		if brush := controlBackground(); brush != 0 {
			FillRect(wParam, hwnd, brush)
			return 1
		}
	}
	r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

// settingsScrollMetrics 是设置页滚动范围的纯函数版，便于单测。
// viewportTop/Bottom 是表单可视区，contentBottom 是表单内容底部。
func settingsScrollMetrics(viewportTop, viewportBottom, contentBottom int32) (viewport, maxScroll int32) {
	viewport = viewportBottom - viewportTop
	if viewport < 1 {
		viewport = 1
	}
	if contentBottom <= viewportTop+viewport {
		return viewport, 0
	}
	return viewport, contentBottom - viewportTop - viewport
}

var (
	ctlBrushOnce sync.Once
	ctlBrush     uintptr
)

// controlBackground 是与主窗口自绘背景一致的刷子。
// 原生 STATIC / BUTTON 默认按 COLOR_BTNFACE 擦背景，不换成同色刷子
// 会在浅灰页面上留下明显的灰块。
func controlBackground() uintptr {
	ctlBrushOnce.Do(func() {
		ctlBrush, _, _ = procCreateSolidBr.Call(mainBackground)
	})
	return ctlBrush
}

// settingsLayout 是设置页的物理像素坐标。
//
// 采用**固定单列 + 纵向滚动**：两列在窄窗口或高 DPI 下必然有一列被裁掉，
// 且需要同时保证「列宽 / 标签 / 输入框 / 间距」四者一致，实测极易出错。
// 单列没有横向分配问题，任何窗口宽度都完整可见，内容超长时用滚轮滚动。
type settingsLayout struct {
	leftX, colW  int32
	labelW, rowH int32
	topOverride  int32
	eduRows      [3]int32
	campusRows   [3]int32
	modeRows     [3]int32
	runRows      [5]int32
	eduTop       int32
	campusTop    int32
	modeTop      int32
	runTop       int32
	quietY       int32
	saveY        int32
	hintY        int32
	contentH     int32
}

func (w *mainWindow) settingsGeometry() settingsLayout {
	left, _, right, bottom := w.contentRect()
	g := settingsGeometryForWidth(left, right, w.scale)
	// 保存 / 提示固定在页面底部：表单可能需要滚动，但动作按钮必须始终可点。
	g.saveY = bottom - w.scale(76)
	g.hintY = g.saveY + w.scale(40)
	g.contentH = g.hintY + w.scale(20)
	return g
}

// settingsGeometryForWidth 是纯函数版布局计算，便于单测断言不越界。
func settingsGeometryForWidth(left, right int32, scale func(int32) int32) settingsLayout {
	var g settingsLayout
	g.leftX = left
	// 右侧让出滚动条槽：面板是不透明裁剪容器，盖住滚动条会让设置页滚不动。
	g.colW = right - left - scale(23)
	g.labelW = scale(118)
	g.rowH = scale(28)

	rowStep := scale(38)
	y := scale(112)

	group := func(name string) int32 {
		at := y
		y += scale(30)
		return at
	}
	rows := func(n int, out []int32) {
		for i := 0; i < n; i++ {
			out[i] = y
			y += rowStep
		}
		y += scale(14)
	}

	g.eduTop = group("edu")
	rows(3, g.eduRows[:])
	g.campusTop = group("campus")
	rows(3, g.campusRows[:])
	g.modeTop = group("mode")
	rows(3, g.modeRows[:])
	g.runTop = group("run")
	rows(4, g.runRows[:])
	g.quietY = y
	y += rowStep

	// 这里的 saveY 只是「表单内容结束高度」，真实位置由 settingsGeometry
	// 按窗口底部重新计算（滚动时按钮不能跟着滚走）。
	g.saveY = y + scale(10)
	g.hintY = g.saveY + scale(44)
	g.contentH = g.hintY + scale(40)
	return g
}

func (w *mainWindow) buildSettingsControls() error {
	ctl := w.ctl

	if err := RegisterClass(settingsPanelClass, settingsPanelProc, 0); err != nil {
		return err
	}
	panel := CreateChild(settingsPanelClass, "", wsClipChildren, w.hwnd, IDSetPanel)
	ctl[IDSetPanel] = panel

	label := func(id int, text string) uintptr {
		h := CreateChild(classNameStatic, text, SS_LEFT, panel, id)
		ctl[id] = h
		if f := mainFont(w.hwnd, 14, 400); f != 0 {
			procSendMessage.Call(h, WM_SETFONT, f, 1)
		}
		return h
	}
	group := func(id int, text string) {
		h := CreateChild(classNameStatic, text, SS_LEFT, panel, id)
		ctl[id] = h
		if f := mainFont(w.hwnd, 16, 700); f != 0 {
			procSendMessage.Call(h, WM_SETFONT, f, 1)
		}
	}
	field := func(id int, password bool) uintptr {
		st := uint32(ES_AUTOHSCROLL | WS_BORDER | WS_TABSTOP)
		if password {
			st |= ES_PASSWORD
		}
		e := CreateChild(classNameEdit, "", st, panel, id)
		ctl[id] = e
		if f := mainFont(w.hwnd, 14, 400); f != 0 {
			procSendMessage.Call(e, WM_SETFONT, f, 1)
		}
		return e
	}
	checkbox := func(id int, text string) {
		c := CreateChild(classNameButton, text, BS_AUTOCHECKBOX|WS_TABSTOP, panel, id)
		ctl[id] = c
		if f := mainFont(w.hwnd, 14, 400); f != 0 {
			procSendMessage.Call(c, WM_SETFONT, f, 1)
		}
		subclassChildWheel(c)
	}

	group(IDSetGroupFirst, "教务与电费")
	label(IDSetLabelFirst, "学号")
	field(IDSetStudentID, false)
	label(IDSetLabelFirst+1, "教务密码")
	field(IDSetEduPass, true)
	label(IDSetLabelFirst+2, "学校代码")
	field(IDSetCustomer, false)

	group(IDSetGroupSecond, "校园网认证（独立凭据）")
	label(IDSetLabelFirst+3, "账号")
	field(IDSetCampusUser, false)
	label(IDSetLabelFirst+4, "密码")
	field(IDSetCampusPass, true)
	label(IDSetLabelFirst+5, "指纹参数")
	field(IDSetCampusFing, false)

	group(IDSetGroupThird, "运行模式")
	checkbox(IDSetAutoMode, "未认证时自动登录（默认手动）")
	checkbox(IDSetWiredOnly, "自动认证仅限有线网络（避免与宿舍路由器冲突）")
	checkbox(IDSetAutoStart, "开机登录时自启动（只启动一次，不保活）")

	group(IDSetGroupThird+1, "轮询与提醒")
	label(IDSetLabelFirst+6, "电费间隔（秒）")
	field(IDSetPollSecs, false)
	label(IDSetLabelFirst+7, "探测间隔（秒）")
	field(IDSetProbeSecs, false)
	label(IDSetLabelFirst+8, "余额阈值（度）")
	field(IDSetBalance, false)
	label(IDSetLabelFirst+9, "可用天数（天）")
	field(IDSetDays, false)
	label(IDSetLabelFirst+10, "静默时段（时）")
	field(IDSetQuietFrom, false)
	sep := CreateChild(classNameStatic, "至", SS_LEFT, panel, IDSetQuietSep)
	ctl[IDSetQuietSep] = sep
	if f := mainFont(w.hwnd, 14, 400); f != 0 {
		procSendMessage.Call(sep, WM_SETFONT, f, 1)
	}
	field(IDSetQuietTo, false)

	registerCueEdit(ctl[IDSetStudentID], "用于电费与课表查询")
	registerCueEdit(ctl[IDSetEduPass], "留空保持不变")
	registerCueEdit(ctl[IDSetCustomer], "留空使用内置默认 3126")
	registerCueEdit(ctl[IDSetCampusUser], "校园网账号，独立于教务")
	registerCueEdit(ctl[IDSetCampusPass], "留空保持不变")
	registerCueEdit(ctl[IDSetCampusFing], "可留空")
	registerCueEdit(ctl[IDSetPollSecs], "默认 14400，最低 600")
	registerCueEdit(ctl[IDSetProbeSecs], "默认 120，最低 30")
	registerCueEdit(ctl[IDSetBalance], "默认 10")
	registerCueEdit(ctl[IDSetDays], "默认 3")
	registerCueEdit(ctl[IDSetQuietFrom], "23")
	registerCueEdit(ctl[IDSetQuietTo], "7")

	// 保存 / 还原 / 提示固定在页面底部，不放进滚动面板。
	hint := CreateChild(classNameStatic, "", SS_LEFT, w.hwnd, IDSetHint)
	ctl[IDSetHint] = hint
	if f := mainFont(w.hwnd, 13, 400); f != 0 {
		procSendMessage.Call(hint, WM_SETFONT, f, 1)
	}

	save := CreateChild(classNameButton, "保存", BS_PUSHBUTTON|WS_TABSTOP, w.hwnd, IDSetSave)
	ctl[IDSetSave] = save
	registerMainButton(save, btnPrimary)
	revert := CreateChild(classNameButton, "还原", BS_PUSHBUTTON|WS_TABSTOP, w.hwnd, IDSetRevert)
	ctl[IDSetRevert] = revert
	registerMainButton(revert, btnGhost)
	return nil
}

// layoutSettings 在页面可见时重新摆放控件；窗口缩放后也走这里。
// 表单控件摆在滚动面板内（坐标相对面板左上角），滚动只移动面板内容；
// 保存 / 还原 / 提示是主窗口直接子控件，固定在底部不随滚动。
func (w *mainWindow) layoutSettings() {
	if w == nil || len(w.ctl) == 0 {
		return
	}
	g := w.settingsGeometry()
	rowH := g.rowH
	fieldW := g.colW - g.labelW
	offset := w.settingsScroll
	top, bottom := w.settingsViewport()
	if bottom < top {
		bottom = top
	}
	Move(w.ctl[IDSetPanel], g.leftX, top, g.colW, bottom-top)

	place := func(id int, x, y, width, height int32) {
		if h := w.ctl[id]; h != 0 {
			Move(h, x-g.leftX, y-top-offset, width, height)
		}
	}
	// 标签与输入框之间留 8px 缝隙，避免边框压在文字上。
	labelAt := func(id int, y int32) {
		place(id, g.leftX, y+w.scale(3), g.labelW-w.scale(8), rowH)
	}
	fieldAt := func(id int, y int32) {
		// 右侧留 8px：输入框边框紧贴内容区右边缘会显得被裁。
		place(id, g.leftX+g.labelW, y, fieldW-w.scale(8), rowH)
	}

	place(IDSetGroupFirst, g.leftX, g.eduTop, g.colW, w.scale(26))
	labelAt(IDSetLabelFirst, g.eduRows[0])
	fieldAt(IDSetStudentID, g.eduRows[0])
	labelAt(IDSetLabelFirst+1, g.eduRows[1])
	fieldAt(IDSetEduPass, g.eduRows[1])
	labelAt(IDSetLabelFirst+2, g.eduRows[2])
	fieldAt(IDSetCustomer, g.eduRows[2])

	place(IDSetGroupSecond, g.leftX, g.campusTop, g.colW, w.scale(26))
	labelAt(IDSetLabelFirst+3, g.campusRows[0])
	fieldAt(IDSetCampusUser, g.campusRows[0])
	labelAt(IDSetLabelFirst+4, g.campusRows[1])
	fieldAt(IDSetCampusPass, g.campusRows[1])
	labelAt(IDSetLabelFirst+5, g.campusRows[2])
	fieldAt(IDSetCampusFing, g.campusRows[2])

	place(IDSetGroupThird, g.leftX, g.modeTop, g.colW, w.scale(26))
	place(IDSetAutoMode, g.leftX, g.modeRows[0], g.colW, rowH)
	place(IDSetWiredOnly, g.leftX, g.modeRows[1], g.colW, rowH)
	place(IDSetAutoStart, g.leftX, g.modeRows[2], g.colW, rowH)

	place(IDSetGroupThird+1, g.leftX, g.runTop, g.colW, w.scale(26))
	for i, id := range []int{IDSetPollSecs, IDSetProbeSecs, IDSetBalance, IDSetDays} {
		labelAt(IDSetLabelFirst+6+i, g.runRows[i])
		fieldAt(id, g.runRows[i])
	}
	// 静默时段：两个小输入框并排放在同一行，中间「至」分隔。
	labelAt(IDSetLabelFirst+10, g.quietY)
	place(IDSetQuietFrom, g.leftX+g.labelW, g.quietY, w.scale(58), rowH)
	place(IDSetQuietSep, g.leftX+g.labelW+w.scale(66), g.quietY+w.scale(3), w.scale(36), rowH)
	place(IDSetQuietTo, g.leftX+g.labelW+w.scale(110), g.quietY, w.scale(58), rowH)

	w.move(IDSetSave, g.leftX, g.saveY, w.scale(104), w.scale(32))
	w.move(IDSetRevert, g.leftX+w.scale(116), g.saveY, w.scale(88), w.scale(32))
	w.move(IDSetHint, g.leftX, g.hintY, g.colW, w.scale(40))

	w.showSettingsPage(true)
}

func (w *mainWindow) right() int32 {
	_, _, right, _ := w.contentRect()
	return right
}

func (w *mainWindow) showSettingsPage(visible bool) {
	for id := range w.ctl {
		if id >= IDSetPanel {
			w.show(id, visible)
		}
	}
}

// fillSettingsForm 写入表单；两个密码框一律留空，表示「不修改」。
func (w *mainWindow) fillSettingsForm(form desktop.SettingsForm) {
	set := func(id int, v string) {
		if h := w.ctl[id]; h != 0 {
			SetText(h, v)
		}
	}
	set(IDSetStudentID, form.StudentID)
	set(IDSetEduPass, "")
	set(IDSetCustomer, form.CustomerCode)
	set(IDSetCampusUser, form.CampusUser)
	set(IDSetCampusPass, "")
	set(IDSetCampusFing, form.CampusFingerprint)
	set(IDSetPollSecs, strconv.Itoa(form.PollSeconds))
	set(IDSetProbeSecs, strconv.Itoa(form.CampusProbeSecs))
	set(IDSetBalance, form.BalanceThreshold)
	set(IDSetDays, strconv.Itoa(form.DaysThreshold))
	set(IDSetQuietFrom, strconv.Itoa(form.QuietFrom))
	set(IDSetQuietTo, strconv.Itoa(form.QuietTo))
	if h := w.ctl[IDSetAutoMode]; h != 0 {
		SetChecked(h, form.AutoMode)
	}
	if h := w.ctl[IDSetWiredOnly]; h != 0 {
		SetChecked(h, form.WiredOnly)
	}
	if h := w.ctl[IDSetAutoStart]; h != 0 {
		SetChecked(h, form.AutoStart)
	}
	if h := w.ctl[IDSetHint]; h != 0 {
		SetText(h, "")
	}
}

func (w *mainWindow) collectSettings() desktop.SettingsResult {
	get := func(id int) string {
		if h := w.ctl[id]; h != 0 {
			return Text(h)
		}
		return ""
	}
	checked := func(id int) bool {
		if h := w.ctl[id]; h != 0 {
			return Checked(h)
		}
		return false
	}
	res := desktop.SettingsResult{
		StudentID:         strings.TrimSpace(get(IDSetStudentID)),
		EduPassword:       get(IDSetEduPass),
		CustomerCode:      strings.TrimSpace(get(IDSetCustomer)),
		CampusUser:        strings.TrimSpace(get(IDSetCampusUser)),
		CampusPassword:    get(IDSetCampusPass),
		CampusFingerprint: strings.TrimSpace(get(IDSetCampusFing)),
		AutoMode:          checked(IDSetAutoMode),
		WiredOnly:         checked(IDSetWiredOnly),
		AutoStart:         checked(IDSetAutoStart),
		PollSeconds:       strings.TrimSpace(get(IDSetPollSecs)),
		CampusProbeSecs:   strings.TrimSpace(get(IDSetProbeSecs)),
		BalanceThreshold:  strings.TrimSpace(get(IDSetBalance)),
		DaysThreshold:     strings.TrimSpace(get(IDSetDays)),
		QuietFrom:         strings.TrimSpace(get(IDSetQuietFrom)),
		QuietTo:           strings.TrimSpace(get(IDSetQuietTo)),
	}
	return res
}

// commandSettings 处理设置页按钮；其余控件（输入框、复选框）不需要派生动作。
func (w *mainWindow) commandSettings(id int) {
	switch id {
	case IDSetSave:
		if w.handler.OnSaveSettings == nil {
			return
		}
		if h := w.ctl[IDSetHint]; h != 0 {
			SetText(h, w.handler.OnSaveSettings(w.collectSettings()))
		}
	case IDSetRevert:
		w.fillSettingsForm(w.state.SettingsForm)
	}
}
