//go:build windows

package ui

import "unsafe"

// WM_GETMINMAXINFO / WM_DPICHANGED 的 lParam 是 Win32 在同步窗口过程中
// 提供的原生结构地址。RtlMoveMemory 可直接读写这些地址，避免 Go 里把
// uintptr 强转 unsafe.Pointer；地址不会跨回调保存。

var procRtlMoveMemory = kernel32.NewProc("RtlMoveMemory")

func applyMainMinTrackSize(address uintptr, width, height int32) {
	if address == 0 {
		return
	}
	value := [2]int32{width, height}
	procRtlMoveMemory.Call(address+unsafe.Offsetof(mainMinMaxInfo{}.MinTrackSize),
		uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
}

func mainSuggestedWindowRect(address uintptr) (left, top, width, height int32) {
	if address == 0 {
		return
	}
	var r RECT
	procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&r)), address, unsafe.Sizeof(r))
	return r.Left, r.Top, r.Right - r.Left, r.Bottom - r.Top
}
