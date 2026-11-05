//go:build windows

// Package ui 是 kemaolife-agent 的原生 Win32 薄封装。
//
// 只用 Go 标准库 syscall 直调 user32 / gdi32 / comctl32，不引入 golang.org/x/sys，
// 也不使用任何浏览器内核（AGENTS §2.2 A1、§2 A5）。
//
// 这一层只做「窗口、控件、菜单、图标」，不含任何业务判断：
// 业务状态由 cmd/kemaolife-agent 传入，UI 只负责显示与回调。
package ui

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
)

var (
	procRegisterClassEx = user32.NewProc("RegisterClassExW")
	procCreateWindowEx  = user32.NewProc("CreateWindowExW")
	procDefWindowProc   = user32.NewProc("DefWindowProcW")
	procGetMessage      = user32.NewProc("GetMessageW")
	procTranslateMsg    = user32.NewProc("TranslateMessage")
	procDispatchMsg     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage = user32.NewProc("PostQuitMessage")
	procPostMessage     = user32.NewProc("PostMessageW")
	procDestroyWindow   = user32.NewProc("DestroyWindow")
	procShowWindow      = user32.NewProc("ShowWindow")
	procUpdateWindow    = user32.NewProc("UpdateWindow")
	procSetWindowText   = user32.NewProc("SetWindowTextW")
	procGetWindowText   = user32.NewProc("GetWindowTextW")
	procGetWindowTextLn = user32.NewProc("GetWindowTextLengthW")
	procSendMessage     = user32.NewProc("SendMessageW")
	procMessageBox      = user32.NewProc("MessageBoxW")
	procCreatePopupMenu = user32.NewProc("CreatePopupMenu")
	procAppendMenu      = user32.NewProc("AppendMenuW")
	procTrackPopupMenu  = user32.NewProc("TrackPopupMenu")
	procDestroyMenu     = user32.NewProc("DestroyMenu")
	procSetForeground   = user32.NewProc("SetForegroundWindow")
	procGetCursorPos    = user32.NewProc("GetCursorPos")
	procLoadCursor      = user32.NewProc("LoadCursorW")
	procLoadIconChecked = user32.NewProc("LoadIconW")
	procSetFocus        = user32.NewProc("SetFocus")
	procEnableWindow    = user32.NewProc("EnableWindow")
	procSetWindowPos    = user32.NewProc("SetWindowPos")
	procAdjustRect      = user32.NewProc("AdjustWindowRectEx")
	procGetSystemMetr   = user32.NewProc("GetSystemMetrics")
	procGetDpiForWindow = user32.NewProc("GetDpiForWindow")
	procSetDpiAwareness = user32.NewProc("SetProcessDpiAwarenessContext")
	procCreateIconRes   = user32.NewProc("CreateIconFromResourceEx")
	procSetTimer        = user32.NewProc("SetTimer")
	procKillTimer       = user32.NewProc("KillTimer")
	procGetDC           = user32.NewProc("GetDC")
	procReleaseDC       = user32.NewProc("ReleaseDC")
	procInvalidateRect  = user32.NewProc("InvalidateRect")

	procCreateMutex   = kernel32.NewProc("CreateMutexW")
	procGetModuleHndl = kernel32.NewProc("GetModuleHandleW")
	procGetLastError  = kernel32.NewProc("GetLastError")

	procCreateSolidBr = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject  = gdi32.NewProc("DeleteObject")
)

// 窗口消息（只列用到的）。
const (
	WM_DESTROY       = 0x0002
	WM_CLOSE         = 0x0010
	WM_PAINT         = 0x000F
	WM_COMMAND       = 0x0111
	WM_TIMER         = 0x0113
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONDBLCLK = 0x0206
	WM_RBUTTONUP     = 0x0205
	WM_APP           = 0x8000
)

// 控件样式与通用常量。
const (
	WS_OVERLAPPED       = 0x00000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_MINIMIZEBOX      = 0x00020000
	WS_CHILD            = 0x40000000
	WS_VISIBLE          = 0x10000000
	WS_BORDER           = 0x00800000
	WS_TABSTOP          = 0x00010000
	WS_VSCROLL          = 0x00200000
	WS_EX_DLGMODALFRAME = 0x00000001
	WS_EX_TOPMOST       = 0x00000008

	SW_HIDE = 0
	SW_SHOW = 5

	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010

	MF_STRING    = 0x0000
	MF_POPUP     = 0x0010
	MF_SEPARATOR = 0x0800
	MF_GRAYED    = 0x0001
	MF_CHECKED   = 0x0008
	MF_UNCHECKED = 0x0000

	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100

	MB_OK            = 0x00000000
	MB_ICONERROR     = 0x00000010
	MB_ICONINFO      = 0x00000040
	MB_ICONWARNING   = 0x00000030
	MB_SETFOREGROUND = 0x00010000

	IDC_ARROW       = 32512
	IDI_APPLICATION = 32512

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	BN_CLICKED = 0

	COLOR_WINDOW  = 5
	COLOR_BTNFACE = 15

	// 控件类名。
	classNameStatic = "STATIC"
	classNameEdit   = "EDIT"
	classNameButton = "BUTTON"

	// BS_* 按钮样式。
	BS_PUSHBUTTON   = 0x00000000
	BS_AUTOCHECKBOX = 0x00000003
	BS_GROUPBOX     = 0x00000007
	ES_AUTOHSCROLL  = 0x0080
	ES_PASSWORD     = 0x0020
	SS_LEFT         = 0x0000

	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
	dpiPerMonitorAwareV2 = ^uintptr(2) // (HANDLE)-4
)

// POINT 对应 Win32 POINT。
type POINT struct {
	X, Y int32
}

// MSG 对应 Win32 MSG。
type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

// WNDCLASSEX 对应 Win32 WNDCLASSEXW。
type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

// EnablePerMonitorDPI 让窗口在高分屏上按物理像素绘制，避免位图缩放导致文字发虚。
func EnablePerMonitorDPI() {
	procSetDpiAwareness.Call(dpiPerMonitorAwareV2)
}

// Scale 把设计用的逻辑像素换算成当前 DPI 下的物理像素。
func Scale(hwnd uintptr, dp int32) int32 {
	dpi := uint32(96)
	if hwnd != 0 {
		if d, _, err := procGetDpiForWindow.Call(hwnd); err == nil && d >= 72 {
			dpi = uint32(d)
		}
	}
	return int32(float64(dp) * float64(dpi) / 96.0)
}

// ScreenSize 返回主屏分辨率（用于托盘菜单定位的兜底）。
func ScreenSize() (int32, int32) {
	cx, _, _ := procGetSystemMetr.Call(SM_CXSCREEN)
	cy, _, _ := procGetSystemMetr.Call(SM_CYSCREEN)
	return int32(cx), int32(cy)
}

func cursorPos() POINT {
	var p POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

// instance 返回当前进程模块句柄。
func instance() uintptr {
	h, _, _ := procGetModuleHndl.Call(0)
	return h
}

var (
	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
)

// CreateMutex 创建（或打开）命名互斥量，返回句柄与「是否已存在」。
//
// alreadyExists 为 true 说明另一个实例正在运行；调用方必须立刻 CloseHandle
// 并退出，不要继续初始化（否则会出现两个托盘图标、双倍轮询请求）。
func CreateMutex(name string) (handle uintptr, alreadyExists bool, err error) {
	const errorAlreadyExists = 183
	h, _, e := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(strPtr(name))))
	if h == 0 {
		return 0, false, e
	}
	if errno, ok := e.(syscall.Errno); ok && errno == errorAlreadyExists {
		return h, true, nil
	}
	return h, false, nil
}

// CloseHandle 释放内核句柄。
func CloseHandle(h uintptr) {
	if h != 0 {
		procCloseHandle.Call(h)
	}
}

// AcquireSingleInstance 用命名互斥量保证单实例。
//
// 双开会出两个托盘图标、各自轮询，既重复请求又会「关掉一个还剩一个」。
// 互斥量创建失败不阻断启动：单实例是体验优化，不是功能前提。
func AcquireSingleInstance() (release func(), ok bool) {
	h, exists, err := CreateMutex("KemaolifeAgent.SingleInstance")
	if err != nil {
		return func() {}, true
	}
	if exists {
		CloseHandle(h)
		return func() {}, false
	}
	return func() { CloseHandle(h) }, true
}
