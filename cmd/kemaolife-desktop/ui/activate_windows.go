//go:build windows

package ui

import "unsafe"

// ShowExistingMainWindow 把已有实例的主窗口恢复到前台。
// 单实例互斥量只负责阻止第二个轮询循环；重复启动的交互语义由这里完成。
//
// 已有实例以 --background 驻留时主窗口尚未创建：改为通知托盘窗口，
// 由它的 UI 线程创建并显示主窗口（跨进程不能直接操作对方的窗口控件）。
func ShowExistingMainWindow() {
	findWindow := user32.NewProc("FindWindowW")
	hwnd, _, _ := findWindow.Call(uintptr(unsafe.Pointer(strPtr(MainCls))), 0)
	if hwnd != 0 {
		Restore(hwnd)
		procSetForeground.Call(hwnd)
		return
	}
	tray, _, _ := findWindow.Call(uintptr(unsafe.Pointer(strPtr(TrayClass))), 0)
	if tray != 0 {
		procPostMessage.Call(tray, WM_SHOW_MAIN, 0, 0)
	}
}
