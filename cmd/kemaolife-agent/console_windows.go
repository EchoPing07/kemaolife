//go:build windows

package main

import (
	"syscall"
)

// 本程序是托盘常驻程序，必须构建成 GUI 子系统（Windows PE subsystem = 2）：
//
//	go build -ldflags "-H=windowsgui" -o kemaolife-agent.exe ./cmd/kemaolife-agent
//
// 默认的 console 子系统（subsystem = 3）会让系统为它分配一个终端窗口，用户关掉
// 那个窗口就等于结束进程，「擦掉窗口仍在托盘运行」这一条验收直接不成立（docs/002 §5.2）。
//
// 构建参数是人写的，漏一次就退化成「带黑框的托盘程序」，所以启动时自己再兜一层：
// 若当前确实挂着一个控制台窗口，就主动 FreeConsole 摘掉。这样即使忘记加
// -H=windowsgui，程序也是无黑框的托盘常驻。

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procFreeConsole      = kernel32.NewProc("FreeConsole")
)

// detachConsole 无控制台时是空操作，可重复调用。
//
// FreeConsole 只解除本进程与控制台的关联，不结束进程；当本进程是控制台的唯一
// 使用者时，控制台窗口随之关闭，进程继续在消息循环里跑托盘。
func detachConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return // GUI 子系统：本来就没有控制台
	}
	procFreeConsole.Call()
}
