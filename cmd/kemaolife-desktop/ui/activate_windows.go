//go:build windows

package ui

import (
	"time"
	"unsafe"
)

// ShowExistingMainWindow 把已有实例的主窗口恢复到前台。
// 单实例互斥量只负责阻止第二个轮询循环；重复启动的交互语义由这里完成。
//
// 已有实例以 --background 驻留时主窗口尚未创建：改为通知托盘窗口，
// 由它的 UI 线程创建并显示主窗口（跨进程不能直接操作对方的窗口控件）。
func ShowExistingMainWindow() {
	findWindow := user32.NewProc("FindWindowW")
	// 主窗口由框架持有；统一经托盘窗口回到它的 UI 线程唤起。
	// 首个实例可能仍在初始化：第二个实例只在这里短暂等待并转发一次，
	// 不启动自己的轮询，也不要求首个实例的主窗口已经创建。
	for range 100 {
		tray, _, _ := findWindow.Call(uintptr(unsafe.Pointer(strPtr(TrayClass))), 0)
		if tray != 0 {
			procPostMessage.Call(tray, WM_SHOW_MAIN, 0, 0)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
