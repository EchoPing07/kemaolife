//go:build windows

package ui

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

// WndProc 是窗口过程签名。
type WndProc func(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr

// 窗口类名（注册一次，多窗口复用同一过程）。
const (
	TrayClass   = "KemaolifeTrayWindow"
	SettingsCls = "KemaolifeSettingsWindow"
)

// 自定义消息：后台 goroutine 通过它把状态变化送回 UI 线程。
//
// Win32 控件只能在创建它的线程里操作，后台轮询线程必须 PostMessage 回来，
// 不能直接改控件文本 —— 否则表现为「界面卡死」或「文本不刷新」。
const (
	WM_STATUS_UPDATE = WM_APP + 1 // 状态文本变了
	WM_QUERY_DONE    = WM_APP + 3 // 一次轮询结束（触发菜单重建）
	WM_OPEN_SETTINGS = WM_APP + 4 // 请求打开设置窗口
)

// wmTrayCallback 是 Shell_NotifyIcon 回调消息号（与通知实现里的常量一致）。
const wmTrayCallback = WM_APP + 100

var (
	mu        sync.Mutex
	wndProcs  = map[string]WndProc{}
	classAtom = map[string]uint16{}
	// classNames 持有类名的 UTF-16 缓冲：窗口类注册后系统仍会按名字查找
	// （FindWindow / CreateWindow），让 Go 回收这块内存会让类名变成乱码。
	classNames     []*uint16
	classCallbacks []uintptr
)

// RegisterClass 注册窗口类（同名重复注册时复用）。
func RegisterClass(name string, proc WndProc, hIcon uintptr) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := classAtom[name]; ok {
		return nil
	}

	// 用 SetWindowLongPtr 保存 Go 闭包不可行（Go 指针不能进 Win32 内存），
	// 所以按类名分发：全局过程查表跳转到具体 Go 函数。
	cb := syscall.NewCallback(func(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
		mu.Lock()
		p, ok := wndProcs[name]
		mu.Unlock()
		if ok && p != nil {
			return p(hwnd, msg, wParam, lParam)
		}
		r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	})

	hCursor, _, _ := procLoadCursor.Call(0, IDC_ARROW)
	cn := strPtr(name) // 必须活到 RegisterClassEx 之后：类名指针被内核读取
	classNames = append(classNames, cn)
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   cb,
		HInstance:     instance(),
		HIcon:         hIcon,
		HIconSm:       hIcon,
		HCursor:       hCursor,
		HbrBackground: COLOR_BTNFACE + 1,
		LpszClassName: cn,
	}
	if r, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("注册窗口类 %s 失败: %w", name, err)
	}
	runtime.KeepAlive(cn)

	wndProcs[name] = proc
	classAtom[name] = 1
	classCallbacks = append(classCallbacks, cb)
	return nil
}

// CreateWindow 创建窗口，className 必须是已注册的类。
func CreateWindow(className, title string, style, exStyle uint32, parent, menu uintptr) (uintptr, error) {
	cn := strPtr(className)
	tt := strPtr(title)
	h, _, err := procCreateWindowEx.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(cn)),
		uintptr(unsafe.Pointer(tt)),
		uintptr(style),
		0, 0, 0, 0,
		parent, menu, instance(), 0,
	)
	runtime.KeepAlive(cn)
	runtime.KeepAlive(tt)
	if h == 0 {
		return 0, fmt.Errorf("创建窗口 %s 失败: %w", className, err)
	}
	return h, nil
}

// CreateChild 创建一个子控件（STATIC / EDIT / BUTTON）。
func CreateChild(class, text string, style uint32, parent uintptr, id int) uintptr {
	c := strPtr(class)
	t := strPtr(text)
	h, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(c)),
		uintptr(unsafe.Pointer(t)),
		uintptr(style|WS_CHILD|WS_VISIBLE),
		0, 0, 0, 0,
		parent, uintptr(id), instance(), 0,
	)
	// 引用保持到调用返回之后：Go 的 GC 不知道 Win32 读过这块内存，
	// 提前回收会让控件文本变成乱码甚至让 CreateWindowEx 崩溃。
	runtime.KeepAlive(c)
	runtime.KeepAlive(t)
	return h
}

// Move 设置控件 / 窗口在父窗口内的位置与大小（物理像素）。
func Move(hwnd uintptr, x, y, w, h int32) {
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		SWP_NOZORDER|SWP_NOACTIVATE)
}

// SetText 设置控件文本。
func SetText(hwnd uintptr, s string) {
	p := strPtr(s)
	procSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
}

// Text 读取控件文本。
func Text(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLn.Call(hwnd)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

// SetChecked 设置复选框勾选状态（BS_AUTOCHECKBOX 由用户点击切换，这里用于初始化）。
func SetChecked(hwnd uintptr, on bool) {
	// BM_SETCHECK = 0x00F1
	var v uintptr
	if on {
		v = 1 // BST_CHECKED
	}
	procSendMessage.Call(hwnd, 0x00F1, v, 0)
}

// Checked 读取复选框状态。
func Checked(hwnd uintptr) bool {
	// BM_GETCHECK = 0x00F0
	r, _, _ := procSendMessage.Call(hwnd, 0x00F0, 0, 0)
	return r == 1
}

// Enable 启用 / 禁用控件（按模式切换按钮时用，避免「点了没反应」）。
func Enable(hwnd uintptr, on bool) {
	var v uintptr
	if on {
		v = 1
	}
	procEnableWindow.Call(hwnd, v)
}

// Show 显示 / 隐藏窗口。
func Show(hwnd uintptr, show bool) {
	if !show {
		procShowWindow.Call(hwnd, SW_HIDE)
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)
	// UpdateWindow 强制立即重绘：只 ShowWindow 时窗口会等到下一条消息才画，
	// 在高负载机器上表现为「窗口出来了但是空的」。
	procUpdateWindow.Call(hwnd)
	procSetForeground.Call(hwnd)
	procInvalidateRect.Call(hwnd, 0, 1) // TRUE = 擦除背景后重绘
}

// pendingTexts 是「后台线程 → UI 线程」的文本投递箱。
//
// 为什么不用 PostMessage + LocalAlloc 传字符串指针：
// 那样要把 Win32 裸地址转回 Go 指针（unsafe.Pointer(uintptr)），
// 既会被 vet 判为误用，也真的依赖「Win32 内存生命周期」这种隐性约定。
// 用索引投递则只在 UI 线程读文本，跨线程只传一个整数。
var (
	pendingMu    sync.Mutex
	pendingTexts = map[uintptr]string{}
	pendingSeq   uintptr
)

// PostStatus 从任意 goroutine 把状态文本送回 UI 线程。
func PostStatus(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	pendingMu.Lock()
	pendingSeq++
	id := pendingSeq
	pendingTexts[id] = text
	pendingMu.Unlock()

	procPostMessage.Call(hwnd, WM_STATUS_UPDATE, id, 0)
}

// TakeText 取出 PostStatus 投递的文本（由 UI 线程在 WM_STATUS_UPDATE 里调用）。
func TakeText(wParam uintptr) string {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	text := pendingTexts[wParam]
	delete(pendingTexts, wParam)
	return text
}

// PostTrigger 从任意 goroutine 请求 UI 线程执行一个动作（例如重建菜单）。
func PostTrigger(hwnd uintptr, msg uint32) {
	if hwnd == 0 {
		return
	}
	procPostMessage.Call(hwnd, uintptr(msg), 0, 0)
}

// Run 进入消息循环，阻塞当前 OS 线程直到 PostQuitMessage。
//
// 注意：必须在主 goroutine 上调用并 LockOSThread —— Win32 消息队列属于线程，
// 换线程 GetMessage 就收不到自己的窗口消息。
func Run() int {
	var msg MSG
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			return int(r)
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMsg.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// Quit 结束消息循环。
func Quit() { procPostQuitMessage.Call(0) }

// MessageBox 弹一个原生消息框（用于启动失败这类无法走托盘提示的场景）。
func MessageBox(title, text string, flags uint32) {
	procMessageBox.Call(0,
		uintptr(unsafe.Pointer(strPtr(text))),
		uintptr(unsafe.Pointer(strPtr(title))),
		uintptr(flags))
}

func strPtr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		z := uint16(0)
		return &z
	}
	return p
}

// RECT 对应 Win32 RECT。
type RECT struct {
	Left, Top, Right, Bottom int32
}

// AdjustToClient 由客户区尺寸算出窗口外框尺寸，保证控件不被标题栏挤掉。
func AdjustToClient(hwnd uintptr, style, exStyle uint32, cw, ch int32) (int32, int32) {
	r := RECT{0, 0, cw, ch}
	procAdjustRect.Call(uintptr(unsafe.Pointer(&r)), uintptr(style), 0, uintptr(exStyle))
	return r.Right - r.Left, r.Bottom - r.Top
}

// CenterOnScreen 把窗口放到主屏中央。
func CenterOnScreen(hwnd uintptr, w, h int32) {
	// 先把窗口（含非客户区）尺寸设好，再算居中位置。
	screenW, screenH := ScreenSize()
	x := (screenW - w) / 2
	y := (screenH - h) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		SWP_NOZORDER|SWP_NOACTIVATE)
}

// SetWindowIcon 给窗口设置图标（任务栏与 Alt+Tab 都用它）。
func SetWindowIcon(hwnd, icon uintptr) {
	// WM_SETICON = 0x0080；ICON_SMALL=0，ICON_BIG=1
	procSendMessage.Call(hwnd, 0x0080, 0, icon)
	procSendMessage.Call(hwnd, 0x0080, 1, icon)
}

// Focus 把输入焦点交给控件（设置窗口打开时定位到第一个输入框）。
func Focus(hwnd uintptr) { procSetFocus.Call(hwnd) }

// 绘制相关常量与辅助。
const (
	WM_ERASEBKGND     = 0x0014
	WM_CTLCOLORBTN    = 0x0135
	WM_CTLCOLORSTATIC = 0x0138
	transparentMode   = 1 // TRANSPARENT
)

var (
	procSetTextColor = gdi32.NewProc("SetTextColor")
	procSetBkMode    = gdi32.NewProc("SetBkMode")
	brushOnce        sync.Once
	dialogBrushCache uintptr
)

// dialogBrush 返回与窗口类背景一致的刷子（惰性创建，进程内复用）。
func dialogBrush() uintptr {
	brushOnce.Do(func() {
		// COLOR_BTNFACE 的系统颜色值：RGB(240,240,240)。
		dialogBrushCache, _, _ = procCreateSolidBr.Call(rgb(240, 240, 240))
	})
	return dialogBrushCache
}

// rgb 把 RGB 分量压成 Win32 的 COLORREF（0x00BBGGRR）。
func rgb(r, g, b uint32) uintptr { return uintptr(r | g<<8 | b<<16) }

// SetTextColor 设置控件文字颜色。
func SetTextColor(hdc, color uintptr) { procSetTextColor.Call(hdc, color) }

// SetBkMode 设置背景模式（transparent 让文字不描背景框）。
func SetBkMode(hdc uintptr, mode int32) { procSetBkMode.Call(hdc, uintptr(mode)) }

// FillRect 用刷子填充客户区（WM_ERASEBKGND 里自绘底色）。
func FillRect(hdc, hwnd, brush uintptr) {
	var r RECT
	var info winRect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&info)))
	r.Left, r.Top = 0, 0
	r.Right, r.Bottom = info.Right, info.Bottom
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brush)
}

type winRect struct {
	Left, Top, Right, Bottom int32
}

var (
	procGetClientRect = user32.NewProc("GetClientRect")
	procFillRect      = user32.NewProc("FillRect")
)

var (
	procCreateFont = gdi32.NewProc("CreateFontW")
	procSendDlgMsg = user32.NewProc("SendMessageW")
)

var (
	fontMu    sync.Mutex
	fontCache = map[int32]uintptr{}
)

// SetFontBold 给控件设置粗体 / 常规字体（分节标题用）。
//
// 字体句柄缓存在进程级 map 里：每次调用都 CreateFontW 会在反复开关
// 设置窗口后耗尽 GDI 对象（Windows 有 10k 上限）。
func SetFontBold(hwnd uintptr, bold bool) {
	weight := int32(400)
	if bold {
		weight = 700
	}
	if font := cachedFont(weight); font != 0 {
		procSendDlgMsg.Call(hwnd, WM_SETFONT, font, 1)
	}
}

// WM_SETFONT 消息号。
const WM_SETFONT = 0x0030

// cachedFont 返回指定字重的「微软雅黑 9pt」字体句柄（进程内复用）。
//
// 用雅黑而不是系统默认宋体：中文界面在 125%/150% 缩放下雅黑更清晰。
func cachedFont(weight int32) uintptr {
	fontMu.Lock()
	defer fontMu.Unlock()
	if f, ok := fontCache[weight]; ok {
		return f
	}
	face := strPtr("Microsoft YaHei UI")
	// CreateFontW(h, w, esc, orient, weight, italic, underline, strikeout,
	//             charset, outprec, clipprec, quality, pitch, face)
	h, _, _ := procCreateFont.Call(
		uintptr(16), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1,       // DEFAULT_CHARSET
		0, 0, 5, // CLEARTYPE_QUALITY
		0, uintptr(unsafe.Pointer(face)),
	)
	if h == 0 {
		return 0
	}
	fontCache[weight] = h
	return h
}
