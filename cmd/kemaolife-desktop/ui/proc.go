//go:build windows

package ui

import (
	"sync"
)

// 窗口过程注册：托盘窗口与设置窗口各一个，二者共享同一进程的消息循环。

// 托盘命令回调：由 cmd 层注入，UI 层只负责把命令 ID 翻译成动作。
type TrayHandler struct {
	// BuildMenu 每次右键都调用：状态与模式可能在上一次点击后改变。
	BuildMenu func() []MenuItem
	// OnCommand 收到菜单命令（0 表示用户点了菜单外部取消）。
	OnCommand func(id int)
	// OnTrayClick 收到托盘图标点击（左键 / 右键之外的场景，如双击）。
	OnTrayClick func()
	// OnStatusText 收到后台 PostStatus 的文本，更新托盘提示。
	OnStatusText func(text string)
	// OnQueryDone 后台一轮轮询结束：刷新提示并重建菜单状态。
	OnQueryDone func()
}

var (
	trayMu      sync.Mutex
	trayHandler TrayHandler
	trayHwnd    uintptr
)

// BindTray 绑定托盘窗口与命令回调。
func BindTray(hwnd uintptr, h TrayHandler) {
	trayMu.Lock()
	defer trayMu.Unlock()
	trayHwnd = hwnd
	trayHandler = h
}

// handleTrayMessage 是托盘窗口的消息处理核心。
func handleTrayMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	switch msg {
	case WM_STATUS_UPDATE:
		text := TakeText(wParam)
		trayMu.Lock()
		cb := trayHandler.OnStatusText
		trayMu.Unlock()
		if cb != nil {
			cb(text)
		}
		return 0, true

	case WM_QUERY_DONE:
		trayMu.Lock()
		cb := trayHandler.OnQueryDone
		trayMu.Unlock()
		if cb != nil {
			cb()
		}
		return 0, true

	case TrayCallbackMessage:
		// lParam 低字是鼠标消息：右键弹菜单，左键单击交给上层（打开设置）。
		//
		// 左键单击唤起主窗口「今天」：双击在 Win11 的溢出面板里
		// 经常被面板自身吞掉，单击更可靠。
		// 复选：双击图标没反应的问题在 Win11 溢出面板里常见，两个消息都认。
		switch uint32(lParam) & 0xFFFF {
		case WM_RBUTTONUP:
			showTrayMenu(hwnd)
			return 0, true
		case WM_LBUTTONUP, WM_LBUTTONDBLCLK:
			// 实测托盘左键下发的是 WM_LBUTTONDBLCLK（0x203），部分环境是
			// WM_LBUTTONUP（0x202），两个都要认，否则「双击图标没反应」。
			trayMu.Lock()
			cb := trayHandler.OnTrayClick
			trayMu.Unlock()
			if cb != nil {
				cb()
			}
			return 0, true
		}
		return 0, true

	case WM_COMMAND:
		// 托盘菜单的命令也会以 WM_COMMAND 形式到达（TrackPopupMenu 返回 ID 时
		// 走的是返回值路径，这里兜住键盘触发的情况）。
		id := int(wParam & 0xFFFF)
		trayMu.Lock()
		cb := trayHandler.OnCommand
		trayMu.Unlock()
		if cb != nil {
			cb(id)
		}
		return 0, true

	case WM_DESTROY:
		Quit()
		return 0, true
	}
	return 0, false
}

// showTrayMenu 构建并弹出菜单，然后把用户选择交给上层。
func showTrayMenu(hwnd uintptr) {
	trayMu.Lock()
	build := trayHandler.BuildMenu
	cb := trayHandler.OnCommand
	trayMu.Unlock()

	if build == nil {
		return
	}
	items := build()
	menu, err := BuildMenu(items)
	if err != nil {
		return
	}
	defer menu.Destroy()

	id := menu.PopupAtCursor(hwnd)
	if id == 0 {
		return // 用户取消
	}
	if cb != nil {
		cb(id)
	}
}

// TrayProc 是托盘窗口过程。
func TrayProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if r, ok := handleTrayMessage(hwnd, msg, wParam, lParam); ok {
		return r
	}
	r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}
