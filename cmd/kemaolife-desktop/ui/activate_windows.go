//go:build windows

package ui

import "unsafe"

// ShowExistingMainWindow 把已有实例的主窗口恢复到前台。
// 单实例互斥量只负责阻止第二个轮询循环；重复启动的交互语义由这里完成。
func ShowExistingMainWindow() {
	findWindow := user32.NewProc("FindWindowW")
	hwnd, _, _ := findWindow.Call(uintptr(unsafe.Pointer(strPtr("KemaolifeMainWindow"))), 0)
	if hwnd == 0 {
		return
	}
	const swRestore = 9
	procShowWindow.Call(hwnd, swRestore)
	procSetForeground.Call(hwnd)
}
